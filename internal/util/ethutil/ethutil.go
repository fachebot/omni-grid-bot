package ethutil

import (
	"crypto/ecdsa"
	"errors"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
)

func ParsePrivateKey(privateKey string) (*ecdsa.PrivateKey, common.Address, error) {
	if strings.HasPrefix(privateKey, "0x") ||
		strings.HasPrefix(privateKey, "0X") {
		privateKey = privateKey[2:]
	}

	pk, err := crypto.HexToECDSA(privateKey)
	if err != nil {
		return nil, common.Address{}, err
	}

	publicKey := pk.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, common.Address{}, errors.New("publicKey is not of type *ecdsa.PublicKey")
	}

	address := crypto.PubkeyToAddress(*publicKeyECDSA)
	return pk, address, nil
}
