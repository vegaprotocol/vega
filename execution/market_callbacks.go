package execution

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func (m *Market) OnMarketMinProbabilityOfTradingLPOrdersUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnMinProbabilityOfTradingLPOrdersUpdate(d)
}

func (m *Market) BondPenaltyFactorUpdate(ctx context.Context, d num.Decimal) {
	m.bondPenaltyFactor = d
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

func (m *Market) OnFeeFactorsMakerFeeUpdate(ctx context.Context, d num.Decimal) error {
	m.fee.OnFeeFactorsMakerFeeUpdate(d)
	m.mkt.Fees.Factors.MakerFee = d
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) OnFeeFactorsInfrastructureFeeUpdate(ctx context.Context, d num.Decimal) error {
	m.fee.OnFeeFactorsInfrastructureFeeUpdate(d)
	m.mkt.Fees.Factors.InfrastructureFee = d
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) OnSuppliedStakeToObligationFactorUpdate(d num.Decimal) {
	m.liquidity.OnSuppliedStakeToObligationFactorUpdate(d)
}

func (m *Market) OnMarketValueWindowLengthUpdate(d time.Duration) {
	m.marketValueWindowLength = d
}

func (m *Market) OnMarketLiquidityProvidersFeeDistribitionTimeStep(d time.Duration) {
	m.lpFeeDistributionTimeStep = d
}

func (m *Market) OnMarketTargetStakeTimeWindowUpdate(d time.Duration) {
	m.tsCalc.UpdateTimeWindow(d)
}

func (m *Market) OnMarketTargetStakeScalingFactorUpdate(d num.Decimal) error {
	return m.tsCalc.UpdateScalingFactor(d)
}

func (m *Market) OnMarketLiquidityProvisionShapesMaxSizeUpdate(v int64) error {
	return m.liquidity.OnMarketLiquidityProvisionShapesMaxSizeUpdate(v)
}

func (m *Market) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(d num.Decimal) {
	m.liquidity.OnMaximumLiquidityFeeFactorLevelUpdate(d)
}

func (m *Market) OnMarketProbabilityOfTradingTauScalingUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnProbabilityOfTradingTauScalingUpdate(d)
}

func (m *Market) OnMarketLiquidityTargetStakeTriggeringRatio(ctx context.Context, d num.Decimal) {
	m.lMonitor.UpdateTargetStakeTriggerRatio(ctx, d)
	// TODO: Send an event containing updated parameter
}

func (m *Market) OnMarketAuctionMinimumDurationUpdate(ctx context.Context, d time.Duration) {
	m.pMonitor.SetMinDuration(d)
	m.lMonitor.SetMinDuration(d)
	evt := m.as.UpdateMinDuration(ctx, d)
	// we were in an auction, and the duration of the auction was updated
	if evt != nil {
		m.broker.Send(evt)
	}
}
