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
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/execution/amm"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/liquidation"
	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/liquidity/target"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/markets"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/monitor"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/settlement"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"golang.org/x/exp/maps"
)

func NewMarketFromSnapshot(
	ctx context.Context,
	log *logging.Logger,
	em *types.ExecMarket,
	riskConfig risk.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	matchingConfig matching.Config,
	feeConfig fee.Config,
	liquidityConfig liquidity.Config,
	collateralEngine common.Collateral,
	oracleEngine products.OracleEngine,
	timeService common.TimeService,
	broker common.Broker,
	stateVarEngine common.StateVarEngine,
	assetDetails *assets.Asset,
	marketActivityTracker *common.MarketActivityTracker,
	peggedOrderNotify func(int64),
	referralDiscountRewardService fee.ReferralDiscountRewardService,
	volumeDiscountService fee.VolumeDiscountService,
	banking common.Banking,
) (*Market, error) {
	mkt := em.Market

	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(mkt.PositionDecimalPlaces))
	if len(em.Market.ID) == 0 {
		return nil, common.ErrEmptyMarketID
	}

	assetDecimals := assetDetails.DecimalPlaces()
	tradableInstrument, err := markets.NewTradableInstrumentFromSnapshot(ctx, log, mkt.TradableInstrument, em.Market.ID,
		timeService, oracleEngine, broker, em.Product, uint32(assetDecimals))
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate a new market: %w", err)
	}

	as := monitor.NewAuctionStateFromSnapshot(mkt, em.AuctionState)

	if vgcontext.InProgressUpgradeFrom(ctx, "v0.73.13") {
		// protocol upgrade from v0.73.12, lets populate the new liquidity-fee-settings with a default marginal-cost method
		log.Info("migrating liquidity fee settings for existing market", logging.String("mid", mkt.ID))
		mkt.Fees.LiquidityFeeSettings = &types.LiquidityFeeSettings{
			Method: types.LiquidityFeeMethodMarginalCost,
		}

		// if the market is in a none opening-auction we need to tell the instrument
		if as.InAuction() && !as.IsOpeningAuction() {
			tradableInstrument.Instrument.Product.UpdateAuctionState(ctx, true)
		}
	}

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewCachedOrderBook(
		log, matchingConfig, mkt.ID, as.InAuction(), peggedOrderNotify)
	asset := tradableInstrument.Instrument.Product.GetAsset()

	// this needs to stay
	riskEngine := risk.NewEngine(log,
		riskConfig,
		tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel,
		book,
		as,
		timeService,
		broker,
		mkt.ID,
		asset,
		stateVarEngine,
		positionFactor,
		em.RiskFactorConsensusReached,
		&types.RiskFactor{Market: mkt.ID, Short: em.ShortRiskFactor, Long: em.LongRiskFactor},
		mkt.LinearSlippageFactor,
		mkt.QuadraticSlippageFactor,
	)

	settleEngine := settlement.NewSnapshotEngine(
		log,
		settlementConfig,
		tradableInstrument.Instrument.Product,
		mkt.ID,
		timeService,
		broker,
		positionFactor,
	)
	positionEngine := positions.NewSnapshotEngine(log, positionConfig, mkt.ID, broker)

	var feeEngine *fee.Engine
	if em.FeesStats != nil {
		feeEngine, err = fee.NewFromState(log, feeConfig, *mkt.Fees, asset, positionFactor, em.FeesStats)
		if err != nil {
			return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
		}
	} else {
		feeEngine, err = fee.New(log, feeConfig, *mkt.Fees, asset, positionFactor)
		if err != nil {
			return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
		}
	}

	tsCalc := target.NewSnapshotEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, positionEngine, mkt.ID, positionFactor)

	pMonitor, err := price.NewMonitorFromSnapshot(mkt.ID, asset, em.PriceMonitor, mkt.PriceMonitoringSettings, tradableInstrument.RiskModel, as, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}

	exp := assetDecimals - mkt.DecimalPlaces
	priceFactor := num.UintZero().Exp(num.NewUint(10), num.NewUint(exp))

	// TODO(jeremy): remove this once the upgrade with the .73 have run on mainnet
	// this is required to support the migration to SLA liquidity
	if !(mkt.LiquiditySLAParams != nil) {
		mkt.LiquiditySLAParams = ptr.From(liquidity.DefaultSLAParameters)
	}

	liquidityEngine := liquidity.NewSnapshotEngine(
		liquidityConfig, log, timeService, broker, tradableInstrument.RiskModel,
		pMonitor, book, as, asset, mkt.ID, stateVarEngine, positionFactor, mkt.LiquiditySLAParams)
	equityShares := common.NewEquitySharesFromSnapshot(em.EquityShare)

	// just check for nil first just in case we are on a protocol upgrade from a version were AMM were not supported.
	// @TODO pass in AMM
	marketLiquidity := common.NewMarketLiquidity(
		log, liquidityEngine, collateralEngine, broker, book, equityShares, marketActivityTracker,
		feeEngine, common.FutureMarketType, mkt.ID, asset, priceFactor, mkt.LiquiditySLAParams.PriceRange, nil,
	)
	if err := marketLiquidity.SetState(em.MarketLiquidity); err != nil {
		return nil, fmt.Errorf("unable to restore the market liquidity state: %w", err)
	}

	// backward compatibility check for nil
	stopOrders := stoporders.New(log)
	if em.StopOrders != nil {
		stopOrders = stoporders.NewFromProto(log, em.StopOrders)
	} else {
		// use the last markPrice for the market to initialise stopOrders price
		if em.LastTradedPrice != nil {
			stopOrders.PriceUpdated(em.LastTradedPrice.Clone())
		}
	}

	expiringStopOrders := common.NewExpiringOrders()
	if em.ExpiringStopOrders != nil {
		expiringStopOrders = common.NewExpiringOrdersFromState(em.ExpiringStopOrders)
	}
	// @TODO same as in the non-snapshot market constructor: default to legacy liquidation strategy for the time being
	// this can be removed once this parameter is no longer optional
	if mkt.LiquidationStrategy == nil {
		mkt.LiquidationStrategy = liquidation.GetLegacyStrat()
	}
	le := liquidation.New(log, mkt.LiquidationStrategy, mkt.GetID(), broker, book, as, timeService, marketLiquidity, positionEngine)

	partyMargin := make(map[string]num.Decimal, len(em.PartyMarginFactors))
	for _, pmf := range em.PartyMarginFactors {
		partyMargin[pmf.Party], _ = num.DecimalFromString(pmf.MarginFactor)
	}

	now := timeService.GetTimeNow()
	marketType := mkt.MarketType()

	markPriceCalculator := common.NewCompositePriceCalculatorFromSnapshot(ctx, em.CurrentMarkPrice, timeService, oracleEngine, em.MarkPriceCalculator)
	market := &Market{
		log:                           log,
		mkt:                           mkt,
		closingAt:                     time.Unix(0, mkt.MarketTimestamps.Close),
		timeService:                   timeService,
		matching:                      book,
		tradableInstrument:            tradableInstrument,
		risk:                          riskEngine,
		position:                      positionEngine,
		settlement:                    settleEngine,
		collateral:                    collateralEngine,
		broker:                        broker,
		fee:                           feeEngine,
		referralDiscountRewardService: referralDiscountRewardService,
		volumeDiscountService:         volumeDiscountService,
		liquidityEngine:               liquidityEngine,
		liquidity:                     marketLiquidity,
		parties:                       map[string]struct{}{},
		tsCalc:                        tsCalc,
		feeSplitter:                   common.NewFeeSplitterFromSnapshot(em.FeeSplitter, now),
		as:                            as,
		pMonitor:                      pMonitor,
		peggedOrders:                  common.NewPeggedOrdersFromSnapshot(log, timeService, em.PeggedOrders),
		expiringOrders:                common.NewExpiringOrdersFromState(em.ExpiringOrders),
		equityShares:                  equityShares,
		lastBestBidPrice:              em.LastBestBid.Clone(),
		lastBestAskPrice:              em.LastBestAsk.Clone(),
		lastMidBuyPrice:               em.LastMidBid.Clone(),
		lastMidSellPrice:              em.LastMidAsk.Clone(),
		lastTradedPrice:               em.LastTradedPrice,
		priceFactor:                   priceFactor,
		lastMarketValueProxy:          em.LastMarketValueProxy,
		marketActivityTracker:         marketActivityTracker,
		positionFactor:                positionFactor,
		stateVarEngine:                stateVarEngine,
		settlementDataInMarket:        em.SettlementData,
		settlementAsset:               asset,
		stopOrders:                    stopOrders,
		expiringStopOrders:            expiringStopOrders,
		perp:                          marketType == types.MarketTypePerp,
		partyMarginFactor:             partyMargin,
		liquidation:                   le,
		banking:                       banking,
		markPriceCalculator:           markPriceCalculator,
	}

	markPriceCalculator.SetOraclePriceScalingFunc(market.scaleOracleData)

	if em.InternalCompositePriceCalculator != nil {
		market.internalCompositePriceCalculator = common.NewCompositePriceCalculatorFromSnapshot(ctx, nil, timeService, oracleEngine, em.InternalCompositePriceCalculator)
		market.internalCompositePriceCalculator.SetOraclePriceScalingFunc(market.scaleOracleData)
	}

	// just check for nil first just in case we are on a protocol upgrade from a version were AMM were not supported.
	if em.Amm == nil {
		market.amm = amm.New(log, broker, collateralEngine, market, market.risk, market.position, market.priceFactor, positionFactor)
	} else {
		market.amm = amm.NewFromProto(log, broker, collateralEngine, market, market.risk, market.position, em.Amm, market.priceFactor, positionFactor)
	}
	// now we can set the AMM on the market liquidity engine.
	market.liquidity.SetAMM(market.amm)

	book.SetOffbookSource(market.amm)

	// now set AMM engine on liquidity market.
	market.liquidity.SetAMM(market.amm)

	for _, p := range em.Parties {
		market.parties[p] = struct{}{}
	}

	market.assetDP = uint32(assetDecimals)
	switch marketType {
	case types.MarketTypeFuture:
		market.tradableInstrument.Instrument.Product.NotifyOnTradingTerminated(market.tradingTerminated)
		market.tradableInstrument.Instrument.Product.NotifyOnSettlementData(market.settlementData)
	case types.MarketTypePerp:
		market.tradableInstrument.Instrument.Product.NotifyOnSettlementData(market.settlementDataPerp)
	case types.MarketTypeSpot:
	default:
		log.Panic("unexpected market type", logging.Int("type", int(marketType)))
	}

	if em.SettlementData != nil {
		// ensure oracle has the settlement data
		market.tradableInstrument.Instrument.Product.RestoreSettlementData(em.SettlementData.Clone())
	}

	liquidityEngine.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)

	if mkt.State == types.MarketStateTradingTerminated {
		market.tradableInstrument.Instrument.UnsubscribeTradingTerminated(ctx)
	}

	if em.Closed {
		market.closed = true
		market.tradableInstrument.Instrument.Unsubscribe(ctx)
		market.markPriceCalculator.Close(ctx)
		if market.internalCompositePriceCalculator != nil {
			market.internalCompositePriceCalculator.Close(ctx)
		}
		stateVarEngine.UnregisterStateVariable(asset, mkt.ID)
	}
	return market, nil
}

func (m *Market) GetNewStateProviders() []types.StateProvider {
	return []types.StateProvider{
		m.position, m.matching, m.tsCalc,
		m.liquidityEngine.V1StateProvider(), m.liquidityEngine.V2StateProvider(),
		m.settlement, m.liquidation,
	}
}

func (m *Market) GetState() *types.ExecMarket {
	rf := m.risk.GetRiskFactors()
	var sp *num.Numeric
	if m.settlementDataInMarket != nil {
		sp = m.settlementDataInMarket.Clone()
	}

	parties := maps.Keys(m.parties)
	sort.Strings(parties)
	assetQuantum, _ := m.collateral.GetAssetQuantum(m.settlementAsset)

	partyMarginFactors := make([]*snapshot.PartyMarginFactor, 0, len(m.partyMarginFactor))
	for k, d := range m.partyMarginFactor {
		partyMarginFactors = append(partyMarginFactors, &snapshot.PartyMarginFactor{Party: k, MarginFactor: d.String()})
	}
	sort.Slice(partyMarginFactors, func(i, j int) bool {
		return partyMarginFactors[i].Party < partyMarginFactors[j].Party
	})

	em := &types.ExecMarket{
		Market:                         m.mkt.DeepClone(),
		PriceMonitor:                   m.pMonitor.GetState(),
		AuctionState:                   m.as.GetState(),
		PeggedOrders:                   m.peggedOrders.GetState(),
		ExpiringOrders:                 m.expiringOrders.GetState(),
		LastBestBid:                    m.lastBestBidPrice.Clone(),
		LastBestAsk:                    m.lastBestAskPrice.Clone(),
		LastMidBid:                     m.lastMidBuyPrice.Clone(),
		LastMidAsk:                     m.lastMidSellPrice.Clone(),
		LastMarketValueProxy:           m.lastMarketValueProxy,
		LastTradedPrice:                m.lastTradedPrice,
		EquityShare:                    m.equityShares.GetState(),
		RiskFactorConsensusReached:     m.risk.IsRiskFactorInitialised(),
		ShortRiskFactor:                rf.Short,
		LongRiskFactor:                 rf.Long,
		FeeSplitter:                    m.feeSplitter.GetState(),
		SettlementData:                 sp,
		NextMTM:                        m.nextMTM.UnixNano(),
		NextInternalCompositePriceCalc: m.nextInternalCompositePriceCalc.UnixNano(),
		Parties:                        parties,
		Closed:                         m.closed,
		IsSucceeded:                    m.succeeded,
		StopOrders:                     m.stopOrders.ToProto(),
		ExpiringStopOrders:             m.expiringStopOrders.GetState(),
		Product:                        m.tradableInstrument.Instrument.Product.Serialize(),
		FeesStats:                      m.fee.GetState(assetQuantum),
		PartyMarginFactors:             partyMarginFactors,
		MarkPriceCalculator:            m.markPriceCalculator.IntoProto(),
		Amm:                            m.amm.IntoProto(),
		MarketLiquidity:                m.liquidity.GetState(),
	}
	if m.perp && m.internalCompositePriceCalculator != nil {
		em.InternalCompositePriceCalculator = m.internalCompositePriceCalculator.IntoProto()
	}

	return em
}
