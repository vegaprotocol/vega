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

package execution

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/idgeneration"
	"code.vegaprotocol.io/vega/core/liquidity"
	liquiditytarget "code.vegaprotocol.io/vega/core/liquidity/spottarget"
	"code.vegaprotocol.io/vega/core/matching"
	"code.vegaprotocol.io/vega/core/metrics"
	"code.vegaprotocol.io/vega/core/monitor"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/risk"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/statevar"
	vegacontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
)

type SpotMarketCollateral interface {
	CreateSpotMarketAccounts(ctx context.Context, marketID, quoteAsset string)
	ClearSpotMarket(ctx context.Context, mktID string) ([]*types.LedgerMovement, error)
	HasGeneralAccount(party, asset string) bool
	TransferFeesContinuousTrading(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	TransferFees(ctx context.Context, marketID string, assetID string, ft events.FeesTransfer) ([]*types.LedgerMovement, error)
	GetMarketLiquidityFeeAccount(market, asset string) (*types.Account, error)
	PartyHasSufficientBalance(asset, partyID string, balance *num.Uint) error
	TransferToHoldingAccount(ctx context.Context, transfer *types.Transfer) (*types.LedgerMovement, error)
	ReleaseFromHoldingAccount(ctx context.Context, transfer *types.Transfer) (*types.LedgerMovement, error)
	TransferSpot(ctx context.Context, partyID, toPartyID, asset string, quantity *num.Uint) (*types.LedgerMovement, error)
}

type SpotTargetStakeCalculator interface {
}

// SpotMarket represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transactions.
type SpotMarket struct {
	log   *logging.Logger
	idgen IDGenerator

	mkt *types.Market

	closingAt   time.Time
	timeService TimeService

	mu sync.Mutex

	lastTradedPrice *num.Uint
	markPrice       *num.Uint
	priceFactor     *num.Uint

	// own engines
	matching  *matching.CachedOrderBook
	fee       *fee.Engine
	liquidity *liquidity.SnapshotEngine

	// deps engines
	collateral SpotMarketCollateral

	broker Broker
	closed bool

	parties map[string]struct{}

	pMonitor                PriceMonitor
	lpPriceRange            num.Decimal
	linearSlippageFactor    num.Decimal
	quadraticSlippageFactor num.Decimal

	tsCalc SpotTargetStakeCalculator

	as AuctionState

	peggedOrders   *PeggedOrders
	expiringOrders *ExpiringOrders

	// Store the previous price values so we can see what has changed
	lastBestBidPrice *num.Uint
	lastBestAskPrice *num.Uint
	lastMidBuyPrice  *num.Uint
	lastMidSellPrice *num.Uint

	lastMarketValueProxy num.Decimal
	//TODO @ze97286 is this used?
	bondPenaltyFactor       num.Decimal
	marketValueWindowLength time.Duration

	// Liquidity Fee
	feeSplitter                *FeeSplitter
	lpFeeDistributionTimeStep  time.Duration
	lastEquityShareDistributed time.Time
	equityShares               *EquityShares
	minLPStakeQuantumMultiple  num.Decimal

	stateVarEngine        StateVarEngine
	marketActivityTracker *MarketActivityTracker
	baseFactor            num.Decimal // 10^(baseDP-pdp)
	positionFactor        num.Decimal // 10^pdp
	baseAsset             string
	quoteAsset            string

	// TODO @ze97286 decide if I want this
	orderHoldingTracker *HoldingAccountTracker
}

// NewSpotMarket creates a new market using the market framework configuration and creates underlying engines.
func NewSpotMarket(
	ctx context.Context,
	log *logging.Logger,
	matchingConfig matching.Config,
	feeConfig fee.Config,
	liquidityConfig liquidity.Config,
	collateralEngine SpotMarketCollateral,
	mkt *types.Market,
	timeService TimeService,
	broker Broker,
	as *monitor.AuctionState,
	stateVarEngine StateVarEngine,
	marketActivityTracker *MarketActivityTracker,
	baseAssetDetails *assets.Asset,
	quoteAssetDetails *assets.Asset,
	peggedOrderNotify func(int64),
) (*SpotMarket, error) {
	if len(mkt.ID) == 0 {
		return nil, ErrEmptyMarketID
	}

	positionFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(mkt.PositionDecimalPlaces))
	priceFactor := num.NewUint(1)
	if exp := quoteAssetDetails.DecimalPlaces() - mkt.DecimalPlaces; exp != 0 {
		priceFactor.Exp(num.NewUint(10), num.NewUint(exp))
	}
	baseFactor := num.DecimalFromFloat(10).Pow(num.DecimalFromInt64(int64(baseAssetDetails.DecimalPlaces()) - mkt.PositionDecimalPlaces))

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewCachedOrderBook(log, matchingConfig, mkt.ID, as.InAuction(), peggedOrderNotify)
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

	pMonitor, err := price.NewMonitor(quoteAsset, mkt.ID, riskModel, as, mkt.PriceMonitoringSettings, stateVarEngine, log)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}

	now := timeService.GetTimeNow()
	liqEngine := liquidity.NewSnapshotEngine(
		liquidityConfig, log, timeService, broker, riskModel, pMonitor, book, quoteAsset, mkt.ID, stateVarEngine, priceFactor.Clone(), positionFactor)

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

	market := &SpotMarket{
		log:                       log,
		idgen:                     nil,
		mkt:                       mkt,
		matching:                  book,
		collateral:                collateralEngine,
		timeService:               timeService,
		broker:                    broker,
		fee:                       feeEngine,
		liquidity:                 liqEngine,
		parties:                   map[string]struct{}{},
		as:                        as,
		pMonitor:                  pMonitor,
		tsCalc:                    tsCalc,
		peggedOrders:              NewPeggedOrders(log, timeService),
		expiringOrders:            NewExpiringOrders(),
		feeSplitter:               NewFeeSplitter(),
		equityShares:              NewEquityShares(num.DecimalZero()),
		lastBestAskPrice:          num.UintZero(),
		lastMidSellPrice:          num.UintZero(),
		lastMidBuyPrice:           num.UintZero(),
		lastBestBidPrice:          num.UintZero(),
		stateVarEngine:            stateVarEngine,
		marketActivityTracker:     marketActivityTracker,
		priceFactor:               priceFactor,
		baseFactor:                baseFactor,
		minLPStakeQuantumMultiple: num.MustDecimalFromString("1"),
		positionFactor:            positionFactor,
		lpPriceRange:              mkt.LPPriceRange,
		baseAsset:                 baseAsset,
		quoteAsset:                quoteAsset,
		orderHoldingTracker:       NewHoldingAccountTracker(collateralEngine),
	}

	liqEngine.SetGetStaticPricesFunc(market.getBestStaticPricesDecimal)

	return market, nil
}

func (m *SpotMarket) Update(ctx context.Context, config *types.Market) error {
	config.TradingMode = m.mkt.TradingMode
	config.State = m.mkt.State
	config.MarketTimestamps = m.mkt.MarketTimestamps
	m.mkt = config

	m.tsCalc.UpdateParameters(config.LiquidityMonitoringParameters.TargetStakeParameters)
	riskModel, err := risk.NewModel(config.TradableInstrument.RiskModel, m.quoteAsset)
	if err != nil {
		return err
	}
	m.pMonitor.UpdateSettings(riskModel, m.mkt.PriceMonitoringSettings)
	m.lpPriceRange = m.mkt.LPPriceRange
	m.liquidity.UpdateMarketConfig(riskModel, m.pMonitor)

	m.updateLiquidityFee(ctx)
	return nil
}

// GetID returns the id of the given market.
func (m *SpotMarket) GetID() string {
	return m.mkt.ID
}

// PostRestore restores market price in orders after snapshot reload.
func (m *SpotMarket) PostRestore(ctx context.Context) error {
	// tell the matching engine about the markets price factor so it can finish restoring orders
	m.matching.RestoreWithMarketPriceFactor(m.priceFactor)
	return nil
}

func (m *SpotMarket) GetMarketState() types.MarketState {
	return m.mkt.State
}

// TODO @ze97286 should we have spot market data instead?
func (m *SpotMarket) GetMarketData() types.MarketData {
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

	var targetStake string
	if m.as.InAuction() {
		targetStake = m.getTheoreticalTargetStake().String()
	} else {
		targetStake = m.getTargetStake().String()
	}
	bounds := m.pMonitor.GetCurrentBounds()
	for _, b := range bounds {
		priceToMarketPrecision(b.MaxValidPrice, m.priceFactor) // effictively floors this
		priceToMarketPrecision(b.MinValidPrice, m.priceFactor)

		rp, _ := num.UintFromDecimal(b.ReferencePrice)
		priceToMarketPrecision(rp, m.priceFactor)
		b.ReferencePrice = num.DecimalFromUint(rp)

		if m.priceFactor.NEQ(one) {
			b.MinValidPrice.AddSum(one) // ceil
		}
	}

	// TODO should we have spot market data?
	return types.MarketData{
		Market:                    m.GetID(),
		BestBidPrice:              priceToMarketPrecision(bestBidPrice, m.priceFactor),
		BestBidVolume:             bestBidVolume,
		BestOfferPrice:            priceToMarketPrecision(bestOfferPrice, m.priceFactor),
		BestOfferVolume:           bestOfferVolume,
		BestStaticBidPrice:        priceToMarketPrecision(bestStaticBidPrice, m.priceFactor),
		BestStaticBidVolume:       bestStaticBidVolume,
		BestStaticOfferPrice:      priceToMarketPrecision(bestStaticOfferPrice, m.priceFactor),
		BestStaticOfferVolume:     bestStaticOfferVolume,
		MidPrice:                  priceToMarketPrecision(midPrice, m.priceFactor),
		StaticMidPrice:            priceToMarketPrecision(staticMidPrice, m.priceFactor),
		LastTradedPrice:           priceToMarketPrecision(m.getLastTradedPrice(), m.priceFactor),
		Timestamp:                 m.timeService.GetTimeNow().UnixNano(),
		IndicativePrice:           priceToMarketPrecision(indicativePrice, m.priceFactor),
		IndicativeVolume:          indicativeVolume,
		AuctionStart:              auctionStart,
		AuctionEnd:                auctionEnd,
		MarketTradingMode:         m.as.Mode(),
		MarketState:               m.mkt.State,
		Trigger:                   m.as.Trigger(),
		ExtensionTrigger:          m.as.ExtensionTrigger(),
		TargetStake:               targetStake,
		SuppliedStake:             m.getSuppliedStake().String(),
		PriceMonitoringBounds:     bounds,
		MarketValueProxy:          m.lastMarketValueProxy.BigInt().String(),
		LiquidityProviderFeeShare: lpsToLiquidityProviderFeeShare(m.equityShares.lps, m.liquidity.GetAverageLiquidityScores()),
	}
}

func (m *SpotMarket) IntoType() types.Market {
	return *m.mkt.DeepClone()
}

func (m *SpotMarket) Hash() []byte {
	mID := logging.String("market-id", m.GetID())
	matchingHash := m.matching.Hash()
	m.log.Debug("orderbook state hash", logging.Hash(matchingHash), mID)
	return matchingHash
}

// ReloadConf will trigger a reload of all the config settings in the market and all underlying engines
// this is required when hot-reloading any config changes, eg. logger level.
func (m *SpotMarket) ReloadConf(matchingConfig matching.Config, feeConfig fee.Config) {
	m.log.Info("reloading configuration")
	m.matching.ReloadConf(matchingConfig)
	m.fee.ReloadConf(feeConfig)
}

// addParty adds the party to the market mapping.
func (m *SpotMarket) addParty(party string) {
	if _, ok := m.parties[party]; !ok {
		m.parties[party] = struct{}{}
	}
}

// removePeggedOrder looks through the pegged and parked list and removes the matching order if found.
func (m *SpotMarket) removePeggedOrder(order *types.Order) {
	// remove if order was expiring
	m.expiringOrders.RemoveOrder(order.ExpiresAt, order.ID)
	// unpark will remove the order from the pegged orders data structure
	m.peggedOrders.Unpark(order.ID)
}

// RemoveExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked.
func (m *SpotMarket) removeExpiredOrders(ctx context.Context, timestamp int64) []*types.Order {
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
		m.checkForReferenceMoves(ctx, expired, false)
	}

	return expired
}

// parkAllPeggedOrders parks all pegged orders.
func (m *SpotMarket) parkAllPeggedOrders(ctx context.Context) []*types.Order {
	toParkIDs := m.matching.GetActivePeggedOrderIDs()

	parked := make([]*types.Order, 0, len(toParkIDs))
	for _, order := range toParkIDs {
		parked = append(parked, m.parkOrder(ctx, order))
	}
	return parked
}

// validatePeggedOrder validates pegged order.
func (m *SpotMarket) validatePeggedOrder(order *types.Order) types.OrderError {
	if order.Type != types.OrderTypeLimit {
		// All pegged orders must be LIMIT orders
		return types.ErrPeggedOrderMustBeLimitOrder
	}

	if order.TimeInForce != types.OrderTimeInForceGTT && order.TimeInForce != types.OrderTimeInForceGTC && order.TimeInForce != types.OrderTimeInForceGFN {
		// Pegged orders can only be GTC or GTT
		return types.ErrPeggedOrderMustBeGTTOrGTC
	}

	if order.PeggedOrder.Reference == types.PeggedReferenceUnspecified {
		// We must specify a valid reference
		return types.ErrPeggedOrderWithoutReferencePrice
	}

	if order.Side == types.SideBuy {
		switch order.PeggedOrder.Reference {
		case types.PeggedReferenceBestAsk:
			return types.ErrPeggedOrderBuyCannotReferenceBestAskPrice
		case types.PeggedReferenceMid:
			if order.PeggedOrder.Offset.IsZero() {
				return types.ErrPeggedOrderOffsetMustBeGreaterThanZero
			}
		}
	} else {
		switch order.PeggedOrder.Reference {
		case types.PeggedReferenceBestBid:
			return types.ErrPeggedOrderSellCannotReferenceBestBidPrice
		case types.PeggedReferenceMid:
			if order.PeggedOrder.Offset.IsZero() {
				return types.ErrPeggedOrderOffsetMustBeGreaterThanZero
			}
		}
	}
	return types.OrderErrorUnspecified
}

// OnTick notifies the market of a new time event/update.
// todo: make this a more generic function name e.g. OnTimeUpdateEvent
func (m *SpotMarket) OnTick(ctx context.Context, t time.Time) bool {
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
	}

	// some engines still needs to get updates:
	m.pMonitor.OnTimeUpdate(t)
	m.feeSplitter.SetCurrentTime(t)

	// TODO(): This also assume that the market is not
	// being closed before the market is leaving
	// the opening auction, but settlement at expiry is
	// not even specced or implemented as of now...
	// if the state of the market is just PROPOSED,
	// we will just skip everything there as nothing apply.
	if m.mkt.State == types.MarketStateProposed {
		return false
	}

	// if trading is terminated, we have nothing to do here.
	// we just need to wait for the settlementData to arrive through oracle
	if m.mkt.State == types.MarketStateTradingTerminated {
		return false
	}

	// TODO maybe this needs to change
	// distribute liquidity fees each `m.lpFeeDistributionTimeStep`
	if t.Sub(m.lastEquityShareDistributed) > m.lpFeeDistributionTimeStep {
		m.lastEquityShareDistributed = t

		if err := m.distributeLiquidityFees(ctx); err != nil {
			m.log.Panic("liquidity fee distribution error", logging.Error(err))
		}
	}

	// check auction, if any. If we leave auction, MTM is performed in this call
	m.checkAuction(ctx, t)
	timer.EngineTimeCounterAdd()

	m.updateMarketValueProxy()
	m.updateLiquidityScores()
	m.updateLiquidityFee(ctx)
	m.broker.Send(events.NewMarketTick(ctx, m.mkt.ID, t))
	return m.closed
}

//
// Auctions
//

// StartOpeningAuction kicks off opening auction.
func (m *SpotMarket) StartOpeningAuction(ctx context.Context) error {
	if m.mkt.State != types.MarketStateProposed {
		return ErrCannotStartOpeningAuctionForMarketNotInProposedState
	}

	// now we start the opening auction
	if m.as.AuctionStart() {
		// we are now in a pending state
		m.mkt.State = types.MarketStatePending
		m.mkt.MarketTimestamps.Pending = m.timeService.GetTimeNow().UnixNano()
		m.mkt.TradingMode = types.MarketTradingModeOpeningAuction
		m.enterAuction(ctx)
	} else {
		// TODO(): to be removed once we don't have market starting
		// without an opening auction
		m.mkt.State = types.MarketStateActive
		m.mkt.TradingMode = types.MarketTradingModeContinuous
	}

	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

// CanLeaveOpeningAuction checks if the market can leave the opening auction based on whether floating point consensus has been reached on all 2 vars.
func (m *SpotMarket) CanLeaveOpeningAuction() bool {
	boundFactorsInitialised := m.pMonitor.IsBoundFactorsInitialised()
	potInitialised := m.liquidity.IsPoTInitialised()
	canLeave := boundFactorsInitialised && potInitialised
	if !canLeave {
		m.log.Info("Cannot leave opening auction", logging.String("market", m.mkt.ID), logging.Bool("bound-factors-initialised", boundFactorsInitialised), logging.Bool("pot-initialised", potInitialised))
	}
	return canLeave
}

// OnOpeningAuctionFirstUncrossingPrice is triggered when the opening auction sees an uncrossing price for the first time and emits
// an event to the state variable engine.
func (m *SpotMarket) OnOpeningAuctionFirstUncrossingPrice() {
	m.log.Info("OnOpeningAuctionFirstUncrossingPrice event fired", logging.String("market", m.mkt.ID))
	assetKey := m.baseAsset + "_" + m.quoteAsset
	m.stateVarEngine.ReadyForTimeTrigger(assetKey, m.mkt.ID)
	m.stateVarEngine.NewEvent(assetKey, m.mkt.ID, statevar.EventTypeOpeningAuctionFirstUncrossingPrice)
}

// OnAuctionEnded is called whenever an auction is ended and emits an event to the state var engine.
func (m *SpotMarket) OnAuctionEnded() {
	m.log.Info("OnAuctionEnded event fired", logging.String("market", m.mkt.ID))
	assetKey := m.baseAsset + "_" + m.quoteAsset
	m.stateVarEngine.NewEvent(assetKey, m.mkt.ID, statevar.EventTypeAuctionEnded)
}

// processFeesTransfersOnEnterAuction handles the transfer from general account to holding account of fees to cover the trades that can take place
// during auction. This is necessary as during auction the fees are split between the participating parties of a trade rather than paid by the aggressor.
func (m *SpotMarket) processFeesTransfersOnEnterAuction(ctx context.Context) {
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
			fees, err := m.calculateFees(party, o.Remaining, o.Price, o.Side)
			if err != nil {
				m.log.Error("error calculating fees for order", logging.Order(o), logging.Error(err))
				ordersToCancel = append(ordersToCancel, o)
				continue
			}
			if fees.IsZero() {
				continue
			}
			// if the side is buy then the fees are paid directly by the buyer which must have an account in quote asset
			// with sufficient funds
			if o.Side == types.SideBuy {
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
	}
	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}
	// cancel all orders with insufficient funds
	for _, o := range ordersToCancel {
		m.cancelOrder(ctx, o.Party, o.ID)
	}
}

// processFeesReleaseOnLeaveAuction releases any fees locked for the duration of an auction
func (m *SpotMarket) processFeesReleaseOnLeaveAuction(ctx context.Context) {
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

// EnterAuction : Prepare the order book to be run as an auction.
// when entering an auction we need to make sure there's sufficent funds in the holding account to cover the potential trade + fees.
// If there isn't, the order must be cancelled.
func (m *SpotMarket) enterAuction(ctx context.Context) {
	// Change market type to auction
	ordersToCancel := m.matching.EnterAuction()

	// Move into auction mode to prevent pegged order repricing
	event := m.as.AuctionStarted(ctx, m.timeService.GetTimeNow())

	// this is at least the size of the orders to be cancelled
	updatedOrders := make([]*types.Order, 0, len(ordersToCancel))

	// Cancel all the orders that were invalid
	for _, order := range ordersToCancel {
		_, err := m.cancelOrder(ctx, order.Party, order.ID)
		if err != nil {
			m.log.Debug("error cancelling order when entering auction",
				logging.MarketID(m.GetID()),
				logging.OrderID(order.ID),
				logging.Error(err))
		}
		updatedOrders = append(updatedOrders, order)
	}

	// now that all orders that don't fit in auctions have been cancelled, process necessary transfer of fees from the general account of the
	// buyers to the holding account. Orders with insufficient cover of buyer or where the quantity to be delivered to the seller does not cover
	// for the due fees during auction are cancelled here.
	m.processFeesTransfersOnEnterAuction(ctx)

	// now update all special orders
	m.enterAuctionSpecialOrders(ctx, updatedOrders)

	// Send an event bus update
	m.broker.Send(event)

	if m.as.InAuction() && m.as.IsPriceAuction() {
		m.mkt.State = types.MarketStateSuspended
		m.mkt.TradingMode = types.MarketTradingModeMonitoringAuction
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	}
}

// leaveAuction : Return the orderbook and market to continuous trading.
func (m *SpotMarket) leaveAuction(ctx context.Context, now time.Time) {
	defer func() {
		if !m.as.InAuction() && (m.mkt.State == types.MarketStateSuspended || m.mkt.State == types.MarketStatePending) {
			if m.mkt.State == types.MarketStatePending {
				// the market is now properly open,
				// so set the timestamp to when the opening auction actually ended
				m.mkt.MarketTimestamps.Open = now.UnixNano()
			}
			m.processFeesReleaseOnLeaveAuction(ctx)
			m.mkt.State = types.MarketStateActive
			m.mkt.TradingMode = types.MarketTradingModeContinuous
			m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
			m.updateLiquidityFee(ctx)
			m.OnAuctionEnded()
		}
	}()

	// Change market type to continuous trading
	uncrossedOrders, ordersToCancel, err := m.matching.LeaveAuction(m.timeService.GetTimeNow())
	if err != nil {
		m.log.Error("Error leaving auction", logging.Error(err))
	}

	// Process each confirmation & apply fee calculations to each trade
	evts := make([]events.Event, 0, len(uncrossedOrders))
	for _, uncrossedOrder := range uncrossedOrders {
		// then do the confirmation
		m.handleConfirmation(ctx, uncrossedOrder)

		if uncrossedOrder.Order.Remaining == 0 {
			uncrossedOrder.Order.Status = types.OrderStatusFilled
		}
		evts = append(evts, events.NewOrderEvent(ctx, uncrossedOrder.Order))
	}

	// send order events in a single batch, it's more efficient
	m.broker.SendBatch(evts)

	// will hold all orders which have been updated by the uncrossing
	// or which were cancelled at end of auction
	updatedOrders := []*types.Order{}

	// Process each order we have to cancel
	for _, order := range ordersToCancel {
		conf, err := m.cancelOrder(ctx, order.Party, order.ID)
		if err != nil {
			m.log.Panic("Failed to cancel order",
				logging.Error(err),
				logging.String("OrderID", order.ID))
		}

		updatedOrders = append(updatedOrders, conf.Order)
	}

	// update auction state, so we know what the new tradeMode ought to be
	endEvt := m.as.Left(ctx, now)

	for _, uncrossedOrder := range uncrossedOrders {
		updatedOrders = append(updatedOrders, uncrossedOrder.Order)
		updatedOrders = append(
			updatedOrders, uncrossedOrder.PassiveOrdersAffected...)
	}

	m.checkForReferenceMoves(ctx, updatedOrders, true)

	if !m.as.InAuction() {
		// only send the auction-left event if we actually *left* the auction.
		m.broker.Send(endEvt)
	}
}

//
// Order submission
//

// getNewPeggedPrice calculates pegged price based on the pegged reference and current prices.
func (m *SpotMarket) getNewPeggedPrice(order *types.Order) (*num.Uint, error) {
	if m.as.InAuction() {
		return num.UintZero(), ErrCannotRepriceDuringAuction
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
		return num.UintZero(), ErrUnableToReprice
	}

	offset := num.UintZero().Mul(order.PeggedOrder.Offset, m.priceFactor)
	if order.Side == types.SideSell {
		return price.AddSum(offset), nil
	}

	if price.LTE(offset) {
		return num.UintZero(), ErrUnableToReprice
	}

	return num.UintZero().Sub(price, offset), nil
}

// Reprice a pegged order. This only updates the price on the order.
func (m *SpotMarket) repricePeggedOrder(order *types.Order) error {
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

func (m *SpotMarket) releaseOrderFromHoldingAccount(ctx context.Context, orderID, party string, side types.Side) {
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

// removeOrders removes orders from the book when the market is stopped.
func (m *SpotMarket) removeOrders(ctx context.Context) {
	// remove all order from the book
	// and send events with the stopped status
	orders := append(m.matching.Settled(), m.peggedOrders.Settled()...)
	orderEvents := make([]events.Event, 0, len(orders))
	for _, v := range orders {
		orderEvents = append(orderEvents, events.NewOrderEvent(ctx, v))
		m.releaseOrderFromHoldingAccount(ctx, v.ID, v.Party, v.Side)
	}
	m.broker.SendBatch(orderEvents)
}

// parkOrder removes the given order from the orderbook. parkOrder will panic if it encounters errors, which means that it reached an
// invalid state. When the order is parked, the funds from the holding account are released to the general account.
func (m *SpotMarket) parkOrder(ctx context.Context, orderID string) *types.Order {
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

// unregisterAndReject
// I *think* this is called when aggressing, i.e. the order didn't go to the book therefore there's no need to release funds from the holding account.
func (m *SpotMarket) unregisterAndReject(ctx context.Context, order *types.Order, err error) error {
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

// CancelAllOrders cancels all orders in the market.
func (m *SpotMarket) CancelAllOrders(ctx context.Context, partyID string) ([]*types.OrderCancellationConfirmation, error) {
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
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

	// now we extract all liquidity provision order out of the list.
	// cancelling some order may trigger repricing, and repricing
	// liquidity order, which also trigger cancelling...
	// by filtering the list now, we are sure that we will
	// never try to
	// 1. remove a lp order
	// 2. have invalid order referencing lp order which have been canceleld
	okOrders := []*types.Order{}
	for _, order := range orders {
		if order.IsLiquidityOrder() {
			continue
		}
		okOrders = append(okOrders, order)
	}

	cancellations := make([]*types.OrderCancellationConfirmation, 0, len(orders))

	// now iterate over all orders and cancel one by one.
	cancelledOrders := make([]*types.Order, 0, len(okOrders))
	for _, order := range okOrders {
		cancellation, err := m.cancelOrder(ctx, partyID, order.ID)
		if err != nil {
			return nil, err
		}
		cancellations = append(cancellations, cancellation)
		cancelledOrders = append(cancelledOrders, cancellation.Order)
	}

	m.checkForReferenceMoves(ctx, cancelledOrders, false)

	return cancellations, nil
}

// CancelOrder canels a single order in the market.
func (m *SpotMarket) CancelOrder(ctx context.Context, partyID, orderID string, deterministicID string) (oc *types.OrderCancellationConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.CancelOrderWithIDGenerator(ctx, partyID, orderID, idgen)
}

// CancelOrderWithIDGenerator cancels an order in the market.
func (m *SpotMarket) CancelOrderWithIDGenerator(ctx context.Context, partyID, orderID string, idgen IDGenerator) (oc *types.OrderCancellationConfirmation, _ error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// cancelling and amending an order that is part of the LP commitment isn't allowed
	if o, err := m.matching.GetOrderByID(orderID); err == nil && o.IsLiquidityOrder() {
		return nil, types.ErrEditNotAllowed
	}

	conf, err := m.cancelOrder(ctx, partyID, orderID)
	if err != nil {
		return conf, err
	}

	if !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, []*types.Order{conf.Order}, false)
	}

	return conf, nil
}

// CancelOrder cancels the given order. If the order is found on the book, we release locked funds from holdingn account to the general account of the party.
func (m *SpotMarket) cancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
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

// AmendOrder amend an existing order from the order book.
func (m *SpotMarket) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment, party string, deterministicID string) (oc *types.OrderConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.AmendOrderWithIDGenerator(ctx, orderAmendment, party, idgen)
}

// AmendOrderWithIDGenerator amends an order.
func (m *SpotMarket) AmendOrderWithIDGenerator(ctx context.Context, orderAmendment *types.OrderAmendment, party string, idgen IDGenerator) (oc *types.OrderConfirmation, _ error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	conf, updatedOrders, err := m.amendOrder(ctx, orderAmendment, party)
	if err != nil {
		return nil, err
	}

	allUpdatedOrders := append(
		[]*types.Order{conf.Order},
		conf.PassiveOrdersAffected...,
	)
	allUpdatedOrders = append(
		allUpdatedOrders,
		updatedOrders...,
	)

	if !m.as.InAuction() {
		m.checkForReferenceMoves(ctx, allUpdatedOrders, false)
	}
	return conf, nil
}

func (m *SpotMarket) amendOrder(ctx context.Context, orderAmendment *types.OrderAmendment, party string) (cnf *types.OrderConfirmation, orderUpdates []*types.Order, returnedErr error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, nil, ErrMarketClosed
	}

	existingOrder, foundOnBook, err := m.findOrderAndEnsureOwnership(
		orderAmendment.OrderID, party, m.GetID())
	if err != nil {
		return nil, nil, err
	}

	if err := m.validateOrderAmendment(existingOrder, orderAmendment); err != nil {
		return nil, nil, err
	}

	amendedOrder, err := m.applyOrderAmendment(existingOrder, orderAmendment)
	if err != nil {
		return nil, nil, err
	}

	// We do this first, just in case the party would also have
	// change the expiry, and that would have been catched by
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

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		return m.orderCancelReplace(ctx, existingOrder, amendedOrder)
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease || timeInForceChange {
		ret := m.orderAmendInPlace(existingOrder, amendedOrder)
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
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

func (m *SpotMarket) orderCancelReplace(ctx context.Context, existingOrder, newOrder *types.Order) (conf *types.OrderConfirmation, orders []*types.Order, err error) {
	defer func() {
		if err != nil {
			return
		}

		orders = m.handleConfirmation(ctx, conf)
		if conf.Order.IsFinished() {
			m.releaseOrderFromHoldingAccount(ctx, conf.Order.ID, conf.Order.Party, conf.Order.Side)
		}
		m.broker.Send(events.NewOrderEvent(ctx, conf.Order))
	}()

	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "orderCancelReplace")
	defer timer.EngineTimeCounterAdd()

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

		return conf, nil, nil
	}
	// first we call the order book to evaluate auction triggers and get the list of trades
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

// applyOrderAmendment assumes the amendment have been validated before.
func (m *SpotMarket) applyOrderAmendment(existingOrder *types.Order, amendment *types.OrderAmendment) (order *types.Order, err error) {
	order = existingOrder.Clone()
	order.UpdatedAt = m.timeService.GetTimeNow().UnixNano()
	order.Version++

	if existingOrder.PeggedOrder != nil {
		order.PeggedOrder = &types.PeggedOrder{
			Reference: existingOrder.PeggedOrder.Reference,
			Offset:    existingOrder.PeggedOrder.Offset,
		}
	}

	var amendPrice *num.Uint
	if amendment.Price != nil {
		amendPrice = amendment.Price.Clone()
		amendPrice.Mul(amendPrice, m.priceFactor)
	}
	// apply price changes
	if amendment.Price != nil && existingOrder.Price.NEQ(amendPrice) {
		order.Price = amendPrice.Clone()
		order.OriginalPrice = amendment.Price.Clone()
	}

	// apply size changes
	if delta := amendment.SizeDelta; delta != 0 {
		if delta < 0 {
			order.Size -= uint64(-delta)
			if order.Remaining > uint64(-delta) {
				order.Remaining -= uint64(-delta)
			} else {
				order.Remaining = 0
			}
		} else {
			order.Size += uint64(delta)
			order.Remaining += uint64(delta)
		}
	}

	// apply tif
	if amendment.TimeInForce != types.OrderTimeInForceUnspecified {
		order.TimeInForce = amendment.TimeInForce
		if amendment.TimeInForce != types.OrderTimeInForceGTT {
			order.ExpiresAt = 0
		}
	}
	if amendment.ExpiresAt != nil {
		order.ExpiresAt = *amendment.ExpiresAt
	}

	// apply pegged order values
	if order.PeggedOrder != nil {
		if amendment.PeggedOffset != nil {
			order.PeggedOrder.Offset = amendment.PeggedOffset.Clone()
		}

		if amendment.PeggedReference != types.PeggedReferenceUnspecified {
			order.PeggedOrder.Reference = amendment.PeggedReference
		}
		if verr := m.validatePeggedOrder(order); verr != types.OrderErrorUnspecified {
			err = verr
		}
	}

	return order, err
}

// findOrderAndEnsureOwnership checks that the party is actually the owner of the order ID.
func (m *SpotMarket) findOrderAndEnsureOwnership(orderID, partyID, marketID string) (exitingOrder *types.Order, foundOnBook bool, err error) {
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

	if existingOrder.IsLiquidityOrder() {
		return nil, false, types.ErrEditNotAllowed
	}

	return existingOrder, foundOnBook, err
}

// orderAmendWhenParked amends a parked pegged order.
func (m *SpotMarket) orderAmendWhenParked(amendOrder *types.Order) *types.OrderConfirmation {
	amendOrder.Status = types.OrderStatusParked
	amendOrder.Price = num.UintZero()
	amendOrder.OriginalPrice = num.UintZero()
	m.peggedOrders.AmendParked(amendOrder)

	return &types.OrderConfirmation{
		Order: amendOrder,
	}
}

// orderAmendInPlace amends the order in the order book.
func (m *SpotMarket) orderAmendInPlace(originalOrder, amendOrder *types.Order) *types.OrderConfirmation {
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

func (m *SpotMarket) validateOrderAmendment(order *types.Order, amendment *types.OrderAmendment) error {
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
	if amendment.Price != nil || amendment.SizeDelta != 0 && order.Side == types.SideBuy {
		remaining := order.Remaining
		if amendment.SizeDelta < 0 {
			if remaining > uint64(-amendment.SizeDelta) {
				remaining -= uint64(-amendment.SizeDelta)
			} else {
				remaining = 0
			}
		} else {
			remaining += uint64(amendment.SizeDelta)
		}

		if remaining == 0 && amendment.Price.LTE(order.Price) {
			return nil
		}
		existingHoldingQty, existingHoldingFee := m.orderHoldingTracker.getCurrentHolding(order.ID)
		oldHoldingRequirement := num.Sum(existingHoldingQty, existingHoldingFee)
		newFeesRequirement := num.UintZero()
		if m.as.InAuction() {
			newFeesRequirement, _ = m.calculateFees(order.Party, order.Remaining, order.Price, order.Side)
		}
		newHoldingRequirement := num.Sum(m.calculateAmountBySide(order.Party, order.Side, amendment.Price, remaining), newFeesRequirement)

		if newHoldingRequirement.GT(oldHoldingRequirement) {
			if m.collateral.PartyHasSufficientBalance(m.quoteAsset, order.Party, num.UintZero().Sub(newHoldingRequirement, oldHoldingRequirement)) != nil {
				return fmt.Errorf("party does not have sufficient balance to cover the trade and fees")
			}
		}
	}

	// if the side is sell and we want to sell more, need to check we're good for it
	if amendment.SizeDelta > 0 && order.Side == types.SideSell {
		if m.collateral.PartyHasSufficientBalance(m.baseAsset, order.Party, scaleBaseQuantityToAssetDP(uint64(amendment.SizeDelta), m.baseFactor)) != nil {
			return fmt.Errorf("party does not have sufficient balance to cover the new size")
		}
	}

	return nil
}

// SubmitOrder submits the given order.
func (m *SpotMarket) SubmitOrder(ctx context.Context, orderSubmission *types.OrderSubmission, party string, deterministicID string) (oc *types.OrderConfirmation, _ error) {
	idgen := idgeneration.New(deterministicID)
	return m.SubmitOrderWithIDGeneratorAndOrderID(
		ctx, orderSubmission, party, idgen, idgen.NextID(),
	)
}

// SubmitOrderWithIDGeneratorAndOrderID submits the given order.
func (m *SpotMarket) SubmitOrderWithIDGeneratorAndOrderID(ctx context.Context, orderSubmission *types.OrderSubmission, party string, idgen IDGenerator, orderID string) (oc *types.OrderConfirmation, _ error) {
	m.idgen = idgen
	defer func() { m.idgen = nil }()

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
		return nil, ErrTradingNotAllowed
	}

	conf, orderUpdates, err := m.submitOrder(ctx, order)
	if err != nil {
		return nil, err
	}

	allUpdatedOrders := append(
		[]*types.Order{conf.Order}, conf.PassiveOrdersAffected...)
	allUpdatedOrders = append(allUpdatedOrders, orderUpdates...)

	if !m.as.InAuction() {
		m.checkForReferenceMoves(
			ctx, allUpdatedOrders, false)
	}

	return conf, nil
}

// submitOrder validates and submits an order.
func (m *SpotMarket) submitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, []*types.Order, error) {
	timer := metrics.NewTimeCounter(m.mkt.ID, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.ID, orderValidity)
	}()

	// set those at the beginning as even rejected order get through the buffers
	order.Version = InitialOrderVersion
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

// validateOrder checks that the order parameters are valid for the market.
func (m *SpotMarket) validateOrder(ctx context.Context, order *types.Order) (err error) {
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
		return ErrGFNOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceIOC {
		order.Reason = types.OrderErrorCannotSendIOCOrderDuringAuction
		return ErrIOCOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.OrderTimeInForceFOK {
		order.Reason = types.OrderErrorCannotSendFOKOrderDurinAuction
		return ErrFOKOrderReceivedAuctionTrading
	}

	if !isAuction && order.TimeInForce == types.OrderTimeInForceGFA {
		order.Reason = types.OrderErrorGFAOrderDuringContinuousTrading
		return ErrGFAOrderReceivedDuringContinuousTrading
	}

	// Check the expiry time is valid
	if order.ExpiresAt > 0 && order.ExpiresAt < order.CreatedAt {
		order.Reason = types.OrderErrorInvalidExpirationDatetime
		return ErrInvalidExpiresAtTime
	}

	if m.closed {
		// adding order to the buffer first
		order.Reason = types.OrderErrorMarketClosed
		return ErrMarketClosed
	}

	if order.Type == types.OrderTypeNetwork {
		order.Reason = types.OrderErrorInvalidType
		return ErrInvalidOrderType
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
		if reason := m.validatePeggedOrder(order); reason != types.OrderErrorUnspecified {
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
func (m *SpotMarket) validateAccounts(ctx context.Context, order *types.Order) error {
	if (order.Side == types.SideBuy && !m.collateral.HasGeneralAccount(order.Party, m.quoteAsset)) ||
		(order.Side == types.SideSell && !m.collateral.HasGeneralAccount(order.Party, m.baseAsset)) {
		// adding order to the buffer first
		order.Status = types.OrderStatusRejected
		order.Reason = types.OrderErrorInsufficientAssetBalance
		m.broker.Send(events.NewOrderEvent(ctx, order))

		// party should be created before even trying to post order
		return ErrPartyInsufficientAssetBalance
	}

	if err := m.checkSufficientFunds(order.Party, order.Side, order.Price, order.Size); err != nil {
		return err
	}

	// from this point we know the party have the necessary accounts and balances
	// we had it to the list of parties.
	m.addParty(order.Party)
	return nil
}

// submitValidatedOrder submits a new order.
func (m *SpotMarket) submitValidatedOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, []*types.Order, error) {
	isPegged := order.PeggedOrder != nil
	if isPegged {
		order.Status = types.OrderStatusParked
		order.Reason = types.OrderErrorUnspecified

		if m.as.InAuction() {
			// as the order can't trade we don't transfer from the general account to the holding account in this case.
			m.peggedOrders.Park(order)
			// If we are in an auction, we don't insert this order into the book
			// Maybe should return an orderConfirmation with order state PARKED
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil, nil
		}
		// Reprice
		err := m.repricePeggedOrder(order)
		if err != nil {
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

	// if an auction was trigger, and we are a pegged order
	// or a liquidity order, let's return now.
	if m.as.InAuction() && (isPegged || order.IsLiquidityOrder()) {
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
	// to the holding account for the market/asset
	if !order.IsFinished() && order.Remaining > 0 {
		amt := m.calculateAmountBySide(order.Party, order.Side, order.Price, order.Remaining)
		fees := num.UintZero()
		if m.as.InAuction() {
			fees, err = m.calculateFees(order.Party, order.Remaining, order.Price, order.Side)
			if err != nil {
				return nil, nil, m.unregisterAndReject(ctx, order, err)
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
	}

	// we replace the trades in the confirmation with the one we got initially
	// the contains the fees information
	// NB: I have to say this this is a weird way of doing it, why are we doing it twice?
	confirmation.Trades = trades

	// Send out the order update here as handling the confirmation message
	// below might trigger an action that can change the order details.
	m.broker.Send(events.NewOrderEvent(ctx, order))

	orderUpdates := m.handleConfirmation(ctx, confirmation)
	return confirmation, orderUpdates, nil
}

func (m *SpotMarket) handleTrade(ctx context.Context, trade *types.Trade) ([]*types.LedgerMovement, error) {
	transfers := []*types.LedgerMovement{}
	// we need to transfer base from the seller to the buyer,
	// quote from buyer to the seller.
	// if we're in auction we first need to release the fee funds for the buyer
	// and release the funds for both sides from the holding accounts.
	fees, err := m.calculateFeesForTrades([]*types.Trade{trade})
	if err != nil {
		m.log.Panic("failed to calculate fees for trade", logging.Trade(trade))
	}
	// release the quote quantity + fees if any from the holding to the general account of the buyer
	if trade.Aggressor == types.SideUnspecified {
		transfer, err := m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.BuyOrder, trade.Buyer, m.quoteAsset, scaleQuoteQuantityToAssetDP(trade.Size, trade.Price, m.positionFactor), fees.TotalFeesAmountPerParty()[trade.Buyer])
		if err != nil {
			m.log.Panic("failed to release funds from holding account for trade", logging.Trade(trade))
		}
		transfers = append(transfers, transfer)

		transfer, err = m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.SellOrder, trade.Seller, m.baseAsset, scaleBaseQuantityToAssetDP(trade.Size, m.baseFactor), fees.TotalFeesAmountPerParty()[trade.Seller])
		if err != nil {
			m.log.Panic("failed to release funds from holding account for trade", logging.Trade(trade))
		}
		transfers = append(transfers, transfer)
	} else {
		// only need to release for the passive side
		if trade.Aggressor == types.SideSell {
			transfer, err := m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.BuyOrder, trade.Buyer, m.quoteAsset, scaleQuoteQuantityToAssetDP(trade.Size, trade.Price, m.positionFactor), fees.TotalFeesAmountPerParty()[trade.Buyer])
			if err != nil {
				m.log.Panic("failed to release funds from holding account for trade", logging.Trade(trade))
			}
			transfers = append(transfers, transfer)
		} else {
			transfer, err := m.orderHoldingTracker.ReleaseQuantityHoldingAccount(ctx, trade.SellOrder, trade.Seller, m.baseAsset, scaleBaseQuantityToAssetDP(trade.Size, m.baseFactor), fees.TotalFeesAmountPerParty()[trade.Seller])
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
	sellerFees, ok := fees.TotalFeesAmountPerParty()[trade.Seller]
	sellerPayoutAmt := scaleQuoteQuantityToAssetDP(trade.Size, trade.Price, m.positionFactor)
	if ok && !sellerFees.IsZero() {
		sellerPayoutAmt.Sub(sellerPayoutAmt, sellerFees)
	}
	transfer, err = m.collateral.TransferSpot(ctx, trade.Buyer, trade.Seller, m.quoteAsset, sellerPayoutAmt)
	if err != nil {
		m.log.Panic("failed to complete spot transfer", logging.Trade(trade))
	}
	transfers = append(transfers, transfer)

	// now pay fees - always from the buyer
	m.applyFees(ctx, trade.Buyer, fees)
	return transfers, nil
}

func (m *SpotMarket) handleConfirmationPassiveOrders(ctx context.Context, conf *types.OrderConfirmation) {
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

func (m *SpotMarket) handleConfirmation(ctx context.Context, conf *types.OrderConfirmation) []*types.Order {
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

	m.handleConfirmationPassiveOrders(ctx, conf)
	end := m.as.CanLeave()
	orderUpdates := make([]*types.Order, 0, len(conf.PassiveOrdersAffected)+1)
	orderUpdates = append(orderUpdates, conf.Order)
	orderUpdates = append(orderUpdates, conf.PassiveOrdersAffected...)

	if len(conf.Trades) == 0 {
		return orderUpdates
	}
	m.setLastTradedPrice(conf.Trades[len(conf.Trades)-1])

	// Insert all trades resulted from the executed order
	tradeEvts := make([]events.Event, 0, len(conf.Trades))
	tradedValue, _ := num.UintFromDecimal(
		conf.TradedValue().ToDecimal().Div(m.positionFactor))

	transfers := []*types.LedgerMovement{}
	for idx, trade := range conf.Trades {
		trade.SetIDs(m.idgen.NextID(), conf.Order, conf.PassiveOrdersAffected[idx])

		tradeTransfers, err := m.handleTrade(ctx, trade)
		if err != nil {
			continue
		}
		transfers = append(transfers, tradeTransfers...)
		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))
	}

	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}

	// prepare transfers for the base/quote assets (fees have already been done)

	m.feeSplitter.AddTradeValue(tradedValue)
	m.marketActivityTracker.AddValueTraded(m.mkt.ID, tradedValue)
	m.broker.SendBatch(tradeEvts)
	// check reference moves if we have order updates, and we are not in an auction (or leaving an auction)
	// we handle reference moves in confirmMTM when leaving an auction already
	if len(orderUpdates) > 0 && !end && !m.as.InAuction() {
		m.checkForReferenceMoves(
			ctx, orderUpdates, false)
	}

	return orderUpdates
}

// checkPriceAndGetTrades calculates the trades that would be generated from the given order.
func (m *SpotMarket) checkPriceAndGetTrades(ctx context.Context, order *types.Order) ([]*types.Trade, error) {
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

// TODO
func (m *SpotMarket) applyFees(ctx context.Context, buyer string, fees events.FeesTransfer) error {
	var (
		transfers []*types.LedgerMovement
		err       error
	)

	buyerPaidFees := m.fee.ConsolidateFeePayoutOnBuyer(fees, buyer)

	// because all fees are paid from the quote quantity of the buyer

	if !m.as.InAuction() {
		transfers, err = m.collateral.TransferFeesContinuousTrading(ctx, m.GetID(), m.quoteAsset, buyerPaidFees)
	} else if m.as.IsMonitorAuction() {
		// @TODO handle this properly
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), m.quoteAsset, buyerPaidFees)
	} else if m.as.IsFBA() {
		// @TODO implement transfer for auction types
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), m.quoteAsset, buyerPaidFees)
	}

	// send transfers through the broker
	if len(transfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, transfers))
	}

	m.marketActivityTracker.UpdateFeesFromTransfers(m.GetID(), fees.Transfers())
	return err
}

//
// utilities
//

// calculateFees calculate the amount of fees a party is due to pay given a side/price/size.
// during opening auction there are no fees.
func (m *SpotMarket) calculateFees(party string, size uint64, price *num.Uint, side types.Side) (*num.Uint, error) {
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

	return fees.TotalFeesAmountPerParty()[party], err
}

func (m *SpotMarket) calculateFeesForTrades(trades []*types.Trade) (events.FeesTransfer, error) {
	var (
		fees events.FeesTransfer
		err  error
	)
	if !m.as.InAuction() {
		fees, err = m.fee.CalculateForContinuousMode(trades)
	} else if m.as.IsMonitorAuction() {
		// we are in auction mode
		fees, err = m.fee.CalculateForAuctionMode(trades)
	} else if m.as.IsFBA() {
		fees, err = m.fee.CalculateForFrequentBatchesAuctionMode(trades)
	}
	return fees, err
}

// calculateAmountBySide calculates the amount including fees in the asset decimals.
// Fees are included if we are in monitoring auction.
func (m *SpotMarket) calculateAmountBySide(party string, side types.Side, price *num.Uint, size uint64) *num.Uint {
	fees := num.UintZero()
	if side == types.SideBuy {
		return num.Sum(scaleQuoteQuantityToAssetDP(size, price, m.positionFactor), fees)
	} else {
		return scaleBaseQuantityToAssetDP(size, m.baseFactor)
	}
}

// checkSufficientFunds checks if the aggressor party has in their general account sufficient funds to cover the trade + fees.
func (m *SpotMarket) checkSufficientFunds(party string, side types.Side, price *num.Uint, size uint64) error {
	required := m.calculateAmountBySide(party, side, price, size)
	if side == types.SideBuy {
		if m.collateral.PartyHasSufficientBalance(m.quoteAsset, party, required) != nil {
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

//
// market lifecycle
//

// cleanupOnReject removes all resources created while the market was on PREPARED state.
// at this point no fees would have been collected or anything like this.
func (m *SpotMarket) cleanupOnReject(ctx context.Context) {
	err := m.stopAllLiquidityProvisionOnReject(ctx)
	if err != nil {
		m.log.Debug("could not stop all liquidity provision on market rejection",
			logging.MarketID(m.GetID()),
			logging.Error(err))
	}

	tresps, err := m.collateral.ClearSpotMarket(ctx, m.GetID())
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

// canTrade returns true if the market state is active pending or suspended.
func (m *SpotMarket) canTrade() bool {
	return m.mkt.State == types.MarketStateActive ||
		m.mkt.State == types.MarketStatePending ||
		m.mkt.State == types.MarketStateSuspended
}

// terminateTrading terminates a market - this can be triggered only via governance,
func (m *SpotMarket) TerminateTrading(ctx context.Context, tt bool) {
	// ignore trading termination while the governance proposal hasn't been enacted
	if m.mkt.State == types.MarketStateProposed {
		return
	}

	if m.mkt.State != types.MarketStatePending {
		m.markPrice = m.lastTradedPrice
		m.mkt.State = types.MarketStateTradingTerminated
		m.mkt.TradingMode = types.MarketTradingModeNoTrading
		m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
		if err := m.closeMarket(ctx, m.timeService.GetTimeNow()); err != nil {
			m.log.Error("could not close market", logging.Error(err))
		}
		m.closed = m.mkt.State == types.MarketStateSettled
		m.broker.Send(events.NewMarketSettled(ctx, m.GetID(), m.timeService.GetTimeNow().UnixNano(), m.lastTradedPrice, m.positionFactor))
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

// cleanMarketWithState clears the collateral state of the market and clears up state vars and sets the terminated state of the market
// NB: should it actually go to settled?.
func (m *SpotMarket) cleanMarketWithState(ctx context.Context, mktState types.MarketState) error {
	clearMarketTransfers, err := m.collateral.ClearSpotMarket(ctx, m.GetID())
	if err != nil {
		m.log.Error("Clear market error",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return err
	}

	// unregister state-variables
	m.stateVarEngine.UnregisterStateVariable(m.baseAsset+"_"+m.quoteAsset, m.mkt.ID)

	if len(clearMarketTransfers) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, clearMarketTransfers))
	}

	m.mkt.State = mktState
	m.mkt.TradingMode = types.MarketTradingModeNoTrading
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

// closeCancelledMarket cleans up after a cancelled market.
func (m *SpotMarket) closeCancelledMarket(ctx context.Context) error {
	if err := m.cleanMarketWithState(ctx, types.MarketStateCancelled); err != nil {
		return err
	}

	if err := m.stopAllLiquidityProvisionOnReject(ctx); err != nil {
		m.log.Debug("could not stop all liquidity provision on market rejection",
			logging.MarketID(m.GetID()),
			logging.Error(err))
	}

	m.closed = true
	return nil
}

// closeMarket
// NB: this is currently called immediately from terminate trading.
func (m *SpotMarket) closeMarket(ctx context.Context, t time.Time) error {
	// final distribution of liquidity fees
	m.distributeLiquidityFees(ctx)

	err := m.cleanMarketWithState(ctx, types.MarketStateSettled)
	if err != nil {
		return err
	}

	m.removeOrders(ctx)

	for _, party := range m.liquidity.ProvisionsPerParty().Slice() {
		// we don't care about the actual orders as they will be cancelled in the book as part of settlement anyways.
		err := m.liquidity.StopLiquidityProvision(ctx, party.Party)
		if err != nil {
			return err
		}
	}

	return nil
}

// Reject a market if the market state allow.
func (m *SpotMarket) Reject(ctx context.Context) error {
	if m.mkt.State != types.MarketStateProposed {
		return ErrCannotRejectMarketNotInProposedState
	}

	// we closed all parties accounts
	m.cleanupOnReject(ctx)
	m.mkt.State = types.MarketStateRejected
	m.mkt.TradingMode = types.MarketTradingModeNoTrading
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

// price utils
//
// getLastTradedPrice returns the last traded price.
func (m *SpotMarket) getLastTradedPrice() *num.Uint {
	if m.lastTradedPrice == nil {
		return num.UintZero()
	}
	return m.lastTradedPrice.Clone()
}

// getCurrentMarkPrice returns the current mark price.
// NB: in spot market there's no mark to market so not sure what's the point of this but keeping for now.
func (m *SpotMarket) getCurrentMarkPrice() *num.Uint {
	if m.markPrice == nil {
		return num.UintZero()
	}
	return m.markPrice.Clone()
}

// getReferencePrice returns mark price when market is not in auction, otherwise indicative uncrossing price is returned.
func (m *SpotMarket) getReferencePrice() *num.Uint {
	if !m.as.InAuction() {
		return m.getCurrentMarkPrice()
	}
	ip := m.matching.GetIndicativePrice() // can be zero
	if ip.IsZero() {
		return m.getCurrentMarkPrice()
	}
	return ip
}

// getMarketObservable returns current mark price once market is out of opening auction.
// During opening auction the indicative uncrossing price is returned.
func (m *SpotMarket) getMarketObservable(fallbackPrice *num.Uint) *num.Uint {
	// during opening auction we don't have a last traded price, so we use the indicative price instead
	if m.as.IsOpeningAuction() {
		if ip := m.matching.GetIndicativePrice(); !ip.IsZero() {
			return ip
		}
		// we don't have an indicative price yet so we use the supplied price
		return fallbackPrice
	}
	return m.getCurrentMarkPrice()
}

// minValidPrice returns the minimal price and price increment.
func (m *SpotMarket) minValidPrice() *num.Uint {
	return m.priceFactor
}

// GetTotalOrderBookLevelCount returns the total number of levels in the order book.
func (m *SpotMarket) GetTotalOrderBookLevelCount() uint64 {
	return m.matching.GetOrderBookLevelCount()
}

// GetTotalPeggedOrderCount returns the total number of pegged orders.
func (m *SpotMarket) GetTotalPeggedOrderCount() uint64 {
	return m.matching.GetPeggedOrdersCount()
}

// GetTotalOpenPositionCount returns the total number of open positions.
func (m *SpotMarket) GetTotalOpenPositionCount() uint64 {
	return 0
}

// GetTotalLPShapeCount returns the total number of LP shapes.
func (m *SpotMarket) GetTotalLPShapeCount() uint64 {
	return m.liquidity.GetLPShapeCount()
}

// getOrderBy looks for the order in the order book and in the list
// of pegged orders in the market. Returns the order if found, a bool
// representing if the order was found on the order book and any error code.
func (m *SpotMarket) getOrderByID(orderID string) (*types.Order, bool, error) {
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
	return nil, false, ErrOrderNotFound
}

func (m *SpotMarket) getBestStaticAskPrice() (*num.Uint, error) {
	return m.matching.GetBestStaticAskPrice()
}

func (m *SpotMarket) getBestStaticAskPriceAndVolume() (*num.Uint, uint64, error) {
	return m.matching.GetBestStaticAskPriceAndVolume()
}

func (m *SpotMarket) getBestStaticBidPrice() (*num.Uint, error) {
	return m.matching.GetBestStaticBidPrice()
}

func (m *SpotMarket) getBestStaticBidPriceAndVolume() (*num.Uint, uint64, error) {
	return m.matching.GetBestStaticBidPriceAndVolume()
}

func (m *SpotMarket) getBestStaticPricesDecimal() (bid, ask num.Decimal, err error) {
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

func (m *SpotMarket) getStaticMidPrice(side types.Side) (*num.Uint, error) {
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

//
// target stake/liquidity
//

// updateLiquidityFee computes the current LiquidityProvision fee and updates
// the fee engine.
func (m *SpotMarket) updateLiquidityFee(ctx context.Context) {
	stake := m.getTargetStake()
	fee := m.liquidity.ProvisionsPerParty().FeeForTarget(stake)
	if !fee.Equals(m.getLiquidityFee()) {
		m.fee.SetLiquidityFee(fee)
		m.setLiquidityFee(fee)
		m.broker.Send(
			events.NewMarketUpdatedEvent(ctx, *m.mkt),
		)
	}
}

func (m *SpotMarket) setLiquidityFee(fee num.Decimal) {
	m.mkt.Fees.Factors.LiquidityFee = fee
}

func (m *SpotMarket) getLiquidityFee() num.Decimal {
	return m.mkt.Fees.Factors.LiquidityFee
}

func (m *SpotMarket) setLastTradedPrice(trade *types.Trade) {
	m.lastTradedPrice = trade.Price.Clone()
}

// MAYBE COMMON - liquidity
func (m *SpotMarket) updateMarketValueProxy() {
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
	ts := m.liquidity.ProvisionsPerParty().TotalStake()
	m.lastMarketValueProxy = m.feeSplitter.MarketValueProxy(
		m.marketValueWindowLength, ts)
}

// NOT COMMON
func (m *SpotMarket) getTheoreticalTargetStake() *num.Uint {
	rf := m.risk.GetRiskFactors()

	// Ignoring the error as GetTheoreticalTargetStake handles trades==nil and len(trades)==0
	trades, _ := m.matching.GetIndicativeTrades()

	return m.tsCalc.GetTheoreticalTargetStake(
		*rf, m.timeService.GetTimeNow(), m.getReferencePrice(), trades)
}

// NOT COMMON
func (m *SpotMarket) getTargetStake() *num.Uint {
	return m.tsCalc.GetTargetStake(*m.risk.GetRiskFactors(), m.timeService.GetTimeNow(), m.getCurrentMarkPrice())
}

// NOT COMMON
func (m *SpotMarket) getSuppliedStake() *num.Uint {
	return m.liquidity.CalculateSuppliedStake()
}

// COMMON
func (m *SpotMarket) canSubmitCommitment() bool {
	return m.canTrade() || m.mkt.State == types.MarketStateProposed
}

// TODO revisit with liquidity engine
func (m *SpotMarket) stopAllLiquidityProvisionOnReject(ctx context.Context) error {
	for party := range m.liquidity.ProvisionsPerParty() {
		// here we ignore  the list of orders that could have been
		// created with this party liquidity provision. At this point
		// if we are calling this function, the market is in a PENDING
		// state, which means that liquidity provision can be submitted
		// but orders would never be able to be deployed, so it's safe
		// to ignorethe second return as it shall be an empty slice.
		err := m.liquidity.StopLiquidityProvision(ctx, party)
		if err != nil {
			return err
		}
	}
	return nil
}

// REVIEW
func (m *SpotMarket) distributeLiquidityFees(ctx context.Context) error {
	acc, err := m.collateral.GetMarketLiquidityFeeAccount(m.mkt.GetID(), m.quoteAsset)
	if err != nil {
		return fmt.Errorf("failed to get market liquidity fee account: %w", err)
	}

	// We can't distribute any share when no balance.
	if acc.Balance.IsZero() {
		// reset next distribution period
		m.liquidity.ResetAverageLiquidityScores()
		return nil
	}

	shares := m.equityShares.SharesExcept(m.liquidity.GetInactiveParties())
	if len(shares) == 0 {
		return nil
	}

	// get liquidity scores and reset for next period
	shares = m.updateSharesWithLiquidityScores(shares)

	feeTransfer := m.fee.BuildLiquidityFeeDistributionTransfer(shares, acc)
	if feeTransfer == nil {
		return nil
	}

	m.marketActivityTracker.UpdateFeesFromTransfers(m.GetID(), feeTransfer.Transfers())
	resp, err := m.collateral.TransferFees(ctx, m.GetID(), m.quoteAsset, feeTransfer)
	if err != nil {
		return fmt.Errorf("failed to transfer fees: %w", err)
	}

	if len(resp) > 0 {
		m.broker.Send(events.NewLedgerMovements(ctx, resp))
	}

	return nil
}

func (m *SpotMarket) OnMarketMinLpStakeQuantumMultipleUpdate(_ context.Context, d num.Decimal) {
	m.minLPStakeQuantumMultiple = d
}

func (m *SpotMarket) OnMarketMinProbabilityOfTradingLPOrdersUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnMinProbabilityOfTradingLPOrdersUpdate(d)
}

func (m *SpotMarket) BondPenaltyFactorUpdate(ctx context.Context, d num.Decimal) {
	m.bondPenaltyFactor = d
}

func (m *SpotMarket) OnFeeFactorsMakerFeeUpdate(ctx context.Context, d num.Decimal) error {
	m.fee.OnFeeFactorsMakerFeeUpdate(d)
	m.mkt.Fees.Factors.MakerFee = d
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *SpotMarket) OnFeeFactorsInfrastructureFeeUpdate(ctx context.Context, d num.Decimal) error {
	m.fee.OnFeeFactorsInfrastructureFeeUpdate(d)
	m.mkt.Fees.Factors.InfrastructureFee = d
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *SpotMarket) OnSuppliedStakeToObligationFactorUpdate(d num.Decimal) {
	m.liquidity.OnSuppliedStakeToObligationFactorUpdate(d)
}

func (m *SpotMarket) OnMarketValueWindowLengthUpdate(d time.Duration) {
	m.marketValueWindowLength = d
}

func (m *SpotMarket) OnMarketLiquidityProvidersFeeDistribitionTimeStep(d time.Duration) {
	m.lpFeeDistributionTimeStep = d
}

func (m *SpotMarket) OnMarketTargetStakeTimeWindowUpdate(d time.Duration) {
	m.tsCalc.UpdateTimeWindow(d)
}

func (m *SpotMarket) OnMarketTargetStakeScalingFactorUpdate(d num.Decimal) error {
	return m.tsCalc.UpdateScalingFactor(d)
}

func (m *SpotMarket) OnMarketLiquidityProvisionShapesMaxSizeUpdate(v int64) error {
	return m.liquidity.OnMarketLiquidityProvisionShapesMaxSizeUpdate(v)
}

func (m *SpotMarket) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(d num.Decimal) {
	m.liquidity.OnMaximumLiquidityFeeFactorLevelUpdate(d)
}

func (m *SpotMarket) OnMarketProbabilityOfTradingTauScalingUpdate(_ context.Context, d num.Decimal) {
	m.liquidity.OnProbabilityOfTradingTauScalingUpdate(d)
}

func (m *SpotMarket) OnMarketAuctionMinimumDurationUpdate(ctx context.Context, d time.Duration) {
	m.pMonitor.SetMinDuration(d)
	evt := m.as.UpdateMinDuration(ctx, d)
	// we were in an auction, and the duration of the auction was updated
	if evt != nil {
		m.broker.Send(evt)
	}
}

func (m *SpotMarket) checkAuction(ctx context.Context, now time.Time) {
	if !m.as.InAuction() {
		if m.as.AuctionStart() {
			m.enterAuction(ctx)
		}
		return
	}

	// as soon as we have an indicative uncrossing price in opening auction it needs to be passed into the price monitoring engine so statevar calculation can start
	isOpening := m.as.IsOpeningAuction()
	if isOpening && !m.pMonitor.Initialised() {
		trades, err := m.matching.GetIndicativeTrades()
		if err != nil {
			m.log.Panic("Can't get indicative trades")
		}
		if len(trades) > 0 {
			// pass the first uncrossing trades to price engine so state variables depending on it can be initialised
			m.pMonitor.CheckPrice(ctx, m.as, trades, true)
			m.OnOpeningAuctionFirstUncrossingPrice()
		}
	}

	if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
		return
	}
	trades, err := m.matching.GetIndicativeTrades()
	if err != nil {
		m.log.Panic("Can't get indicative trades")
	}

	// opening auction
	if isOpening {
		if len(trades) == 0 {
			return
		}

		if !m.as.CanLeave() {
			if e := m.as.AuctionExtended(ctx, now); e != nil {
				m.broker.Send(e)
			}
			return
		}
		// opening auction requirements satisfied at this point, other requirements still need to be checked downstream though
		m.as.SetReadyToLeave()
		m.pMonitor.CheckPrice(ctx, m.as, trades, true)
		if m.as.ExtensionTrigger() == types.AuctionTriggerPrice {
			// this should never, ever happen
			m.log.Panic("Leaving opening auction somehow triggered price monitoring to extend the auction")
		}

		// if we don't have yet consensus for the floating point parameters, stay in the opening auction
		if !m.CanLeaveOpeningAuction() {
			m.log.Info("cannot leave opening auction - waiting for floating point to complete the first round")
			return
		}
		m.log.Info("leaving opening auction for market", logging.String("market-id", m.mkt.ID))
		m.leaveAuction(ctx, now)

		m.equityShares.OpeningAuctionEnded()
		// start the market fee window
		m.feeSplitter.TimeWindowStart(now)
		return
	}
	// price and liquidity auctions
	if endTS := m.as.ExpiresAt(); endTS == nil || !endTS.Before(now) {
		return
	}
	isPrice := m.as.IsPriceAuction() || m.as.IsPriceExtension()
	if isPrice || m.as.CanLeave() {
		m.pMonitor.CheckPrice(ctx, m.as, trades, true)
	}
	end := m.as.CanLeave()
	if evt := m.as.AuctionExtended(ctx, m.timeService.GetTimeNow()); evt != nil {
		m.broker.Send(evt)
		end = false
	}
	// price monitoring engine and liquidity monitoring engine both indicated auction can end
	if end {
		// can we leave based on the book state?
		m.leaveAuction(ctx, now)
	}
}
