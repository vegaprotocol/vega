package v1

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

func NewTransactionV1(pubKey, data []byte, signature *Signature) *Transaction {
	return &Transaction{
		InputData: data,
		Signature: signature,
		From:      NewTransactionAddress(pubKey),
		Version:   1,
	}
}

func NewTransactionAddress(pubKey []byte) *Transaction_Address {
	return &Transaction_Address{
		Address: hex.EncodeToString(pubKey),
	}
}

func NewInputData() *InputData {
	return &InputData{
		Nonce:       makeNonce(),
		BlockHeight: 0,
	}
}

func makeNonce() uint64 {
	max := &big.Int{}
	// set it to the max value of the uint64
	max.SetUint64(^uint64(0))
	nonce, _ := rand.Int(rand.Reader, max)
	return nonce.Uint64()
}
