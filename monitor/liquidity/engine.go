package liquidity

import (
	"context"
	"sync"
	"time"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/monitor/liquidity AuctionState
type AuctionState interface {
	IsLiquidityAuction() bool
	StartLiquidityAuction(t time.Time, d *types.AuctionDuration)
	SetReadyToLeave()
	InAuction() bool
	ExtendAuctionLiquidity(delta types.AuctionDuration)
	ExpiresAt() *time.Time
}

// TargetStakeCalculator interface
//go:generate go run github.com/golang/mock/mockgen -destination mocks/target_stake_calculator_mock.go -package mocks code.vegaprotocol.io/vega/monitor/liquidity TargetStakeCalculator
type TargetStakeCalculator interface {
	GetTheoreticalTargetStake(rf types.RiskFactor, now time.Time, markPrice *num.Uint, trades []*types.Trade) *num.Uint
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

func (e *Engine) UpdateParameters(parameters *types.LiquidityMonitoringParameters) {
	e.params = parameters
}

func (e *Engine) SetMinDuration(d time.Duration) {
	e.mu.Lock()
	e.minDuration = d
	e.mu.Unlock()
}

func (e *Engine) UpdateTargetStakeTriggerRatio(ctx context.Context, ratio num.Decimal) {
	e.mu.Lock()
	e.params.TriggeringRatio = ratio
	// @TODO emit event
	e.mu.Unlock()
}

// CheckLiquidity Starts or Ends a Liquidity auction given the current and target stakes along with best static bid and ask volumes.
// The constant c1 represents the netparam `MarketLiquidityTargetStakeTriggeringRatio`.
func (e *Engine) CheckLiquidity(as AuctionState, t time.Time, currentStake *num.Uint, trades []*types.Trade,
	rf types.RiskFactor, refPrice *num.Uint, bestStaticBidVolume, bestStaticAskVolume uint64, persistent bool) error {
	exp := as.ExpiresAt()
	if exp != nil && exp.After(t) {
		// we're in auction, and the auction isn't expiring yet, so we don't have to do anything yet
		return nil
	}
	e.mu.Lock()
	c1 := e.params.TriggeringRatio
	md := int64(e.minDuration / time.Second)
	e.mu.Unlock()
	targetStake := e.tsCalc.GetTheoreticalTargetStake(rf, t, refPrice.Clone(), trades)
	ext := types.AuctionDuration{
		Duration: e.params.AuctionExtension,
	}
	// if we're in liquidity auction already, the auction has expired, and we can end/extend the auction
	// @TODO we don't have the ability to support volume limited auctions yet

	if exp != nil && as.IsLiquidityAuction() {
		if currentStake.GTE(targetStake) && bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
			as.SetReadyToLeave()
			return nil // all done
		}
		// we're still in trouble, extend the auction
		as.ExtendAuctionLiquidity(ext)
		return nil
	}
	// multiply target stake by triggering ratio
	scaledTargetStakeDec := targetStake.ToDecimal().Mul(c1)
	scaledTargetStake, _ := num.UintFromDecimal(scaledTargetStakeDec)
	stakeUndersupplied := currentStake.LT(scaledTargetStake)
	if stakeUndersupplied || bestStaticBidVolume == 0 || bestStaticAskVolume == 0 {
		if stakeUndersupplied && trades != nil && !persistent {
			// non-persistent order cannot trigger auction by raising target stake
			// we're going to stay in continuous trading
			return proto.OrderError_ORDER_ERROR_INVALID_PERSISTENCE
		}
		if exp != nil {
			as.ExtendAuctionLiquidity(ext)

			return nil
		}
		as.StartLiquidityAuction(t, &types.AuctionDuration{
			Duration: md, // we multiply this by a second later on
		})
	}
	return nil

	//TODO: Don't leave ANY auction type if supplied < target, only
}
