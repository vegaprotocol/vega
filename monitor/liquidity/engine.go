package liquidity

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/monitor/liquidity AuctionState
type AuctionState interface {
	IsLiquidityAuction() bool
	StartLiquidityAuction(t time.Time, d *types.AuctionDuration)
	AuctionEnd() bool
}

type Engine struct{}

func NewMonitor() *Engine {
	return nil
}

// CheckTarget Starts of Ends a Liquidity auction given the current and target stakes.
// The constant c1 represents the netparam `MarketLiquidityTargetStakeTriggeringRatio`.
func (e *Engine) CheckTarget(as AuctionState, t time.Time, c1, current, target float64) {
	if as.IsLiquidityAuction() {
		if current >= target {
			as.AuctionEnd()
		}
	} else {
		if current < (target * c1) {
			as.StartLiquidityAuction(t, nil)
		}
	}
}
