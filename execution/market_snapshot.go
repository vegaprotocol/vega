package execution

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/liquidity"
	"code.vegaprotocol.io/vega/liquidity/target"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/monitor"
	lmon "code.vegaprotocol.io/vega/monitor/liquidity"
	"code.vegaprotocol.io/vega/monitor/price"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/products"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
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
	collateralEngine MarketCollateral,
	oracleEngine products.OracleEngine,
	now time.Time,
	broker Broker,
	stateVarEngine StateVarEngine,
	assetDetails *assets.Asset,
	feesTracker *FeesTracker,
	marketTracker *MarketTracker,
) (*Market, error) {
	mkt := em.Market
	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(int64(mkt.PositionDecimalPlaces)))
	if len(em.Market.ID) == 0 {
		return nil, ErrEmptyMarketID
	}

	tradableInstrument, err := markets.NewTradableInstrument(ctx, log, mkt.TradableInstrument, oracleEngine)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate a new market: %w", err)
	}

	as := monitor.NewAuctionStateFromSnapshot(mkt, em.AuctionState)

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewCachedOrderBook(
		log, matchingConfig, mkt.ID, as.InAuction())
	asset := tradableInstrument.Instrument.Product.GetAsset()

	// this needs to stay
	riskEngine := risk.NewEngine(
		log,
		riskConfig,
		tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel,
		book,
		as,
		broker,
		now.UnixNano(),
		mkt.ID,
		asset,
		stateVarEngine,
		&types.RiskFactor{
			Market: mkt.ID,
			Short:  em.ShortRiskFactor,
			Long:   em.LongRiskFactor,
		},
		em.RiskFactorConsensusReached,
		positionFactor,
	)

	settleEngine := settlement.New(
		log,
		settlementConfig,
		tradableInstrument.Instrument.Product,
		mkt.ID,
		broker,
		positionFactor,
	)
	positionEngine := positions.NewSnapshotEngine(log, positionConfig, mkt.ID, broker)

	feeEngine, err := fee.New(log, feeConfig, *mkt.Fees, asset, positionFactor)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
	}

	tsCalc := target.NewSnapshotEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, positionEngine, mkt.ID, positionFactor)

	pMonitor, err := price.NewMonitorFromSnapshot(mkt.ID, asset, em.PriceMonitor, mkt.PriceMonitoringSettings, tradableInstrument.RiskModel, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}

	exp := assetDetails.DecimalPlaces() - mkt.DecimalPlaces
	priceFactor := num.Zero().Exp(num.NewUint(10), num.NewUint(exp))
	lMonitor := lmon.NewMonitor(tsCalc, mkt.LiquidityMonitoringParameters)

	liqEngine := liquidity.NewSnapshotEngine(liquidityConfig, log, broker, tradableInstrument.RiskModel, pMonitor, asset, mkt.ID, stateVarEngine, priceFactor.Clone(), positionFactor)
	// call on chain time update straight away, so
	// the time in the engine is being updatedat creation
	liqEngine.OnChainTimeUpdate(ctx, now)

	market := &Market{
		log:                        log,
		mkt:                        mkt,
		closingAt:                  time.Unix(0, mkt.MarketTimestamps.Close),
		currentTime:                now,
		matching:                   book,
		tradableInstrument:         tradableInstrument,
		risk:                       riskEngine,
		position:                   positionEngine,
		settlement:                 settleEngine,
		collateral:                 collateralEngine,
		broker:                     broker,
		fee:                        feeEngine,
		liquidity:                  liqEngine,
		parties:                    map[string]struct{}{}, // parties will be restored on chain time update
		lMonitor:                   lMonitor,
		tsCalc:                     tsCalc,
		feeSplitter:                NewFeeSplitterFromSnapshot(em.FeeSplitter, now),
		as:                         as,
		pMonitor:                   pMonitor,
		peggedOrders:               NewPeggedOrdersFromSnapshot(em.PeggedOrders),
		expiringOrders:             NewExpiringOrdersFromState(em.ExpiringOrders),
		equityShares:               NewEquitySharesFromSnapshot(em.EquityShare),
		lastBestBidPrice:           em.LastBestBid.Clone(),
		lastBestAskPrice:           em.LastBestAsk.Clone(),
		lastMidBuyPrice:            em.LastMidBid.Clone(),
		lastMidSellPrice:           em.LastMidAsk.Clone(),
		markPrice:                  em.CurrentMarkPrice.Clone(),
		stateChanged:               true,
		priceFactor:                priceFactor,
		lastMarketValueProxy:       em.LastMarketValueProxy,
		lastEquityShareDistributed: time.Unix(0, em.LastEquityShareDistributed),
		feesTracker:                feesTracker,
		marketTracker:              marketTracker,
		positionFactor:             positionFactor,
	}

	market.tradableInstrument.Instrument.Product.NotifyOnTradingTerminated(market.tradingTerminated)
	market.tradableInstrument.Instrument.Product.NotifyOnSettlementPrice(market.settlementPrice)
	liqEngine.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)

	return market, nil
}

func (m *Market) changed() bool {
	return (m.stateChanged ||
		m.pMonitor.Changed() ||
		m.as.Changed() ||
		m.peggedOrders.Changed() ||
		m.expiringOrders.Changed() ||
		m.equityShares.Changed() ||
		m.feeSplitter.Changed())
}

func (m *Market) getState() *types.ExecMarket {
	rf, _ := m.risk.GetRiskFactors()
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
		LastEquityShareDistributed: m.lastEquityShareDistributed.UnixNano(),
		EquityShare:                m.equityShares.GetState(),
		RiskFactorConsensusReached: m.risk.IsRiskFactorInitialised(),
		ShortRiskFactor:            rf.Short,
		LongRiskFactor:             rf.Long,
		FeeSplitter:                m.feeSplitter.GetState(),
	}

	m.stateChanged = false

	return em
}
