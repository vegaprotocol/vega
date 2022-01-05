package validators

import (
	"code.vegaprotocol.io/vega/nodewallets"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_wallet_mock.go -package mocks code.vegaprotocol.io/vega/validators NodeWallets
type NodeWallets interface {
	GetVega() Wallet
	GetTendermintPubkey() string
	GetEthereumAddress() string
}

type NodeWalletsWrapper struct {
	*nodewallets.NodeWallets
}

func WrapNodeWallets(nw *nodewallets.NodeWallets) *NodeWalletsWrapper {
	return &NodeWalletsWrapper{nw}
}

func (w *NodeWalletsWrapper) GetVega() Wallet {
	return w.Vega
}

func (w *NodeWalletsWrapper) GetEthereumAddress() string {
	return w.Ethereum.PubKey().Hex()
}

func (w *NodeWalletsWrapper) GetTendermintPubkey() string {
	return w.Tendermint.Pubkey
}
