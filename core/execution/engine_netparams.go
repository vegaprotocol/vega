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

package execution

import (
	"context"
	"errors"
	"time"

	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/future"
	"code.vegaprotocol.io/vega/core/execution/spot"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

type netParamsValues struct {
	feeDistributionTimeStep               time.Duration
	marketValueWindowLength               time.Duration
	suppliedStakeToObligationFactor       num.Decimal
	infrastructureFee                     num.Decimal
	makerFee                              num.Decimal
	treasuryFee                           num.Decimal
	buyBackFee                            num.Decimal
	scalingFactors                        *types.ScalingFactors
	maxLiquidityFee                       num.Decimal
	bondPenaltyFactor                     num.Decimal
	auctionMinDuration                    time.Duration
	auctionMaxDuration                    time.Duration
	probabilityOfTradingTauScaling        num.Decimal
	minProbabilityOfTradingLPOrders       num.Decimal
	minLpStakeQuantumMultiple             num.Decimal
	marketCreationQuantumMultiple         num.Decimal
	markPriceUpdateMaximumFrequency       time.Duration
	internalCompositePriceUpdateFrequency time.Duration
	marketPartiesMaximumStopOrdersUpdate  *num.Uint

	// Liquidity version 2.
	liquidityV2BondPenaltyFactor                 num.Decimal
	liquidityV2EarlyExitPenalty                  num.Decimal
	liquidityV2MaxLiquidityFee                   num.Decimal
	liquidityV2SLANonPerformanceBondPenaltyMax   num.Decimal
	liquidityV2SLANonPerformanceBondPenaltySlope num.Decimal
	liquidityV2StakeToCCYVolume                  num.Decimal
	liquidityV2ProvidersFeeCalculationTimeStep   time.Duration
	liquidityELSFeeFraction                      num.Decimal

	// AMM
	ammCommitmentQuantum *num.Uint
	ammCalculationLevels *num.Uint

	// only used for protocol upgrade to v0.74
	chainID uint64

	// network wide auction duration
	lbadTable *types.LongBlockAuctionDurationTable
}

func defaultNetParamsValues() netParamsValues {
	return netParamsValues{
		feeDistributionTimeStep:         -1,
		marketValueWindowLength:         -1,
		suppliedStakeToObligationFactor: num.DecimalFromInt64(-1),
		infrastructureFee:               num.DecimalFromInt64(-1),
		makerFee:                        num.DecimalFromInt64(-1),
		buyBackFee:                      num.DecimalFromInt64(-1),
		treasuryFee:                     num.DecimalFromInt64(-1),
		scalingFactors:                  nil,
		maxLiquidityFee:                 num.DecimalFromInt64(-1),
		bondPenaltyFactor:               num.DecimalFromInt64(-1),

		auctionMinDuration:                    -1,
		probabilityOfTradingTauScaling:        num.DecimalFromInt64(-1),
		minProbabilityOfTradingLPOrders:       num.DecimalFromInt64(-1),
		minLpStakeQuantumMultiple:             num.DecimalFromInt64(-1),
		marketCreationQuantumMultiple:         num.DecimalFromInt64(-1),
		markPriceUpdateMaximumFrequency:       5 * time.Second, // default is 5 seconds, should come from net params though
		internalCompositePriceUpdateFrequency: 5 * time.Second,
		marketPartiesMaximumStopOrdersUpdate:  num.UintZero(),

		// Liquidity version 2.
		liquidityV2BondPenaltyFactor:                 num.DecimalFromInt64(-1),
		liquidityV2EarlyExitPenalty:                  num.DecimalFromInt64(-1),
		liquidityV2MaxLiquidityFee:                   num.DecimalFromInt64(-1),
		liquidityV2SLANonPerformanceBondPenaltyMax:   num.DecimalFromInt64(-1),
		liquidityV2SLANonPerformanceBondPenaltySlope: num.DecimalFromInt64(-1),
		liquidityV2StakeToCCYVolume:                  num.DecimalFromInt64(-1),
		liquidityV2ProvidersFeeCalculationTimeStep:   time.Second * 5,

		ammCommitmentQuantum: num.UintZero(),
		ammCalculationLevels: num.NewUint(100),
	}
}

func (e *Engine) OnMarketAuctionMinimumDurationUpdate(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, d)
	}
	e.npv.auctionMinDuration = d
	return nil
}

func (e *Engine) OnMarketAuctionMaximumDurationUpdate(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.allMarketsCpy {
		if mkt.IsOpeningAuction() {
			mkt.OnMarketAuctionMaximumDurationUpdate(ctx, d)
		}
	}
	e.npv.auctionMaxDuration = d
	return nil
}

func (e *Engine) OnMarkPriceUpdateMaximumFrequency(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, d)
	}
	e.npv.markPriceUpdateMaximumFrequency = d
	return nil
}

func (e *Engine) OnInternalCompositePriceUpdateFrequency(ctx context.Context, d time.Duration) error {
	for _, mkt := range e.futureMarkets {
		mkt.OnInternalCompositePriceUpdateFrequency(ctx, d)
	}
	e.npv.internalCompositePriceUpdateFrequency = d
	return nil
}

// OnMarketLiquidityV2BondPenaltyUpdate stores net param on execution engine and applies to markets at the start of new epoch.
func (e *Engine) OnMarketLiquidityV2BondPenaltyUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market liquidity bond penalty (liquidity v2)",
			logging.Decimal("bond-penalty-factor", d),
		)
	}

	// Set immediately during opening auction
	for _, mkt := range e.allMarketsCpy {
		if mkt.IsOpeningAuction() {
			mkt.OnMarketLiquidityV2BondPenaltyFactorUpdate(d)
		}
	}

	e.npv.liquidityV2BondPenaltyFactor = d
	return nil
}

// OnMarketLiquidityV2EarlyExitPenaltyUpdate stores net param on execution engine and applies to markets
// at the start of new epoch.
func (e *Engine) OnMarketLiquidityV2EarlyExitPenaltyUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market liquidity early exit penalty (liquidity v2)",
			logging.Decimal("early-exit-penalty", d),
		)
	}

	// Set immediately during opening auction
	for _, mkt := range e.allMarketsCpy {
		if mkt.IsOpeningAuction() {
			mkt.OnMarketLiquidityV2EarlyExitPenaltyUpdate(d)
		}
	}

	e.npv.liquidityV2EarlyExitPenalty = d
	return nil
}

// OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate stores net param on execution engine and
// applies at the start of new epoch.
func (e *Engine) OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity provision max liquidity fee factor (liquidity v2)",
			logging.Decimal("max-liquidity-fee", d),
		)
	}

	// Set immediately during opening auction
	for _, mkt := range e.allMarketsCpy {
		if mkt.IsOpeningAuction() {
			mkt.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(d)
		}
	}

	e.npv.liquidityV2MaxLiquidityFee = d
	return nil
}

// OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate stores net param on execution engine and applies to markets at the
// start of new epoch.
func (e *Engine) OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market SLA non performance bond penalty slope (liquidity v2)",
			logging.Decimal("bond-penalty-slope", d),
		)
	}

	// Set immediately during opening auction
	for _, mkt := range e.allMarketsCpy {
		if mkt.IsOpeningAuction() {
			mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(d)
		}
	}

	e.npv.liquidityV2SLANonPerformanceBondPenaltySlope = d
	return nil
}

// OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate stores net param on execution engine and applies to markets
// at the start of new epoch.
func (e *Engine) OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market SLA non performance bond penalty max (liquidity v2)",
			logging.Decimal("bond-penalty-max", d),
		)
	}

	for _, m := range e.futureMarketsCpy {
		// Set immediately during opening auction
		if m.IsOpeningAuction() {
			m.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(d)
		}
	}

	e.npv.liquidityV2SLANonPerformanceBondPenaltyMax = d
	return nil
}

// OnMarketLiquidityV2StakeToCCYVolumeUpdate stores net param on execution engine and applies to markets
// at the start of new epoch.
func (e *Engine) OnMarketLiquidityV2StakeToCCYVolumeUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market stake to CCYVolume (liquidity v2)",
			logging.Decimal("stake-to-ccy-volume", d),
		)
	}

	for _, m := range e.futureMarketsCpy {
		// Set immediately during opening auction
		if m.IsOpeningAuction() {
			m.OnMarketLiquidityV2StakeToCCYVolume(d)
		}
	}

	e.npv.liquidityV2StakeToCCYVolume = d
	return nil
}

// OnMarketLiquidityV2ProvidersFeeCalculationTimeStep stores net param on execution engine and applies to markets
// at the start of new epoch.
func (e *Engine) OnMarketLiquidityV2ProvidersFeeCalculationTimeStep(_ context.Context, d time.Duration) error {
	if e.log.IsDebug() {
		e.log.Debug("update market SLA providers fee calculation time step (liquidity v2)",
			logging.Duration("providersFeeCalculationTimeStep", d),
		)
	}

	for _, m := range e.allMarketsCpy {
		// Set immediately during opening auction
		if m.IsOpeningAuction() {
			m.OnMarketLiquidityV2ProvidersFeeCalculationTimeStep(d)
		}
	}

	e.npv.liquidityV2ProvidersFeeCalculationTimeStep = d
	return nil
}

func (e *Engine) OnNetworkWideAuctionDurationUpdated(ctx context.Context, v interface{}) error {
	if e.log.IsDebug() {
		e.log.Debug("update network wide auction duration",
			logging.Reflect("network-wide-auction-duration", v),
		)
	}
	lbadTable, ok := v.(*vega.LongBlockAuctionDurationTable)
	if !ok {
		return errors.New("invalid long block auction duration table")
	}
	lbads, err := types.LongBlockAuctionDurationTableFromProto(lbadTable)
	if err != nil {
		return err
	}
	e.npv.lbadTable = lbads
	return nil
}

func (e *Engine) OnMarketMarginScalingFactorsUpdate(ctx context.Context, v interface{}) error {
	if e.log.IsDebug() {
		e.log.Debug("update market scaling factors",
			logging.Reflect("scaling-factors", v),
		)
	}

	pscalingFactors, ok := v.(*vega.ScalingFactors)
	if !ok {
		return errors.New("invalid types for Margin ScalingFactors")
	}
	scalingFactors := types.ScalingFactorsFromProto(pscalingFactors)
	for _, mkt := range e.futureMarketsCpy {
		if err := mkt.OnMarginScalingFactorsUpdate(ctx, scalingFactors); err != nil {
			return err
		}
	}
	e.npv.scalingFactors = scalingFactors
	return nil
}

func (e *Engine) OnMarketFeeFactorsMakerFeeUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update maker fee in market fee factors",
			logging.Decimal("maker-fee", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnFeeFactorsMakerFeeUpdate(ctx, d)
	}
	e.npv.makerFee = d
	return nil
}

func (e *Engine) OnMarketFeeFactorsTreasuryFeeUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update treasury fee in market fee factors",
			logging.Decimal("treasury-fee", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnFeeFactorsTreasuryFeeUpdate(ctx, d)
	}
	e.npv.treasuryFee = d
	return nil
}

func (e *Engine) OnMarketFeeFactorsBuyBackFeeUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update buy back fee in market fee factors",
			logging.Decimal("buy-back-fee", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnFeeFactorsBuyBackFeeUpdate(ctx, d)
	}
	e.npv.buyBackFee = d
	return nil
}

func (e *Engine) OnMarketFeeFactorsInfrastructureFeeUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update infrastructure fee in market fee factors",
			logging.Decimal("infrastructure-fee", d),
		)
	}
	for _, mkt := range e.allMarketsCpy {
		mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, d)
	}
	e.npv.infrastructureFee = d
	return nil
}

func (e *Engine) OnMarketValueWindowLengthUpdate(_ context.Context, d time.Duration) error {
	if e.log.IsDebug() {
		e.log.Debug("update market value window length",
			logging.Duration("window-length", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketValueWindowLengthUpdate(d)
	}
	e.npv.marketValueWindowLength = d
	return nil
}

// to be removed and replaced by its v2 counterpart. in use only for future.
func (e *Engine) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update liquidity provision max liquidity fee factor",
			logging.Decimal("max-liquidity-fee", d),
		)
	}

	for _, mkt := range e.futureMarketsCpy {
		mkt.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(d)
	}
	e.npv.maxLiquidityFee = d

	return nil
}

func (e *Engine) OnMarketLiquidityEquityLikeShareFeeFractionUpdate(_ context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market liquidity equityLikeShareFeeFraction",
			logging.Decimal("market.liquidity.equityLikeShareFeeFraction", d),
		)
	}
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketLiquidityEquityLikeShareFeeFractionUpdate(d)
	}
	e.npv.liquidityELSFeeFraction = d
	return nil
}

func (e *Engine) OnMarketProbabilityOfTradingTauScalingUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update probability of trading tau scaling",
			logging.Decimal("probability-of-trading-tau-scaling", d),
		)
	}
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketProbabilityOfTradingTauScalingUpdate(ctx, d)
	}
	e.npv.probabilityOfTradingTauScaling = d
	return nil
}

func (e *Engine) OnMarketMinProbabilityOfTradingForLPOrdersUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update min probability of trading tau scaling",
			logging.Decimal("min-probability-of-trading-lp-orders", d),
		)
	}

	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketMinProbabilityOfTradingLPOrdersUpdate(ctx, d)
	}
	e.npv.minProbabilityOfTradingLPOrders = d
	return nil
}

func (e *Engine) OnMinLpStakeQuantumMultipleUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update min lp stake quantum multiple",
			logging.Decimal("min-lp-stake-quantum-multiple", d),
		)
	}
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketMinLpStakeQuantumMultipleUpdate(ctx, d)
	}
	e.npv.minLpStakeQuantumMultiple = d
	return nil
}

func (e *Engine) OnMarketCreationQuantumMultipleUpdate(ctx context.Context, d num.Decimal) error {
	if e.log.IsDebug() {
		e.log.Debug("update market creation quantum multiple",
			logging.Decimal("market-creation-quantum-multiple", d),
		)
	}
	e.npv.marketCreationQuantumMultiple = d
	return nil
}

func (e *Engine) OnMarketPartiesMaximumStopOrdersUpdate(ctx context.Context, u *num.Uint) error {
	if e.log.IsDebug() {
		e.log.Debug("update market parties maxiumum stop orders",
			logging.BigUint("value", u),
		)
	}
	e.npv.marketPartiesMaximumStopOrdersUpdate = u
	for _, mkt := range e.allMarketsCpy {
		mkt.OnMarketPartiesMaximumStopOrdersUpdate(ctx, u)
	}
	return nil
}

func (e *Engine) OnMaxPeggedOrderUpdate(ctx context.Context, max *num.Uint) error {
	if e.log.IsDebug() {
		e.log.Debug("update max pegged orders",
			logging.Uint64("max-pegged-orders", max.Uint64()),
		)
	}
	e.maxPeggedOrders = max.Uint64()
	return nil
}

func (e *Engine) OnMarketAMMMinCommitmentQuantum(ctx context.Context, c *num.Uint) error {
	if e.log.IsDebug() {
		e.log.Debug("update amm min commitment quantum",
			logging.BigUint("commitment-quantum", c),
		)
	}
	e.npv.ammCommitmentQuantum = c
	for _, m := range e.allMarketsCpy {
		m.OnAMMMinCommitmentQuantumUpdate(ctx, c.Clone())
	}

	return nil
}

func (e *Engine) OnMarketAMMMaxCalculationLevels(ctx context.Context, c *num.Uint) error {
	if e.log.IsDebug() {
		e.log.Debug("update amm max calculation levels",
			logging.BigUint("ccalculation-levels", c),
		)
	}
	e.npv.ammCalculationLevels = c
	for _, m := range e.allMarketsCpy {
		m.OnMarketAMMMaxCalculationLevels(ctx, c.Clone())
	}
	return nil
}

func (e *Engine) propagateSpotInitialNetParams(ctx context.Context, mkt *spot.Market, isRestore bool) error {
	if !e.npv.minLpStakeQuantumMultiple.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketMinLpStakeQuantumMultipleUpdate(ctx, e.npv.minLpStakeQuantumMultiple)
	}
	if e.npv.auctionMinDuration != -1 {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, e.npv.auctionMinDuration)
	}
	if e.npv.auctionMaxDuration > 0 {
		mkt.OnMarketAuctionMaximumDurationUpdate(ctx, e.npv.auctionMaxDuration)
	}
	if !e.npv.infrastructureFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, e.npv.infrastructureFee)
	}

	if !e.npv.makerFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsMakerFeeUpdate(ctx, e.npv.makerFee)
	}

	if !e.npv.buyBackFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsBuyBackFeeUpdate(ctx, e.npv.buyBackFee)
	}

	if !e.npv.treasuryFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsTreasuryFeeUpdate(ctx, e.npv.treasuryFee)
	}

	if e.npv.marketValueWindowLength != -1 {
		mkt.OnMarketValueWindowLengthUpdate(e.npv.marketValueWindowLength)
	}

	if e.npv.markPriceUpdateMaximumFrequency > 0 {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, e.npv.markPriceUpdateMaximumFrequency)
	}

	if !e.npv.liquidityV2EarlyExitPenalty.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2EarlyExitPenaltyUpdate(e.npv.liquidityV2EarlyExitPenalty)
	}

	if !e.npv.liquidityV2MaxLiquidityFee.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(e.npv.liquidityV2MaxLiquidityFee)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltySlope.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltySlope)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltyMax.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltyMax)
	}

	if !e.npv.liquidityV2StakeToCCYVolume.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2StakeToCCYVolume(e.npv.liquidityV2StakeToCCYVolume)
	}

	mkt.OnMarketPartiesMaximumStopOrdersUpdate(ctx, e.npv.marketPartiesMaximumStopOrdersUpdate)
	mkt.OnMinimalHoldingQuantumMultipleUpdate(e.minHoldingQuantumMultiplier)

	e.propagateSLANetParams(ctx, mkt, isRestore)

	if !e.npv.liquidityELSFeeFraction.IsZero() {
		mkt.OnMarketLiquidityEquityLikeShareFeeFractionUpdate(e.npv.liquidityELSFeeFraction)
	}
	return nil
}

func (e *Engine) propagateInitialNetParamsToFutureMarket(ctx context.Context, mkt *future.Market, isRestore bool) error {
	if !e.npv.probabilityOfTradingTauScaling.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketProbabilityOfTradingTauScalingUpdate(ctx, e.npv.probabilityOfTradingTauScaling)
	}
	if !e.npv.minProbabilityOfTradingLPOrders.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketMinProbabilityOfTradingLPOrdersUpdate(ctx, e.npv.minProbabilityOfTradingLPOrders)
	}
	if !e.npv.minLpStakeQuantumMultiple.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketMinLpStakeQuantumMultipleUpdate(ctx, e.npv.minLpStakeQuantumMultiple)
	}
	if e.npv.auctionMinDuration != -1 {
		mkt.OnMarketAuctionMinimumDurationUpdate(ctx, e.npv.auctionMinDuration)
	}
	if e.npv.auctionMaxDuration > 0 {
		mkt.OnMarketAuctionMaximumDurationUpdate(ctx, e.npv.auctionMaxDuration)
	}

	if !e.npv.infrastructureFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsInfrastructureFeeUpdate(ctx, e.npv.infrastructureFee)
	}

	if !e.npv.makerFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsMakerFeeUpdate(ctx, e.npv.makerFee)
	}

	if !e.npv.buyBackFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsBuyBackFeeUpdate(ctx, e.npv.buyBackFee)
	}

	if !e.npv.treasuryFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnFeeFactorsTreasuryFeeUpdate(ctx, e.npv.treasuryFee)
	}

	if e.npv.scalingFactors != nil {
		if err := mkt.OnMarginScalingFactorsUpdate(ctx, e.npv.scalingFactors); err != nil {
			return err
		}
	}

	if e.npv.marketValueWindowLength != -1 {
		mkt.OnMarketValueWindowLengthUpdate(e.npv.marketValueWindowLength)
	}

	if !e.npv.maxLiquidityFee.Equal(num.DecimalFromInt64(-1)) {
		mkt.OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(e.npv.maxLiquidityFee)
	}
	if e.npv.markPriceUpdateMaximumFrequency > 0 {
		mkt.OnMarkPriceUpdateMaximumFrequency(ctx, e.npv.markPriceUpdateMaximumFrequency)
	}
	if e.npv.internalCompositePriceUpdateFrequency > 0 {
		mkt.OnInternalCompositePriceUpdateFrequency(ctx, e.npv.internalCompositePriceUpdateFrequency)
	}
	if !e.npv.liquidityELSFeeFraction.IsZero() {
		mkt.OnMarketLiquidityEquityLikeShareFeeFractionUpdate(e.npv.liquidityELSFeeFraction)
	}

	mkt.OnMarketPartiesMaximumStopOrdersUpdate(ctx, e.npv.marketPartiesMaximumStopOrdersUpdate)
	mkt.OnMinimalMarginQuantumMultipleUpdate(e.minMaintenanceMarginQuantumMultiplier)

	mkt.OnAMMMinCommitmentQuantumUpdate(ctx, e.npv.ammCommitmentQuantum)
	mkt.OnMarketAMMMaxCalculationLevels(ctx, e.npv.ammCalculationLevels)

	e.propagateSLANetParams(ctx, mkt, isRestore)

	return nil
}

func (e *Engine) propagateSLANetParams(_ context.Context, mkt common.CommonMarket, isRestore bool) {
	if !e.npv.liquidityV2BondPenaltyFactor.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2BondPenaltyFactorUpdate(e.npv.liquidityV2BondPenaltyFactor)
	}

	if !e.npv.liquidityV2EarlyExitPenalty.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2EarlyExitPenaltyUpdate(e.npv.liquidityV2EarlyExitPenalty)
	}

	if !e.npv.liquidityV2MaxLiquidityFee.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2MaximumLiquidityFeeFactorLevelUpdate(e.npv.liquidityV2MaxLiquidityFee)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltySlope.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltySlopeUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltySlope)
	}

	if !e.npv.liquidityV2SLANonPerformanceBondPenaltyMax.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2SLANonPerformanceBondPenaltyMaxUpdate(e.npv.liquidityV2SLANonPerformanceBondPenaltyMax)
	}

	if !e.npv.liquidityV2StakeToCCYVolume.Equal(num.DecimalFromInt64(-1)) { //nolint:staticcheck
		mkt.OnMarketLiquidityV2StakeToCCYVolume(e.npv.liquidityV2StakeToCCYVolume)
	}

	if !isRestore && e.npv.liquidityV2ProvidersFeeCalculationTimeStep != 0 {
		mkt.OnMarketLiquidityV2ProvidersFeeCalculationTimeStep(e.npv.liquidityV2ProvidersFeeCalculationTimeStep)
	}
}
