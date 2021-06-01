package liquidity

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/monitor/liquidity AuctionState
type AuctionState interface {
	IsLiquidityAuction() bool
	StartLiquidityAuction(t time.Time, d *types.AuctionDuration)
	EndAuction()
	InAuction() bool
	ExtendAuctionLiquidity(delta types.AuctionDuration)
	ExpiresAt() *time.Time
}

// TargetStakeCalculator interface
//go:generate go run github.com/golang/mock/mockgen -destination mocks/target_stake_calculator_mock.go -package mocks code.vegaprotocol.io/vega/monitor/liquidity TargetStakeCalculator
type TargetStakeCalculator interface {
	GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice uint64, trades []*types.Trade) float64
}

type Engine struct {
	mu          *sync.Mutex
	params      *types.LiquidityMonitoringParameters
	minDuration time.Duration
	tsCalc      TargetStakeCalculator
}

func NewMonitor(tsCalc TargetStakeCalculator, params *types.LiquidityMonitoringParameters) *Engine {
	// temp hard-coded duration of 1 until we can make these parameters required
	if params.AuctionExtension == 0 {
		params.AuctionExtension = 1
	}
	e := &Engine{
		mu:     &sync.Mutex{},
		params: params,
		tsCalc: tsCalc,
	}
	if e.minDuration < 1 {
		e.minDuration = time.Second
	}
	return e
}

func (e *Engine) SetMinDuration(d time.Duration) {
	e.mu.Lock()
	e.minDuration = d
	e.mu.Unlock()
}

func (e *Engine) UpdateTargetStakeTriggerRatio(ctx context.Context, ratio float64) {
	e.mu.Lock()
	e.params.TriggeringRatio = ratio
	// @TODO emit event
	e.mu.Unlock()
}

// CheckLiquidity Starts or Ends a Liquidity auction given the current and target stakes along with best static bid and ask volumes.
// The constant c1 represents the netparam `MarketLiquidityTargetStakeTriggeringRatio`.
func (e *Engine) CheckLiquidity(as AuctionState, t time.Time, currentStake float64, trades []*types.Trade, rf types.RiskFactor, markPrice uint64, bestStaticBidVolume, bestStaticAskVolume uint64) {
	exp := as.ExpiresAt()
	if exp != nil && exp.After(t) {
		// we're in auction, and the auction isn't expiring yet, so we don't have to do anything yet
		return
	}
	e.mu.Lock()
	c1 := e.params.TriggeringRatio
	md := int64(e.minDuration / time.Second)
	e.mu.Unlock()
	targetStake := e.tsCalc.GetTheoreticalTargetStake(rf, t, markPrice, trades)
	ext := types.AuctionDuration{
		Duration: e.params.AuctionExtension,
	}
	// if we're in liquidity auction already, the auction has expired, and we can end/extend the auction
	// @TODO we don't have the ability to support volume limited auctions yet
	if exp != nil && as.IsLiquidityAuction() {
		if currentStake >= targetStake && bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
			as.EndAuction()
			return // all done
		}
		// we're still in trouble, extend the auction
		as.ExtendAuctionLiquidity(ext)
		return
	}
	if currentStake < (targetStake*c1) || bestStaticBidVolume == 0 || bestStaticAskVolume == 0 {
		if exp != nil {
			as.ExtendAuctionLiquidity(ext)
			return
		}
		as.StartLiquidityAuction(t, &types.AuctionDuration{
			Duration: md, // we multiply this by a second later on
		})
	}
}
