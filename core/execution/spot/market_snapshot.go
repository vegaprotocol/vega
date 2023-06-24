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
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/liquidity"
	liquiditytarget "code.vegaprotocol.io/vega/core/liquidity/target/spot"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/monitor"
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
	baseAssetDetails *assets.Asset,
	quoteAssetDetails *assets.Asset,
	marketActivityTracker *common.MarketActivityTracker,
	peggedOrderNotify func(int64),
) (*Market, error) {
	mkt := em.Market
	if len(em.Market.ID) == 0 {
		return nil, common.ErrEmptyMarketID
	}
	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(mkt.PositionDecimalPlaces))

	priceFactor := num.NewUint(1)
	if exp := quoteAssetDetails.DecimalPlaces() - mkt.DecimalPlaces; exp != 0 {
		priceFactor.Exp(num.NewUint(10), num.NewUint(exp))
	}
	baseFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(int64(baseAssetDetails.DecimalPlaces()) - mkt.PositionDecimalPlaces))
	as := monitor.NewAuctionStateFromSnapshot(mkt, em.AuctionState)

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewCachedOrderBook(
		log, matchingConfig, mkt.ID, as.InAuction(), peggedOrderNotify)
	assets, err := mkt.GetAssets()
	if err != nil {
		return nil, err
	}

	if len(assets) != 2 {
		return nil, fmt.Errorf("expecting base asset and quote asset for spot market")
	}

	baseAsset := assets[0]
	quoteAsset := assets[1]
	feeEngine, err := fee.New(log, feeConfig, *mkt.Fees, quoteAsset, positionFactor)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
	}

	tsCalc := liquiditytarget.NewSnapshotEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, mkt.ID, positionFactor)
	riskModel, err := risk.NewModel(mkt.TradableInstrument.RiskModel, quoteAsset)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate risk model: %w", err)
	}
	pMonitor, err := price.NewMonitor(quoteAsset, mkt.ID, riskModel, as, mkt.PriceMonitoringSettings, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}
	liquidity := &DummyLiquidity{}

	now := timeService.GetTimeNow()
	market := &Market{
		log:                        log,
		mkt:                        mkt,
		closingAt:                  time.Unix(0, mkt.MarketTimestamps.Close),
		timeService:                timeService,
		matching:                   book,
		collateral:                 collateralEngine,
		broker:                     broker,
		fee:                        feeEngine,
		liquidity:                  liquidity,
		parties:                    map[string]struct{}{},
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
		baseFactor:                 baseFactor,
		baseAsset:                  baseAsset,
		quoteAsset:                 quoteAsset,
	}

	for _, p := range em.Parties {
		market.parties[p] = struct{}{}
	}

	if em.Closed {
		market.closed = true
		stateVarEngine.UnregisterStateVariable(baseAsset+"_"+quoteAsset, mkt.ID)
	}
	return market, nil
}

func (m *Market) GetState() *types.ExecSpotMarket {
	parties := maps.Keys(m.parties)
	sort.Strings(parties)

	em := &types.ExecSpotMarket{
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
		FeeSplitter:                m.feeSplitter.GetState(),
		NextMTM:                    m.nextMTM.UnixNano(),
		Parties:                    parties,
		Closed:                     m.closed,
		HasTraded:                  m.hasTraded,
	}

	return em
}

func (m *Market) GetNewStateProviders() []types.StateProvider {
	return []types.StateProvider{m.matching, m.tsCalc, m.orderHoldingTracker}
}
