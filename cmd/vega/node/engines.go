package node

import (
	"code.vegaprotocol.io/vega/evtforward"
	evtfwdeth "code.vegaprotocol.io/vega/evtforward/ethereum"
	"code.vegaprotocol.io/vega/types"
)

type EventForwarderEngine interface {
	ReloadConf(evtforward.Config)
	StartEthereumEngine(evtfwdeth.Client, evtfwdeth.Forwarder, evtfwdeth.Config, *types.EthereumConfig) error
	Stop()
}
