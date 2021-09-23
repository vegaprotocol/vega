package nodewallet

import (
	"errors"
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/nodewallets/eth"
	"code.vegaprotocol.io/vega/nodewallets/vega"
)

var (
	ErrEthereumWalletAlreadyExists = errors.New("the Ethereum node wallet already exists")
	ErrVegaWalletAlreadyExists     = errors.New("the Vega node wallet already exists")
)

func GetEthereumWallet(vegaPaths paths.Paths, registryPassphrase string) (*eth.Wallet, error) {
	registryLoader, err := InitialiseRegistry(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if registry.Ethereum == nil {
		return nil, ErrEthereumWalletIsMissing
	}

	walletLoader, err := eth.InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Ethereum node wallet loader: %w", err)
	}

	wallet, err := walletLoader.Load(registry.Ethereum.Name, registry.Ethereum.Passphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load Ethereum node wallet: %w", err)
	}

	return wallet, nil
}

func GetVegaWallet(vegaPaths paths.Paths, registryPassphrase string) (*vega.Wallet, error) {
	registryLoader, err := InitialiseRegistry(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
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

func GetNodeWallets(vegaPaths paths.Paths, registryPassphrase string) (*NodeWallets, error) {
	nodeWallets := &NodeWallets{}

	registryLoader, err := InitialiseRegistry(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if registry.Ethereum != nil {
		ethWalletLoader, err := eth.InitialiseWalletLoader(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Ethereum node wallet loader: %w", err)
		}

		nodeWallets.Ethereum, err = ethWalletLoader.Load(registry.Ethereum.Name, registry.Ethereum.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't load Ethereum node wallet: %w", err)
		}
	}

	if registry.Vega != nil {
		vegaWalletLoader, err := vega.InitialiseWalletLoader(vegaPaths)
		if err != nil {
			return nil, fmt.Errorf("couldn't initialise Vega node wallet loader: %w", err)
		}

		nodeWallets.Vega, err = vegaWalletLoader.Load(registry.Vega.Name, registry.Vega.Passphrase)
		if err != nil {
			return nil, fmt.Errorf("couldn't load Vega node wallet: %w", err)
		}
	}

	return nodeWallets, nil
}

func GenerateEthereumWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase string, overwrite bool) (map[string]string, error) {
	registryLoader, err := InitialiseRegistry(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if !overwrite && registry.Ethereum != nil {
		return nil, ErrEthereumWalletAlreadyExists
	}

	ethWalletLoader, err := eth.InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Ethereum node wallet loader: %w", err)
	}

	w, data, err := ethWalletLoader.Generate(walletPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't generate Ethereum node wallet: %w", err)
	}

	registry.Ethereum = &RegisteredEthereumWallet{
		Name:       w.Name(),
		Passphrase: walletPassphrase,
	}

	if err := registryLoader.SaveRegistry(registry, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}

func GenerateVegaWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase string, overwrite bool) (map[string]string, error) {
	registryLoader, err := InitialiseRegistry(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if !overwrite && registry.Vega != nil {
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

	registry.Vega = &RegisteredVegaWallet{
		Name:       w.Name(),
		Passphrase: walletPassphrase,
	}

	if err := registryLoader.SaveRegistry(registry, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}

func ImportEthereumWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase, sourceFilePath string, overwrite bool) (map[string]string, error) {
	if !filepath.IsAbs(sourceFilePath) {
		return nil, fmt.Errorf("path to the wallet file need to be absolute")
	}

	registryLoader, err := InitialiseRegistry(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if !overwrite && registry.Ethereum != nil {
		return nil, ErrEthereumWalletAlreadyExists
	}

	ethWalletLoader, err := eth.InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise Ethereum node wallet loader: %w", err)
	}

	w, data, err := ethWalletLoader.Import(sourceFilePath, walletPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't import Ethereum node wallet: %w", err)
	}

	registry.Ethereum = &RegisteredEthereumWallet{
		Name:       w.Name(),
		Passphrase: walletPassphrase,
	}

	if err := registryLoader.SaveRegistry(registry, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}

func ImportVegaWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase, sourceFilePath string, overwrite bool) (map[string]string, error) {
	if !filepath.IsAbs(sourceFilePath) {
		return nil, fmt.Errorf("path to the wallet file need to be absolute")
	}

	registryLoader, err := InitialiseRegistry(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if !overwrite && registry.Vega != nil {
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

	registry.Vega = &RegisteredVegaWallet{
		Name:       w.Name(),
		Passphrase: walletPassphrase,
	}

	if err := registryLoader.SaveRegistry(registry, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}
