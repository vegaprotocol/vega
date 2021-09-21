package faucet

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/shared/paths"
)

var (
	ErrFaucetConfigAlreadyExists = errors.New("faucet configuration already exists")
)

type InitialisationResult struct {
	Wallet         *WalletGenerationResult
	ConfigFilePath string
}

func Initialise(vegaPaths paths.Paths, passphrase string, rewrite bool) (*InitialisationResult, error) {
	walletLoader, err := InitialiseWalletLoader(vegaPaths)
	if err != nil {
		return nil, fmt.Errorf("couldn't initialise faucet wallet loader: %w", err)
	}

	walletGenResult, err := walletLoader.GenerateWallet(passphrase)
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
