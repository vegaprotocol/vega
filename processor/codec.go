package processor

import (
	"encoding/hex"
	"fmt"

	"code.vegaprotocol.io/vega/blockchain/abci"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/wallet/crypto"

	"github.com/golang/protobuf/proto"
)

type codec struct {
}

// Decode takes a raw input from a Tendermint Tx and decodes into a vega Tx,
// the decoding process involves a signature verification.
func (c *codec) Decode(payload []byte) (abci.Tx, error) {
	bundle := &types.SignedBundle{}
	if err := proto.Unmarshal(payload, bundle); err != nil {
		return nil, fmt.Errorf("unable to unmarshal signed bundle: %w", err)
	}

	protoTx := &types.Transaction{}
	if err := proto.Unmarshal(bundle.Tx, protoTx); err != nil {
		return nil, fmt.Errorf("unable to unmarshal transaction from signed bundle: %w", err)
	}

	tx, err := NewTx(protoTx, bundle.Sig.Sig)
	if err != nil {
		return nil, err
	}

	if err := verifyBundle(bundle, protoTx.GetPubKey()); err != nil {
		return nil, err
	}

	return tx, nil
}

func verifyBundle(bundle *types.SignedBundle, pubkey []byte) error {
	// build new signature algorithm using the algo from the sig
	validator, err := crypto.NewSignatureAlgorithm(bundle.Sig.Algo)
	if err != nil {
		return fmt.Errorf("unable to instanciate new algorithm: %w", err)
	}

	ok, err := validator.Verify(pubkey, bundle.Tx, bundle.Sig.Sig)
	if err != nil {
		return fmt.Errorf("unable to verify bundle: %w", err)
	}

	if !ok {
		hexPubKey := hex.EncodeToString(pubkey)
		return fmt.Errorf("invalid tx signature '%s': %w", hexPubKey, ErrInvalidSignature)
	}

	return nil
}
