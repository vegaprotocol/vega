package v1

import (
	"encoding/hex"

	"code.vegaprotocol.io/vega/crypto"
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
		PubKey: hex.EncodeToString(pubKey),
	}
}

func NewInputData(height uint64) *InputData {
	return &InputData{
		Nonce:       crypto.NewNonce(),
		BlockHeight: height,
	}
}

func NewSignature(sig []byte, algo string, version uint32) *Signature {
	return &Signature{
		Value:   hex.EncodeToString(sig),
		Algo:    algo,
		Version: version,
	}
}
