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

package spot

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/libs/num"
)

func (m *Market) OnMarketMinLpStakeQuantumMultipleUpdate(_ context.Context, d num.Decimal) {
	m.minLPStakeQuantumMultiple = d
	m.liquidity.OnMinLPStakeQuantumMultiple((d))
}

func (m *Market) OnMarketMinProbabilityOfTradingLPOrdersUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnMinProbabilityOfTradingLPOrdersUpdate(d)
}

func (m *Market) OnMarketProbabilityOfTradingTauScalingUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnProbabilityOfTradingTauScalingUpdate(d)
}

func (m *Market) OnFeeFactorsMakerFeeUpdate(ctx context.Context, d num.Decimal) {
	m.fee.OnFeeFactorsMakerFeeUpdate(d)
	m.mkt.Fees.Factors.MakerFee = d
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
}

func (m *Market) OnFeeFactorsInfrastructureFeeUpdate(ctx context.Context, d num.Decimal) {
	m.fee.OnFeeFactorsInfrastructureFeeUpdate(d)
	m.mkt.Fees.Factors.InfrastructureFee = d
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
}

func (m *Market) OnMarketValueWindowLengthUpdate(d time.Duration) {
	m.marketValueWindowLength = d
}

func (m *Market) OnMarketTargetStakeTimeWindowUpdate(d time.Duration) {
	m.tsCalc.UpdateTimeWindow(d)
}

func (m *Market) OnMarketTargetStakeScalingFactorUpdate(d num.Decimal) error {
	return m.tsCalc.UpdateScalingFactor(d)
}

func (m *Market) OnMarketAuctionMinimumDurationUpdate(ctx context.Context, d time.Duration) {
	m.minDuration = d
	m.pMonitor.SetMinDuration(d)
	evt := m.as.UpdateMinDuration(ctx, d)
	// we were in an auction, and the duration of the auction was updated
	if evt != nil {
		m.broker.Send(evt)
	}
}

func (m *Market) OnMarkPriceUpdateMaximumFrequency(ctx context.Context, d time.Duration) {
	if !m.nextMTM.IsZero() {
		m.nextMTM = m.nextMTM.Add(-m.mtmDelta)
	}
	m.nextMTM = m.nextMTM.Add(d)
	m.mtmDelta = d
}

func (m *Market) OnMarketPartiesMaximumStopOrdersUpdate(ctx context.Context, u *num.Uint) {
	m.maxStopOrdersPerParties = u.Clone()
}

func (m *Market) OnMarketLiquidityV2EarlyExitPenaltyUpdate(d num.Decimal) {
	m.liquidity.OnEarlyExitPenalty(d)
}

func (m *Market) OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(d num.Decimal) {
	m.liquidity.OnMaximumLiquidityFeeFactorLevelUpdate(d)
}

func (m *Market) OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(d num.Decimal) {
	m.liquidity.OnNonPerformanceBondPenaltySlopeUpdate(d)
}

func (m *Market) OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(d num.Decimal) {
	m.liquidity.OnNonPerformanceBondPenaltyMaxUpdate(d)
}

func (m *Market) OnMarketLiquidityV2SuppliedStakeToObligationFactorUpdate(d num.Decimal) {
	m.liquidity.OnStakeToCcyVolumeUpdate(d)
}
