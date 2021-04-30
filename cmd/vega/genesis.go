package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet"
	"code.vegaprotocol.io/vega/nodewallet/eth"

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/jessevdk/go-flags"
	tmconfig "github.com/tendermint/tendermint/config"
)

type genesisCmd struct {
	config.RootPathFlag
	// We've unified the passphrase flag as config.PassphraseFlag, which uses --passphrase.
	// As systemtests uses --vega-wallet-passphrase we'll define the flag directly here
	// TODO: uncomment this line and remove the Passphrase field.
	// config.PassphraseFlag
	Passphrase config.Passphrase `short:"p" long:"nodewallet-passphrase" description:"A file containing the passphrase for the nodewallet, if empty will prompt for input"`

	InPlace bool   `short:"i" long:"in-place" description:"Edit the genesis file in-place"`
	TmRoot  string `short:"t" long:"tm-root" description:"The root path of tendermint"`
}

func (opts *genesisCmd) Execute(_ []string) error {
	tmCfg := tmconfig.DefaultConfig()
	tmCfg.SetRoot(os.ExpandEnv(opts.TmRoot))

	pass, err := opts.Passphrase.Get("node wallet")
	if err != nil {
		return err
	}

	// just for a nicer output, if we do not enter a new line
	// the genesis output is dump straight on the password field
	fmt.Printf("\n")

	// load tm pubkey
	tmKey, err := loadTMPubkey(tmCfg)
	if err != nil {
		return fmt.Errorf("unable to load TM pubkey: %w", err)
	}

	log := logging.NewLoggerFromConfig(
		logging.NewDefaultConfig(),
	)
	defer log.AtExit()

	// load vega pubkey
	vegaKey, err := loadVegaPubKey(log, opts.RootPath, pass)
	if err != nil {
		return err
	}

	// Update the default state with the validators info
	gs := genesis.DefaultGenesisState()
	gs.Validators[tmKey] = vegaKey

	// dump or write
	if opts.InPlace {
		return genesis.UpdateInPlace(&gs, tmCfg.BaseConfig.GenesisFile())
	}
	dump, err := genesis.Dump(&gs)
	if err != nil {
		return err
	}
	fmt.Printf("%v\n", dump)
	return nil
}

// loadTMPubkey returns the hex encoded value of publickey from `priv_validator_key.json`
// It exits the program there is a problem loading the key.
func loadTMPubkey(cfg *tmconfig.Config) (string, error) {
	keyJSONBytes, err := ioutil.ReadFile(cfg.PrivValidatorKeyFile())
	if err != nil {
		return "", nil
	}

	pvKey := struct {
		PubKey struct {
			Value string `json:"value"`
		} `json:"pub_key"`
	}{}

	if err := json.Unmarshal(keyJSONBytes, &pvKey); err != nil {
		return "", nil
	}

	return pvKey.PubKey.Value, nil
}

func loadVegaPubKey(log *logging.Logger, rootPath, pass string) (string, error) {
	conf, err := config.Read(rootPath)
	if err != nil {
		return "", err
	}

	// instantiate the ETHClient
	var ethclt eth.ETHClient = nil
	if conf.NodeWallet.ETH.Address != "" {
		ethclt, err = ethclient.Dial(conf.NodeWallet.ETH.Address)
		if err != nil {
			return "", fmt.Errorf("failed to connect to Ethereum at %s: %w", conf.NodeWallet.ETH.Address, err)
		}
	}

	nw, err := nodewallet.New(log, conf.NodeWallet, pass, ethclt)
	if err != nil {
		return "", err
	}

	w, ok := nw.Get("vega")
	if !ok {
		return "", errors.New("no vega wallet stored in node wallet")
	}

	vegaKey := w.PubKeyOrAddress()
	return hex.EncodeToString(vegaKey), nil
}

func Genesis(ctx context.Context, parser *flags.Parser) error {
	rootPath := config.NewRootPathFlag()
	_, err := parser.AddCommand(
		"genesis",
		"Generates the genesis file",
		"Generate a default genesis state for a vega network",
		&genesisCmd{
			RootPathFlag: rootPath,
			TmRoot:       "$HOME/.tendermint",
		},
	)
	return err
}
