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

package nodewallets

import (
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/nodewallets/eth"
	"code.vegaprotocol.io/vega/nodewallets/eth/clef"
	"code.vegaprotocol.io/vega/nodewallets/eth/keystore"
	"code.vegaprotocol.io/vega/nodewallets/registry"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

func GetEthereumWallet(config eth.Config, vegaPaths paths.Paths, registryPassphrase string) (*eth.Wallet, error) {
	registryLoader, err := registry.NewLoader(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.Get(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if registry.Ethereum == nil {
		return nil, ErrEthereumWalletIsMissing
	}

	return getEthereumWalletWithRegistry(config, vegaPaths, registry)
}

func getEthereumWalletWithRegistry(config eth.Config, vegaPaths paths.Paths, reg *registry.Registry) (*eth.Wallet, error) {
	switch walletRegistry := reg.Ethereum.Details.(type) {
	case registry.EthereumClefWallet:
		ethAddress := ethcommon.HexToAddress(walletRegistry.AccountAddress)

		client, err := rpc.Dial(config.ClefAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to dial Clef daemon: %w", err)
		}

		w, err := clef.NewWallet(client, config.ClefAddress, ethAddress)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Ethereum Clef node wallet: %w", err)
		}

		return eth.NewWallet(w), nil
	case registry.EthereumKeyStoreWallet:
		walletLoader, err := keystore.InitialiseWalletLoader(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Ethereum key store node wallet loader: %w", err)
		}

		w, err := walletLoader.Load(walletRegistry.Name, walletRegistry.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't load Ethereum key store node wallet: %w", err)
		}

		return eth.NewWallet(w), nil
	default:
		return nil, fmt.Errorf("could not create unknown Ethereum wallet type %q", reg.Ethereum.Type)
	}
}

func GenerateEthereumWallet(
	config eth.Config,
	vegaPaths paths.Paths,
	registryPassphrase,
	walletPassphrase string,
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

	if !overwrite && reg.Ethereum != nil {
		return nil, ErrEthereumWalletAlreadyExists
	}

	var data map[string]string

	if config.ClefAddress != "" {
		client, err := rpc.Dial(config.ClefAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to dial Clef daemon: %w", err)
		}

		w, err := clef.GenerateNewWallet(client, config.ClefAddress)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate Ethereum clef node wallet: %w", err)
		}

		data = map[string]string{
			"clefAddress":    config.ClefAddress,
			"accountAddress": w.PubKey().Hex(),
		}

		reg.Ethereum = &registry.RegisteredEthereumWallet{
			Type: registry.EthereumWalletTypeClef,
			Details: registry.EthereumClefWallet{
				Name:           w.Name(),
				AccountAddress: w.PubKey().Hex(),
				ClefAddress:    config.ClefAddress,
			},
		}
	} else {
		keyStoreLoader, err := keystore.InitialiseWalletLoader(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Ethereum key store node wallet loader: %w", err)
		}

		w, d, err := keyStoreLoader.Generate(walletPassphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't generate Ethereum key store node wallet: %w", err)
		}

		data = d

		reg.Ethereum = &registry.RegisteredEthereumWallet{
			Type: registry.EthereumWalletTypeKeyStore,
			Details: registry.EthereumKeyStoreWallet{
				Name:       w.Name(),
				Passphrase: walletPassphrase,
			},
		}
	}

	if err := registryLoader.Save(reg, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}

func ImportEthereumWallet(
	config eth.Config,
	vegaPaths paths.Paths,
	registryPassphrase,
	walletPassphrase,
	accountAddress,
	sourceFilePath string,
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

	if !overwrite && reg.Ethereum != nil {
		return nil, ErrEthereumWalletAlreadyExists
	}

	var data map[string]string

	if config.ClefAddress != "" {
		if !ethcommon.IsHexAddress(accountAddress) {
			return nil, fmt.Errorf("invalid Ethereum hex address %q", accountAddress)
		}

		ethAddress := ethcommon.HexToAddress(accountAddress)

		client, err := rpc.Dial(config.ClefAddress)
		if err != nil {
			return nil, fmt.Errorf("failed to dial Clef daemon: %w", err)
		}

		w, err := clef.NewWallet(client, config.ClefAddress, ethAddress)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Ethereum Clef node wallet: %w", err)
		}

		data = map[string]string{
			"clefAddress":    config.ClefAddress,
			"accountAddress": w.PubKey().Hex(),
		}

		reg.Ethereum = &registry.RegisteredEthereumWallet{
			Type: registry.EthereumWalletTypeClef,
			Details: registry.EthereumClefWallet{
				Name:           w.Name(),
				AccountAddress: w.PubKey().Hex(),
				ClefAddress:    config.ClefAddress,
			},
		}
	} else {
		if !filepath.IsAbs(sourceFilePath) {
			return nil, fmt.Errorf("path to the wallet file need to be absolute")
		}

		ethWalletLoader, err := keystore.InitialiseWalletLoader(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Ethereum node wallet loader: %w", err)
		}

		w, d, err := ethWalletLoader.Import(sourceFilePath, walletPassphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't import Ethereum node wallet: %w", err)
		}

		data = d

		reg.Ethereum = &registry.RegisteredEthereumWallet{
			Type: registry.EthereumWalletTypeKeyStore,
			Details: registry.EthereumKeyStoreWallet{
				Name:       w.Name(),
				Passphrase: walletPassphrase,
			},
		}
	}

	if err := registryLoader.Save(reg, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}
