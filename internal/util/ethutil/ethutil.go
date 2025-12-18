package ethutil

import (
	"crypto/ecdsa"
	"errors"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/shopspring/decimal"
)

func ParseUnits(value *big.Int, decimals uint8) decimal.Decimal {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromInt32(int32(decimals)))
	num, _ := decimal.NewFromString(value.String())
	result := num.DivRound(mul, int32(decimals)).Truncate(int32(decimals))
	return result
}

func FormatUnits(amount decimal.Decimal, decimals uint8) *big.Int {
	mul := decimal.NewFromFloat(float64(10)).Pow(decimal.NewFromInt32(int32(decimals)))
	result := amount.Mul(mul)

	wei := big.NewInt(0)
	wei.SetString(result.String(), 10)
	return wei
}

func GetAddress(privateKey *ecdsa.PrivateKey) (common.Address, error) {
	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return common.Address{}, errors.New("publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return address, nil
}

func ParsePrivateKey(privateKey string) (*ecdsa.PrivateKey, common.Address, error) {
	if strings.HasPrefix(privateKey, "0x") ||
		strings.HasPrefix(privateKey, "0X") {
		privateKey = privateKey[2:]
	}

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, common.Address{}, err
	}

	address, err := GetAddress(pk)
	return pk, address, err
}
