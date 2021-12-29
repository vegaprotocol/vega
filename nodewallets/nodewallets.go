package nodewallets

import (
	"errors"

	"code.vegaprotocol.io/vega/nodewallets/eth"
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
}

func (w *NodeWallets) Verify() error {
	if w.Vega == nil {
		return ErrVegaWalletIsMissing
	}
	if w.Ethereum == nil {
		return ErrEthereumWalletIsMissing
	}
	if w.Tendermint == nil {
		return ErrEthereumWalletIsMissing
	}
	return nil
}
