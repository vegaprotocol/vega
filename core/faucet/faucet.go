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

package faucet

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/paths"
)

var ErrFaucetConfigAlreadyExists = errors.New("faucet configuration already exists")

type InitialisationResult struct {
	Wallet         *WalletGenerationResult
	ConfigFilePath string
}

func Initialise(vegaPaths paths.Paths, passphrase string, rewrite bool) (*InitialisationResult, error) {
	walletGenResult, err := GenerateWallet(vegaPaths, passphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate a new faucet wallet: %w", err)
	}

	cfgLoader, err := InitialiseConfigLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise faucet configuration loader: %w", err)
	}

	configExists, err := cfgLoader.ConfigExists()
	if err != nil {
		return nil, fmt.Errorf("couldn't verify faucet configuration presence: %w", err)
	}

	if configExists {
		if rewrite {
			cfgLoader.RemoveConfig()
		} else {
			return nil, ErrFaucetConfigAlreadyExists
		}
	}

	cfg := NewDefaultConfig()
	cfg.WalletName = walletGenResult.Name

	if err := cfgLoader.SaveConfig(&cfg); err != nil {
		return nil, fmt.Errorf("couldn't save faucet configuration: %w", err)
	}

	return &InitialisationResult{
		Wallet:         walletGenResult,
		ConfigFilePath: cfgLoader.ConfigFilePath(),
	}, nil
}
