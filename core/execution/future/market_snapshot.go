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

package future

import (
	"context"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/liquidity"
	"code.vegaprotocol.io/vega/core/liquidity/target"
	"code.vegaprotocol.io/vega/core/markets"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/monitor"
	lmon "code.vegaprotocol.io/vega/core/monitor/liquidity"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/positions"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/settlement"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
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
) (*Market, error) {
	mkt := em.Market
	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(mkt.PositionDecimalPlaces))
	if len(em.Market.ID) == 0 {
		return nil, common.ErrEmptyMarketID
	}

	tradableInstrument, err := markets.NewTradableInstrument(ctx, log, mkt.TradableInstrument, oracleEngine)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate a new market: %w", err)
	}

	as := monitor.NewAuctionStateFromSnapshot(mkt, em.AuctionState)

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

	feeEngine, err := fee.New(log, feeConfig, *mkt.Fees, asset, positionFactor)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
	}

	tsCalc := target.NewSnapshotEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, positionEngine, mkt.ID, positionFactor)

	pMonitor, err := price.NewMonitorFromSnapshot(mkt.ID, asset, em.PriceMonitor, mkt.PriceMonitoringSettings, tradableInstrument.RiskModel, as, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}

	exp := assetDetails.DecimalPlaces() - mkt.DecimalPlaces
	priceFactor := num.UintZero().Exp(num.NewUint(10), num.NewUint(exp))
	lMonitor := lmon.NewMonitor(tsCalc, mkt.LiquidityMonitoringParameters)

	liqEngine := liquidity.NewSnapshotEngine(liquidityConfig, log, timeService, broker, tradableInstrument.RiskModel, pMonitor, book, asset, mkt.ID, stateVarEngine, priceFactor.Clone(), positionFactor)

	now := timeService.GetTimeNow()
	market := &Market{
		log:                        log,
		mkt:                        mkt,
		closingAt:                  time.Unix(0, mkt.MarketTimestamps.Close),
		timeService:                timeService,
		matching:                   book,
		tradableInstrument:         tradableInstrument,
		risk:                       riskEngine,
		position:                   positionEngine,
		settlement:                 settleEngine,
		collateral:                 collateralEngine,
		broker:                     broker,
		fee:                        feeEngine,
		liquidity:                  liqEngine,
		parties:                    map[string]struct{}{},
		lMonitor:                   lMonitor,
		tsCalc:                     tsCalc,
		feeSplitter:                common.NewFeeSplitterFromSnapshot(em.FeeSplitter, now),
		as:                         as,
		pMonitor:                   pMonitor,
		peggedOrders:               common.NewPeggedOrdersFromSnapshot(log, timeService, em.PeggedOrders),
		expiringOrders:             common.NewExpiringOrdersFromState(em.ExpiringOrders),
		equityShares:               common.NewEquitySharesFromSnapshot(em.EquityShare),
		lastBestBidPrice:           em.LastBestBid.Clone(),
		lastBestAskPrice:           em.LastBestAsk.Clone(),
		lastMidBuyPrice:            em.LastMidBid.Clone(),
		lastMidSellPrice:           em.LastMidAsk.Clone(),
		markPrice:                  em.CurrentMarkPrice.Clone(),
		lastTradedPrice:            em.LastTradedPrice.Clone(),
		priceFactor:                priceFactor,
		lastMarketValueProxy:       em.LastMarketValueProxy,
		lastEquityShareDistributed: time.Unix(0, em.LastEquityShareDistributed),
		marketActivityTracker:      marketActivityTracker,
		positionFactor:             positionFactor,
		stateVarEngine:             stateVarEngine,
		settlementDataInMarket:     em.SettlementData,
		lpPriceRange:               mkt.LPPriceRange,
		linearSlippageFactor:       mkt.LinearSlippageFactor,
		quadraticSlippageFactor:    mkt.QuadraticSlippageFactor,
		settlementAsset:            asset,
	}

	for _, p := range em.Parties {
		market.parties[p] = struct{}{}
	}

	market.assetDP = uint32(assetDetails.DecimalPlaces())
	market.tradableInstrument.Instrument.Product.NotifyOnTradingTerminated(market.tradingTerminated)
	market.tradableInstrument.Instrument.Product.NotifyOnSettlementData(market.settlementData)
	if em.SettlementData != nil {
		// ensure oracle has the settlement data
		market.tradableInstrument.Instrument.Product.RestoreSettlementData(em.SettlementData.Clone())
	}
	liqEngine.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)

	if mkt.State == types.MarketStateTradingTerminated {
		market.tradableInstrument.Instrument.Product.UnsubscribeTradingTerminated(ctx)
	}

	if em.Closed {
		market.closed = true
		stateVarEngine.UnregisterStateVariable(asset, mkt.ID)
	}
	return market, nil
}

func (m *Market) GetNewStateProviders() []types.StateProvider {
	return []types.StateProvider{m.position, m.matching, m.tsCalc, m.liquidity, m.settlement}
}

func (m *Market) GetState() *types.ExecMarket {
	rf := m.risk.GetRiskFactors()
	var sp *num.Numeric
	if m.settlementDataInMarket != nil {
		sp = m.settlementDataInMarket.Clone()
	}

	parties := maps.Keys(m.parties)
	sort.Strings(parties)

	em := &types.ExecMarket{
		Market:                     m.mkt.DeepClone(),
		PriceMonitor:               m.pMonitor.GetState(),
		AuctionState:               m.as.GetState(),
		PeggedOrders:               m.peggedOrders.GetState(),
		ExpiringOrders:             m.expiringOrders.GetState(),
		LastBestBid:                m.lastBestBidPrice.Clone(),
		LastBestAsk:                m.lastBestAskPrice.Clone(),
		LastMidBid:                 m.lastMidBuyPrice.Clone(),
		LastMidAsk:                 m.lastMidSellPrice.Clone(),
		LastMarketValueProxy:       m.lastMarketValueProxy,
		CurrentMarkPrice:           m.getCurrentMarkPrice(),
		LastTradedPrice:            m.getLastTradedPrice(),
		LastEquityShareDistributed: m.lastEquityShareDistributed.UnixNano(),
		EquityShare:                m.equityShares.GetState(),
		RiskFactorConsensusReached: m.risk.IsRiskFactorInitialised(),
		ShortRiskFactor:            rf.Short,
		LongRiskFactor:             rf.Long,
		FeeSplitter:                m.feeSplitter.GetState(),
		SettlementData:             sp,
		NextMTM:                    m.nextMTM.UnixNano(),
		Parties:                    parties,
		Closed:                     m.closed,
		IsSucceeded:                m.succeeded,
	}

	return em
}
