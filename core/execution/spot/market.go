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

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/stoporders"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/idgeneration"
	liquiditytarget "code.vegaprotocol.io/vega/core/liquidity/target/spot"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

const (
	BaseAssetIndex  = 0
	QuoteAssetIndex = 1
)

type TargetStakeCalculator interface {
	types.StateProvider
	RecordTotalStake(oi uint64, now time.Time) error
	GetTargetStake(now time.Time) *num.Uint
	UpdateScalingFactor(sFactor num.Decimal) error
	UpdateTimeWindow(tWindow time.Duration)
	StopSnapshots()
	UpdateParameters(types.TargetStakeParameters)
}

// Market represents an instance of a market in vega and is in charge of calling the engines in order to process all transactions.
type Market struct {
	log   *logging.Logger
	idgen common.IDGenerator

	mkt *types.Market

	closingAt   time.Time
	timeService common.TimeService

	mu sync.Mutex

	lastTradedPrice *num.Uint
	markPrice       *num.Uint
	priceFactor     *num.Uint

	// own engines
	matching                      *matching.CachedOrderBook
	fee                           *fee.Engine
	referralDiscountRewardService fee.ReferralDiscountRewardService
	volumeDiscountService         fee.VolumeDiscountService
	liquidity                     common.MarketLiquidityEngine
	liquidityEngine               common.LiquidityEngine

	// deps engines
	collateral common.Collateral
	banking    common.Banking

	broker common.Broker
	closed bool

	parties map[string]struct{}

	pMonitor common.PriceMonitor

	tsCalc TargetStakeCalculator

	as common.AuctionState

	peggedOrders   *common.PeggedOrders
	expiringOrders *common.ExpiringOrders

	// Store the previous price values so we can see what has changed
	lastBestBidPrice *num.Uint
	lastBestAskPrice *num.Uint
	lastMidBuyPrice  *num.Uint
	lastMidSellPrice *num.Uint

	lastMarketValueProxy    num.Decimal
	marketValueWindowLength time.Duration

	// Liquidity Fee
	feeSplitter                *common.FeeSplitter
	lastEquityShareDistributed time.Time
	equityShares               *common.EquityShares
	minLPStakeQuantumMultiple  num.Decimal

	stateVarEngine        common.StateVarEngine
	marketActivityTracker *common.MarketActivityTracker
	baseFactor            num.Decimal // 10^(baseDP-pdp)
	positionFactor        num.Decimal // 10^pdp

	orderHoldingTracker *HoldingAccountTracker

	nextMTM    time.Time
	mtmDelta   time.Duration
	hasTraded  bool
	baseAsset  string
	quoteAsset string

	maxStopOrdersPerParties *num.Uint
	stopOrders              *stoporders.Pool
	expiringStopOrders      *common.ExpiringOrders

	minDuration time.Duration
}

// NewMarket creates a new market using the market framework configuration and creates underlying engines.
func NewMarket(
	log *logging.Logger,
	matchingConfig matching.Config,
	feeConfig fee.Config,
	liquidityConfig liquidity.Config,
	collateralEngine common.Collateral,
	mkt *types.Market,
	timeService common.TimeService,
	broker common.Broker,
	as common.AuctionState,
	stateVarEngine common.StateVarEngine,
	marketActivityTracker *common.MarketActivityTracker,
	baseAssetDetails *assets.Asset,
	quoteAssetDetails *assets.Asset,
	peggedOrderNotify func(int64),
	referralDiscountRewardService fee.ReferralDiscountRewardService,
	volumeDiscountService fee.VolumeDiscountService,
	banking common.Banking,
) (*Market, error) {
	if len(mkt.ID) == 0 {
		return nil, common.ErrEmptyMarketID
	}

	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(mkt.PositionDecimalPlaces))
	priceFactor := num.NewUint(1)
	if exp := quoteAssetDetails.DecimalPlaces() - mkt.DecimalPlaces; exp != 0 {
		priceFactor.Exp(num.NewUint(10), num.NewUint(exp))
	}
	baseFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(int64(baseAssetDetails.DecimalPlaces()) - mkt.PositionDecimalPlaces))
	book := matching.NewCachedOrderBook(log, matchingConfig, mkt.ID, as.InAuction(), peggedOrderNotify)
	assets, err := mkt.GetAssets()
	if err != nil {
		return nil, err
	}

	if len(assets) != 2 {
		return nil, fmt.Errorf("expecting base asset and quote asset for spot market")
	}

	baseAsset := assets[BaseAssetIndex]
	quoteAsset := assets[QuoteAssetIndex]
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

	now := timeService.GetTimeNow()

	// The market is initially created in a proposed state
	mkt.State = types.MarketStateProposed
	mkt.TradingMode = types.MarketTradingModeNoTrading

	// Populate the market timestamps
	ts := &types.MarketTimestamps{
		Proposed: now.UnixNano(),
		Pending:  now.UnixNano(),
	}

	if mkt.OpeningAuction != nil {
		ts.Open = now.Add(time.Duration(mkt.OpeningAuction.Duration)).UnixNano()
	} else {
		ts.Open = now.UnixNano()
	}

	mkt.MarketTimestamps = ts
	liquidity := liquidity.NewSnapshotEngine(liquidityConfig, log, timeService, broker, riskModel, pMonitor, book, as, quoteAsset, mkt.ID, stateVarEngine, positionFactor, mkt.LiquiditySLAParams)
	els := common.NewEquityShares(num.DecimalZero())
	marketLiquidity := common.NewMarketLiquidity(log, liquidity, collateralEngine, broker, book, els, marketActivityTracker, feeEngine, common.SpotMarketType, mkt.ID, quoteAsset, priceFactor, mkt.LiquiditySLAParams.PriceRange)
	market := &Market{
		log:                           log,
		idgen:                         nil,
		mkt:                           mkt,
		matching:                      book,
		collateral:                    collateralEngine,
		timeService:                   timeService,
		broker:                        broker,
		fee:                           feeEngine,
		referralDiscountRewardService: referralDiscountRewardService,
		volumeDiscountService:         volumeDiscountService,
		parties:                       map[string]struct{}{},
		as:                            as,
		pMonitor:                      pMonitor,
		liquidity:                     marketLiquidity,
		liquidityEngine:               liquidity,
		tsCalc:                        tsCalc,
		peggedOrders:                  common.NewPeggedOrders(log, timeService),
		expiringOrders:                common.NewExpiringOrders(),
		feeSplitter:                   common.NewFeeSplitter(),
		equityShares:                  els,
		lastBestAskPrice:              num.UintZero(),
		lastMidSellPrice:              num.UintZero(),
		lastMidBuyPrice:               num.UintZero(),
		lastBestBidPrice:              num.UintZero(),
		stateVarEngine:                stateVarEngine,
		marketActivityTracker:         marketActivityTracker,
		priceFactor:                   priceFactor,
		baseFactor:                    baseFactor,
		minLPStakeQuantumMultiple:     num.MustDecimalFromString("1"),
		positionFactor:                positionFactor,
		baseAsset:                     baseAsset,
		quoteAsset:                    quoteAsset,
		orderHoldingTracker:           NewHoldingAccountTracker(mkt.ID, log, collateralEngine),
		nextMTM:                       time.Time{}, // default to zero time
		maxStopOrdersPerParties:       num.UintZero(),
		stopOrders:                    stoporders.New(log),
		expiringStopOrders:            common.NewExpiringOrders(),
		banking:                       banking,
	}
	liquidity.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)

	return market, nil
}

func (m *Market) IsOpeningAuction() bool {
	return m.as.IsOpeningAuction()
}

func (m *Market) GetPartiesStats() *types.MarketStats {
	return &types.MarketStats{}
}

func (m *Market) Update(ctx context.Context, config *types.Market) error {
	config.TradingMode = m.mkt.TradingMode
	config.State = m.mkt.State
	config.MarketTimestamps = m.mkt.MarketTimestamps
	m.mkt = config

	m.tsCalc.UpdateParameters(*config.LiquidityMonitoringParameters.TargetStakeParameters)
	riskModel, err := risk.NewModel(config.TradableInstrument.RiskModel, m.quoteAsset)
	if err != nil {
		return err
	}
	m.pMonitor.UpdateSettings(riskModel, m.mkt.PriceMonitoringSettings)
	m.liquidity.UpdateMarketConfig(riskModel, m.pMonitor)
	m.updateLiquidityFee(ctx)

	// update immediately during opening auction
	if m.as.IsOpeningAuction() {
		m.liquidity.UpdateSLAParameters(m.mkt.LiquiditySLAParams)
	}

	return nil
}

func (m *Market) GetEquityShares() *common.EquityShares {
	return m.equityShares
}

func (m *Market) SetNextMTM(tm time.Time) {
	m.nextMTM = tm
}

func (m *Market) GetNextMTM() time.Time {
	return m.nextMTM
}

func (m *Market) midPrice() *num.Uint {
	bestBidPrice, _, _ := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, _, _ := m.matching.BestOfferPriceAndVolume()
	two := num.NewUint(2)
	midPrice := num.UintZero()
	if !bestBidPrice.IsZero() && !bestOfferPrice.IsZero() {
		midPrice = midPrice.Div(num.Sum(bestBidPrice, bestOfferPrice), two)
	}
	return midPrice
}

func (m *Market) IntoType() types.Market {
	return *m.mkt.DeepClone()
}

func (m *Market) Hash() []byte {
	mID := logging.String("market-id", m.GetID())
	matchingHash := m.matching.Hash()
	m.log.Debug("orderbook state hash", logging.Hash(matchingHash), mID)
	return matchingHash
}

func (m *Market) GetMarketState() types.MarketState {
	return m.mkt.State
}

func (m *Market) priceToMarketPrecision(price *num.Uint) *num.Uint {
	return price.Div(price, m.priceFactor)
}

func (m *Market) GetMarketData() types.MarketData {
	bestBidPrice, bestBidVolume, _ := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, bestOfferVolume, _ := m.matching.BestOfferPriceAndVolume()
	bestStaticBidPrice, bestStaticBidVolume, _ := m.getBestStaticBidPriceAndVolume()
	bestStaticOfferPrice, bestStaticOfferVolume, _ := m.getBestStaticAskPriceAndVolume()

	// Auction related values
	indicativePrice := num.UintZero()
	indicativeVolume := uint64(0)
	var auctionStart, auctionEnd int64
	if m.as.InAuction() {
		indicativePrice, indicativeVolume, _ = m.matching.GetIndicativePriceAndVolume()
		if t := m.as.Start(); !t.IsZero() {
			auctionStart = t.UnixNano()
		}
		if t := m.as.ExpiresAt(); t != nil {
			auctionEnd = t.UnixNano()
		}
	}

	// If we do not have one of the best_* prices, leave the mid price as zero
	two := num.NewUint(2)
	midPrice := num.UintZero()
	if !bestBidPrice.IsZero() && !bestOfferPrice.IsZero() {
		midPrice = midPrice.Div(num.Sum(bestBidPrice, bestOfferPrice), two)
	}

	staticMidPrice := num.UintZero()
	if !bestStaticBidPrice.IsZero() && !bestStaticOfferPrice.IsZero() {
		staticMidPrice = staticMidPrice.Div(num.Sum(bestStaticBidPrice, bestStaticOfferPrice), two)
	}

	targetStake := m.getTargetStake().String()
	bounds := m.pMonitor.GetCurrentBounds()
	for _, b := range bounds {
		m.priceToMarketPrecision(b.MaxValidPrice) // effictively floors this
		m.priceToMarketPrecision(b.MinValidPrice)

		rp, _ := num.UintFromDecimal(b.ReferencePrice)
		m.priceToMarketPrecision(rp)
		b.ReferencePrice = num.DecimalFromUint(rp)

		if m.priceFactor.NEQ(common.One) {
			b.MinValidPrice.AddSum(common.One) // ceil
		}
	}
	mode := m.as.Mode()
	if m.mkt.TradingMode == types.MarketTradingModeNoTrading {
		mode = m.mkt.TradingMode
	}

	return types.MarketData{
		Market:                    m.GetID(),
		BestBidPrice:              m.priceToMarketPrecision(bestBidPrice),
		BestBidVolume:             bestBidVolume,
		BestOfferPrice:            m.priceToMarketPrecision(bestOfferPrice),
		BestOfferVolume:           bestOfferVolume,
		BestStaticBidPrice:        m.priceToMarketPrecision(bestStaticBidPrice),
		BestStaticBidVolume:       bestStaticBidVolume,
		BestStaticOfferPrice:      m.priceToMarketPrecision(bestStaticOfferPrice),
		BestStaticOfferVolume:     bestStaticOfferVolume,
		NextMTM:                   m.nextMTM.UnixNano(),
		MidPrice:                  m.priceToMarketPrecision(midPrice),
		StaticMidPrice:            m.priceToMarketPrecision(staticMidPrice),
		MarkPrice:                 m.priceToMarketPrecision(m.getCurrentMarkPrice()),
		LastTradedPrice:           m.priceToMarketPrecision(m.getLastTradedPrice()),
		Timestamp:                 m.timeService.GetTimeNow().UnixNano(),
		IndicativePrice:           m.priceToMarketPrecision(indicativePrice),
		IndicativeVolume:          indicativeVolume,
		AuctionStart:              auctionStart,
		AuctionEnd:                auctionEnd,
		MarketTradingMode:         mode,
		MarketState:               m.mkt.State,
		Trigger:                   m.as.Trigger(),
		ExtensionTrigger:          m.as.ExtensionTrigger(),
		TargetStake:               targetStake,
		SuppliedStake:             m.getSuppliedStake().String(),
		PriceMonitoringBounds:     bounds,
		MarketValueProxy:          m.lastMarketValueProxy.BigInt().String(),
		LiquidityProviderFeeShare: m.equityShares.LpsToLiquidityProviderFeeShare(m.liquidity.GetAverageLiquidityScores()),
		LiquidityProviderSLA:      m.liquidityEngine.LiquidityProviderSLAStats(m.timeService.GetTimeNow()),
	}
}

func (m *Market) uncrossOnLeaveAuction(ctx context.Context) ([]*types.OrderConfirmation, []*types.Order) {
	uncrossedOrders, ordersToCancel, err := m.matching.LeaveAuction(m.timeService.GetTimeNow())
	if err != nil {
		m.log.Error("Error leaving auction", logging.Error(err))
	}
	evts := make([]events.Event, 0, len(uncrossedOrders))
	for _, uncrossedOrder := range uncrossedOrders {
		m.handleConfirmation(ctx, uncrossedOrder)
		if uncrossedOrder.Order.Remaining == 0 {
			uncrossedOrder.Order.Status = types.OrderStatusFilled
		}
		evts = append(evts, events.NewOrderEvent(ctx, uncrossedOrder.Order))
	}

	for _, uncrossedOrder := range uncrossedOrders {
		m.handleConfirmationPassiveOrders(ctx, uncrossedOrder)
	}

	// send order events in a single batch, it's more efficient
	m.broker.SendBatch(evts)
	return uncrossedOrders, ordersToCancel
}

func (m *Market) uncrossOrderAtAuctionEnd(ctx context.Context) {
	if !m.as.InAuction() || m.as.IsOpeningAuction() {
		return
	}
	m.uncrossOnLeaveAuction(ctx)
}

func (m *Market) UpdateMarketState(ctx context.Context, changes *types.MarketStateUpdateConfiguration) error {
	_, blockHash := vegacontext.TraceIDFromContext(ctx)
	// make deterministic ID for this market, concatenate
	// the block hash and the market ID
	m.idgen = idgeneration.New(blockHash + crypto.HashStrToHex(m.GetID()))
	// and we call next ID on this directly just so we don't have an ID which have
	// a different from others, we basically burn the first ID.
	_ = m.idgen.NextID()
	defer func() { m.idgen = nil }()
	if changes.UpdateType == types.MarketStateUpdateTypeTerminate {
		m.uncrossOrderAtAuctionEnd(ctx)
		// terminate and settle
		m.closeSpotMarket(ctx)
	} else if changes.UpdateType == types.MarketStateUpdateTypeSuspend {
		m.mkt.State = types.MarketStateSuspendedViaGovernance
		m.mkt.TradingMode = types.MarketTradingModeSuspendedViaGovernance
		if m.as.InAuction() {
			m.as.ExtendAuctionSuspension(types.AuctionDuration{Duration: int64(m.minDuration)})
			evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow())
			if evt != nil {
				m.broker.Send(evt)
			}
		} else {
			m.as.StartGovernanceSuspensionAuction(m.timeService.GetTimeNow())
			m.enterAuction(ctx)
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
		}
	} else if changes.UpdateType == types.MarketStateUpdateTypeResume && m.mkt.State == types.MarketStateSuspendedViaGovernance {
		if m.as.GetState().Trigger == types.AuctionTriggerGovernanceSuspension && m.as.GetState().Extension == types.AuctionTriggerUnspecified {
			m.as.EndGovernanceSuspensionAuction()
			m.leaveAuction(ctx, m.timeService.GetTimeNow())
		} else {
			if m.as.GetState().Trigger == types.AuctionTriggerOpening {
				m.mkt.State = types.MarketStatePending
				m.mkt.TradingMode = types.MarketTradingModeOpeningAuction
			} else {
				m.mkt.State = types.MarketStateSuspended
				m.mkt.TradingMode = types.MarketTradingModeMonitoringAuction
			}
			defer func() { m.idgen = nil }()
			m.checkAuction(ctx, m.timeService.GetTimeNow(), m.idgen)
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
		}
	}
	return nil
}

// ReloadConf will trigger a reload of all the config settings in the market and all underlying engines
// this is required when hot-reloading any config changes, eg. logger level.
func (m *Market) ReloadConf(matchingConfig matching.Config, feeConfig fee.Config) {
	m.log.Info("reloading configuration")
	m.matching.ReloadConf(matchingConfig)
	m.fee.ReloadConf(feeConfig)
}

func (m *Market) GetAssetForProposerBonus() string {
	return m.quoteAsset
}

// Reject a market if the market state allow.
func (m *Market) Reject(ctx context.Context) error {
	if m.mkt.State != types.MarketStateProposed {
		return common.ErrCannotRejectMarketNotInProposedState
	}

	// we closed all parties accounts
	m.cleanupOnReject(ctx)
	m.mkt.State = types.MarketStateRejected
	m.mkt.TradingMode = types.MarketTradingModeNoTrading
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

// CanLeaveOpeningAuction checks if the market can leave the opening auction based on whether floating point consensus has been reached on all 2 vars.
func (m *Market) CanLeaveOpeningAuction() bool {
	boundFactorsInitialised := m.pMonitor.IsBoundFactorsInitialised()
	potInitialised := m.liquidity.IsProbabilityOfTradingInitialised()

	canLeave := boundFactorsInitialised && potInitialised
	if !canLeave {
		m.log.Info("Cannot leave opening auction", logging.String("market", m.mkt.ID), logging.Bool("bound-factors-initialised", boundFactorsInitialised), logging.Bool("pot-initialised", potInitialised))
	}
	return canLeave
}

// StartOpeningAuction kicks off opening auction.
func (m *Market) StartOpeningAuction(ctx context.Context) error {
	if m.mkt.State != types.MarketStateProposed {
		return common.ErrCannotStartOpeningAuctionForMarketNotInProposedState
	}

	// now we start the opening auction
	if m.as.AuctionStart() {
		// we are now in a pending state
		m.mkt.State = types.MarketStatePending
		m.mkt.TradingMode = types.MarketTradingModeOpeningAuction
		m.enterAuction(ctx)
	} else {
		m.mkt.State = types.MarketStateActive
		m.mkt.TradingMode = types.MarketTradingModeContinuous
	}

	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

// GetID returns the id of the given market.
func (m *Market) GetID() string {
	return m.mkt.ID
}

// PostRestore restores market price in orders after snapshot reload.
func (m *Market) PostRestore(ctx context.Context) error {
	// tell the matching engine about the markets price factor so it can finish restoring orders
	m.matching.RestoreWithMarketPriceFactor(m.priceFactor)
	return nil
}

// OnTick notifies the market of a new time event/update.
func (m *Market) OnTick(ctx context.Context, t time.Time) bool {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "OnTick")
	m.mu.Lock()
	defer m.mu.Unlock()

	_, blockHash := vegacontext.TraceIDFromContext(ctx)
	// make deterministic ID for this market, concatenate
	// the block hash and the market ID
	m.idgen = idgeneration.New(blockHash + crypto.HashStrToHex(m.GetID()))
	// and we call next ID on this directly just so we don't have an ID which have
	// a different from others, we basically burn the first ID.
	_ = m.idgen.NextID()
	defer func() { m.idgen = nil }()

	if m.closed {
		return true
	}

	// first we expire orders
	if !m.closed && m.canTrade() {
		expired := m.removeExpiredOrders(ctx, t.UnixNano())
		metrics.OrderGaugeAdd(-len(expired), m.GetID())
		confirmations := m.removeExpiredStopOrders(ctx, t.UnixNano(), m.idgen)

		stopsExpired := 0
		for _, v := range confirmations {
			stopsExpired++
			for _, v := range v.PassiveOrdersAffected {
				if v.Status != types.OrderStatusActive {
					stopsExpired++
				}
			}
		}
		metrics.OrderGaugeAdd(-stopsExpired, m.GetID())
	}

	// some engines still needs to get updates:
	m.pMonitor.OnTimeUpdate(t)
	m.feeSplitter.SetCurrentTime(t)

	if m.mkt.State == types.MarketStateProposed {
		return false
	}

	m.checkAuction(ctx, t, m.idgen)
	timer.EngineTimeCounterAdd()
	m.updateMarketValueProxy()
	m.updateLiquidityFee(ctx)
	m.liquidity.OnTick(ctx, t)
	m.broker.Send(events.NewMarketTick(ctx, m.mkt.ID, t))
	return m.closed
}
func (m *Market) BeginBlock(_ context.Context) {}

// BlockEnd notifies the market of the end of the block.
func (m *Market) BlockEnd(_ context.Context) {
	// simplified version of updating mark price every MTM interval
	mp := m.getLastTradedPrice()
	if !m.hasTraded && m.markPrice != nil {
		// no trades happened, make sure we're just using the current mark price
		mp = m.markPrice.Clone()
	}
	t := m.timeService.GetTimeNow()
	if mp != nil && !mp.IsZero() && !m.as.InAuction() && (m.nextMTM.IsZero() || !m.nextMTM.After(t)) {
		m.markPrice = mp
		m.nextMTM = t.Add(m.mtmDelta) // add delta here

		// last traded price should not reflect the closeout trades
		m.lastTradedPrice = mp.Clone()
		m.hasTraded = false
	}
	m.tsCalc.RecordTotalStake(m.liquidity.CalculateSuppliedStake().Uint64(), m.timeService.GetTimeNow())
	m.liquidity.EndBlock(m.markPrice, m.midPrice(), m.positionFactor)
}

func (m *Market) updateMarketValueProxy() {
	// if windows length is reached, reset fee splitter
	if mvwl := m.marketValueWindowLength; m.feeSplitter.Elapsed() > mvwl {
		// AvgTradeValue calculates the rolling average trade value to include the current window (which is ending)
		m.equityShares.AvgTradeValue(m.feeSplitter.AvgTradeValue())
		// this increments the internal window counter
		m.feeSplitter.TimeWindowStart(m.timeService.GetTimeNow())
		// m.equityShares.UpdateVirtualStake() // this should always set the vStake >= physical stake?
	}

	// these need to happen every block
	// but also when new LP is submitted just so we are sure we do
	// not have a mvp of 0
	ts := m.liquidity.CalculateSuppliedStake()
	m.lastMarketValueProxy = m.feeSplitter.MarketValueProxy(
		m.marketValueWindowLength, ts)
}

// removeOrders removes orders from the book when the market is stopped.
func (m *Market) removeOrders(ctx context.Context) {
	// remove all order from the book
	// and send events with the stopped status
	orders := append(m.matching.Settled(), m.peggedOrders.Settled()...)
	orderEvents := make([]events.Event, 0, len(orders))
	for _, v := range orders {
		orderEvents = append(orderEvents, events.NewOrderEvent(ctx, v))
		// release any locked funds for the order from the holding account
		m.releaseOrderFromHoldingAccount(ctx, v.ID, v.Party, v.Side)
	}
	m.broker.SendBatch(orderEvents)
}

// cleanMarketWithState clears the collateral state of the market and clears up state vars and sets the terminated state of the market
// NB: should it actually go to settled?.
func (m *Market) cleanMarketWithState(ctx context.Context, mktState types.MarketState) error {
	clearMarketTransfers, err := m.collateral.ClearSpotMarket(ctx, m.GetID(), m.quoteAsset)
	if err != nil {
		m.log.Error("Clear market error",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	m.stateVarEngine.UnregisterStateVariable(m.baseAsset+"_"+m.quoteAsset, m.mkt.ID)
	if len(clearMarketTransfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, clearMarketTransfers))
	}

	m.mkt.State = mktState
	m.mkt.TradingMode = types.MarketTradingModeNoTrading
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	m.closed = true
	return nil
}

// closeCancelledMarket cleans up after a cancelled market.
func (m *Market) closeCancelledMarket(ctx context.Context) error {
	if err := m.cleanMarketWithState(ctx, types.MarketStateCancelled); err != nil {
		return err
	}

	m.liquidity.StopAllLiquidityProvision(ctx)
	m.closed = true
	return nil
}

// closeMarket
// NB: this is currently called immediately from terminate trading.
func (m *Market) closeMarket(ctx context.Context) error {
	// final distribution of liquidity fees
	m.liquidity.OnMarketClosed(ctx, m.timeService.GetTimeNow())
	err := m.cleanMarketWithState(ctx, types.MarketStateClosed)
	if err != nil {
		return err
	}

	m.removeOrders(ctx)
	m.liquidity.StopAllLiquidityProvision(ctx)
	return nil
}

// unregisterAndReject - the order didn't go to the book therefore there's no need to release funds from the holding account.
func (m *Market) unregisterAndReject(ctx context.Context, order *types.Order, err error) error {
	order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
	order.Status = types.OrderStatusRejected
	if oerr, ok := types.IsOrderError(err); ok {
		// the order wasn't invalid, so stopped is a better status, rather than rejected.
		if types.IsStoppingOrder(oerr) {
			order.Status = types.OrderStatusStopped
		}
		order.Reason = oerr
	} else {
		// should not happened but still...
		order.Reason = types.OrderErrorInternalError
	}
	m.broker.Send(events.NewOrderEvent(ctx, order))
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Failure after submitting order to matching engine",
			logging.Order(*order),
			logging.Error(err))
	}
	return err
}

// getNewPeggedPrice calculates pegged price based on the pegged reference and current prices.
func (m *Market) getNewPeggedPrice(order *types.Order) (*num.Uint, error) {
	if m.as.InAuction() {
		return num.UintZero(), common.ErrCannotRepriceDuringAuction
	}

	var (
		err   error
		price *num.Uint
	)

	switch order.PeggedOrder.Reference {
	case types.PeggedReferenceMid:
		price, err = m.getStaticMidPrice(order.Side)
	case types.PeggedReferenceBestBid:
		price, err = m.getBestStaticBidPrice()
	case types.PeggedReferenceBestAsk:
		price, err = m.getBestStaticAskPrice()
	}
	if err != nil {
		return num.UintZero(), common.ErrUnableToReprice
	}

	offset := num.UintZero().Mul(order.PeggedOrder.Offset, m.priceFactor)
	if order.Side == types.SideSell {
		return price.AddSum(offset), nil
	}

	if price.LTE(offset) {
		return num.UintZero(), common.ErrUnableToReprice
	}

	return num.UintZero().Sub(price, offset), nil
}

// Reprice a pegged order. This only updates the price on the order.
func (m *Market) repricePeggedOrder(order *types.Order) error {
	// Work out the new price of the order
	price, err := m.getNewPeggedPrice(order)
	if err != nil {
		return err
	}
	original := price.Clone()
	order.OriginalPrice = original.Div(original, m.priceFactor) // set original price in market precision
	order.Price = price
	return nil
}

// parkAllPeggedOrders parks all pegged orders.
func (m *Market) parkAllPeggedOrders(ctx context.Context) {
	toParkIDs := m.matching.GetActivePeggedOrderIDs()
	for _, order := range toParkIDs {
		m.parkOrder(ctx, order)
	}
}

// EnterAuction : Prepare the order book to be run as an auction.
// when entering an auction we need to make sure there's sufficient funds in the holding account to cover the potential trade + fees.
// If there isn't, the order must be cancelled.
func (m *Market) enterAuction(ctx context.Context) {
	// Change market type to auction
	ordersToCancel := m.matching.EnterAuction()

	// Move into auction mode to prevent pegged order repricing
	event := m.as.AuctionStarted(ctx, m.timeService.GetTimeNow())

	// Cancel all the orders that were invalid
	for _, order := range ordersToCancel {
		_, err := m.cancelOrder(ctx, order.Party, order.ID)
		if err != nil {
			m.log.Debug("error cancelling order when entering auction",
				logging.MarketID(m.GetID()),
				logging.OrderID(order.ID),
				logging.Error(err))
		}
	}

	// now update all special orders
	m.enterAuctionSpecialOrders(ctx)

	// now that all orders that don't fit in auctions have been cancelled, process necessary transfer of fees from the general account of the
	// buyers to the holding account. Orders with insufficient cover of buyer or where the quantity to be delivered to the seller does not cover
	// for the due fees during auction are cancelled here.
	m.processFeesTransfersOnEnterAuction(ctx)

	// Send an event bus update
	m.broker.Send(event)

	if m.as.InAuction() && m.as.IsPriceAuction() {
		m.mkt.State = types.MarketStateSuspended
		m.mkt.TradingMode = types.MarketTradingModeMonitoringAuction
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	}
}

// OnOpeningAuctionFirstUncrossingPrice is triggered when the opening auction sees an uncrossing price for the first time and emits
// an event to the state variable engine.
func (m *Market) OnOpeningAuctionFirstUncrossingPrice() {
	m.log.Info("OnOpeningAuctionFirstUncrossingPrice event fired", logging.String("market", m.mkt.ID))
	m.stateVarEngine.ReadyForTimeTrigger(m.quoteAsset, m.mkt.ID)
	m.stateVarEngine.NewEvent(m.quoteAsset, m.mkt.ID, statevar.EventTypeOpeningAuctionFirstUncrossingPrice)
}

// OnAuctionEnded is called whenever an auction is ended and emits an event to the state var engine.
func (m *Market) OnAuctionEnded() {
	m.log.Info("OnAuctionEnded event fired", logging.String("market", m.mkt.ID))
	m.stateVarEngine.NewEvent(m.quoteAsset, m.mkt.ID, statevar.EventTypeAuctionEnded)
}

// leaveAuction : Return the orderbook and market to continuous trading.
func (m *Market) leaveAuction(ctx context.Context, now time.Time) {
	defer func() {
		if !m.as.InAuction() && (m.mkt.State == types.MarketStateSuspended || m.mkt.State == types.MarketStatePending || m.mkt.State == types.MarketStateSuspendedViaGovernance) {
			if m.mkt.State == types.MarketStatePending {
				// the market is now properly open,
				// so set the timestamp to when the opening auction actually ended
				m.mkt.MarketTimestamps.Open = now.UnixNano()
			}
			if m.mkt.TradingMode != types.MarketTradingModeOpeningAuction {
				// if we're leaving a price monitoring auction we can release the fees funds locked for the duration of the auction for any uncrossed orders
				m.processFeesReleaseOnLeaveAuction(ctx)
			}
			m.mkt.State = types.MarketStateActive
			m.mkt.TradingMode = types.MarketTradingModeContinuous
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
			m.updateLiquidityFee(ctx)
			m.OnAuctionEnded()
		}
	}()

	_, ordersToCancel := m.uncrossOnLeaveAuction(ctx)

	// Process each order we have to cancel
	for _, order := range ordersToCancel {
		_, err := m.cancelOrder(ctx, order.Party, order.ID)
		if err != nil {
			m.log.Panic("Failed to cancel order",
				logging.Error(err),
				logging.String("OrderID", order.ID))
		}
	}

	// update auction state, so we know what the new tradeMode ought to be
	endEvt := m.as.Left(ctx, now)

	previousMarkPrice := m.getCurrentMarkPrice()
	// set the mark price here so that margins checks for special orders use the correct value
	m.markPrice = m.getLastTradedPrice()

	m.checkForReferenceMoves(ctx, true)
	if !m.as.InAuction() {
		// only send the auction-left event if we actually *left* the auction.
		m.broker.Send(endEvt)
		m.nextMTM = m.timeService.GetTimeNow().Add(m.mtmDelta)
	} else {
		// revert to old mark price if we're not leaving the auction after all
		m.markPrice = previousMarkPrice
	}
}

// validateOrder checks that the order parameters are valid for the market.
func (m *Market) validateOrder(ctx context.Context, order *types.Order) (err error) {
	defer func() {
		if err != nil {
			order.Status = types.OrderStatusRejected
			m.broker.Send(events.NewOrderEvent(ctx, order))
		}
	}()

	// Check we are allowed to handle this order type with the current market status
	isAuction := m.as.InAuction()
	if isAuction && order.TimeInForce == types.OrderTimeInForceGFN {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorCannotSendGFNOrderDuringAnAuction
		return common.ErrGFNOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceIOC {
		order.Reason = types.OrderErrorCannotSendIOCOrderDuringAuction
		return common.ErrIOCOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceFOK {
		order.Reason = types.OrderErrorCannotSendFOKOrderDurinAuction
		return common.ErrFOKOrderReceivedAuctionTrading
	}

	if !isAuction && order.TimeInForce == types.OrderTimeInForceGFA {
		order.Reason = types.OrderErrorGFAOrderDuringContinuousTrading
		return common.ErrGFAOrderReceivedDuringContinuousTrading
	}

	// Check the expiry time is valid
	if order.ExpiresAt > 0 && order.ExpiresAt < order.CreatedAt {
		order.Reason = types.OrderErrorInvalidExpirationDatetime
		return common.ErrInvalidExpiresAtTime
	}

	if m.closed {
		// adding order to the buffer first
		order.Reason = types.OrderErrorMarketClosed
		return common.ErrMarketClosed
	}

	if order.Type == types.OrderTypeNetwork {
		order.Reason = types.OrderErrorInvalidType
		return common.ErrInvalidOrderType
	}

	// Validate market
	if order.MarketID != m.mkt.ID {
		// adding order to the buffer first
		order.Reason = types.OrderErrorInvalidMarketID
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.Order(*order),
				logging.String("market", m.mkt.ID))
		}
		return types.ErrInvalidMarketID
	}

	// Validate pegged orders
	if order.PeggedOrder != nil {
		if reason := order.ValidatePeggedOrder(); reason != types.OrderErrorUnspecified {
			order.Reason = reason
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failed to validate pegged order details",
					logging.Order(*order),
					logging.String("market", m.mkt.ID))
			}
			return reason
		}
	}

	return nil
}

// validateAccounts checks that the party has the required accounts and that they have sufficient funds in the account to cover for the trade and
// any fees due.
func (m *Market) validateAccounts(ctx context.Context, order *types.Order) error {
	if (order.Side == types.SideBuy && !m.collateral.HasGeneralAccount(order.Party, m.quoteAsset)) ||
		(order.Side == types.SideSell && !m.collateral.HasGeneralAccount(order.Party, m.baseAsset)) {
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorInsufficientAssetBalance
		m.broker.Send(events.NewOrderEvent(ctx, order))

		// party should be created before even trying to post order
		return common.ErrPartyInsufficientAssetBalance
	}

	price := order.Price
	// pegged order would not have a price at this point so unless we're in auction we need to get a price for it first
	if order.PeggedOrder != nil && !m.as.InAuction() {
		p, err := m.getNewPeggedPrice(order)
		if err != nil {
			return err
		}
		price = p
	}
	// if the order is not pegged or it is pegged and we're not in an auction, check the party has sufficient balance
	if order.PeggedOrder == nil || !m.as.InAuction() {
		if err := m.checkSufficientFunds(order.Party, order.Side, price, order.TrueRemaining(), order.PeggedOrder != nil); err != nil {
			return err
		}
	}

	// from this point we know the party have the necessary accounts and balances
	// we had it to the list of parties.
	m.addParty(order.Party)
	return nil
}

func rejectStopOrders(rejectionReason types.StopOrderRejectionReason, orders ...*types.StopOrder) {
	for _, o := range orders {
		if o != nil {
			o.Status = types.StopOrderStatusRejected
			o.RejectionReason = ptr.From(rejectionReason)
		}
	}
}

func (m *Market) SubmitStopOrdersWithIDGeneratorAndOrderIDs(
	ctx context.Context,
	submission *types.StopOrdersSubmission,
	party string,
	idgen common.IDGenerator,
	fallsBelowID, risesAboveID *string,
) (*types.OrderConfirmation, error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

	fallsBelow, risesAbove := submission.IntoStopOrders(
		party, ptr.UnBox(fallsBelowID), ptr.UnBox(risesAboveID), m.timeService.GetTimeNow())

	defer func() {
		evts := []events.Event{}
		if fallsBelow != nil {
			evts = append(evts, events.NewStopOrderEvent(ctx, fallsBelow))
		}
		if risesAbove != nil {
			evts = append(evts, events.NewStopOrderEvent(ctx, risesAbove))
		}

		if len(evts) > 0 {
			m.broker.SendBatch(evts)
		}
	}()

	if m.IsOpeningAuction() {
		rejectStopOrders(types.StopOrderRejectionNotAllowedDuringOpeningAuction, fallsBelow, risesAbove)
		return nil, common.ErrStopOrderNotAllowedDuringOpeningAuction
	}

	if !m.canTrade() {
		rejectStopOrders(types.StopOrderRejectionTradingNotAllowed, fallsBelow, risesAbove)
		return nil, common.ErrTradingNotAllowed
	}

	orderCnt := 0

	if fallsBelow != nil {
		if fallsBelow.OrderSubmission.Side == types.SideBuy && !m.collateral.HasGeneralAccount(party, m.quoteAsset) {
			rejectStopOrders(types.StopOrderRejectionNotClosingThePosition, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderSideNotClosingThePosition
		}
		if !m.collateral.HasGeneralAccount(party, m.baseAsset) {
			rejectStopOrders(types.StopOrderRejectionNotClosingThePosition, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderSideNotClosingThePosition
		}
		orderCnt++
	}
	if risesAbove != nil {
		if risesAbove.OrderSubmission.Side == types.SideBuy && !m.collateral.HasGeneralAccount(party, m.quoteAsset) {
			rejectStopOrders(types.StopOrderRejectionNotClosingThePosition, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderSideNotClosingThePosition
		}
		if !m.collateral.HasGeneralAccount(party, m.baseAsset) {
			rejectStopOrders(types.StopOrderRejectionNotClosingThePosition, fallsBelow, risesAbove)
			return nil, common.ErrStopOrderSideNotClosingThePosition
		}
		orderCnt++
	}

	// now check if that party hasn't exceeded the max amount per market
	if m.stopOrders.CountForParty(party)+uint64(orderCnt) > m.maxStopOrdersPerParties.Uint64() {
		rejectStopOrders(types.StopOrderRejectionMaxStopOrdersPerPartyReached, fallsBelow, risesAbove)
		return nil, common.ErrMaxStopOrdersPerPartyReached
	}

	fallsBelowTriggered, risesAboveTriggered := m.stopOrderWouldTriggerAtSubmission(fallsBelow), m.stopOrderWouldTriggerAtSubmission(risesAbove)
	triggered := fallsBelowTriggered || risesAboveTriggered

	// if we are in an auction
	// or no order is triggered
	// let's just submit it straight away
	if m.as.InAuction() || !triggered {
		m.poolStopOrders(ctx, fallsBelow, risesAbove)
		return nil, nil
	}

	var confirmation *types.OrderConfirmation
	var err error
	// now would the order get trigger straight away?
	switch {
	case fallsBelowTriggered:
		fallsBelow.Status = types.StopOrderStatusTriggered
		if risesAbove != nil {
			risesAbove.Status = types.StopOrderStatusStopped
		}
		fallsBelow.OrderID = idgen.NextID()
		confirmation, err = m.SubmitOrderWithIDGeneratorAndOrderID(
			ctx, fallsBelow.OrderSubmission, party, idgen, fallsBelow.OrderID, true,
		)
		if err != nil && confirmation != nil {
			fallsBelow.OrderID = confirmation.Order.ID
		}
	case risesAboveTriggered:
		risesAbove.Status = types.StopOrderStatusTriggered
		if fallsBelow != nil {
			fallsBelow.Status = types.StopOrderStatusStopped
		}
		risesAbove.OrderID = idgen.NextID()
		confirmation, err = m.SubmitOrderWithIDGeneratorAndOrderID(
			ctx, risesAbove.OrderSubmission, party, idgen, risesAbove.OrderID, true,
		)
		if err != nil && confirmation != nil {
			risesAbove.OrderID = confirmation.Order.ID
		}
	}

	return confirmation, err
}

func (m *Market) poolStopOrders(
	ctx context.Context,
	fallsBelow, risesAbove *types.StopOrder,
) {
	evts := []events.Event{}
	if fallsBelow != nil {
		m.stopOrders.Insert(fallsBelow)
		if fallsBelow.Expiry.Expires() {
			m.expiringStopOrders.Insert(fallsBelow.ID, fallsBelow.CreatedAt.UnixNano())
		}
		evts = append(evts, events.NewStopOrderEvent(ctx, fallsBelow))
	}
	if risesAbove != nil {
		m.stopOrders.Insert(risesAbove)
		if risesAbove.Expiry.Expires() {
			m.expiringStopOrders.Insert(risesAbove.ID, risesAbove.CreatedAt.UnixNano())
		}
		evts = append(evts, events.NewStopOrderEvent(ctx, risesAbove))
	}

	m.broker.SendBatch(evts)
}

func (m *Market) stopOrderWouldTriggerAtSubmission(
	stopOrder *types.StopOrder,
) bool {
	if m.lastTradedPrice == nil || stopOrder == nil || stopOrder.Trigger.IsTrailingPercentOffset() {
		return false
	}

	switch stopOrder.Trigger.Direction {
	case types.StopOrderTriggerDirectionFallsBelow:
		if m.lastTradedPrice.LTE(stopOrder.Trigger.Price()) {
			return true
		}
	case types.StopOrderTriggerDirectionRisesAbove:
		if m.lastTradedPrice.GTE(stopOrder.Trigger.Price()) {
			return true
		}
	}
	return false
}

func (m *Market) triggerStopOrders(
	ctx context.Context,
	idgen common.IDGenerator,
) []*types.OrderConfirmation {
	if m.lastTradedPrice == nil {
		return nil
	}

	triggered, cancelled := m.stopOrders.PriceUpdated(m.lastTradedPrice)

	if len(triggered) <= 0 {
		return nil
	}

	evts := make([]events.Event, 0, len(cancelled))
	for _, v := range cancelled {
		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	confirmations := m.submitStopOrders(ctx, triggered, types.StopOrderStatusTriggered, idgen)

	return append(m.triggerStopOrders(ctx, idgen), confirmations...)
}

// SubmitOrder submits the given order.
func (m *Market) SubmitOrder(ctx context.Context, orderSubmission *types.OrderSubmission, party string, deterministicID string) (oc *types.OrderConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.SubmitOrderWithIDGeneratorAndOrderID(ctx, orderSubmission, party, idgen, idgen.NextID(), true)
}

// SubmitOrderWithIDGeneratorAndOrderID submits the given order.
func (m *Market) SubmitOrderWithIDGeneratorAndOrderID(ctx context.Context, orderSubmission *types.OrderSubmission, party string, idgen common.IDGenerator, orderID string, checkForTriggers bool) (oc *types.OrderConfirmation, _ error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

	defer func() {
		if !checkForTriggers {
			return
		}

		m.triggerStopOrders(ctx, idgen)
	}()

	order := orderSubmission.IntoOrder(party)
	if order.Price != nil {
		order.OriginalPrice = order.Price.Clone()
		order.Price.Mul(order.Price, m.priceFactor)
	}
	order.CreatedAt = m.timeService.GetTimeNow().UnixNano()
	order.ID = orderID

	if !m.canTrade() {
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorMarketClosed
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, common.ErrTradingNotAllowed
	}

	conf, _, err := m.submitOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	if !m.as.InAuction() {
		m.checkForReferenceMoves(
			ctx, false)
	}

	return conf, nil
}

// submitOrder validates and submits an order.
func (m *Market) submitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, []*types.Order, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.ID, orderValidity)
	}()

	// set those at the beginning as even rejected order get through the buffers
	order.Version = common.InitialOrderVersion
	order.Status = types.OrderStatusActive

	if err := m.validateOrder(ctx, order); err != nil {
		return nil, nil, err
	}

	if err := m.validateAccounts(ctx, order); err != nil {
		return nil, nil, err
	}

	// Now that validation is handled, call the code to place the order
	orderConf, orderUpdates, err := m.submitValidatedOrder(ctx, order)
	if err == nil {
		orderValidity = "valid"
	}

	if order.PeggedOrder != nil && order.IsFinished() {
		// remove the pegged order from anywhere
		m.removePeggedOrder(order)
	}

	// insert an expiring order if it's either in the book
	// or in the parked list
	if order.IsExpireable() && !order.IsFinished() {
		m.expiringOrders.Insert(order.ID, order.ExpiresAt)
	}

	return orderConf, orderUpdates, err
}

// submitValidatedOrder submits a new order.
func (m *Market) submitValidatedOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, []*types.Order, error) {
	isPegged := order.PeggedOrder != nil
	if isPegged {
		order.Status = types.OrderStatusParked
		order.Reason = types.OrderErrorUnspecified

		if m.as.InAuction() {
			order.SetIcebergPeaks()
			// as the order can't trade we don't transfer from the general account to the holding account in this case.
			m.peggedOrders.Park(order)
			// If we are in an auction, we don't insert this order into the book
			// Maybe should return an orderConfirmation with order state PARKED
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil, nil
		}
		err := m.repricePeggedOrder(order)
		if err != nil {
			order.SetIcebergPeaks()
			m.peggedOrders.Park(order)
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil, nil // nolint
		}
	}
	var trades []*types.Trade
	// we're not in auction (not opening, not any other auction
	if !m.as.InAuction() {
		// first we call the order book to evaluate auction triggers and get the list of trades
		var err error
		trades, err = m.checkPriceAndGetTrades(ctx, order)
		if err != nil {
			return nil, nil, m.unregisterAndReject(ctx, order, err)
		}
		// NB we don't apply fees here because if this is a sell the fees are taken from the quantity that the buyer pays (in quote asset)
		// so this is deferred to handling confirmations - by this point the aggressor must have sufficient funds to cover for fees so this should
		// not be an issue
	}

	// if an auction is ongoing and the order is pegged, park it and return
	if m.as.InAuction() && isPegged {
		if isPegged {
			m.peggedOrders.Park(order)
		}
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return &types.OrderConfirmation{Order: order}, nil, nil
	}

	order.Status = types.OrderStatusActive

	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(order)
	if err != nil {
		return nil, nil, m.unregisterAndReject(ctx, order, err)
	}

	// if the order is not finished and remaining is non zero, we need to transfer the remaining base/quote from the general account
	// to the holding account for the market/asset. If an auction is on-going we also need to account for potential fees (applicable for buy orders only)
	if !order.IsFinished() && order.Remaining > 0 {
		err := m.transferToHoldingAccount(ctx, order)
		if err != nil {
			return nil, nil, m.unregisterAndReject(ctx, order, err)
		}
	}

	// we replace the trades in the confirmation with the one we got initially
	// the contains the fees information
	// NB: I have to say this this is a weird way of doing it, why are we doing it twice?
	confirmation.Trades = trades

	// Send out the order update here as handling the confirmation message
	// below might trigger an action that can change the order details.
	m.broker.Send(events.NewOrderEvent(ctx, order))

	orderUpdates := m.handleConfirmation(ctx, confirmation)
	m.handleConfirmationPassiveOrders(ctx, confirmation)
	return confirmation, orderUpdates, nil
}

// checkPriceAndGetTrades calculates the trades that would be generated from the given order.
func (m *Market) checkPriceAndGetTrades(ctx context.Context, order *types.Order) ([]*types.Trade, error) {
	trades, err := m.matching.GetTrades(order)
	if err != nil {
		return nil, err
	}

	if order.PostOnly && len(trades) > 0 {
		return nil, types.OrderErrorPostOnlyOrderWouldTrade
	}

	persistent := true
	switch order.TimeInForce {
	case types.OrderTimeInForceFOK, types.OrderTimeInForceGFN, types.OrderTimeInForceIOC:
		persistent = false
	}

	if m.pMonitor.CheckPrice(ctx, m.as, trades, persistent) {
		return nil, types.OrderErrorNonPersistentOrderOutOfPriceBounds
	}

	if evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow()); evt != nil {
		m.broker.Send(evt)
	}

	// start the  monitoring auction if required?
	if m.as.AuctionStart() {
		m.enterAuction(ctx)
		return nil, nil
	}

	return trades, nil
}

// addParty adds the party to the market mapping.
func (m *Market) addParty(party string) {
	if _, ok := m.parties[party]; !ok {
		m.parties[party] = struct{}{}
	}
}

// applyFees handles transfer of fee payment from the *buyer* to the fees account.
func (m *Market) applyFees(ctx context.Context, fees events.FeesTransfer) error {
	var (
		transfers []*types.LedgerMovement
		err       error
	)

	if !m.as.InAuction() {
		transfers, err = m.collateral.TransferSpotFeesContinuousTrading(ctx, m.GetID(), m.quoteAsset, fees)
	} else if m.as.IsMonitorAuction() {
		transfers, err = m.collateral.TransferSpotFees(ctx, m.GetID(), m.quoteAsset, fees)
	} else if m.as.IsFBA() {
		transfers, err = m.collateral.TransferSpotFees(ctx, m.GetID(), m.quoteAsset, fees)
	}

	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}

	m.marketActivityTracker.UpdateFeesFromTransfers(m.quoteAsset, m.GetID(), fees.Transfers())
	return err
}

func (m *Market) handleConfirmationPassiveOrders(ctx context.Context, conf *types.OrderConfirmation) {
	le := []*types.LedgerMovement{}

	if conf.PassiveOrdersAffected != nil {
		evts := make([]events.Event, 0, len(conf.PassiveOrdersAffected))

		// Insert or update passive orders siting on the book
		for _, order := range conf.PassiveOrdersAffected {
			// set the `updatedAt` value as these orders have changed
			order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
			evts = append(evts, events.NewOrderEvent(ctx, order))

			// If the order is a pegged order and is complete we must remove it from the pegged list
			if order.PeggedOrder != nil {
				if order.Remaining == 0 || order.Status != types.OrderStatusActive {
					m.removePeggedOrder(order)
				}
			}

			if order.IsFinished() {
				m.releaseOrderFromHoldingAccount(ctx, order.ID, order.Party, order.Side)
			}

			// remove the order from the expiring list
			// if it was a GTT order
			if order.IsExpireable() && order.IsFinished() {
				m.expiringOrders.RemoveOrder(order.ExpiresAt, order.ID)
			}
		}
		if len(le) > 0 {
			m.broker.Send(events.NewLedgerMovements(ctx, le))
		}
		m.broker.SendBatch(evts)
	}
}

func (m *Market) handleConfirmation(ctx context.Context, conf *types.OrderConfirmation) []*types.Order {
	// When re-submitting liquidity order, it happen that the pricing is putting
	// the order at a price which makes it uncross straight away.
	// then triggering this handleConfirmation flow, etc.
	// Although the order is considered aggressive, and we never expect in the flow
	// for an aggressive order to be pegged, so we never remove them from the pegged
	// list. All this impact the float of EnterAuction, which if triggered from there
	// will try to park all pegged orders, including this order which have never been
	// removed from the pegged list. We add this check to make sure  that if the
	// aggressive order is pegged, we then do remove it from the list.
	if conf.Order.PeggedOrder != nil {
		if conf.Order.Remaining == 0 || conf.Order.Status != types.OrderStatusActive {
			m.removePeggedOrder(conf.Order)
		}
	}

	end := m.as.CanLeave()
	orderUpdates := make([]*types.Order, 0, len(conf.PassiveOrdersAffected)+1)
	orderUpdates = append(orderUpdates, conf.Order)
	orderUpdates = append(orderUpdates, conf.PassiveOrdersAffected...)

	if len(conf.Trades) == 0 {
		return orderUpdates
	}
	m.setLastTradedPrice(conf.Trades[len(conf.Trades)-1])
	m.hasTraded = true

	// Insert all trades resulted from the executed order
	tradeEvts := make([]events.Event, 0, len(conf.Trades))
	tradedValue, _ := num.UintFromDecimal(
		conf.TradedValue().ToDecimal().Div(m.positionFactor))

	transfers := []*types.LedgerMovement{}
	for idx, trade := range conf.Trades {
		trade.SetIDs(m.idgen.NextID(), conf.Order, conf.PassiveOrdersAffected[idx])

		tradeTransfers := m.handleTrade(ctx, trade)
		transfers = append(transfers, tradeTransfers...)
		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))
	}
	if conf.Order.IsFinished() {
		m.releaseOrderFromHoldingAccount(ctx, conf.Order.ID, conf.Order.Party, conf.Order.Side)
	}

	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}

	m.feeSplitter.AddTradeValue(tradedValue)
	m.marketActivityTracker.AddValueTraded(m.quoteAsset, m.mkt.ID, tradedValue)
	m.broker.SendBatch(tradeEvts)
	// check reference moves if we have order updates, and we are not in an auction (or leaving an auction)
	// we handle reference moves in confirmMTM when leaving an auction already
	if len(orderUpdates) > 0 && !end && !m.as.InAuction() {
		m.checkForReferenceMoves(
			ctx, false)
	}

	return orderUpdates
}

// updateLiquidityFee computes the current LiquidityProvision fee and updates
// the fee engine.
func (m *Market) updateLiquidityFee(ctx context.Context) {
	var fee num.Decimal
	switch m.mkt.Fees.LiquidityFeeSettings.Method {
	case types.LiquidityFeeMethodConstant:
		fee = m.mkt.Fees.LiquidityFeeSettings.FeeConstant
	case types.LiquidityFeeMethodMarginalCost:
		fee = m.liquidityEngine.ProvisionsPerParty().FeeForTarget(m.getTargetStake())
	case types.LiquidityFeeMethodWeightedAverage:
		fee = m.liquidityEngine.ProvisionsPerParty().FeeForWeightedAverage()
	default:
		m.log.Panic("unknown liquidity fee method")
	}

	if !fee.Equals(m.getLiquidityFee()) {
		m.fee.SetLiquidityFee(fee)
		m.setLiquidityFee(fee)
		m.broker.Send(
			events.NewMarketUpdatedEvent(ctx, *m.mkt),
		)
	}
}

func (m *Market) setLiquidityFee(fee num.Decimal) {
	m.mkt.Fees.Factors.LiquidityFee = fee
}

func (m *Market) getLiquidityFee() num.Decimal {
	return m.mkt.Fees.Factors.LiquidityFee
}

func (m *Market) setLastTradedPrice(trade *types.Trade) {
	m.lastTradedPrice = trade.Price.Clone()
}

func (m *Market) CancelAllStopOrders(ctx context.Context, partyID string) error {
	if !m.canTrade() {
		return common.ErrTradingNotAllowed
	}

	stopOrders, err := m.stopOrders.Cancel(partyID, "")
	if err != nil {
		return err
	}

	m.removeCancelledExpiringStopOrders(stopOrders)

	evts := make([]events.Event, 0, len(stopOrders))
	for _, v := range stopOrders {
		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	return nil
}

// CancelAllOrders cancels all orders in the market.
func (m *Market) CancelAllOrders(ctx context.Context, partyID string) ([]*types.OrderCancellationConfirmation, error) {
	if !m.canTrade() {
		return nil, common.ErrTradingNotAllowed
	}

	// get all order for this party in the book
	orders := m.matching.GetOrdersPerParty(partyID)

	// add all orders being eventually parked
	orders = append(orders, m.peggedOrders.GetAllParkedForParty(partyID)...)

	// just an early exit, there's just no orders...
	if len(orders) <= 0 {
		return nil, nil
	}

	// now we eventually dedup them
	uniq := map[string]*types.Order{}
	for _, v := range orders {
		uniq[v.ID] = v
	}

	// put them back in the slice, and sort them
	orders = make([]*types.Order, 0, len(uniq))
	for _, v := range uniq {
		orders = append(orders, v)
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].ID < orders[j].ID
	})

	cancellations := make([]*types.OrderCancellationConfirmation, 0, len(orders))

	// now iterate over all orders and cancel one by one.
	for _, order := range orders {
		cancellation, err := m.cancelOrder(ctx, partyID, order.ID)
		if err != nil {
			return nil, err
		}
		cancellations = append(cancellations, cancellation)
	}

	m.checkForReferenceMoves(ctx, false)

	return cancellations, nil
}

func (m *Market) CancelStopOrder(
	ctx context.Context,
	partyID, orderID string,
) error {
	if !m.canTrade() {
		return common.ErrTradingNotAllowed
	}

	stopOrders, err := m.stopOrders.Cancel(partyID, orderID)
	if err != nil {
		return err
	}

	m.removeCancelledExpiringStopOrders(stopOrders)

	evts := []events.Event{events.NewStopOrderEvent(ctx, stopOrders[0])}
	if len(stopOrders) > 1 {
		evts = append(evts, events.NewStopOrderEvent(ctx, stopOrders[1]))
	}

	m.broker.SendBatch(evts)

	return nil
}

func (m *Market) removeCancelledExpiringStopOrders(
	stopOrders []*types.StopOrder,
) {
	for _, o := range stopOrders {
		if o.Expiry.Expires() {
			m.expiringStopOrders.RemoveOrder(o.Expiry.ExpiresAt.UnixNano(), o.ID)
		}
	}
}

// CancelOrder canels a single order in the market.
func (m *Market) CancelOrder(ctx context.Context, partyID, orderID string, deterministicID string) (oc *types.OrderCancellationConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.CancelOrderWithIDGenerator(ctx, partyID, orderID, idgen)
}

// CancelOrderWithIDGenerator cancels an order in the market.
func (m *Market) CancelOrderWithIDGenerator(ctx context.Context, partyID, orderID string, idgen common.IDGenerator) (oc *types.OrderCancellationConfirmation, _ error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

	if !m.canTrade() {
		return nil, common.ErrTradingNotAllowed
	}

	conf, err := m.cancelOrder(ctx, partyID, orderID)
	if err != nil {
		return conf, err
	}

	if !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, false)
	}

	return conf, nil
}

// CancelOrder cancels the given order. If the order is found on the book, we release locked funds from holding account to the general account of the party.
func (m *Market) cancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, common.ErrMarketClosed
	}

	order, foundOnBook, err := m.getOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	// Only allow the original order creator to cancel their order
	if order.Party != partyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Party ID mismatch",
				logging.String("party-id", partyID),
				logging.String("order-id", orderID),
				logging.String("market", m.mkt.ID))
		}
		return nil, types.ErrInvalidPartyID
	}

	if foundOnBook {
		cancellation, err := m.matching.CancelOrder(order)
		if cancellation == nil || err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failure after cancel order from matching engine",
					logging.String("party-id", partyID),
					logging.String("order-id", orderID),
					logging.String("market", m.mkt.ID),
					logging.Error(err))
			}
			return nil, err
		}
		m.releaseOrderFromHoldingAccount(ctx, orderID, order.Party, order.Side)
	}

	if order.IsExpireable() {
		m.expiringOrders.RemoveOrder(order.ExpiresAt, order.ID)
	}

	// If this is a pegged order, remove from pegged and parked lists
	if order.PeggedOrder != nil {
		m.removePeggedOrder(order)
		order.Status = types.OrderStatusCancelled
	}

	// Publish the changed order details
	order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
	m.broker.Send(events.NewOrderEvent(ctx, order))

	return &types.OrderCancellationConfirmation{Order: order}, nil
}

// parkOrder removes the given order from the orderbook. parkOrder will panic if it encounters errors, which means that it reached an
// invalid state. When the order is parked, the funds from the holding account are released to the general account.
func (m *Market) parkOrder(ctx context.Context, orderID string) *types.Order {
	order, err := m.matching.RemoveOrder(orderID)
	if err != nil {
		m.log.Panic("Failure to remove order from matching engine",
			logging.OrderID(orderID),
			logging.Error(err))
	}
	m.releaseOrderFromHoldingAccount(ctx, orderID, order.Party, order.Side)
	m.peggedOrders.Park(order)
	m.broker.Send(events.NewOrderEvent(ctx, order))
	return order
}

// AmendOrder amend an existing order from the order book.
func (m *Market) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment, party string, deterministicID string) (oc *types.OrderConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.AmendOrderWithIDGenerator(ctx, orderAmendment, party, idgen)
}

// AmendOrderWithIDGenerator amends an order.
func (m *Market) AmendOrderWithIDGenerator(ctx context.Context, orderAmendment *types.OrderAmendment, party string, idgen common.IDGenerator) (oc *types.OrderConfirmation, _ error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

	defer func() {
		m.triggerStopOrders(ctx, idgen)
	}()

	if !m.canTrade() {
		return nil, common.ErrTradingNotAllowed
	}

	conf, _, err := m.amendOrder(ctx, orderAmendment, party)
	if err != nil {
		return nil, err
	}

	if !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, false)
	}
	return conf, nil
}

// findOrderAndEnsureOwnership checks that the party is actually the owner of the order ID.
func (m *Market) findOrderAndEnsureOwnership(orderID, partyID, marketID string) (exitingOrder *types.Order, foundOnBook bool, err error) {
	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, foundOnBook, err := m.getOrderByID(orderID)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid order ID",
				logging.OrderID(orderID),
				logging.PartyID(partyID),
				logging.MarketID(marketID),
				logging.Error(err))
		}
		return nil, false, types.ErrInvalidOrderID
	}

	// We can only amend this order if we created it
	if existingOrder.Party != partyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid party ID",
				logging.String("original party id:", existingOrder.Party),
				logging.PartyID(partyID),
			)
		}
		return nil, false, types.ErrInvalidPartyID
	}

	// Validate Market
	if existingOrder.MarketID != marketID {
		// we should never reach this point
		m.log.Panic("Market ID mismatch",
			logging.MarketID(m.mkt.ID),
			logging.Order(*existingOrder),
			logging.Error(types.ErrInvalidMarketID),
		)
	}

	return existingOrder, foundOnBook, err
}

func (m *Market) amendOrder(ctx context.Context, orderAmendment *types.OrderAmendment, party string) (cnf *types.OrderConfirmation, orderUpdates []*types.Order, returnedErr error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, nil, common.ErrMarketClosed
	}

	existingOrder, foundOnBook, err := m.findOrderAndEnsureOwnership(orderAmendment.OrderID, party, m.GetID())
	if err != nil {
		return nil, nil, err
	}

	if err := m.validateOrderAmendment(existingOrder, orderAmendment); err != nil {
		return nil, nil, err
	}

	amendedOrder, err := existingOrder.ApplyOrderAmendment(orderAmendment, m.timeService.GetTimeNow().UnixNano(), m.priceFactor)
	if err != nil {
		return nil, nil, err
	}

	// We do this first, just in case the party would also have
	// change the expiry, and that would have been caught by
	// the follow up checks, so we do not insert a non-existing
	// order in the expiring orders
	// if remaining is reduces <= 0, then order is cancelled
	if amendedOrder.Remaining <= 0 {
		confirm, err := m.cancelOrder(
			ctx, existingOrder.Party, existingOrder.ID)
		if err != nil {
			return nil, nil, err
		}
		return &types.OrderConfirmation{
			Order: confirm.Order,
		}, nil, nil
	}

	// If we have a pegged order that is no longer expiring, we need to remove it
	var (
		needToRemoveExpiry, needToAddExpiry bool
		expiresAt                           int64
	)

	defer func() {
		// no errors, amend most likely happened properly
		if returnedErr == nil {
			if needToRemoveExpiry {
				m.expiringOrders.RemoveOrder(expiresAt, existingOrder.ID)
			}
			// need to make sure the order haven't been matched with the
			// amend, consuming the remain volume as well or we would
			// add an order while it's not needed to the expiring list
			if needToAddExpiry && cnf != nil && !cnf.Order.IsFinished() {
				m.expiringOrders.Insert(amendedOrder.ID, amendedOrder.ExpiresAt)
			}
		}
	}()

	// if we are amending from GTT to GTC, flag ready to remove from expiry list
	if existingOrder.IsExpireable() && !amendedOrder.IsExpireable() {
		// We no longer need to handle the expiry
		needToRemoveExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if we are amending from GTC to GTT, flag ready to add to expiry list
	if !existingOrder.IsExpireable() && amendedOrder.IsExpireable() {
		// We need to handle the expiry
		needToAddExpiry = true
	}

	// if both where expireable but we changed the duration
	// then we need to remove, then reinsert...
	if existingOrder.IsExpireable() && amendedOrder.IsExpireable() &&
		existingOrder.ExpiresAt != amendedOrder.ExpiresAt {
		// Still expiring but needs to be updated in the expiring
		// orders pool
		needToRemoveExpiry = true
		needToAddExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if expiration has changed and is before the original creation time, reject this amend
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < existingOrder.CreatedAt {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Amended expiry before original creation time",
				logging.Int64("existing-created-at", existingOrder.CreatedAt),
				logging.Int64("amended-expires-at", amendedOrder.ExpiresAt),
				logging.Order(*existingOrder))
		}
		return nil, nil, types.ErrInvalidExpirationDatetime
	}

	// if expiration has changed and is not 0, and is before currentTime
	// then we expire the order
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < amendedOrder.UpdatedAt {
		needToAddExpiry = false
		// remove the order from the expiring
		// at this point the order is still referenced at the time of expiry of the existingOrder
		if existingOrder.IsExpireable() {
			m.expiringOrders.RemoveOrder(existingOrder.ExpiresAt, amendedOrder.ID)
		}

		// Update the existing message in place before we cancel it
		if foundOnBook {
			// Do not amend in place, the amend could be something
			// not supported for an amend in place, and not pass
			// the validation of the order book
			cancellation, err := m.matching.CancelOrder(existingOrder)
			if cancellation == nil || err != nil {
				m.log.Panic("Failure to cancel order from matching engine",
					logging.String("party-id", amendedOrder.Party),
					logging.String("order-id", amendedOrder.ID),
					logging.String("market", m.mkt.ID),
					logging.Error(err))
			}
			m.releaseOrderFromHoldingAccount(ctx, existingOrder.ID, existingOrder.Party, existingOrder.Side)
		}

		// Update the order in our stores (will be marked as cancelled)
		// set the proper status
		amendedOrder.Status = types.OrderStatusExpired
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		m.removePeggedOrder(amendedOrder)

		return &types.OrderConfirmation{
			Order: amendedOrder,
		}, nil, nil
	}

	if existingOrder.PeggedOrder != nil {
		// Amend in place during an auction
		if m.as.InAuction() {
			ret := m.orderAmendWhenParked(amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		}
		err := m.repricePeggedOrder(amendedOrder)
		if err != nil {
			// Failed to reprice so we have to park the order
			if amendedOrder.Status != types.OrderStatusParked {
				// If we are live then park
				m.parkOrder(ctx, existingOrder.ID)
			}
			ret := m.orderAmendWhenParked(amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		}
		// We got a new valid price, if we are parked we need to unpark
		if amendedOrder.Status == types.OrderStatusParked {
			// we were parked, need to unpark
			m.peggedOrders.Unpark(amendedOrder.ID)
			return m.submitValidatedOrder(ctx, amendedOrder)
		}
	}

	priceShift := amendedOrder.Price.NEQ(existingOrder.Price)
	sizeIncrease := amendedOrder.Size > existingOrder.Size
	sizeDecrease := amendedOrder.Size < existingOrder.Size
	expiryChange := amendedOrder.ExpiresAt != existingOrder.ExpiresAt
	timeInForceChange := amendedOrder.TimeInForce != existingOrder.TimeInForce

	// If nothing changed, amend in place to update updatedAt and version number
	if !priceShift && !sizeIncrease && !sizeDecrease && !expiryChange && !timeInForceChange {
		ret := m.orderAmendInPlace(existingOrder, amendedOrder)
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		return ret, nil, nil
	}

	icebergSizeIncrease := false
	if amendedOrder.IcebergOrder != nil && sizeIncrease {
		// iceberg orders size changes can always be done in-place because they either:
		// 1) decrease the size, which is already done in-place for all orders
		// 2) increase the size, which only increases the reserved remaining and not the "active" remaining of the iceberg
		// so we set an icebergSizeIncrease to skip the cancel-replace flow.
		sizeIncrease = false
		icebergSizeIncrease = true
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		return m.orderCancelReplace(ctx, existingOrder, amendedOrder)
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease || timeInForceChange || icebergSizeIncrease {
		m.releaseOrderFromHoldingAccount(ctx, amendedOrder.ID, amendedOrder.Party, amendedOrder.Side)
		ret := m.orderAmendInPlace(existingOrder, amendedOrder)
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
		amt := m.calculateAmountBySide(ret.Order.Side, ret.Order.Price, ret.Order.TrueRemaining())
		fees := num.UintZero()
		var err error
		if m.as.InAuction() {
			fees, err = m.calculateFees(ret.Order.Party, ret.Order.TrueRemaining(), ret.Order.Price, ret.Order.Side)
			if err != nil {
				return nil, nil, m.unregisterAndReject(ctx, ret.Order, err)
			}
		}
		asset := m.quoteAsset
		if ret.Order.Side == types.SideSell {
			asset = m.baseAsset
		}
		transfer, err := m.orderHoldingTracker.TransferToHoldingAccount(ctx, ret.Order.ID, ret.Order.Party, asset, amt, fees)
		if err != nil {
			m.log.Panic("failed to transfer funds to holding account for order", logging.Order(ret.Order), logging.Error(err))
		}
		m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{transfer}))
		return ret, nil, nil
	}

	// we should never reach this point as amendment was validated before
	// and every kind should have been handled down here.
	m.log.Panic(
		"invalid amend did not match any amendment combination",
		logging.String("amended-order", amendedOrder.String()),
		logging.String("existing-order", amendedOrder.String()),
	)

	return nil, nil, types.ErrEditNotAllowed
}

func (m *Market) validateOrderAmendment(order *types.Order, amendment *types.OrderAmendment) error {
	if err := amendment.Validate(); err != nil {
		return err
	}
	// check TIME_IN_FORCE and expiry
	if amendment.TimeInForce == types.OrderTimeInForceGTT {
		// if expiresAt is before or equal to created at
		// we return an error, we know ExpiresAt is set because of amendment.Validate
		if *amendment.ExpiresAt <= order.CreatedAt {
			return types.OrderErrorExpiryAtBeforeCreatedAt
		}
	}

	if (amendment.TimeInForce == types.OrderTimeInForceGFN ||
		amendment.TimeInForce == types.OrderTimeInForceGFA) &&
		amendment.TimeInForce != order.TimeInForce {
		// We cannot amend to a GFA/GFN orders
		return types.OrderErrorCannotAmendToGFAOrGFN
	}

	if (order.TimeInForce == types.OrderTimeInForceGFN ||
		order.TimeInForce == types.OrderTimeInForceGFA) &&
		(amendment.TimeInForce != order.TimeInForce &&
			amendment.TimeInForce != types.OrderTimeInForceUnspecified) {
		// We cannot amend from a GFA/GFN orders
		return types.OrderErrorCannotAmendFromGFAOrGFN
	}

	if order.PeggedOrder == nil {
		// We cannot change a pegged orders details on a non pegged order
		if amendment.PeggedOffset != nil ||
			amendment.PeggedReference != types.PeggedReferenceUnspecified {
			return types.OrderErrorCannotAmendPeggedOrderDetailsOnNonPeggedOrder
		}
	} else if amendment.Price != nil {
		// We cannot change the price on a pegged order
		return types.OrderErrorUnableToAmendPriceOnPeggedOrder
	}

	// if side is buy we need to check that the party has sufficient funds in their general account to cover for the change in quote asset required
	if order.Side == types.SideBuy && (amendment.Price != nil || amendment.SizeDelta != 0) {
		remaining := order.Remaining
		// calculate the effective remaining after the change
		if amendment.SizeDelta < 0 {
			if remaining > uint64(-amendment.SizeDelta) {
				remaining -= uint64(-amendment.SizeDelta)
			} else {
				remaining = 0
			}
		} else {
			remaining += uint64(amendment.SizeDelta)
		}

		// if nothing remains then no need to check anything
		if remaining == 0 {
			return nil
		}

		// if the order is pegged and we're in an auction, we're done here
		if order.PeggedOrder != nil && m.as.InAuction() {
			return nil
		}

		existingHoldingQty, existingHoldingFee := m.orderHoldingTracker.GetCurrentHolding(order.ID)
		oldHoldingRequirement := num.Sum(existingHoldingQty, existingHoldingFee)
		newFeesRequirement := num.UintZero()
		if m.as.InAuction() {
			newFeesRequirement, _ = m.calculateFees(order.Party, remaining, amendment.Price, order.Side)
		}
		price := order.Price
		if amendment.Price != nil {
			price = num.UintZero().Mul(amendment.Price, m.priceFactor)
		}
		if order.PeggedOrder != nil {
			p, err := m.getNewPeggedPrice(order)
			if err != nil {
				return err
			}
			price = p
		}
		newHoldingRequirement := num.Sum(m.calculateAmountBySide(order.Side, price, remaining), newFeesRequirement)
		if newHoldingRequirement.GT(oldHoldingRequirement) {
			if m.collateral.PartyHasSufficientBalance(m.quoteAsset, order.Party, num.UintZero().Sub(newHoldingRequirement, oldHoldingRequirement)) != nil {
				return fmt.Errorf("party does not have sufficient balance to cover the trade and fees")
			}
		}
	}

	// if the side is sell and we want to sell more, need to check we're good for it
	if order.Side == types.SideSell && amendment.SizeDelta > 0 {
		if m.collateral.PartyHasSufficientBalance(m.baseAsset, order.Party, scaleBaseQuantityToAssetDP(uint64(amendment.SizeDelta), m.baseFactor)) != nil {
			return fmt.Errorf("party does not have sufficient balance to cover the new size")
		}
	}

	return nil
}

func (m *Market) GetQuoteAsset() string {
	return m.quoteAsset
}

func (m *Market) Mkt() *types.Market {
	return m.mkt
}

func (m *Market) StopSnapshots() {
	m.matching.StopSnapshots()
	m.tsCalc.StopSnapshots()
	m.liquidityEngine.StopSnapshots()
	m.orderHoldingTracker.StopSnapshots()
}

func (m *Market) orderCancelReplace(ctx context.Context, existingOrder, newOrder *types.Order) (conf *types.OrderConfirmation, orders []*types.Order, err error) {
	defer func() {
		if err != nil {
			return
		}

		orders = m.handleConfirmation(ctx, conf)
		m.handleConfirmationPassiveOrders(ctx, conf)
		if !conf.Order.IsFinished() && !m.as.InAuction() {
			amt := m.calculateAmountBySide(newOrder.Side, newOrder.Price, newOrder.TrueRemaining())
			asset := m.quoteAsset
			if newOrder.Side == types.SideSell {
				asset = m.baseAsset
			}
			transfer, err := m.orderHoldingTracker.TransferToHoldingAccount(ctx, newOrder.ID, newOrder.Party, asset, amt, num.UintZero())
			if err != nil {
				m.log.Panic("failed to transfer funds to holding account for order", logging.Order(newOrder), logging.Error(err))
			}
			m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{transfer}))
		}
		m.broker.Send(events.NewOrderEvent(ctx, conf.Order))
	}()

	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "orderCancelReplace")
	defer timer.EngineTimeCounterAdd()
	// first at this point release the funds of the previous order from holding account
	// because we may be the aggressor
	m.releaseOrderFromHoldingAccount(ctx, newOrder.ID, newOrder.Party, newOrder.Side)

	// make sure the order is on the book, this was done by canceling the order initially, but that could
	// trigger an auction in some cases.
	if o, err := m.matching.GetOrderByID(existingOrder.ID); err != nil || o == nil {
		m.log.Panic("Can't CancelReplace, the original order was not found",
			logging.OrderWithTag(*existingOrder, "existing-order"),
			logging.Error(err))
	}
	// cancel-replace amend during auction is quite simple at this point
	if m.as.InAuction() {
		conf, err := m.matching.ReplaceOrder(existingOrder, newOrder)
		if err != nil {
			m.log.Panic("unable to submit order", logging.Error(err))
		}
		if newOrder.PeggedOrder != nil {
			m.log.Panic("should never reach this point")
		}

		amt := m.calculateAmountBySide(newOrder.Side, newOrder.Price, newOrder.TrueRemaining())
		fees, err := m.calculateFees(newOrder.Party, newOrder.TrueRemaining(), newOrder.Price, newOrder.Side)
		if err != nil {
			return nil, nil, m.unregisterAndReject(ctx, newOrder, err)
		}
		asset := m.quoteAsset
		if newOrder.Side == types.SideSell {
			asset = m.baseAsset
		}
		transfer, err := m.orderHoldingTracker.TransferToHoldingAccount(ctx, newOrder.ID, newOrder.Party, asset, amt, fees)
		if err != nil {
			m.log.Panic("failed to transfer funds to holding account for order", logging.Order(newOrder), logging.Error(err))
		}
		m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{transfer}))

		return conf, nil, nil
	}
	// if its an iceberg order with a price change and it is being submitted aggressively
	// set the visible remaining to the full size
	if newOrder.IcebergOrder != nil {
		newOrder.Remaining += newOrder.IcebergOrder.ReservedRemaining
		newOrder.IcebergOrder.ReservedRemaining = 0
	}

	trades, err := m.checkPriceAndGetTrades(ctx, newOrder)
	if err != nil {
		return nil, nil, errors.New("couldn't insert order in book")
	}

	// "hot-swap" of the orders
	conf, err = m.matching.ReplaceOrder(existingOrder, newOrder)
	if err != nil {
		m.log.Panic("unable to submit order", logging.Error(err))
	}

	// replace the trades in the confirmation to have
	// the ones with the fees embedded
	conf.Trades = trades
	return conf, orders, nil
}

// orderAmendInPlace amends the order in the order book.
func (m *Market) orderAmendInPlace(originalOrder, amendOrder *types.Order) *types.OrderConfirmation {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "orderAmendInPlace")
	defer timer.EngineTimeCounterAdd()

	err := m.matching.AmendOrder(originalOrder, amendOrder)
	if err != nil {
		// panic here, no good reason for a failure at this point
		m.log.Panic("Failure after amend order from matching engine (amend-in-place)",
			logging.OrderWithTag(*amendOrder, "new-order"),
			logging.OrderWithTag(*originalOrder, "old-order"),
			logging.Error(err))
	}

	return &types.OrderConfirmation{
		Order: amendOrder,
	}
}

// orderAmendWhenParked amends a parked pegged order.
func (m *Market) orderAmendWhenParked(amendOrder *types.Order) *types.OrderConfirmation {
	amendOrder.Status = types.OrderStatusParked
	amendOrder.Price = num.UintZero()
	amendOrder.OriginalPrice = num.UintZero()
	m.peggedOrders.AmendParked(amendOrder)

	return &types.OrderConfirmation{
		Order: amendOrder,
	}
}

// submitStopOrders gets a status as parameter.
// this function is used on trigger but also on submission
// at expiry, so just filters out with a parameter.
func (m *Market) submitStopOrders(ctx context.Context, stopOrders []*types.StopOrder, status types.StopOrderStatus, idgen common.IDGenerator) []*types.OrderConfirmation {
	confirmations := []*types.OrderConfirmation{}
	evts := make([]events.Event, 0, len(stopOrders))

	// might contains both the triggered orders and the expired OCO
	for _, v := range stopOrders {
		if v.Status == status {
			conf, err := m.SubmitOrderWithIDGeneratorAndOrderID(
				ctx, v.OrderSubmission, v.Party, idgen, idgen.NextID(), false,
			)
			if err != nil {
				// not much we can do at that point, let's log the error and move on?
				m.log.Error("could not submit stop order",
					logging.StopOrderSubmission(v),
					logging.Error(err))
			}
			if err == nil && conf != nil {
				v.OrderID = conf.Order.ID
				confirmations = append(confirmations, conf)
			}
		}

		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	return confirmations
}

// removeExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked.
func (m *Market) removeExpiredStopOrders(ctx context.Context, timestamp int64, idgen common.IDGenerator) []*types.OrderConfirmation {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "RemoveExpiredStopOrders")
	defer timer.EngineTimeCounterAdd()

	toExpire := m.expiringStopOrders.Expire(timestamp)
	stopOrders := m.stopOrders.RemoveExpired(toExpire)

	evts := []events.Event{}
	filteredOCO := []*types.StopOrder{}
	for _, v := range stopOrders {
		if v.Expiry.Expires() && *v.Expiry.ExpiryStrategy == types.StopOrderExpiryStrategySubmit && len(v.OCOLinkID) <= 0 {
			filteredOCO = append(filteredOCO, v)
			continue
		}
		// nothing to do, can send the event now
		evts = append(evts, events.NewStopOrderEvent(ctx, v))
	}

	m.broker.SendBatch(evts)

	return m.submitStopOrders(ctx, filteredOCO, types.StopOrderStatusExpired, idgen)
}

// RemoveExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked.
func (m *Market) removeExpiredOrders(ctx context.Context, timestamp int64) []*types.Order {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "RemoveExpiredOrders")
	defer timer.EngineTimeCounterAdd()

	expired := []*types.Order{}
	toExp := m.expiringOrders.Expire(timestamp)
	if len(toExp) == 0 {
		return expired
	}
	ids := make([]string, 0, len(toExp))
	for _, orderID := range toExp {
		var order *types.Order
		// The pegged expiry orders are copies and do not reflect the
		// current state of the order, therefore we look it up
		originalOrder, foundOnBook, err := m.getOrderByID(orderID)
		if err != nil {
			// nothing to do there.
			continue
		}
		// assign to the order the order from the book
		// so we get the most recent version from the book
		// to continue with
		order = originalOrder

		// if the order was on the book basically
		// either a pegged + non parked
		// or a non-pegged order
		if foundOnBook {
			m.matching.DeleteOrder(order)
			// release any outstanding funds from the holding account to the general account
			m.releaseOrderFromHoldingAccount(ctx, order.ID, order.Party, order.Side)
		}

		// if this was a pegged order
		// remove from the pegged / parked list
		if order.PeggedOrder != nil {
			m.removePeggedOrder(order)
		}

		// now we add to the list of expired orders
		// and assign the appropriate status
		order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
		order.Status = types.OrderStatusExpired
		expired = append(expired, order)
		ids = append(ids, orderID)
	}
	if len(ids) > 0 {
		m.broker.Send(events.NewExpiredOrdersEvent(ctx, m.mkt.ID, ids))
	}

	// If we have removed an expired order, do we need to reprice any
	// or maybe notify the liquidity engine
	if len(expired) > 0 && !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, false)
	}

	return expired
}

func (m *Market) getBestStaticAskPrice() (*num.Uint, error) {
	return m.matching.GetBestStaticAskPrice()
}

func (m *Market) getBestStaticAskPriceAndVolume() (*num.Uint, uint64, error) {
	return m.matching.GetBestStaticAskPriceAndVolume()
}

func (m *Market) getBestStaticBidPrice() (*num.Uint, error) {
	return m.matching.GetBestStaticBidPrice()
}

func (m *Market) getBestStaticBidPriceAndVolume() (*num.Uint, uint64, error) {
	return m.matching.GetBestStaticBidPriceAndVolume()
}

func (m *Market) getBestStaticPricesDecimal() (bid, ask num.Decimal, err error) {
	ask = num.DecimalZero()
	ubid, err := m.getBestStaticBidPrice()
	if err != nil {
		bid = num.DecimalZero()
		return
	}
	bid = ubid.ToDecimal()
	uask, err := m.getBestStaticAskPrice()
	if err != nil {
		ask = num.DecimalZero()
		return
	}
	ask = uask.ToDecimal()
	return
}

func (m *Market) getStaticMidPrice(side types.Side) (*num.Uint, error) {
	bid, err := m.matching.GetBestStaticBidPrice()
	if err != nil {
		return num.UintZero(), err
	}
	ask, err := m.matching.GetBestStaticAskPrice()
	if err != nil {
		return num.UintZero(), err
	}
	mid := num.UintZero()
	one := num.NewUint(1)
	two := num.Sum(one, one)
	one.Mul(one, m.priceFactor)
	if side == types.SideBuy {
		mid = mid.Div(num.Sum(bid, ask, one), two)
	} else {
		mid = mid.Div(num.Sum(bid, ask), two)
	}

	return mid, nil
}

// removePeggedOrder looks through the pegged and parked list and removes the matching order if found.
func (m *Market) removePeggedOrder(order *types.Order) {
	// remove if order was expiring
	m.expiringOrders.RemoveOrder(order.ExpiresAt, order.ID)
	// unpark will remove the order from the pegged orders data structure
	m.peggedOrders.Unpark(order.ID)
}

// getOrderBy looks for the order in the order book and in the list
// of pegged orders in the market. Returns the order if found, a bool
// representing if the order was found on the order book and any error code.
func (m *Market) getOrderByID(orderID string) (*types.Order, bool, error) {
	order, err := m.matching.GetOrderByID(orderID)
	if err == nil {
		return order, true, nil
	}

	// The pegged order list contains all the pegged orders in the system
	// whether they are parked or live. Check this list of a matching order
	if o := m.peggedOrders.GetParkedByID(orderID); o != nil {
		return o, false, nil
	}

	// We couldn't find it
	return nil, false, common.ErrOrderNotFound
}

func (m *Market) getTargetStake() *num.Uint {
	return m.tsCalc.GetTargetStake(m.timeService.GetTimeNow())
}

func (m *Market) getSuppliedStake() *num.Uint {
	return m.liquidity.CalculateSuppliedStake()
}

// canTrade returns true if the market state is active pending or suspended.
func (m *Market) canTrade() bool {
	return m.mkt.State == types.MarketStateActive ||
		m.mkt.State == types.MarketStatePending ||
		m.mkt.State == types.MarketStateSuspended
}

// cleanupOnReject removes all resources created while the market was on PREPARED state.
// at this point no fees would have been collected or anything like this.
func (m *Market) cleanupOnReject(ctx context.Context) {
	m.stopAllLiquidityProvisionOnReject(ctx)
	tresps, err := m.collateral.ClearSpotMarket(ctx, m.GetID(), m.quoteAsset)
	if err != nil {
		m.log.Panic("unable to cleanup a rejected market",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return
	}

	m.stateVarEngine.UnregisterStateVariable(m.baseAsset+"_"+m.quoteAsset, m.mkt.ID)
	if len(tresps) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, tresps))
	}
}

func (m *Market) stopAllLiquidityProvisionOnReject(ctx context.Context) {
	m.liquidity.StopAllLiquidityProvision(ctx)
}

// GetTotalOrderBookLevelCount returns the total number of levels in the order book.
func (m *Market) GetTotalOrderBookLevelCount() uint64 {
	return m.matching.GetOrderBookLevelCount()
}

// GetTotalPeggedOrderCount returns the total number of pegged orders.
func (m *Market) GetTotalPeggedOrderCount() uint64 {
	return m.matching.GetPeggedOrdersCount()
}

// GetTotalStopOrderCount returns the total number of stop orders.
func (m *Market) GetTotalStopOrderCount() uint64 {
	return m.stopOrders.GetStopOrderCount()
}

// GetTotalOpenPositionCount returns the total number of open positions.
func (m *Market) GetTotalOpenPositionCount() uint64 {
	return 0
}

// GetTotalLPShapeCount returns the total number of LP shapes.
func (m *Market) GetTotalLPShapeCount() uint64 {
	return 9
}

// getCurrentMarkPrice returns the current mark price.
func (m *Market) getCurrentMarkPrice() *num.Uint {
	if m.markPrice == nil {
		return num.UintZero()
	}
	return m.markPrice.Clone()
}

// getLastTradedPrice returns the last traded price.
func (m *Market) getLastTradedPrice() *num.Uint {
	if m.lastTradedPrice == nil {
		return num.UintZero()
	}
	return m.lastTradedPrice.Clone()
}

// spot specific stuff

// processFeesTransfersOnEnterAuction handles the transfer from general account to holding account of fees to cover the trades that can take place
// during auction. This is necessary as during auction the fees are split between the participating parties of a trade rather than paid by the aggressor.
func (m *Market) processFeesTransfersOnEnterAuction(ctx context.Context) {
	parties := make([]string, 0, len(m.parties))
	for v := range m.parties {
		parties = append(parties, v)
	}
	sort.Strings(parties)
	ordersToCancel := []*types.Order{}
	transfers := []*types.LedgerMovement{}
	for _, party := range parties {
		orders := m.matching.GetOrdersPerParty(party)
		for _, o := range orders {
			if o.Side == types.SideSell {
				continue
			}
			// if the side is buy then the fees are paid directly by the buyer which must have an account in quote asset
			// with sufficient funds
			fees, err := m.calculateFees(party, o.TrueRemaining(), o.Price, o.Side)
			if err != nil {
				m.log.Error("error calculating fees for order", logging.Order(o), logging.Error(err))
				ordersToCancel = append(ordersToCancel, o)
				continue
			}
			if fees.IsZero() {
				continue
			}
			if err := m.collateral.PartyHasSufficientBalance(m.quoteAsset, party, fees); err != nil {
				m.log.Error("party has insufficient funds to cover for fees for order", logging.Order(o), logging.Error(err))
				ordersToCancel = append(ordersToCancel, o)
				continue
			}
			// party has sufficient funds to cover for fees - transfer fees from the party general account to the party holding account
			transfer, err := m.orderHoldingTracker.TransferFeeToHoldingAccount(ctx, o.ID, party, m.quoteAsset, fees)
			if err != nil {
				m.log.Error("failed to transfer from general account to holding account", logging.Order(o), logging.Error(err))
				ordersToCancel = append(ordersToCancel, o)
				continue
			}
			transfers = append(transfers, transfer)
		}
	}
	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}
	// cancel all orders with insufficient funds
	for _, o := range ordersToCancel {
		m.cancelOrder(ctx, o.Party, o.ID)
	}
}

// processFeesReleaseOnLeaveAuction releases any fees locked for the duration of an auction.
func (m *Market) processFeesReleaseOnLeaveAuction(ctx context.Context) {
	parties := make([]string, 0, len(m.parties))
	for v := range m.parties {
		parties = append(parties, v)
	}
	sort.Strings(parties)
	transfers := []*types.LedgerMovement{}
	for _, party := range parties {
		orders := m.matching.GetOrdersPerParty(party)
		for _, o := range orders {
			if o.Side == types.SideBuy {
				transfer, err := m.orderHoldingTracker.ReleaseFeeFromHoldingAccount(ctx, o.ID, party, m.quoteAsset)
				if err != nil {
					m.log.Panic("failed to release fee from holding account at the end of an auction", logging.Order(o), logging.Error(err))
					continue
				}
				transfers = append(transfers, transfer)
			}
		}
	}
	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}
}

func (m *Market) handleTrade(ctx context.Context, trade *types.Trade) []*types.LedgerMovement {
	transfers := []*types.LedgerMovement{}
	// we need to transfer base from the seller to the buyer,
	// quote from buyer to the seller.
	// if we're in auction we first need to release the fee funds for the buyer
	// and release the funds for both sides from the holding accounts.
	fees, err := m.calculateFeesForTrades([]*types.Trade{trade})
	if err != nil {
		m.log.Panic("failed to calculate fees for trade", logging.Trade(trade))
	}
	if trade.Aggressor == types.SideUnspecified {
		fee := num.UintZero()
		if fees != nil {
			fee = fees.TotalFeesAmountPerParty()[trade.Buyer]
		}

		// release buyer's trade + fees quote quantity from the holding account
		transfer, err := m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.BuyOrder, trade.Buyer, m.quoteAsset, scaleQuoteQuantityToAssetDP(trade.Size, trade.Price, m.positionFactor), fee)
		if err != nil {
			m.log.Panic("failed to release funds from holding account for trade", logging.Trade(trade), logging.Error(err))
		}
		transfers = append(transfers, transfer)

		// release seller's base quantity from the holding account
		transfer, err = m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.SellOrder, trade.Seller, m.baseAsset, scaleBaseQuantityToAssetDP(trade.Size, m.baseFactor), num.UintZero())
		if err != nil {
			m.log.Panic("failed to release funds from holding account for trade", logging.Trade(trade), logging.Error(err))
		}
		transfers = append(transfers, transfer)
	} else {
		// if there is an aggressor, then we need to release the passive side from the holding account
		if trade.Aggressor == types.SideSell { // the aggressor is the seller so we need to release funds for the buyer from holding
			transfer, err := m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.BuyOrder, trade.Buyer, m.quoteAsset, scaleQuoteQuantityToAssetDP(trade.Size, trade.Price, m.positionFactor), num.UintZero())
			if err != nil {
				m.log.Panic("failed to release funds from holding account for trade", logging.Trade(trade))
			}
			transfers = append(transfers, transfer)
		} else { // the aggressor is the buyer, we release the funds for the seller from holding account
			transfer, err := m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.SellOrder, trade.Seller, m.baseAsset, scaleBaseQuantityToAssetDP(trade.Size, m.baseFactor), num.UintZero())
			if err != nil {
				m.log.Panic("failed to release funds from holding account for trade", logging.Trade(trade))
			}
			transfers = append(transfers, transfer)
		}
	}

	// transfer base to buyer
	transfer, err := m.collateral.TransferSpot(ctx, trade.Seller, trade.Buyer, m.baseAsset, scaleBaseQuantityToAssetDP(trade.Size, m.baseFactor))
	if err != nil {
		m.log.Panic("failed to complete spot transfer", logging.Trade(trade))
	}
	transfers = append(transfers, transfer)
	// transfer quote (potentially minus fees) to the seller
	transfer, err = m.collateral.TransferSpot(ctx, trade.Buyer, trade.Seller, m.quoteAsset, scaleQuoteQuantityToAssetDP(trade.Size, trade.Price, m.positionFactor))
	if err != nil {
		m.log.Panic("failed to complete spot transfer", logging.Trade(trade))
	}
	transfers = append(transfers, transfer)
	if fees != nil {
		m.applyFees(ctx, fees)
	}
	return transfers
}

// transferToHoldingAccount transfers the remaining funds + fees if needed from the general account to the holding account.
func (m *Market) transferToHoldingAccount(ctx context.Context, order *types.Order) error {
	var err error
	amt := m.calculateAmountBySide(order.Side, order.Price, order.TrueRemaining())
	fees := num.UintZero()
	if m.as.InAuction() && order.Side == types.SideBuy {
		fees, err = m.calculateFees(order.Party, order.TrueRemaining(), order.Price, order.Side)
		if err != nil {
			return err
		}
	}
	asset := m.quoteAsset
	if order.Side == types.SideSell {
		asset = m.baseAsset
	}
	transfer, err := m.orderHoldingTracker.TransferToHoldingAccount(ctx, order.ID, order.Party, asset, amt, fees)
	if err != nil {
		m.log.Panic("failed to transfer funds to holding account for order", logging.Order(order), logging.Error(err))
	}
	m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{transfer}))
	return nil
}

// releaseOrderFromHoldingAccount release all funds for a given order from holding account. If there are no funds to release it panics.
func (m *Market) releaseOrderFromHoldingAccount(ctx context.Context, orderID, party string, side types.Side) {
	asset := m.quoteAsset
	if side == types.SideSell {
		asset = m.baseAsset
	}
	transfer, err := m.orderHoldingTracker.ReleaseAllFromHoldingAccount(ctx, orderID, party, asset)
	if err != nil {
		m.log.Panic("could not release funds from holding account", logging.String("order-id", orderID), logging.Error(err))
	}
	if transfer != nil {
		m.broker.Send(events.NewLedgerMovements(ctx, []*types.LedgerMovement{transfer}))
	}
}

// calculateFees calculate the amount of fees a party is due to pay given a side/price/size.
// during opening auction there are no fees.
func (m *Market) calculateFees(party string, size uint64, price *num.Uint, side types.Side) (*num.Uint, error) {
	if m.as.IsOpeningAuction() {
		return num.UintZero(), nil
	}

	fakeTrade := &types.Trade{
		Size:      size,
		Price:     price,
		Aggressor: side,
	}
	if side == types.SideBuy {
		fakeTrade.Buyer = party
	} else {
		fakeTrade.Seller = party
	}

	fees, err := m.calculateFeesForTrades([]*types.Trade{fakeTrade})
	if err != nil {
		return num.UintZero(), err
	}

	// if we're in uncrossing governance auction fees will be nil
	if fees == nil {
		return num.UintZero(), nil
	}

	return fees.TotalFeesAmountPerParty()[party], err
}

func (m *Market) calculateFeesForTrades(trades []*types.Trade) (events.FeesTransfer, error) {
	var (
		fees events.FeesTransfer
		err  error
	)
	if !m.as.InAuction() {
		fees, err = m.fee.CalculateForContinuousMode(trades, m.referralDiscountRewardService, m.volumeDiscountService)
	} else if m.as.IsMonitorAuction() {
		// we are in auction mode
		fees, err = m.fee.CalculateForAuctionMode(trades, m.referralDiscountRewardService, m.volumeDiscountService)
	} else if m.as.IsFBA() {
		fees, err = m.fee.CalculateForFrequentBatchesAuctionMode(trades, m.referralDiscountRewardService, m.volumeDiscountService)
	} else {
		if !m.as.IsOpeningAuction() {
			fees, err = m.fee.CalculateForAuctionMode(trades, m.referralDiscountRewardService, m.volumeDiscountService)
		}
	}
	return fees, err
}

// calculateAmountBySide calculates the amount *excluding* fees in the asset decimals.
func (m *Market) calculateAmountBySide(side types.Side, price *num.Uint, size uint64) *num.Uint {
	if side == types.SideBuy {
		return num.Sum(scaleQuoteQuantityToAssetDP(size, price, m.positionFactor))
	}
	return scaleBaseQuantityToAssetDP(size, m.baseFactor)
}

// checkSufficientFunds checks if the aggressor party has in their general account sufficient funds to cover the trade + fees.
func (m *Market) checkSufficientFunds(party string, side types.Side, price *num.Uint, size uint64, isPegged bool) error {
	required := m.calculateAmountBySide(side, price, size)
	if side == types.SideBuy {
		fees := num.UintZero()
		var err error
		if !isPegged {
			fees, err = m.calculateFees(party, size, price, side)
			if err != nil {
				return err
			}
		}

		if m.collateral.PartyHasSufficientBalance(m.quoteAsset, party, num.Sum(required, fees)) != nil {
			return fmt.Errorf("party does not have sufficient balance to cover the trade and fees")
		}
	} else {
		if m.collateral.PartyHasSufficientBalance(m.baseAsset, party, required) != nil {
			return fmt.Errorf("party does not have sufficient balance to cover the trade and fees")
		}
	}
	return nil
}

// convert the quantity to be transferred to the buyer to the base asset decimals.
func scaleBaseQuantityToAssetDP(sizeUint uint64, baseFactor num.Decimal) *num.Uint {
	size := num.NewUint(sizeUint)
	total := size.ToDecimal().Mul(baseFactor)
	totalI, _ := num.UintFromDecimal(total)
	return totalI
}

// convert the quantity to be transferred to the seller to the quote asset decimals.
func scaleQuoteQuantityToAssetDP(sizeUint uint64, priceInAssetDP *num.Uint, positionFactor num.Decimal) *num.Uint {
	size := num.NewUint(sizeUint)
	total := size.Mul(priceInAssetDP, size).ToDecimal().Div(positionFactor)
	totalI, _ := num.UintFromDecimal(total)
	return totalI
}

// closeSpotMarket terminates a market - this can be triggered only via governance.
func (m *Market) closeSpotMarket(ctx context.Context) {
	if m.mkt.State != types.MarketStatePending {
		m.markPrice = m.lastTradedPrice
		if err := m.closeMarket(ctx); err != nil {
			m.log.Error("could not close market", logging.Error(err))
		}
		return
	}
	for party := range m.parties {
		_, err := m.CancelAllOrders(ctx, party)
		if err != nil {
			m.log.Debug("could not cancel orders for party", logging.PartyID(party), logging.Error(err))
		}
	}
	err := m.closeCancelledMarket(ctx)
	if err != nil {
		m.log.Debug("could not close market", logging.MarketID(m.GetID()))
		return
	}
}

func (m *Market) OnEpochEvent(ctx context.Context, epoch types.Epoch) {
	if m.closed {
		return
	}
	if epoch.Action == vega.EpochAction_EPOCH_ACTION_START {
		m.liquidity.UpdateSLAParameters(m.mkt.LiquiditySLAParams)
		m.liquidity.OnEpochStart(ctx, m.timeService.GetTimeNow(), m.markPrice, m.midPrice(), m.getTargetStake(), m.positionFactor)
	} else if epoch.Action == vega.EpochAction_EPOCH_ACTION_END {
		m.liquidity.OnEpochEnd(ctx, m.timeService.GetTimeNow(), epoch)
		m.updateLiquidityFee(ctx)

		m.banking.RegisterTradingFees(ctx, m.quoteAsset, m.fee.TotalTradingFeesPerParty())

		quoteAssetQuantum, _ := m.collateral.GetAssetQuantum(m.quoteAsset)
		feesStats := m.fee.GetFeesStatsOnEpochEnd(quoteAssetQuantum)
		feesStats.EpochSeq = epoch.Seq
		feesStats.Market = m.GetID()

		m.broker.Send(events.NewFeesStatsEvent(ctx, feesStats))
	}
}

func (m *Market) OnEpochRestore(ctx context.Context, epoch types.Epoch) {
	m.liquidityEngine.OnEpochRestore(epoch)
}

func (m *Market) GetMarketCounters() *types.MarketCounters {
	return &types.MarketCounters{
		StopOrderCounter:    m.GetTotalStopOrderCount(),
		PeggedOrderCounter:  m.GetTotalPeggedOrderCount(),
		OrderbookLevelCount: m.GetTotalOrderBookLevelCount(),
		PositionCount:       0,
	}
}

// IDGen is an id generator for orders.
type IDGen interface {
	NextID() string
}
