package protocol

import (
	"code.vegaprotocol.io/vega/evtforward"
	evtfwdeth "code.vegaprotocol.io/vega/evtforward/ethereum"
	"code.vegaprotocol.io/vega/types"
)

type EventForwarderEngine interface {
	ReloadConf(evtforward.Config)
	SetupEthereumEngine(evtfwdeth.Client, evtfwdeth.Forwarder, evtfwdeth.Config, *types.EthereumConfig, evtfwdeth.Assets) error
	Start()
	Stop()

	// methods used to update starting blocks of the eef
	UpdateStakingStartingBlock(uint64)
	UpdateMultisigControlStartingBlock(uint64)
}
