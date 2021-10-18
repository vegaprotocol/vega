package execution

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func (m *Market) OnMarketMinProbabilityOfTradingLPOrdersUpdate(_ context.Context, f float64) {
	m.liquidity.OnMinProbabilityOfTradingLPOrdersUpdate(num.DecimalFromFloat(f))
}

func (m *Market) BondPenaltyFactorUpdate(ctx context.Context, v float64) {
	m.bondPenaltyFactor = num.DecimalFromFloat(v)
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

func (m *Market) OnFeeFactorsMakerFeeUpdate(ctx context.Context, f float64) error {
	mf := num.DecimalFromFloat(f)
	m.fee.OnFeeFactorsMakerFeeUpdate(mf)
	m.mkt.Fees.Factors.MakerFee = mf
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) OnFeeFactorsInfrastructureFeeUpdate(ctx context.Context, f float64) error {
	inf := num.DecimalFromFloat(f)
	m.fee.OnFeeFactorsInfrastructureFeeUpdate(inf)
	m.mkt.Fees.Factors.InfrastructureFee = inf
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) OnSuppliedStakeToObligationFactorUpdate(v float64) {
	m.liquidity.OnSuppliedStakeToObligationFactorUpdate(num.DecimalFromFloat(v))
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

func (m *Market) OnMarketTargetStakeScalingFactorUpdate(v float64) error {
	return m.tsCalc.UpdateScalingFactor(num.DecimalFromFloat(v))
}

func (m *Market) OnMarketLiquidityProvisionShapesMaxSizeUpdate(v int64) error {
	return m.liquidity.OnMarketLiquidityProvisionShapesMaxSizeUpdate(v)
}

func (m *Market) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(v float64) {
	m.liquidity.OnMaximumLiquidityFeeFactorLevelUpdate(num.DecimalFromFloat(v))
}

func (m *Market) OnMarketProbabilityOfTradingTauScalingUpdate(_ context.Context, v float64) {
	m.liquidity.OnProbabilityOfTradingTauScalingUpdate(num.DecimalFromFloat(v))
}

func (m *Market) OnMarketLiquidityTargetStakeTriggeringRatio(ctx context.Context, v float64) {
	m.lMonitor.UpdateTargetStakeTriggerRatio(ctx, num.DecimalFromFloat(v))
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
