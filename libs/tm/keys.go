package tm

import (
	"encoding/base64"

	tmcrypto "github.com/tendermint/tendermint/proto/tendermint/crypto"
)

func PubKeyToString(pubKey tmcrypto.PublicKey) string {
	return base64.StdEncoding.EncodeToString(pubKey.GetEd25519())
}
