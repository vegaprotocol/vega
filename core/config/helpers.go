// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
