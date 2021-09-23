package main

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/config"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	"code.vegaprotocol.io/shared/paths"

	"github.com/jessevdk/go-flags"
)

type InitCmd struct {
	config.VegaHomeFlag

	Force bool `short:"f" long:"force" description:"Erase exiting vega configuration at the specified path"`
}

var initCmd InitCmd

func (opts *InitCmd) Execute(_ []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	vegaPaths := paths.New(opts.VegaHome)

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
		cfgLoader.Remove()
	}

	cfg := config.NewDefaultConfig()

	if err := cfgLoader.Save(&cfg); err != nil {
		return fmt.Errorf("couldn't save configuration file: %w", err)
	}

	if _, err = storage.InitialiseStorage(vegaPaths); err != nil {
		return fmt.Errorf("couldn't initialise storage: %w", err)
	}

	logger.Info("configuration generated successfully", logging.String("path", cfgLoader.ConfigFilePath()))

	return nil
}

func Init(ctx context.Context, parser *flags.Parser) error {
	initCmd = InitCmd{}

	short := "Initializes a vega node"
	long := "Generate the minimal configuration required for a vega data-node to start"

	_, err := parser.AddCommand("init", short, long, &initCmd)
	return err
}
