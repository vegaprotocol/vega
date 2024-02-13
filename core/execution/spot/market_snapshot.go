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

package spot

import (
	"context"
	"fmt"
	"sort"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/fee"
	liquiditytarget "code.vegaprotocol.io/vega/core/liquidity/target/spot"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
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
	em *types.ExecSpotMarket,
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
	referralDiscountRewardService fee.ReferralDiscountRewardService,
	volumeDiscountService fee.VolumeDiscountService,
	banking common.Banking,
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

	baseAsset := assets[BaseAssetIndex]
	quoteAsset := assets[QuoteAssetIndex]

	var feeEngine *fee.Engine
	if em.FeesStats != nil {
		feeEngine, err = fee.NewFromState(log, feeConfig, *mkt.Fees, quoteAsset, positionFactor, em.FeesStats)
		if err != nil {
			return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
		}
	} else {
		feeEngine, err = fee.New(log, feeConfig, *mkt.Fees, quoteAsset, positionFactor)
		if err != nil {
			return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
		}
	}

	tsCalc := liquiditytarget.NewSnapshotEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, mkt.ID, positionFactor)
	riskModel, err := risk.NewModel(mkt.TradableInstrument.RiskModel, quoteAsset)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate risk model: %w", err)
	}
	pMonitor, err := price.NewMonitorFromSnapshot(mkt.ID, quoteAsset, em.PriceMonitor, mkt.PriceMonitoringSettings, riskModel, as, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}
	els := common.NewEquitySharesFromSnapshot(em.EquityShare)
	liquidity := liquidity.NewSnapshotEngine(liquidityConfig, log, timeService, broker, riskModel, pMonitor, book, as, quoteAsset, mkt.ID, stateVarEngine, positionFactor, mkt.LiquiditySLAParams)
	// @TODO pass in AMM
	marketLiquidity := common.NewMarketLiquidity(log, liquidity, collateralEngine, broker, book, els, marketActivityTracker, feeEngine, common.FutureMarketType, mkt.ID, quoteAsset, priceFactor, mkt.LiquiditySLAParams.PriceRange, nil)

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

	now := timeService.GetTimeNow()
	market := &Market{
		log:                           log,
		mkt:                           mkt,
		closingAt:                     time.Unix(0, mkt.MarketTimestamps.Close),
		timeService:                   timeService,
		matching:                      book,
		collateral:                    collateralEngine,
		broker:                        broker,
		fee:                           feeEngine,
		referralDiscountRewardService: referralDiscountRewardService,
		volumeDiscountService:         volumeDiscountService,
		liquidity:                     marketLiquidity,
		liquidityEngine:               liquidity,
		parties:                       map[string]struct{}{},
		tsCalc:                        tsCalc,
		feeSplitter:                   common.NewFeeSplitterFromSnapshot(em.FeeSplitter, now),
		as:                            as,
		pMonitor:                      pMonitor,
		peggedOrders:                  common.NewPeggedOrdersFromSnapshot(log, timeService, em.PeggedOrders),
		expiringOrders:                common.NewExpiringOrdersFromState(em.ExpiringOrders),
		equityShares:                  els,
		lastBestBidPrice:              em.LastBestBid.Clone(),
		lastBestAskPrice:              em.LastBestAsk.Clone(),
		lastMidBuyPrice:               em.LastMidBid.Clone(),
		lastMidSellPrice:              em.LastMidAsk.Clone(),
		markPrice:                     em.CurrentMarkPrice,
		lastTradedPrice:               em.LastTradedPrice,
		priceFactor:                   priceFactor,
		lastMarketValueProxy:          em.LastMarketValueProxy,
		lastEquityShareDistributed:    time.Unix(0, em.LastEquityShareDistributed),
		marketActivityTracker:         marketActivityTracker,
		positionFactor:                positionFactor,
		stateVarEngine:                stateVarEngine,
		baseFactor:                    baseFactor,
		baseAsset:                     baseAsset,
		quoteAsset:                    quoteAsset,
		stopOrders:                    stopOrders,
		expiringStopOrders:            expiringStopOrders,
		hasTraded:                     em.HasTraded,
		orderHoldingTracker:           NewHoldingAccountTracker(mkt.ID, log, collateralEngine),
		banking:                       banking,
	}
	liquidity.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)
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
	quoteAssetQuantum, _ := m.collateral.GetAssetQuantum(m.quoteAsset)

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
		CurrentMarkPrice:           m.markPrice,
		LastTradedPrice:            m.lastTradedPrice,
		LastEquityShareDistributed: m.lastEquityShareDistributed.UnixNano(),
		EquityShare:                m.equityShares.GetState(),
		FeeSplitter:                m.feeSplitter.GetState(),
		NextMTM:                    m.nextMTM.UnixNano(),
		Parties:                    parties,
		Closed:                     m.closed,
		HasTraded:                  m.hasTraded,
		FeesStats:                  m.fee.GetState(quoteAssetQuantum),
	}

	return em
}

func (m *Market) GetNewStateProviders() []types.StateProvider {
	return []types.StateProvider{m.matching, m.tsCalc, m.orderHoldingTracker, m.liquidityEngine.V2StateProvider()}
}
