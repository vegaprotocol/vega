package oracles

import (
	types "code.vegaprotocol.io/vega/proto"
)

// EthereumEvent represent an ethereum event from the network
type EthereumEvent struct {
	ContractID string
	Event      string
}

func newEthereumEvent(pee *types.EthereumEvent) (*EthereumEvent, error) {
	return &EthereumEvent{
		ContractID: pee.ContractID,
		Event:      pee.Event,
	}, nil
}

// SettlementPrice returns the price communicated by the
// network for the given asset
func (e *EthereumEvent) SettlementPrice() (uint64, error) {
	return 42, nil
}
