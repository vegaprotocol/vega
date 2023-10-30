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

package nodewallets

import (
	"errors"
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/vega/core/nodewallets/registry"
	"code.vegaprotocol.io/vega/core/nodewallets/vega"
	"code.vegaprotocol.io/vega/paths"
)

var (
	ErrEthereumWalletAlreadyExists   = errors.New("the Ethereum node wallet already exists")
	ErrVegaWalletAlreadyExists       = errors.New("the Vega node wallet already exists")
	ErrTendermintPubkeyAlreadyExists = errors.New("the Tendermint pubkey already exists")
)

func GetVegaWallet(vegaPaths paths.Paths, registryPassphrase string) (*vega.Wallet, error) {
	registryLoader, err := registry.NewLoader(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.Get(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if registry.Vega == nil {
		return nil, ErrVegaWalletIsMissing
	}

	walletLoader, err := vega.InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Vega node wallet loader: %w", err)
	}

	wallet, err := walletLoader.Load(registry.Vega.Name, registry.Vega.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load Ethereum node wallet: %w", err)
	}

	return wallet, nil
}

func GetNodeWallets(config Config, vegaPaths paths.Paths, registryPassphrase string) (*NodeWallets, error) {
	nodeWallets := &NodeWallets{}

	registryLoader, err := registry.NewLoader(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	reg, err := registryLoader.Get(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if reg.Ethereum != nil {
		w, err := GetEthereumWalletWithRegistry(vegaPaths, reg)
		if err != nil {
			return nil, err
		}

		nodeWallets.Ethereum = w
	}

	if reg.Vega != nil {
		vegaWalletLoader, err := vega.InitialiseWalletLoader(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Vega node wallet loader: %w", err)
		}

		nodeWallets.Vega, err = vegaWalletLoader.Load(reg.Vega.Name, reg.Vega.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't load Vega node wallet: %w", err)
		}
	}

	if reg.Tendermint != nil {
		nodeWallets.Tendermint = &TendermintPubkey{
			Pubkey: reg.Tendermint.Pubkey,
		}
	}

	return nodeWallets, nil
}

func GenerateVegaWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase string, overwrite bool) (map[string]string, error) {
	registryLoader, err := registry.NewLoader(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	reg, err := registryLoader.Get(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if !overwrite && reg.Vega != nil {
		return nil, ErrVegaWalletAlreadyExists
	}

	vegaWalletLoader, err := vega.InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Vega node wallet loader: %w", err)
	}

	w, data, err := vegaWalletLoader.Generate(walletPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate Vega node wallet: %w", err)
	}

	reg.Vega = &registry.RegisteredVegaWallet{
		Name:       w.Name(),
		Passphrase: walletPassphrase,
	}

	if err := registryLoader.Save(reg, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}

func ImportVegaWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase, sourceFilePath string, overwrite bool) (map[string]string, error) {
	if !filepath.IsAbs(sourceFilePath) {
		return nil, fmt.Errorf("path to the wallet file need to be absolute")
	}

	registryLoader, err := registry.NewLoader(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	reg, err := registryLoader.Get(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if !overwrite && reg.Vega != nil {
		return nil, ErrVegaWalletAlreadyExists
	}

	vegaWalletLoader, err := vega.InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Vega node wallet loader: %w", err)
	}

	w, data, err := vegaWalletLoader.Import(sourceFilePath, walletPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't import Vega node wallet: %w", err)
	}

	reg.Vega = &registry.RegisteredVegaWallet{
		Name:       w.Name(),
		Passphrase: walletPassphrase,
	}

	if err := registryLoader.Save(reg, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}

func ImportTendermintPubkey(
	vegaPaths paths.Paths,
	registryPassphrase, pubkey string,
	overwrite bool,
) (map[string]string, error) {
	registryLoader, err := registry.NewLoader(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	reg, err := registryLoader.Get(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if !overwrite && reg.Tendermint != nil {
		return nil, ErrTendermintPubkeyAlreadyExists
	}

	reg.Tendermint = &registry.RegisteredTendermintPubkey{
		Pubkey: pubkey,
	}

	if err := registryLoader.Save(reg, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	return map[string]string{
		"registryFilePath": registryLoader.RegistryFilePath(),
		"tendermintPubkey": pubkey,
	}, nil
}
