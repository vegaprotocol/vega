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

package config

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/vega/paths"
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
