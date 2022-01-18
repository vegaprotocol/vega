package genesis

import (
	"errors"
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
)

type verifyCmd struct {
	TmHome    string `short:"t" long:"tm-home" description:"The root path of tendermint"`
	Signature string `short:"s" long:"signature" description:"The hex-encoded signature to verify"`
}

func (opts *verifyCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	if len(opts.Signature) == 0 {
		return errors.New("signature is required")
	}

	_, genesisState, err := genesis.GetLocalGenesisState(os.ExpandEnv(opts.TmHome))
	if err != nil {
		return err
	}

	validSignature, err := genesis.VerifyGenesisStateSignature(genesisState, opts.Signature)
	if err != nil {
		return fmt.Errorf("couldn't verify the genesis state signature: %w", err)
	}
	if !validSignature {
		return errors.New("the signature is invalid")
	}

	fmt.Println("the signature is valid")

	return nil
}
