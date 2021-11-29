package genesis

import (
	"encoding/base64"
	"fmt"
	"os"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/validators"
	"github.com/jessevdk/go-flags"
	tmconfig "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"
	tmtypes "github.com/tendermint/tendermint/types"
)

type newValidatorCmd struct {
	Config nodewallets.Config

	TmRoot    string `short:"t" long:"tm-root" description:"The root path of tendermint"`
	Country   string `long:"country" description:"The country from which the validator operates" required:"true"`
	InfoURL   string `long:"info-url" description:"The URL from which people can get to know the validator" required:"true"`
	Name      string `long:"name" description:"The name of the validator node" required:"true"`
	AvatarURL string `long:"avatar-url" description:"An URL to an avatar for the validator"`
}

func (opts *newValidatorCmd) Execute(_ []string) error {
	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	pass, err := genesisCmd.PassphraseFile.Get("node wallet", false)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(genesisCmd.VegaHome)

	if _, err := flags.NewParser(opts, flags.Default|flags.IgnoreUnknown).Parse(); err != nil {
		return err
	}

	vegaKey, ethAddress, walletID, err := loadNodeWalletPubKey(opts.Config, vegaPaths, pass)
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

	b64TmPubKey := base64.StdEncoding.EncodeToString(pubKey.Bytes())

	validatorDataState := map[string]validators.ValidatorData{
		base64.StdEncoding.EncodeToString(pubKey.Bytes()): {
			ID:               walletID,
			VegaPubKey:       vegaKey.value,
			VegaPubKeyNumber: vegaKey.index,
			TmPubKey:         b64TmPubKey,
			EthereumAddress:  ethAddress,
			Country:          opts.Country,
			InfoURL:          opts.InfoURL,
			Name:             opts.Name,
			AvatarURL:        opts.AvatarURL,
		},
	}
	fmt.Println("Info to add in genesis file under `app_state.validators` key")
	return vgjson.PrettyPrint(validatorDataState)
}
