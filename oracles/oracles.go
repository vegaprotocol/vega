package oracles

import (
	"errors"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

var (
	// ErrNilOracle signals that the oracle to instantiate was nil
	ErrNilOracle = errors.New("nil oracle")
	// ErrUnimplementedOracle signals that the oracle specified
	// is still not implemented by the market framework
	ErrUnimplementedOracle = errors.New("unimplemented oracle")
)

// Oracle is an abstraction of an oracle
type Oracle interface {
	SettlementPrice() (uint64, error)
}

// New instantiate a new oracle from a market framework configuration
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
