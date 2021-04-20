package mocks

import (
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/nodewallet"
)

//go:generate go run github.com/golang/mock/mockgen -destination node_wallet_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks NodeWallet
type NodeWallet interface {
	Get(chain nodewallet.Blockchain) (nodewallet.Wallet, bool)
}

//go:generate go run github.com/golang/mock/mockgen -destination broker_mock.go -package mocks code.vegaprotocol.io/vega/cmd/vegabenchmark/mocks Broker
type Broker interface {
	Send(e events.Event)
}
