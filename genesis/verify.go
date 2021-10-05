package genesis

import (
	"encoding/hex"
	"encoding/json"
	"fmt"

	wcrypto "code.vegaprotocol.io/go-wallet/crypto"
)

const (
	PubKey = "6c9848d1e1dc4b34c5c0f3c0d661b6767c795fce2e0563c9ce40cad6af85c99f"
)

func VerifyGenesisStateSignature(genesisState *GenesisState, sig string) (bool, error) {
	sps, err := GetSignedParameters(genesisState)
	if err != nil {
		return false, fmt.Errorf("could get signed parameters from genesis state: %w", err)
	}

	jsonSps, err := json.Marshal(sps)
	if err != nil {
		return false, fmt.Errorf("couldn't marshall signed parameters: %w", err)
	}

	decodedPubKey, err := hex.DecodeString(PubKey)
	if err != nil {
		return false, fmt.Errorf("couldn't decode public key: %w", err)
	}

	decodedSig, err := hex.DecodeString(sig)
	if err != nil {
		return false, fmt.Errorf("couldn't decode signature: %w", err)
	}

	signatureAlgorithm := wcrypto.NewEd25519()
	return signatureAlgorithm.Verify(decodedPubKey, jsonSps, decodedSig)
}
