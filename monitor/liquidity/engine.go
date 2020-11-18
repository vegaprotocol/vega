package liquidity

import (
	"errors"

	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrInvalidParmaterC1 is thrown then the c1 parameter is outside of the expected range.
	ErrInvalidParmaterC1 = errors.New("the c1 paramter needs to be in the (0,1) range")
)

type AuctionState interface {
	InAuction() bool
	IsLiquidityAuction() bool
	StartLiquidityAuction()
	ExtendAuction()
	EndAuction()
}

type Engine struct {
	c1       float64
	target   TargetStakeCalculator
	supplied SuppliedStakeCalculator
}

type TargetStakeCalculator interface {
	Calculate(types.Order) float64
}

type SuppliedStakeCalculator interface {
	Calculate(types.Order) float64
}

// NewMonitor returns a new instance of the liquidity monitoring engine if all parameters are successfully validated and an error otherwise
func NewMonitor(c1 float64, target TargetStakeCalculator, supplied SuppliedStakeCalculator) (*Engine, error) {
	if c1 <= 0 || c1 >= 1 {
		return nil, ErrInvalidParmaterC1
	}
	return &Engine{
		c1:       c1,
		target:   target,
		supplied: supplied,
	}, nil
}

func (e *Engine) CheckOrder(order types.Order, as AuctionState) {

	target := e.target.Calculate(order)
	supplied := e.supplied.Calculate(order)

	//not enough liquidity
	if supplied < e.c1*target {
		if as.InAuction() {
			//TODO: it just needs to be extended until this module says its fine to finish
			as.ExtendAuction()
		} else {
			as.StartLiquidityAuction()
		}
	}

	//if in liquidity auction
	if as.IsLiquidityAuction() && supplied >= target {
		as.EndAuction()
	}
}
