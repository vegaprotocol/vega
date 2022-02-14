package main

import (
	"context"
	"errors"
	"fmt"

	vgjson "code.vegaprotocol.io/shared/libs/json"
	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/nodewallets"
	"github.com/jessevdk/go-flags"
)

type InitCmd struct {
	config.VegaHomeFlag
	config.OutputFlag
	config.Passphrase `long:"nodewallet-passphrase-file"`

	Force bool `short:"f" long:"force" description:"Erase exiting vega configuration at the specified path"`
}

var initCmd InitCmd

func (opts *InitCmd) Execute(args []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	if len(args) != 1 {
		return errors.New("require exactly 1 parameter mode, expected modes [validator, full]")
	}

	mode, err := encoding.NodeModeFromString(args[0])
	if err != nil {
		return err
	}

	output, err := opts.GetOutput()
	if err != nil {
		return err
	}

	vegaPaths := paths.New(opts.VegaHome)

	// a nodewallet will be required only for a validator node
	var nwRegistry *nodewallets.RegistryLoader
	if mode == encoding.NodeModeValidator {
		pass, err := opts.Get("node wallet", true)
		if err != nil {
			return err
		}

		nwRegistry, err = nodewallets.NewRegistryLoader(vegaPaths, pass)
		if err != nil {
			return err
		}
	}

	cfgLoader, err := config.InitialiseLoader(vegaPaths)
	if err != nil {
		return fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	configExists, err := cfgLoader.ConfigExists()
	if err != nil {
		return fmt.Errorf("couldn't verify configuration presence: %w", err)
	}

	if configExists && !opts.Force {
		return fmt.Errorf("configuration already exists at `%s` please remove it first or re-run using -f", cfgLoader.ConfigFilePath())
	}

	if configExists && opts.Force {
		if output.IsHuman() {
			logger.Info("removing existing configuration", logging.String("path", cfgLoader.ConfigFilePath()))
		}
		cfgLoader.Remove()
	}

	cfg := config.NewDefaultConfig()
	cfg.NodeMode = mode

	if err := cfgLoader.Save(&cfg); err != nil {
		return fmt.Errorf("couldn't save configuration file: %w", err)
	}

	if output.IsHuman() {
		logger.Info("configuration generated successfully", logging.String("path", cfgLoader.ConfigFilePath()))
	} else if output.IsJSON() {
		if mode == encoding.NodeModeValidator {
			return vgjson.Print(struct {
				ConfigFilePath           string `json:"configFilePath"`
				NodeWalletConfigFilePath string `json:"nodeWalletConfigFilePath"`
			}{
				ConfigFilePath:           cfgLoader.ConfigFilePath(),
				NodeWalletConfigFilePath: nwRegistry.RegistryFilePath(),
			})
		}
		return vgjson.Print(struct {
			ConfigFilePath string `json:"configFilePath"`
		}{
			ConfigFilePath: cfgLoader.ConfigFilePath(),
		})
	}

	return nil
}

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{}

	var (
		short = "Initializes a vega node"
		long  = "Generate the minimal configuration required for a vega node to start"
	)
	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}
