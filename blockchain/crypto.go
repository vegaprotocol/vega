package blockchain

import (
	"encoding/hex"
	"errors"

	"code.vegaprotocol.io/go-wallet/crypto"
	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
)

func verifyBundle(log *logging.Logger, tx *types.Transaction, bundle *types.SignedBundle) error {
	// build new signature algorithm using the algo from the sig
	validator, err := crypto.NewSignatureAlgorithm(bundle.Sig.Algo, bundle.Sig.Version)
	if err != nil {
		if log != nil {
			log.Error("unable to instantiate new algorithm", logging.Error(err))
		}
		return err
	}
	ok, err := validator.Verify(tx.GetPubKey(), bundle.Tx, bundle.Sig.Sig)
	if err != nil {
		if log != nil {
			log.Error("unable to verify bundle", logging.Error(err))
		}
		return err
	}
	if !ok {
		hexPubKey := hex.EncodeToString(tx.GetPubKey())
		if log != nil {
			log.Error("invalid tx signature", logging.String("pubkey", hexPubKey))
		}
		return ErrInvalidSignature
	}
	return nil
}
