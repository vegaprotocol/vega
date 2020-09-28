package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"

	"code.vegaprotocol.io/vega/genesis"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallet/vega"

	"github.com/spf13/cobra"
	tmconfig "github.com/tendermint/tendermint/config"
)

type genesisCommand struct {
	command

	log              *logging.Logger
	tmRoot           string
	vegaWalletPath   string
	inPlace          bool
	walletPassphrase string
}

func (g *genesisCommand) Init(c *Cli) {
	g.cli = c
	g.cmd = &cobra.Command{
		Use:   "genesis",
		Short: "The genesis subcommand",
		Long:  "Generate a default genesis state for a vega network",
		RunE:  g.Run,
	}

	g.cmd.Flags().StringVarP(&g.tmRoot, "tm-root", "t", "$HOME/.tendermint", "The root path of tendermint")
	g.cmd.Flags().StringVarP(&g.vegaWalletPath, "vega-wallet-path", "v", "", "The path of vega wallet")
	g.cmd.Flags().StringVarP(&g.walletPassphrase, "vega-wallet-passphrase", "p", "", "Vega wallet passphrase")
	g.cmd.Flags().BoolVarP(&g.inPlace, "in-place", "i", false, "Edit the genesis file in-place")

	g.cmd.MarkFlagRequired("vega-wallet-path")
}

func (g *genesisCommand) Run(cmd *cobra.Command, args []string) error {
	// Load the TM config with the defined path
	tmCfg := tmconfig.DefaultConfig()
	tmCfg.SetRoot(os.ExpandEnv(g.tmRoot))

	// load tm pubkey
	tmKey, err := loadTMPubkey(tmCfg)
	if err != nil {
		return err
	}

	// load vega pubkey
	vegaKey, err := loadVegaPubKey(g.vegaWalletPath, g.walletPassphrase)
	if err != nil {
		return err
	}

	// Update the default state with the validators info
	gs := genesis.DefaultGenesisState()
	gs.Validators[tmKey] = vegaKey

	// dump or write
	if g.inPlace {
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
