package bridge

import (
	"crypto/ecdsa"

	"github.com/ethereum/go-ethereum/crypto"
)

type PrivKeySigner struct {
	privateKey *ecdsa.PrivateKey
}

func NewPrivKeySigner(hexPrivKey string) (*PrivKeySigner, error) {
	privateKey, err := crypto.HexToECDSA(hexPrivKey)
	if err != nil {
		return nil, err
	}

	return &PrivKeySigner{
		privateKey: privateKey,
	}, nil
}

func (p *PrivKeySigner) Sign(hash []byte) ([]byte, error) {
	return crypto.Sign(hash, p.privateKey)
}
