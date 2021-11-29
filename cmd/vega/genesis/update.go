package genesis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/nodewallets"
	"code.vegaprotocol.io/vega/validators"

	"github.com/jessevdk/go-flags"
	tmconfig "github.com/tendermint/tendermint/config"
	tmjson "github.com/tendermint/tendermint/libs/json"
	"github.com/tendermint/tendermint/types"
)

type updateCmd struct {
	Config nodewallets.Config

	DryRun  bool   `long:"dry-run" description:"Display the genesis file without writing it"`
	Network string `short:"n" long:"network" choice:"mainnet" choice:"testnet"`
	TmRoot  string `short:"t" long:"tm-root" description:"The root path of tendermint"`
}

func (opts *updateCmd) Execute(_ []string) error {
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
	b64TmPubKey := base64.StdEncoding.EncodeToString(pubKey.Bytes())
	genesisState := genesis.DefaultGenesisState()
	genesisState.Validators[base64.StdEncoding.EncodeToString(pubKey.Bytes())] = validators.ValidatorData{
		ID:               walletID,
		VegaPubKey:       vegaKey.value,
		VegaPubKeyNumber: vegaKey.index,
		EthereumAddress:  ethAddress,
		TmPubKey:         b64TmPubKey,
	}

	if len(opts.Network) != 0 {
		ethConfig := `{"network_id": "%s", "chain_id": "%s", "bridge_address": "%s", "confirmations": %d,  "staking_bridge_addresses": %s}`
		switch opts.Network {
		case "mainnet":
			delete(genesisState.Assets, "VOTE")
			genesisState.Assets["VEGA"] = assets.VegaTokenMainNet
			marshalledBridgeAddresses, _ := json.Marshal([]string{"0xfc9Ad8fE9E0b168999Ee7547797BC39D55d607AA", "0x1B57E5393d949242a9AD6E029E2f8A684BFbBC08"})
			ethConfig = fmt.Sprintf(ethConfig, "3", "3", "0x898b9F9f9Cab971d9Ceb809F93799109Abbe2D10", 3, marshalledBridgeAddresses)
		case "testnet":
			genesisState.Assets["VEGA"] = assets.VegaTokenTestNet
			delete(genesisState.Assets, "VOTE")
			marshalledBridgeAddresses, _ := json.Marshal([]string{"0xfc9Ad8fE9E0b168999Ee7547797BC39D55d607AA", "0x1B57E5393d949242a9AD6E029E2f8A684BFbBC08"})
			ethConfig = fmt.Sprintf(ethConfig, "3", "3", "0x898b9F9f9Cab971d9Ceb809F93799109Abbe2D10", 3, marshalledBridgeAddresses)
		default:
			return fmt.Errorf("network %s is not supported", opts.Network)
		}
		genesisState.NetParams[netparams.BlockchainsEthereumConfig] = ethConfig
	}

	rawGenesisState, err := json.Marshal(genesisState)
	if err != nil {
		return err
	}

	genesisFilePath := tmConfig.GenesisFile()
	data, err := vgfs.ReadFile(genesisFilePath)
	if err != nil {
		return err
	}

	genesisDoc := &types.GenesisDoc{}
	err = tmjson.Unmarshal(data, genesisDoc)
	if err != nil {
		return err
	}

	genesisDoc.AppState = rawGenesisState

	if !opts.DryRun {
		log.Infof("Saving genesis doc at %s", genesisFilePath)
		if err := genesisDoc.SaveAs(genesisFilePath); err != nil {
			return err
		}
	}

	marshalledGenesisDoc, err := tmjson.MarshalIndent(genesisDoc, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(marshalledGenesisDoc))
	return err
}
