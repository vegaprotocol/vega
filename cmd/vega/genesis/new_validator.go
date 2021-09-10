package genesis

import (
	"encoding/base64"
	"fmt"
	"os"

	vgjson "code.vegaprotocol.io/vega/libs/json"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/validators"
	tmconfig "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"
)

type newValidatorCmd struct {
	TmRoot  string `short:"t" long:"tm-root" description:"The root path of tendermint"`
	Country string `long:"country" description:"The country from which the validator operates" required:"true"`
	InfoURL string `long:"info-url" description:"The URL from which people can get to know the validator" required:"true"`
}

func (opts *newValidatorCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := genesisCmd.PassphraseFile.Get("node wallet")
	if err != nil {
		return err
	}

	vegaKey, ethAddress, err := loadNodeWalletPubKey(log, genesisCmd.RootPath, pass)
	if err != nil {
		return err
	}

	// genesis file
	tmConfig := tmconfig.DefaultConfig()
	tmConfig.SetRoot(os.ExpandEnv(opts.TmRoot))

	pubKey, err := loadTendermintPrivateValidatorKey(tmConfig)
	if err != nil {
		return err
	}

	validatorDataDoc := tmtypes.GenesisValidator{
		Address: pubKey.Address(),
		PubKey:  pubKey,
		Power:   10,
	}
	marshalledGenesisDoc, err := tmjson.MarshalIndent(validatorDataDoc, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println("Info to add in genesis file under `validators` key")
	fmt.Println(string(marshalledGenesisDoc))

	validatorDataState := map[string]validators.ValidatorData{
		base64.StdEncoding.EncodeToString(pubKey.Bytes()): {
			VegaPubKey:      vegaKey,
			EthereumAddress: ethAddress,
			Country:         opts.Country,
			InfoURL:         opts.InfoURL,
		},
	}
	fmt.Println("Info to add in genesis file under `app_state.validators` key")
	return vgjson.PrettyPrint(validatorDataState)
}
