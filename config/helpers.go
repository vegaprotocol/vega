package config

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/shared/paths"
)

func EnsureNodeConfig(vegaPaths paths.Paths) (*Loader, *Config, error) {
	cfgLoader, err := InitialiseLoader(vegaPaths)
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't initialise configuration loader: %w", err)
	}

	configExists, err := cfgLoader.ConfigExists()
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't verify configuration presence: %w", err)
	}
	if !configExists {
		return nil, nil, fmt.Errorf("node has not been initialised, please run `%s init`", os.Args[0])
	}

	cfg, err := cfgLoader.Get()
	if err != nil {
		return nil, nil, fmt.Errorf("couldn't get configuration: %w", err)
	}

	return cfgLoader, cfg, nil
}
