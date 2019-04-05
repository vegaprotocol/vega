package oracles

import (
	types "code.vegaprotocol.io/vega/proto"
)

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

func (e *EthereumEvent) SettlementPrice() (uint64, error) {
	return 42, nil
}
