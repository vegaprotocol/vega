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

func verifyBundle(log *logging.Logger, bundle *types.SignedBundle) error {
	validator, err := crypto.NewSignatureAlgorithm(crypto.Ed25519)
	if err != nil {
		if log != nil {
			log.Error("unable to instanciate new algorithm", logging.Error(err))
		}
		return err
	}
	if ok := validator.Verify(bundle.GetPubKey(), bundle.Data, bundle.Sig); !ok {
		hexPubKey := hex.EncodeToString(bundle.GetPubKey())
		if log != nil {
			log.Error("invalid tx signature", logging.String("pubkey", hexPubKey))
		}
		return ErrInvalidSignature
	}
	return nil
}
