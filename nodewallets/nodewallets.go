package nodewallet

import (
	"errors"

	"code.vegaprotocol.io/vega/nodewallets/eth"
	"code.vegaprotocol.io/vega/nodewallets/vega"
)

var (
	ErrVegaWalletIsMissing     = errors.New("the Vega node wallet is missing")
	ErrEthereumWalletIsMissing = errors.New("the Ethereum node wallet is missing")
)

type NodeWallets struct {
	Vega     *vega.Wallet
	Ethereum *eth.Wallet
}

func (w *NodeWallets) Verify() error {
	if w.Vega == nil {
		return ErrVegaWalletIsMissing
	}
	if w.Ethereum == nil {
		return ErrEthereumWalletIsMissing
	}
	return nil
}
