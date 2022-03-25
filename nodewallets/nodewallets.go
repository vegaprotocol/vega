package nodewallets

import (
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/nodewallets/eth"
	"code.vegaprotocol.io/vega/nodewallets/registryloader"
	"code.vegaprotocol.io/vega/nodewallets/vega"
)

var (
	ErrVegaWalletIsMissing       = errors.New("the Vega node wallet is missing")
	ErrEthereumWalletIsMissing   = errors.New("the Ethereum node wallet is missing")
	ErrTendermintPubkeyIsMissing = errors.New("the Tendermint pubkey is missing")
)

type TendermintPubkey struct {
	Pubkey string
}

type NodeWallets struct {
	Vega       *vega.Wallet
	Ethereum   *eth.Wallet
	Tendermint *TendermintPubkey

	registryLoader     *registryloader.RegistryLoader
	registryPassphrase string // @TODO - do not leave this here.. Maybe get it as argument?
}

func (w *NodeWallets) ReloadEthereum() error {
	registry, err := w.registryLoader.GetRegistry(w.registryPassphrase)
	if err != nil {
		return fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	return w.Ethereum.Reload(registry.Ethereum.Details)
}

func (w *NodeWallets) ReloadVega() error {
	registry, err := w.registryLoader.GetRegistry(w.registryPassphrase)
	if err != nil {
		return fmt.Errorf("couldn't load node wallet registry: %v", err)
	}

	return w.Vega.Reload(*registry.Vega)
}

func (w *NodeWallets) Verify() error {
	if w.Vega == nil {
		return ErrVegaWalletIsMissing
	}
	if w.Ethereum == nil {
		return ErrEthereumWalletIsMissing
	}
	if w.Tendermint == nil {
		return ErrTendermintPubkeyIsMissing
	}
	return nil
}
