// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package liquidity

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/auction_state_mock.go -package mocks code.vegaprotocol.io/vega/core/monitor/liquidity AuctionState
type AuctionState interface {
	IsOpeningAuction() bool
	IsLiquidityAuction() bool
	IsLiquidityExtension() bool
	StartLiquidityAuction(t time.Time, d *types.AuctionDuration, trigger types.AuctionTrigger)
	SetReadyToLeave()
	InAuction() bool
	ExtendAuctionLiquidity(delta types.AuctionDuration, trigger types.AuctionTrigger)
	ExpiresAt() *time.Time
}

// TargetStakeCalculator interface
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/target_stake_calculator_mock.go -package mocks code.vegaprotocol.io/vega/core/monitor/liquidity TargetStakeCalculator
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
	e := &Engine{
		mu:     &sync.Mutex{},
		tsCalc: tsCalc,
	}
	e.UpdateParameters(params)
	if e.minDuration < 1 {
		e.minDuration = time.Second
	}
	return e
}

func (e *Engine) UpdateParameters(params *types.LiquidityMonitoringParameters) {
	// temp hard-coded duration of 1 until we can make these parameters required
	if params.AuctionExtension == 0 {
		params.AuctionExtension = 1
	}
	e.params = params
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
// The constant c1 represents the netparam `MarketLiquidityTargetStakeTriggeringRatio`,
// "true" gets returned if non-persistent order should be rejected.
func (e *Engine) CheckLiquidity(as AuctionState, t time.Time, currentStake *num.Uint, trades []*types.Trade,
	rf types.RiskFactor, refPrice *num.Uint, bestStaticBidVolume, bestStaticAskVolume uint64, persistent bool,
) bool {
	exp := as.ExpiresAt()
	if exp != nil && exp.After(t) {
		// we're in auction, and the auction isn't expiring yet, so we don't have to do anything yet
		return false
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

	isOpening := as.IsOpeningAuction()
	if exp != nil && as.IsLiquidityAuction() || as.IsLiquidityExtension() || isOpening {
		if currentStake.GTE(targetStake) && bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
			as.SetReadyToLeave()
			return false // all done
		}
		// we're still in trouble, extend the auction
		trigger := types.AuctionTriggerUnableToDeployLPOrders
		if bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
			trigger = types.AuctionTriggerLiquidityTargetNotMet
		}
		as.ExtendAuctionLiquidity(ext, trigger)
		return false
	}
	// multiply target stake by triggering ratio
	scaledTargetStakeDec := targetStake.ToDecimal().Mul(c1)
	scaledTargetStake, _ := num.UintFromDecimal(scaledTargetStakeDec)
	stakeUndersupplied := currentStake.LT(scaledTargetStake)
	if stakeUndersupplied || bestStaticBidVolume == 0 || bestStaticAskVolume == 0 {
		if stakeUndersupplied && len(trades) > 0 && !persistent {
			// non-persistent order cannot trigger auction by raising target stake
			// we're going to stay in continuous trading
			return true
		}
		if exp != nil {
			trigger := types.AuctionTriggerUnableToDeployLPOrders
			if bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
				trigger = types.AuctionTriggerLiquidityTargetNotMet
			}
			as.ExtendAuctionLiquidity(ext, trigger)

			return false
		}
		trigger := types.AuctionTriggerUnableToDeployLPOrders
		if bestStaticBidVolume > 0 && bestStaticAskVolume > 0 {
			trigger = types.AuctionTriggerLiquidityTargetNotMet
		}
		as.StartLiquidityAuction(t, &types.AuctionDuration{
			Duration: md, // we multiply this by a second later on
		}, trigger)
	}
	return false
}
