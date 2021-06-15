package v1

import (
	"crypto/rand"
	"encoding/hex"
	"math/big"
)

func NewTransaction(pubKey, data []byte, signature *Signature) *Transaction {
	return &Transaction{
		InputData: data,
		Signature: signature,
		From:      NewTransactionPubKey(pubKey),
		Version:   1,
	}
}

func NewTransactionPubKey(pubKey []byte) *Transaction_PubKey {
	return &Transaction_PubKey{
		PubKey: []byte(hex.EncodeToString(pubKey)),
	}
}

func NewInputData(height uint64) *InputData {
	return &InputData{
		Nonce:       makeNonce(),
		BlockHeight: height,
	}
}

func NewSignature(sig []byte, algo string, version uint32) *Signature {
	return &Signature{
		Bytes:   []byte(hex.EncodeToString(sig)),
		Algo:    algo,
		Version: version,
	}
}

func makeNonce() uint64 {
	max := &big.Int{}
	// set it to the max value of the uint64
	max.SetUint64(^uint64(0))
	nonce, _ := rand.Int(rand.Reader, max)
	return nonce.Uint64()
}
