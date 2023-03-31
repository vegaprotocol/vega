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
