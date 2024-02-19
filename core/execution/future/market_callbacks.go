// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package future

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func (m *Market) OnAMMMinCommitmentQuantumUpdate(ctx context.Context, c *num.Uint) {
	m.amm.OnMinCommitmentQuantumUpdate(ctx, c)
}

func (m *Market) OnMarketMinLpStakeQuantumMultipleUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnMinLPStakeQuantumMultiple(d)
}

func (m *Market) OnMarketMinProbabilityOfTradingLPOrdersUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnMinProbabilityOfTradingLPOrdersUpdate(d)
}

func (m *Market) OnMarginScalingFactorsUpdate(ctx context.Context, sf *types.ScalingFactors) error {
	if err := m.risk.OnMarginScalingFactorsUpdate(sf); err != nil {
		return err
	}

	// update our market definition, and dispatch update through the event bus
	m.mkt.TradableInstrument.MarginCalculator.ScalingFactors = sf
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
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

func (m *Market) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(d num.Decimal) {
	m.liquidity.OnMaximumLiquidityFeeFactorLevelUpdate(d)
}

func (m *Market) OnMarketProbabilityOfTradingTauScalingUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnProbabilityOfTradingTauScalingUpdate(d)
}

func (m *Market) OnMarketAuctionMinimumDurationUpdate(ctx context.Context, d time.Duration) {
	m.pMonitor.SetMinDuration(d)
	m.minDuration = d
	evt := m.as.UpdateMinDuration(ctx, d)
	// we were in an auction, and the duration of the auction was updated
	if evt != nil {
		m.broker.Send(evt)
	}
}

func (m *Market) OnMarketAuctionMaximumDurationUpdate(ctx context.Context, d time.Duration) {
	if m.mkt.State == types.MarketStatePending || m.mkt.State == types.MarketStateProposed {
		m.as.UpdateMaxDuration(ctx, d)
	}
}

func (m *Market) OnMarkPriceUpdateMaximumFrequency(ctx context.Context, d time.Duration) {
	if !m.nextMTM.IsZero() {
		m.nextMTM = m.nextMTM.Add(-m.mtmDelta)
	}
	m.nextMTM = m.nextMTM.Add(d)
	m.mtmDelta = d
}

func (m *Market) OnInternalCompositePriceUpdateFrequency(ctx context.Context, d time.Duration) {
	if !m.perp {
		return
	}
	if !m.nextInternalCompositePriceCalc.IsZero() {
		m.nextInternalCompositePriceCalc = m.nextInternalCompositePriceCalc.Add(-m.mtmDelta)
	}
	m.nextInternalCompositePriceCalc = m.nextInternalCompositePriceCalc.Add(d)
	m.internalCompositePriceFrequency = d
}

func (m *Market) OnMarketPartiesMaximumStopOrdersUpdate(ctx context.Context, u *num.Uint) {
	m.maxStopOrdersPerParties = u.Clone()
}

func (m *Market) OnMarketLiquidityV2BondPenaltyFactorUpdate(liquidityV2BondPenaltyFactor num.Decimal) {
	m.bondPenaltyFactor = liquidityV2BondPenaltyFactor

	m.liquidity.OnBondPenaltyFactorUpdate(liquidityV2BondPenaltyFactor)
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

func (m *Market) OnMarketLiquidityV2StakeToCCYVolume(d num.Decimal) {
	m.liquidity.OnStakeToCcyVolumeUpdate(d)
}

func (m *Market) OnMarketLiquidityV2ProvidersFeeCalculationTimeStep(d time.Duration) {
	m.liquidity.OnProvidersFeeCalculationTimeStep(d)
}

func (m *Market) OnMarketLiquidityEquityLikeShareFeeFractionUpdate(d num.Decimal) {
	m.liquidity.SetELSFeeFraction(d)
}
