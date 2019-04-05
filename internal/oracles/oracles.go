package oracles

import (
	"errors"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrNilOracle           = errors.New("nil oracle")
	ErrUnimplementedOracle = errors.New("unimplemented oracle")
)

type Oracle interface {
	SettlementPrice() (uint64, error)
}

func New(po interface{}) (Oracle, error) {
	if po == nil {
		return nil, ErrNilOracle
	}

	switch o := po.(type) {
	case *types.Future_EthereumEvent:
		return newEthereumEvent(o.EthereumEvent)
	default:
		return nil, ErrNilOracle
	}
}
