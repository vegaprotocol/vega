package oracles

import (
	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

// EthereumEvent represent an ethereum event from the network
type EthereumEvent struct {
	ContractID string
	Event      string
	Value      uint64
}

func newEthereumEvent(pee *types.EthereumEvent) (*EthereumEvent, error) {
	return &EthereumEvent{
		ContractID: pee.ContractID,
		Event:      pee.Event,
		Value:      pee.Value,
	}, nil
}

// SettlementPrice returns the price communicated by the
// network for the given asset
func (e *EthereumEvent) SettlementPrice() (uint64, error) {
	return e.Value, nil
}
