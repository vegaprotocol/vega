package blockchain

import (
	"encoding/hex"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/wallet/crypto"
)

var (
	ErrInvalidSignature = errors.New("invalid signature")
)

func verifyBundle(log *logging.Logger, tx *types.Transaction, bundle *types.SignedBundle) error {
	// build new signature algorithm using the algo from the sig
	validator, err := crypto.NewSignatureAlgorithm(bundle.Sig.Algo)
	if err != nil {
		if log != nil {
			log.Error("unable to instanciate new algorithm", logging.Error(err))
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
