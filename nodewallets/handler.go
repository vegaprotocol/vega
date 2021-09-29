package nodewallet

import (
	"errors"
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/nodewallets/vega"
)

var (
	ErrEthereumWalletAlreadyExists = errors.New("the Ethereum node wallet already exists")
	ErrVegaWalletAlreadyExists     = errors.New("the Vega node wallet already exists")
)

func GetVegaWallet(vegaPaths paths.Paths, registryPassphrase string) (*vega.Wallet, error) {
	registryLoader, err := NewRegistryLoader(vegaPaths, registryPassphrase)
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

func GetNodeWallets(config Config, vegaPaths paths.Paths, registryPassphrase string) (*NodeWallets, error) {
	nodeWallets := &NodeWallets{}

	registryLoader, err := NewRegistryLoader(vegaPaths, registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise node wallet registry: %v", err)
	}

	registry, err := registryLoader.GetRegistry(registryPassphrase)
	if err != nil {
		return nil, fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	if registry.Ethereum != nil {
		w, err := getEthereumWalletWithRegistry(config.ETH, vegaPaths, registry)
		if err != nil {
			return nil, err
		}

		nodeWallets.Ethereum = w
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

func GenerateVegaWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase string, overwrite bool) (map[string]string, error) {
	registryLoader, err := NewRegistryLoader(vegaPaths, registryPassphrase)
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

func ImportVegaWallet(vegaPaths paths.Paths, registryPassphrase, walletPassphrase, sourceFilePath string, overwrite bool) (map[string]string, error) {
	if !filepath.IsAbs(sourceFilePath) {
		return nil, fmt.Errorf("path to the wallet file need to be absolute")
	}

	registryLoader, err := NewRegistryLoader(vegaPaths, registryPassphrase)
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
