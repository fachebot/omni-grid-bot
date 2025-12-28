package nado

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/signer/core/apitypes"
)

type SignPlaceOrderParams struct {
	Sender     Sender
	PriceX18   *big.Int
	Amount     *big.Int
	Expiration uint64
	Nonce      uint64
	Appendix   *big.Int
}

type SignCancellationProductsParams struct {
	Sender     Sender
	ProductIds []*big.Int
	Nonce      uint64
}

func GenOrderVerifyingContract(productID uint64) common.Address {
	var beBytes [20]byte

	bigInt := new(big.Int).SetUint64(productID)
	idBytes := bigInt.Bytes()

	copy(beBytes[20-len(idBytes):], idBytes)

	return common.BytesToAddress(beBytes[:])
}

func SignTypedData(typedData apitypes.TypedData, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	domainSeparator, err := typedData.HashStruct("EIP712Domain", typedData.Domain.Map())
	if err != nil {
		return nil, err
	}

	typedDataHash, err := typedData.HashStruct(typedData.PrimaryType, typedData.Message)
	if err != nil {
		return nil, err
	}

	rawData := fmt.Appendf(nil, "\x19\x01%s%s", string(domainSeparator), string(typedDataHash))
	challengeHash := crypto.Keccak256Hash(rawData)

	signature, err := crypto.Sign(challengeHash.Bytes(), privateKey)
	if err != nil {
		return nil, err
	}

	signature[64] += 27

	return signature, nil
}

func SignPlaceOrder(signer *ecdsa.PrivateKey, chainId int64, verifyingContract common.Address, params SignPlaceOrderParams) (string, error) {
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"Order": []apitypes.Type{
				{Name: "sender", Type: "bytes32"},
				{Name: "priceX18", Type: "int128"},
				{Name: "amount", Type: "int128"},
				{Name: "expiration", Type: "uint64"},
				{Name: "nonce", Type: "uint64"},
				{Name: "appendix", Type: "uint128"},
			},
		},
		PrimaryType: "Order",
		Domain: apitypes.TypedDataDomain{
			Name:              "Nado",
			Version:           "0.0.1",
			ChainId:           math.NewHexOrDecimal256(chainId),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"sender":     params.Sender.String(),
			"priceX18":   params.PriceX18,
			"amount":     params.Amount,
			"expiration": big.NewInt(0).SetUint64(params.Expiration),
			"nonce":      big.NewInt(0).SetUint64(params.Nonce),
			"appendix":   params.Appendix,
		},
	}

	signature, err := SignTypedData(typedData, signer)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(signature), nil
}

func SignListTriggerOrders(signer *ecdsa.PrivateKey, chainId int64, verifyingContract common.Address, sender Sender, recvTime uint64) (string, error) {
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"ListTriggerOrders": []apitypes.Type{
				{Name: "sender", Type: "bytes32"},
				{Name: "recvTime", Type: "uint64"},
			},
		},
		PrimaryType: "ListTriggerOrders",
		Domain: apitypes.TypedDataDomain{
			Name:              "Nado",
			Version:           "0.0.1",
			ChainId:           math.NewHexOrDecimal256(chainId),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"sender":   sender.String(),
			"recvTime": big.NewInt(0).SetUint64(recvTime),
		},
	}

	signature, err := SignTypedData(typedData, signer)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(signature), nil
}

func SignCancellationProducts(signer *ecdsa.PrivateKey, chainId int64, verifyingContract common.Address, params SignCancellationProductsParams) (string, error) {
	typedData := apitypes.TypedData{
		Types: apitypes.Types{
			"EIP712Domain": []apitypes.Type{
				{Name: "name", Type: "string"},
				{Name: "version", Type: "string"},
				{Name: "chainId", Type: "uint256"},
				{Name: "verifyingContract", Type: "address"},
			},
			"CancellationProducts": []apitypes.Type{
				{Name: "sender", Type: "bytes32"},
				{Name: "productIds", Type: "uint32[]"},
				{Name: "nonce", Type: "uint64"},
			},
		},
		PrimaryType: "CancellationProducts",
		Domain: apitypes.TypedDataDomain{
			Name:              "Nado",
			Version:           "0.0.1",
			ChainId:           math.NewHexOrDecimal256(chainId),
			VerifyingContract: verifyingContract.Hex(),
		},
		Message: apitypes.TypedDataMessage{
			"sender":     params.Sender.String(),
			"productIds": params.ProductIds,
			"nonce":      big.NewInt(0).SetUint64(params.Nonce),
		},
	}

	signature, err := SignTypedData(typedData, signer)
	if err != nil {
		return "", err
	}
	return "0x" + hex.EncodeToString(signature), nil
}
