package main

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/nodewallet/vega"
	"github.com/jessevdk/go-flags"
	tmconfig "github.com/tendermint/tendermint/config"
)

type genesisCmd struct {
	config.PassphraseFlag
	InPlace    bool   `short:"i" long:"in-place" description:"Edit the genesis file in-place"`
	TmRoot     string `short:"t" long:"tm-root" description:"The root path of tendermint"`
	WalletPath string `short:"v" long:"vega-wallet-path" description:"The path of vega wallet" required:"true"`
}

func (opts *genesisCmd) Execute(_ []string) error {
	tmCfg := tmconfig.DefaultConfig()
	tmCfg.SetRoot(os.ExpandEnv(opts.TmRoot))

	pass, err := opts.Passphrase.Get("vega wallet")
	if err != nil {
		return err
	}

	// load tm pubkey
	tmKey, err := loadTMPubkey(tmCfg)
	if err != nil {
		return err
	}

	vegaKey, err := loadVegaPubKey(opts.WalletPath, pass)
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

func loadVegaPubKey(path, pass string) (string, error) {
	w, err := vega.New(os.ExpandEnv(path), pass)
	if err != nil {
		return "", err
	}

	vegaKey := w.PubKeyOrAddress()
	return hex.EncodeToString(vegaKey), nil
}

func Genesis(ctx context.Context, parser *flags.Parser) error {
	_, err := parser.AddCommand(
		"genesis",
		"Generates the genesis file",
		"Generate a default genesis state for a vega network",
		&genesisCmd{
			TmRoot: "$HOME/.tendermint",
		},
	)
	return err
}
