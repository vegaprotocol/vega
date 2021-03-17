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

// TargetStakeCalculator interface
//go:generate go run github.com/golang/mock/mockgen -destination mocks/target_stake_calculator_mock.go -package mocks code.vegaprotocol.io/vega/monitor/liquidity TargetStakeCalculator
type TargetStakeCalculator interface {
	GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice uint64, trades []*types.Trade) float64
}

type Engine struct {
	tsCalc TargetStakeCalculator
}

func NewMonitor(tsCalc TargetStakeCalculator) *Engine {
	return &Engine{
		tsCalc: tsCalc,
	}
}

// CheckLiquidity Starts or Ends a Liquidity auction given the current and target stakes along with best static bid and ask volumes.
// The constant c1 represents the netparam `MarketLiquidityTargetStakeTriggeringRatio`.
func (e *Engine) CheckLiquidity(as AuctionState, t time.Time, c1, currentStake float64, trades []*types.Trade, rf types.RiskFactor, markPrice uint64, bestStaticBidVolume, bestStaticAskVolume uint64) {
	targetStake := e.tsCalc.GetTheoreticalTargetStake(rf, t, markPrice, trades)
	if as.IsLiquidityAuction() {
		if currentStake >= targetStake && bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
			as.EndAuction()
		}
	} else {
		if currentStake < (targetStake*c1) || bestStaticBidVolume == 0 || bestStaticAskVolume == 0 {
			if as.InAuction() {
				as.ExtendAuction(types.AuctionDuration{})
			} else {
				as.StartLiquidityAuction(t, nil)
			}

		}
	}
}
