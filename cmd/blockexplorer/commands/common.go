// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package commands

import (
	"fmt"

	"code.vegaprotocol.io/vega/blockexplorer/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/jessevdk/go-flags"
)

func loadConfig(logger *logging.Logger, vegaHome string) (*config.Config, error) {
	vegaPaths := paths.New(vegaHome)

	loader, err := config.NewLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("could not create config loader: %w", err)
	}

	exists, err := loader.ConfigExists()
	if err != nil {
		return nil, fmt.Errorf("could not check for existence of config file: %w", err)
	}

	var cfg *config.Config
	if exists {
		cfg, err = loader.Get()
		if err != nil {
			return nil, fmt.Errorf("could not load config: %w", err)
		}
	} else {
		logger.Warn("No config file found; using defaults. Create one with with 'blockexplorer init'")
		defaultCfg := config.NewDefaultConfig()
		cfg = &defaultCfg
	}

	// Apply any command line overrides
	flags.NewParser(cfg, flags.Default|flags.IgnoreUnknown).Parse()
	return cfg, nil
}
