package nodewallets

import (
	"fmt"
	"path/filepath"

	"code.vegaprotocol.io/shared/paths"
	"code.vegaprotocol.io/vega/nodewallets/eth"
	"code.vegaprotocol.io/vega/nodewallets/eth/clef"
	"code.vegaprotocol.io/vega/nodewallets/eth/keystore"
	"code.vegaprotocol.io/vega/nodewallets/registryloader"
	ethcommon "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rpc"
)

func GetEthereumWallet(config eth.Config, vegaPaths paths.Paths, registryPassphrase string) (*eth.Wallet, error) {
	registryLoader, err := registryloader.New(vegaPaths, registryPassphrase)
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

	return getEthereumWalletWithRegistry(config, vegaPaths, registry)
}

func getEthereumWalletWithRegistry(config eth.Config, vegaPaths paths.Paths, registry *registryloader.Registry) (*eth.Wallet, error) {
	switch walletRegistry := registry.Ethereum.Details.(type) {
	case registryloader.EthereumClefWallet:
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
	case registryloader.EthereumKeyStoreWallet:
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
		return nil, fmt.Errorf("could not create unknown Ethereum wallet type %q", registry.Ethereum.Type)
	}
}

func GenerateEthereumWallet(
	config eth.Config,
	vegaPaths paths.Paths,
	registryPassphrase,
	walletPassphrase string,
	overwrite bool,
) (map[string]string, error) {
	registryLoader, err := registryloader.New(vegaPaths, registryPassphrase)
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

		registry.Ethereum = &registryloader.RegisteredEthereumWallet{
			Type: registryloader.EthereumWalletTypeClef,
			Details: registryloader.EthereumClefWallet{
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

		registry.Ethereum = &registryloader.RegisteredEthereumWallet{
			Type: registryloader.EthereumWalletTypeKeyStore,
			Details: registryloader.EthereumKeyStoreWallet{
				Name:       w.Name(),
				Passphrase: walletPassphrase,
			},
		}
	}

	if err := registryLoader.SaveRegistry(registry, registryPassphrase); err != nil {
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
	registryLoader, err := registryloader.New(vegaPaths, registryPassphrase)
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

		registry.Ethereum = &registryloader.RegisteredEthereumWallet{
			Type: registryloader.EthereumWalletTypeClef,
			Details: registryloader.EthereumClefWallet{
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

		registry.Ethereum = &registryloader.RegisteredEthereumWallet{
			Type: registryloader.EthereumWalletTypeKeyStore,
			Details: registryloader.EthereumKeyStoreWallet{
				Name:       w.Name(),
				Passphrase: walletPassphrase,
			},
		}
	}

	if err := registryLoader.SaveRegistry(registry, registryPassphrase); err != nil {
		return nil, fmt.Errorf("couldn't save registry: %w", err)
	}

	data["registryFilePath"] = registryLoader.RegistryFilePath()
	return data, nil
}
