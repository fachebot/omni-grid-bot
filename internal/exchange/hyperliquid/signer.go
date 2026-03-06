package hyperliquid

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
)

type Signer struct {
	privateKey *ecdsa.PrivateKey
	publicKey  string
	isMainnet  bool
	vaultAddr  string
}

func NewSigner(privateKeyHex string, isMainnet bool) (*Signer, error) {
	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	publicKey := crypto.PubkeyToAddress(privateKey.PublicKey)

	return &Signer{
		privateKey: privateKey,
		publicKey:  publicKey.Hex(),
		isMainnet:  isMainnet,
	}, nil
}

func (s *Signer) GetAddress() string {
	return s.publicKey
}

func (s *Signer) IsMainnet() bool {
	return s.isMainnet
}

func (s *Signer) ChainID() int {
	if s.isMainnet {
		return MainnetChainID
	}
	return TestnetChainID
}

func (s *Signer) SignOrder(action map[string]interface{}, nonce int64) (string, error) {
	actionWire, err := actionToWire(action)
	if err != nil {
		return "", err
	}

	hash, err := actionHash(actionWire, s.vaultAddr, nonce)
	if err != nil {
		return "", err
	}

	phantomAgent := constructPhantomAgent(hash, s.isMainnet)
	signature, err := s.signTypedData(phantomAgent)
	if err != nil {
		return "", err
	}

	return signature, nil
}

func (s *Signer) SignCancel(action map[string]interface{}, nonce int64) (string, error) {
	return s.SignOrder(action, nonce)
}

func (s *Signer) SignAction(action map[string]interface{}, nonce int64) (string, error) {
	return s.SignOrder(action, nonce)
}

func (s *Signer) signTypedData(data map[string]interface{}) (string, error) {
	domainSeparator := crypto.Keccak256([]byte(fmt.Sprintf(
		"EIP712Domain(string name,string version,uint256 chainId,address verifyingContract)",
	)))
	domainSeparator = crypto.Keccak256([]byte(fmt.Sprintf(
		"%s%s%s%s%s",
		domainSeparator,
		crypto.Keccak256([]byte("HyperliquidSignTransaction")),
		crypto.Keccak256([]byte("1")),
		crypto.Keccak256([]byte{byte(s.ChainID())}),
		crypto.Keccak256([]byte("0x0000000000000000000000000000000000000000")),
	)))

	messageHash := crypto.Keccak256([]byte(fmt.Sprintf(
		"Agent(string source,bytes32 connectionId)",
	)))

	source, _ := data["source"].(string)
	connectionId, _ := data["connectionId"].(string)
	messageHash = crypto.Keccak256([]byte(fmt.Sprintf(
		"%s%s%s",
		messageHash,
		crypto.Keccak256([]byte(source)),
		crypto.Keccak256([]byte(connectionId)),
	)))

	typedDataHash := crypto.Keccak256([]byte(fmt.Sprintf(
		"\x19\x01%s%s%s",
		domainSeparator,
		messageHash,
		crypto.Keccak256([]byte("")),
	)))

	sig, err := crypto.Sign(typedDataHash, s.privateKey)
	if err != nil {
		return "", err
	}

	return hexutil.Encode(sig), nil
}

func actionToWire(action map[string]interface{}) (map[string]interface{}, error) {
	actionType, ok := action["type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid action type")
	}

	wire := map[string]interface{}{
		"type": actionType,
	}

	switch actionType {
	case "order":
		if orders, ok := action["orders"].([]interface{}); ok {
			orderWires := make([]map[string]interface{}, len(orders))
			for i, o := range orders {
				order := o.(map[string]interface{})
				orderWire := map[string]interface{}{
					"a": order["a"],
					"b": order["b"],
					"p": order["p"],
					"s": order["s"],
					"r": order["r"],
					"t": order["t"],
				}
				if cloid, ok := order["c"]; ok && cloid != nil {
					orderWire["c"] = cloid
				}
				orderWires[i] = orderWire
			}
			wire["orders"] = orderWires
		}
		if grouping, ok := action["grouping"].(string); ok {
			wire["grouping"] = grouping
		}

	case "cancel":
		if cancels, ok := action["cancels"].([]interface{}); ok {
			cancelWires := make([]map[string]interface{}, len(cancels))
			for i, c := range cancels {
				cancel := c.(map[string]interface{})
				cancelWires[i] = map[string]interface{}{
					"a": cancel["a"],
					"o": cancel["o"],
				}
			}
			wire["cancels"] = cancelWires
		}

	case "updateLeverage":
		wire["asset"] = action["asset"]
		wire["isCross"] = action["isCross"]
		wire["leverage"] = action["leverage"]
	}

	return wire, nil
}

func actionHash(action map[string]interface{}, vaultAddress string, nonce int64) ([]byte, error) {
	actionType, ok := action["type"].(string)
	if !ok {
		return nil, fmt.Errorf("invalid action type")
	}

	var data []byte

	switch actionType {
	case "order":
		data = []byte{0x00}
		if orders, ok := action["orders"].([]interface{}); ok {
			data = append(data, varint(len(orders))...)
			for _, o := range orders {
				order := o.(map[string]interface{})
				if asset, ok := order["a"].(int); ok {
					data = append(data, varint(asset)...)
				}
				if isBuy, ok := order["b"].(bool); ok {
					if isBuy {
						data = append(data, 0x01)
					} else {
						data = append(data, 0x00)
					}
				}
				if px, ok := order["p"].(string); ok {
					data = append(data, []byte(px)...)
				}
				if sz, ok := order["s"].(string); ok {
					data = append(data, []byte(sz)...)
				}
			}
		}

	case "cancel":
		data = []byte{0x01}
		if cancels, ok := action["cancels"].([]interface{}); ok {
			data = append(data, varint(len(cancels))...)
			for _, c := range cancels {
				cancel := c.(map[string]interface{})
				if asset, ok := cancel["a"].(int); ok {
					data = append(data, varint(asset)...)
				}
				if oid, ok := cancel["o"].(int64); ok {
					data = append(data, varint(int(oid))...)
				}
			}
		}

	case "updateLeverage":
		data = []byte{0x02}
		if asset, ok := action["asset"].(int); ok {
			data = append(data, varint(asset)...)
		}
		if isCross, ok := action["isCross"].(bool); ok {
			if isCross {
				data = append(data, 0x01)
			} else {
				data = append(data, 0x00)
			}
		}
		if leverage, ok := action["leverage"].(int); ok {
			data = append(data, varint(leverage)...)
		}
	}

	nonceBytes := big.NewInt(nonce).Bytes()
	data = append(data, nonceBytes...)

	if vaultAddress == "" {
		data = append(data, 0x00)
	} else {
		data = append(data, 0x01)
		addrBytes := commonHexToBytes(vaultAddress)
		data = append(data, addrBytes...)
	}

	hash := crypto.Keccak256(data)
	return hash, nil
}

func constructPhantomAgent(hash []byte, isMainnet bool) map[string]interface{} {
	source := "a"
	if !isMainnet {
		source = "b"
	}
	return map[string]interface{}{
		"source":       source,
		"connectionId": hexutil.Encode(hash),
	}
}

func commonHexToBytes(hexAddr string) []byte {
	addr := hexAddr
	if len(addr) > 2 && addr[:2] == "0x" {
		addr = addr[2:]
	}
	bytes, _ := hex.DecodeString(addr)
	return bytes
}

func varint(n int) []byte {
	if n == 0 {
		return []byte{0x00}
	}
	var result []byte
	for n > 0 {
		b := byte(n & 0x7f)
		n >>= 7
		if n > 0 {
			b |= 0x80
		}
		result = append(result, b)
	}
	return result
}
