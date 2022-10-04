// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package commands

import (
	"fmt"

	"code.vegaprotocol.io/vega/blockexplorer/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"github.com/jessevdk/go-flags"
)

func loadConfig(logger *logging.Logger, vegaHome string) (*config.Config, error) {
	paths := paths.New(vegaHome)

	loader, err := config.NewLoader(paths)
	if err != nil {
		return nil, fmt.Errorf("creating config loader: %w", err)
	}

	exists, err := loader.ConfigExists()
	if err != nil {
		return nil, fmt.Errorf("checking for existence of config file: %w", err)
	}

	var cfg *config.Config
	if exists {
		cfg, err = loader.Get()
		if err != nil {
			return nil, fmt.Errorf("loading config: %w", err)
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
