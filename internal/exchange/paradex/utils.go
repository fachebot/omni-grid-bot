package paradex

import (
	"encoding/json"
	"fmt"
	"math/big"
	"strings"
	"time"

	starkcurve "github.com/consensys/gnark-crypto/ecc/stark-curve"
	"github.com/consensys/gnark-crypto/ecc/stark-curve/ecdsa"
	"github.com/consensys/gnark-crypto/ecc/stark-curve/fr"
	"github.com/dontpanicdao/caigo"
	"github.com/dontpanicdao/caigo/types"
	"github.com/fachebot/omni-grid-bot/internal/ent/order"
)

func ConvertOrderStatus(ord *Order) order.Status {
	if ord.CancelReason != "" {
		return order.StatusCanceled
	}

	switch ord.Status {
	case OrderStatusNew, OrderStatusUntriggered:
		return order.StatusPending
	case OrderStatusOpen:
		return order.StatusOpen
	case OrderStatusClosed:
		return order.StatusFilled
	default:
		return order.StatusPending
	}
}

func GetSignatureStr(r, s *big.Int) (string, error) {
	signature := []string{r.String(), s.String()}
	signatureByte, err := json.Marshal(signature)
	return string(signatureByte), err
}

func GetEcdsaPrivateKey(pk string) *ecdsa.PrivateKey {
	privateKey := types.StrToFelt(pk).Big()

	// Generate public key
	_, g := starkcurve.Generators()
	ecdsaPublicKey := new(ecdsa.PublicKey)
	ecdsaPublicKey.A.ScalarMultiplication(&g, privateKey)

	// Generate private key
	pkBytes := privateKey.FillBytes(make([]byte, fr.Bytes))
	buf := append(ecdsaPublicKey.Bytes(), pkBytes...)
	ecdsaPrivateKey := new(ecdsa.PrivateKey)
	ecdsaPrivateKey.SetBytes(buf)
	return ecdsaPrivateKey
}

func GnarkSign(messageHash *big.Int, privateKey string) (r, s *big.Int, err error) {
	ecdsaPrivateKey := GetEcdsaPrivateKey(privateKey)
	sigBin, err := ecdsaPrivateKey.Sign(messageHash.Bytes(), nil)
	if err != nil {
		return nil, nil, err
	}
	r = new(big.Int).SetBytes(sigBin[:fr.Bytes])
	s = new(big.Int).SetBytes(sigBin[fr.Bytes:])
	return r, s, nil
}

func ComputeAddress(config SystemConfigRes, publicKey string) (string, error) {
	publicKeyBN := types.HexToBN(publicKey)
	if publicKeyBN == nil {
		return "", fmt.Errorf("invalid public key")
	}

	paraclearAccountHashBN := types.HexToBN(config.ParaclearAccountHash)
	if paraclearAccountHashBN == nil {
		return "", fmt.Errorf("invalid paraclear_account_hash")
	}
	paraclearAccountProxyHashBN := types.HexToBN(config.ParaclearAccountProxyHash)
	if paraclearAccountProxyHashBN == nil {
		return "", fmt.Errorf("invalid paraclear_account_proxy_hash")
	}

	zero := big.NewInt(0)
	initializeBN := types.GetSelectorFromName("initialize")

	contractAddressPrefix := types.StrToFelt("STARKNET_CONTRACT_ADDRESS").Big()

	constructorCalldata := []*big.Int{
		paraclearAccountHashBN,
		initializeBN,
		big.NewInt(2),
		publicKeyBN,
		zero,
	}
	constructorCalldataHash, _ := caigo.Curve.ComputeHashOnElements(constructorCalldata)

	address := []*big.Int{
		contractAddressPrefix,
		zero,        // deployer address
		publicKeyBN, // salt
		paraclearAccountProxyHashBN,
		constructorCalldataHash,
	}

	addressHash, err := caigo.Curve.ComputeHashOnElements(address)
	if err != nil {
		return "", err
	}

	return types.BigToHex(addressHash), nil
}

func ParseUsdPerpMarket(market string) (string, error) {
	parts := strings.Split(market, "-")
	if len(parts) < 3 {
		return "", fmt.Errorf("invalid market format: %s", market)
	}

	baseCurrency := parts[0]
	quoteCurrency := parts[1]
	contractType := parts[2]

	if strings.ToUpper(quoteCurrency) != "USD" {
		return "", fmt.Errorf("not a USD trading pair: %s", market)
	}

	if !strings.HasSuffix(strings.ToUpper(contractType), "PERP") {
		return "", fmt.Errorf("not a perpetual market: %s", market)
	}

	return baseCurrency, nil
}

func FormatUsdPerpMarket(baseCurrency string) string {
	return fmt.Sprintf("%s-USD-PERP", strings.ToUpper(baseCurrency))
}

func PopulateOrderSignature(req *CreateOrderReq, config SystemConfigRes, dexAccount string, dexPrivateKey string) error {
	sc := caigo.StarkCurve{}
	typedData, err := NewVerificationTypedData("Order", config.ChainId)
	if err != nil {
		return fmt.Errorf("failed to create verification typed data for chainId=%s: %w", config.ChainId, err)
	}

	domEnc, err := typedData.GetTypedMessageHash("StarkNetDomain", typedData.Domain, sc)
	if err != nil {
		return fmt.Errorf("failed to get domain encoded hash: %w", err)
	}

	timestamp := time.Now().UnixMilli()
	orderPayload := &OrderPayload{
		Timestamp: timestamp,
		Market:    req.Market,
		Side:      string(req.Side),
		OrderType: string(req.Type),
		Size:      req.Size.String(),
		Price:     "0",
	}

	if req.Price != "" {
		orderPayload.Price = req.Price
	}

	dexAccountBN := types.HexToBN(dexAccount)
	if dexAccountBN == nil {
		return fmt.Errorf("invalid private key")
	}
	messageHash, err := GnarkGetMessageHash(typedData, domEnc, dexAccountBN, orderPayload, sc)
	if err != nil {
		return fmt.Errorf("failed to compute message hash for account=%s: %w", dexAccount, err)
	}

	r, s, err := GnarkSign(messageHash, dexPrivateKey)
	if err != nil {
		return fmt.Errorf("failed to sign message: %w", err)
	}

	req.SignatureTimestamp = timestamp
	req.Signature, err = GetSignatureStr(r, s)
	if err != nil {
		return fmt.Errorf("failed to generate signature string: %w", err)
	}

	return nil
}
