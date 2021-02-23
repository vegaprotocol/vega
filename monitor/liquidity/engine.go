package liquidity

import (
	"time"

	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/monitor/liquidity AuctionState
type AuctionState interface {
	IsLiquidityAuction() bool
	StartLiquidityAuction(t time.Time, d *types.AuctionDuration)
	EndAuction()
	InAuction() bool
	ExtendAuction(delta types.AuctionDuration)
}

type Engine struct{}

func NewMonitor() *Engine {
	return &Engine{}
}

// CheckLiquidity Starts or Ends a Liquidity auction given the current and target stakes along with best static bid and ask volumes.
// The constant c1 represents the netparam `MarketLiquidityTargetStakeTriggeringRatio`.
func (e *Engine) CheckLiquidity(as AuctionState, t time.Time, c1, current, target float64, bestStaticBidVolume, bestStaticAskVolume uint64) {
	if as.IsLiquidityAuction() {
		if current >= target && bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
			as.EndAuction()
		}
	} else {
		if current < (target*c1) || bestStaticBidVolume == 0 || bestStaticAskVolume == 0 {
			if as.InAuction() {
				as.ExtendAuction(types.AuctionDuration{})
			} else {
				as.StartLiquidityAuction(t, nil)
			}

		}
	}
}
