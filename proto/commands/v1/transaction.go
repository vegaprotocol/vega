package v1

import (
<<<<<<< HEAD
	"encoding/hex"

	"code.vegaprotocol.io/vega/crypto"
=======
	"crypto/rand"
	"encoding/hex"
	"math/big"
>>>>>>> 04ef9cc70 (refactor(events)!: revert change to LossSocialization event)
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
<<<<<<< HEAD
		Nonce:       crypto.NewNonce(),
=======
		Nonce:       makeNonce(),
>>>>>>> 04ef9cc70 (refactor(events)!: revert change to LossSocialization event)
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
<<<<<<< HEAD
=======

func makeNonce() uint64 {
	max := &big.Int{}
	// set it to the max value of the uint64
	max.SetUint64(^uint64(0))
	nonce, _ := rand.Int(rand.Reader, max)
	return nonce.Uint64()
}
>>>>>>> 04ef9cc70 (refactor(events)!: revert change to LossSocialization event)
