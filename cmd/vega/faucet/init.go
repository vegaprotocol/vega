package faucet

import (
	"fmt"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/faucet"
	"code.vegaprotocol.io/vega/logging"
)

type faucetInit struct {
	config.VegaHomeFlag
	config.PassphraseFlag
	config.OutputFlag

	Force         bool `short:"f" long:"force" description:"Erase existing configuration at specified path"`
	UpdateInPlace bool `long:"update-in-place" description:"Update the Vega node configuration with the faucet public key"`
}

func (opts *faucetInit) Execute(_ []string) error {
	logDefaultConfig := logging.NewDefaultConfig()
	log := logging.NewLoggerFromConfig(logDefaultConfig)
	defer log.AtExit()

	output, err := opts.OutputFlag.GetOutput()
	if err != nil {
		return err
	}

	pass, err := opts.PassphraseFile.Get("faucet wallet", true)
	if err != nil {
		return err
	}

	vegaPaths := paths.New(opts.VegaHome)

	initResult, err := faucet.Initialise(vegaPaths, pass, opts.Force)
	if err != nil {
		return fmt.Errorf("couldn't initialise faucet: %w", err)
	}

	var nodeCfgFilePath string
	if opts.UpdateInPlace {
		nodeCfgLoader, nodeCfg, err := config.EnsureNodeConfig(vegaPaths)
		if err != nil {
			return err
		}

		// add the faucet public key to the allowlist
		nodeCfg.EvtForward.BlockchainQueueAllowlist = append(
			nodeCfg.EvtForward.BlockchainQueueAllowlist, initResult.Wallet.PublicKey)

		if err := nodeCfgLoader.Save(nodeCfg); err != nil {
			return fmt.Errorf("couldn't update node configuration: %w", err)
		}

		nodeCfgFilePath = nodeCfgLoader.ConfigFilePath()
	}

	result := struct {
		PublicKey            string `json:"publicKey"`
		NodeConfigFilePath   string `json:"nodeConfigFilePath,omitempty"`
		FaucetConfigFilePath string `json:"faucetConfigFilePath"`
		FaucetWalletFilePath string `json:"faucetWalletFilePath"`
	}{
		NodeConfigFilePath:   nodeCfgFilePath,
		FaucetConfigFilePath: initResult.ConfigFilePath,
		FaucetWalletFilePath: initResult.Wallet.FilePath,
		PublicKey:            initResult.Wallet.PublicKey,
	}

	if output.IsHuman() {
		log.Info("faucet initialised successfully", logging.String("public-key", initResult.Wallet.PublicKey))
		err := vgjson.PrettyPrint(result)
		if err != nil {
			return fmt.Errorf("couldn't pretty print result: %w", err)
		}
	} else if output.IsJSON() {
		return vgjson.Print(result)
	}

	return nil
}
