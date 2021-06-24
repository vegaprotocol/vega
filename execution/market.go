package execution

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/crypto"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/fee"
	"code.vegaprotocol.io/vega/liquidity"
	liquiditytarget "code.vegaprotocol.io/vega/liquidity/target"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/markets"
	"code.vegaprotocol.io/vega/matching"
	"code.vegaprotocol.io/vega/metrics"
	"code.vegaprotocol.io/vega/monitor"
	lmon "code.vegaprotocol.io/vega/monitor/liquidity"
	"code.vegaprotocol.io/vega/monitor/price"
	"code.vegaprotocol.io/vega/positions"
	"code.vegaprotocol.io/vega/products"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"
	"code.vegaprotocol.io/vega/types"

	"github.com/golang/protobuf/proto"
	"github.com/shopspring/decimal"
)

// InitialOrderVersion is set on `Version` field for every new order submission read from the network
const InitialOrderVersion = 1

var (
	// ErrMarketClosed signals that an action have been tried to be applied on a closed market
	ErrMarketClosed = errors.New("market closed")
	// ErrTraderDoNotExists signals that the trader used does not exists
	ErrTraderDoNotExists = errors.New("trader does not exist")
	// ErrMarginCheckFailed signals that a margin check for a position failed
	ErrMarginCheckFailed = errors.New("margin check failed")
	// ErrMarginCheckInsufficient signals that a margin had not enough funds
	ErrMarginCheckInsufficient = errors.New("insufficient margin")
	// ErrMissingGeneralAccountForParty ...
	ErrMissingGeneralAccountForParty = errors.New("missing general account for party")
	// ErrNotEnoughVolumeToZeroOutNetworkOrder ...
	ErrNotEnoughVolumeToZeroOutNetworkOrder = errors.New("not enough volume to zero out network order")
	// ErrInvalidAmendRemainQuantity signals incorrect remaining qty for a reduce by amend
	ErrInvalidAmendRemainQuantity = errors.New("incorrect remaining qty for a reduce by amend")
	// ErrEmptyMarketID is returned if processed market has an empty id
	ErrEmptyMarketID = errors.New("invalid market id (empty)")
	// ErrInvalidOrderType is returned if processed order has an invalid order type
	ErrInvalidOrderType = errors.New("invalid order type")
	// ErrInvalidExpiresAtTime is returned if the expire time is before the createdAt time
	ErrInvalidExpiresAtTime = errors.New("invalid expiresAt time")
	// ErrGFAOrderReceivedDuringContinuousTrading is returned is a gfa order hits the market when the market is in continuous trading state
	ErrGFAOrderReceivedDuringContinuousTrading = errors.New("gfa order received during continuous trading")
	// ErrGFNOrderReceivedAuctionTrading is returned if a gfn order hits the market when in auction state
	ErrGFNOrderReceivedAuctionTrading = errors.New("gfn order received during auction trading")
	// ErrIOCOrderReceivedAuctionTrading is returned if a ioc order hits the market when in auction state
	ErrIOCOrderReceivedAuctionTrading = errors.New("ioc order received during auction trading")
	// ErrFOKOrderReceivedAuctionTrading is returned if a fok order hits the market when in auction state
	ErrFOKOrderReceivedAuctionTrading = errors.New("fok order received during auction trading")
	// ErrUnableToReprice we are unable to get a price required to reprice
	ErrUnableToReprice = errors.New("unable to reprice")
	// ErrOrderNotFound we cannot find the order in the market
	ErrOrderNotFound = errors.New("unable to find the order in the market")
	// ErrTradingNotAllowed no trading related functionalities are allowed in the current state
	ErrTradingNotAllowed = errors.New("trading not allowed")
	// ErrCommitmentSubmissionNotAllowed no commitment submission are permitted in the current state
	ErrCommitmentSubmissionNotAllowed = errors.New("commitment submission not allowed")
	// ErrNotEnoughStake is returned when a LP update results in not enough commitment
	ErrNotEnoughStake = errors.New("commitment submission rejected, not enough stake")

	// ErrCannotRejectMarketNotInProposedState
	ErrCannotRejectMarketNotInProposedState = errors.New("cannot reject a market not in proposed state")
	// ErrCannotStateOpeningAuctionForMarketNotInProposedState
	ErrCannotStartOpeningAuctionForMarketNotInProposedState = errors.New("cannot start the opening auction for a market not in proposed state")
	// ErrCannotRepriceDuringAuction
	ErrCannotRepriceDuringAuction = errors.New("cannot reprice during auction")

	networkPartyID = "network"
)

// PriceMonitor interface to handle price monitoring/auction triggers
// @TODO the interface shouldn't be imported here
type PriceMonitor interface {
	CheckPrice(ctx context.Context, as price.AuctionState, p, v uint64, now time.Time, persistent bool) error
	GetCurrentBounds() []*types.PriceMonitoringBounds
	SetMinDuration(d time.Duration)
	GetValidPriceRange() (float64, float64)
}

// LiquidityMonitor
type LiquidityMonitor interface {
	CheckLiquidity(as lmon.AuctionState, t time.Time, currentStake float64, trades []*types.Trade, rf types.RiskFactor, markPrice uint64, bestStaticBidVolume, bestStaticAskVolume uint64)
	SetMinDuration(d time.Duration)
	UpdateTargetStakeTriggerRatio(ctx context.Context, ratio float64)
}

// TargetStakeCalculator interface
type TargetStakeCalculator interface {
	RecordOpenInterest(oi uint64, now time.Time) error
	GetTargetStake(rf types.RiskFactor, now time.Time, markPrice uint64) float64
	UpdateScalingFactor(sFactor float64) error
	UpdateTimeWindow(tWindow time.Duration)
}

// AuctionState ...
// We can't use the interface yet. AuctionState is passed to the engines, which access different methods
// keep the interface for documentation purposes
type AuctionState interface {
	// are we in auction, and what auction are we in?
	InAuction() bool
	IsOpeningAuction() bool
	IsPriceAuction() bool
	IsLiquidityAuction() bool
	IsFBA() bool
	IsMonitorAuction() bool
	// is it the start/end of an auction
	AuctionStart() bool
	CanLeave() bool
	// when does the auction start/end
	ExpiresAt() *time.Time
	Start() time.Time
	// signal we've started/ended the auction
	AuctionStarted(ctx context.Context) *events.Auction
	Left(ctx context.Context, now time.Time) *events.Auction
	// get some data
	Mode() types.Market_TradingMode
	Trigger() types.AuctionTrigger
	ExtensionTrigger() types.AuctionTrigger
	// UpdateMinDuration works out whether or not the current auction period (if applicable) should be extended
	UpdateMinDuration(ctx context.Context, d time.Duration) *events.Auction
}

// Market represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transactions
type Market struct {
	log   *logging.Logger
	idgen *IDgenerator

	mkt         *types.Market
	closingAt   time.Time
	currentTime time.Time

	mu sync.Mutex

	markPrice uint64

	// own engines
	matching           *matching.CachedOrderBook
	tradableInstrument *markets.TradableInstrument
	risk               *risk.Engine
	position           *positions.Engine
	settlement         *settlement.Engine
	fee                *fee.Engine
	liquidity          *liquidity.Engine

	// deps engines
	collateral *collateral.Engine

	broker Broker
	closed bool

	parties map[string]struct{}

	pMonitor PriceMonitor
	lMonitor LiquidityMonitor

	tsCalc TargetStakeCalculator

	as *monitor.AuctionState // @TODO this should be an interface

	peggedOrders   *PeggedOrders
	expiringOrders *ExpiringOrders

	// Store the previous price values so we can see what has changed
	lastBestBidPrice uint64
	lastBestAskPrice uint64
	lastMidBuyPrice  uint64
	lastMidSellPrice uint64

	lastMarketValueProxy    decimal.Decimal
	bondPenaltyFactor       float64
	marketValueWindowLength time.Duration

	// Liquidity Fee
	feeSplitter                *FeeSplitter
	lpFeeDistributionTimeStep  time.Duration
	lastEquityShareDistributed time.Time
	equityShares               *EquityShares
}

// SetMarketID assigns a deterministic pseudo-random ID to a Market
func SetMarketID(marketcfg *types.Market, seq uint64) error {
	marketcfg.Id = ""
	marketbytes, err := proto.Marshal(marketcfg)
	if err != nil {
		return err
	}
	if len(marketbytes) == 0 {
		return errors.New("failed to marshal market")
	}

	seqbytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(seqbytes, seq)

	h := sha256.New()
	h.Write(marketbytes)
	h.Write(seqbytes)

	d := h.Sum(nil)
	d = d[:20]
	marketcfg.Id = base32.StdEncoding.EncodeToString(d)
	return nil
}

// NewMarket creates a new market using the market framework configuration and creates underlying engines.
func NewMarket(
	ctx context.Context,
	log *logging.Logger,
	riskConfig risk.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	matchingConfig matching.Config,
	feeConfig fee.Config,
	liquidityConfig liquidity.Config,
	collateralEngine *collateral.Engine,
	oracleEngine products.OracleEngine,
	mkt *types.Market,
	now time.Time,
	broker Broker,
	idgen *IDgenerator,
	as *monitor.AuctionState,
) (*Market, error) {

	if len(mkt.Id) == 0 {
		return nil, ErrEmptyMarketID
	}

	tradableInstrument, err := markets.NewTradableInstrument(ctx, log, mkt.TradableInstrument, oracleEngine)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate a new market: %w", err)
	}

	closingAt, err := tradableInstrument.Instrument.GetMarketClosingTime()
	if err != nil {
		return nil, fmt.Errorf("unable to get market closing time: %w", err)
	}

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewCachedOrderBook(
		log, matchingConfig, mkt.Id, as.InAuction())
	asset := tradableInstrument.Instrument.Product.GetAsset()
	riskEngine := risk.NewEngine(
		log,
		riskConfig,
		tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel,
		getInitialFactors(log, mkt, asset),
		book,
		as,
		broker,
		now.UnixNano(),
		mkt.GetId(),
	)
	settleEngine := settlement.New(
		log,
		settlementConfig,
		tradableInstrument.Instrument.Product,
		mkt.Id,
		broker,
	)
	positionEngine := positions.New(log, positionConfig)

	feeEngine, err := fee.New(log, feeConfig, *mkt.Fees, asset)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate fee engine: %w", err)
	}

	tsCalc := liquiditytarget.NewEngine(*mkt.LiquidityMonitoringParameters.TargetStakeParameters, positionEngine)

	pMonitor, err := price.NewMonitor(tradableInstrument.RiskModel, *mkt.PriceMonitoringSettings)
	if err != nil {
		return nil, fmt.Errorf("unable to instantiate price monitoring engine: %w", err)
	}

	lMonitor := lmon.NewMonitor(tsCalc, mkt.LiquidityMonitoringParameters)

	liqEngine := liquidity.NewEngine(liquidityConfig, log, broker, idgen, tradableInstrument.RiskModel, pMonitor, mkt.Id)
	// call on chain time update straight away, so
	// the time in the engine is being updatedat creation
	liqEngine.OnChainTimeUpdate(ctx, now)

	// The market is initially create in a proposed state
	mkt.State = types.Market_STATE_PROPOSED
	mkt.TradingMode = types.Market_TRADING_MODE_CONTINUOUS

	// Populate the market timestamps
	ts := &types.MarketTimestamps{
		Proposed: now.UnixNano(),
		Close:    closingAt.UnixNano(),
	}

	if mkt.OpeningAuction != nil {
		ts.Open = now.Add(time.Duration(mkt.OpeningAuction.Duration)).UnixNano()
	} else {
		ts.Open = now.UnixNano()
	}

	mkt.MarketTimestamps = ts

	market := &Market{
		log:                log,
		idgen:              idgen,
		mkt:                mkt,
		closingAt:          closingAt,
		currentTime:        now,
		matching:           book,
		tradableInstrument: tradableInstrument,
		risk:               riskEngine,
		position:           positionEngine,
		settlement:         settleEngine,
		collateral:         collateralEngine,
		broker:             broker,
		fee:                feeEngine,
		liquidity:          liqEngine,
		parties:            map[string]struct{}{},
		as:                 as,
		pMonitor:           pMonitor,
		lMonitor:           lMonitor,
		tsCalc:             tsCalc,
		peggedOrders:       NewPeggedOrders(),
		expiringOrders:     NewExpiringOrders(),
		feeSplitter:        &FeeSplitter{},
		equityShares:       NewEquityShares(decimal.Zero),
	}

	return market, nil
}

func appendBytes(bz ...[]byte) []byte {
	var out []byte
	for _, b := range bz {
		out = append(out, b...)
	}
	return out
}

func (m *Market) Hash() []byte {
	mID := logging.String("market-id", m.GetID())
	matchingHash := m.matching.Hash()
	m.log.Debug("orderbook state hash", logging.Hash(matchingHash), mID)

	positionHash := m.position.Hash()
	m.log.Debug("positions state hash", logging.Hash(positionHash), mID)

	accountsHash := m.collateral.Hash()
	m.log.Debug("accounts state hash", logging.Hash(accountsHash), mID)

	return crypto.Hash(appendBytes(
		matchingHash, positionHash, accountsHash,
	))
}

func (m *Market) GetMarketData() types.MarketData {
	bestBidPrice, bestBidVolume, _ := m.matching.BestBidPriceAndVolume()
	bestOfferPrice, bestOfferVolume, _ := m.matching.BestOfferPriceAndVolume()
	bestStaticBidPrice, bestStaticBidVolume, _ := m.getBestStaticBidPriceAndVolume()
	bestStaticOfferPrice, bestStaticOfferVolume, _ := m.getBestStaticAskPriceAndVolume()

	// Auction related values
	var indicativePrice, indicativeVolume uint64
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
	var midPrice uint64
	if bestBidPrice > 0 && bestOfferPrice > 0 {
		midPrice = (bestBidPrice + bestOfferPrice) / 2
	}

	var staticMidPrice uint64
	if bestStaticBidPrice > 0 && bestStaticOfferPrice > 0 {
		staticMidPrice = (bestStaticBidPrice + bestStaticOfferPrice) / 2
	}

	return types.MarketData{
		Market:                    m.GetID(),
		BestBidPrice:              bestBidPrice,
		BestBidVolume:             bestBidVolume,
		BestOfferPrice:            bestOfferPrice,
		BestOfferVolume:           bestOfferVolume,
		BestStaticBidPrice:        bestStaticBidPrice,
		BestStaticBidVolume:       bestStaticBidVolume,
		BestStaticOfferPrice:      bestStaticOfferPrice,
		BestStaticOfferVolume:     bestStaticOfferVolume,
		MidPrice:                  midPrice,
		StaticMidPrice:            staticMidPrice,
		MarkPrice:                 m.markPrice,
		Timestamp:                 m.currentTime.UnixNano(),
		OpenInterest:              m.position.GetOpenInterest(),
		IndicativePrice:           indicativePrice,
		IndicativeVolume:          indicativeVolume,
		AuctionStart:              auctionStart,
		AuctionEnd:                auctionEnd,
		MarketTradingMode:         m.as.Mode(),
		Trigger:                   m.as.Trigger(),
		ExtensionTrigger:          m.as.ExtensionTrigger(),
		TargetStake:               strconv.FormatUint(uint64(math.Ceil(m.getTargetStake())), 10),
		SuppliedStake:             strconv.FormatUint(m.getSuppliedStake(), 10),
		PriceMonitoringBounds:     m.pMonitor.GetCurrentBounds(),
		MarketValueProxy:          m.lastMarketValueProxy.String(),
		LiquidityProviderFeeShare: lpsToLiquidityProviderFeeShare(m.equityShares.lps),
	}
}

// ReloadConf will trigger a reload of all the config settings in the market and all underlying engines
// this is required when hot-reloading any config changes, eg. logger level.
func (m *Market) ReloadConf(
	matchingConfig matching.Config,
	riskConfig risk.Config,
	positionConfig positions.Config,
	settlementConfig settlement.Config,
	feeConfig fee.Config,
) {
	m.log.Info("reloading configuration")
	m.matching.ReloadConf(matchingConfig)
	m.risk.ReloadConf(riskConfig)
	m.position.ReloadConf(positionConfig)
	m.settlement.ReloadConf(settlementConfig)
	m.fee.ReloadConf(feeConfig)
}

func (m *Market) Reject(ctx context.Context) error {
	if m.mkt.State != types.Market_STATE_PROPOSED {
		return ErrCannotRejectMarketNotInProposedState
	}

	// we close all parties accounts
	m.cleanupOnReject(ctx)
	m.mkt.State = types.Market_STATE_REJECTED
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

func (m *Market) StartOpeningAuction(ctx context.Context) error {
	if m.mkt.State != types.Market_STATE_PROPOSED {
		return ErrCannotStartOpeningAuctionForMarketNotInProposedState
	}

	// now we start the opening auction
	if m.as.AuctionStart() {
		// we are now in a pending state
		m.mkt.State = types.Market_STATE_PENDING
		m.mkt.MarketTimestamps.Pending = m.currentTime.UnixNano()
		m.EnterAuction(ctx)
	} else {
		// TODO(): to be removed once we don't have market starting
		// without an opening auction
		m.mkt.State = types.Market_STATE_ACTIVE
	}

	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))
	return nil
}

// GetID returns the id of the given market
func (m *Market) GetID() string {
	return m.mkt.Id
}

// OnChainTimeUpdate notifies the market of a new time event/update.
// todo: make this a more generic function name e.g. OnTimeUpdateEvent
func (m *Market) OnChainTimeUpdate(ctx context.Context, t time.Time) (closed bool) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "OnChainTimeUpdate")

	m.mu.Lock()
	defer m.mu.Unlock()

	// some engines still needs to get updates:
	m.currentTime = t
	m.peggedOrders.OnTimeUpdate(t)
	m.liquidity.OnChainTimeUpdate(ctx, t)
	m.risk.OnTimeUpdate(t)
	m.settlement.OnTick(t)
	m.feeSplitter.SetCurrentTime(t)

	// TODO(): This also assume that the market is not
	// being closed before the market is leaving
	// the opening auction, but settlement at expiry is
	// not even specced or implemented as of now...
	// if the state of the market is just PROPOSED,
	// we will just skip everything there as nothing apply.
	if m.mkt.State == types.Market_STATE_PROPOSED {
		return false
	}

	// distribute liquidity fees each `m.lpFeeDistributionTimeStep`
	if t.Sub(m.lastEquityShareDistributed) > m.lpFeeDistributionTimeStep {
		m.lastEquityShareDistributed = t

		if err := m.distributeLiquidityFees(ctx); err != nil {
			m.log.Panic("liquidity fee distribution error", logging.Error(err))
		}
	}

	closed = t.After(m.closingAt)
	m.closed = closed

	// check auction, if any
	m.checkAuction(ctx, t)

	// TODO(): handle market start time

	m.risk.CalculateFactors(ctx, t)
	timer.EngineTimeCounterAdd()

	m.updateMarketValueProxy()

	if !closed {
		m.broker.Send(events.NewMarketTick(ctx, m.mkt.Id, t))
	} else {
		m.closeMarket(ctx, t)
	}

	return
}

func (m *Market) updateMarketValueProxy() {
	// if windows length is reached, reset fee splitter
	if mvwl := m.marketValueWindowLength; m.feeSplitter.Elapsed() > mvwl {
		m.feeSplitter.TimeWindowStart(m.currentTime)
	}

	// these need to happen every block
	// but also when new LP is submitted just so we are sure we do
	// not have a mvp of 0
	ts := m.liquidity.ProvisionsPerParty().TotalStake()
	m.lastMarketValueProxy = m.feeSplitter.MarketValueProxy(
		m.marketValueWindowLength, ts)
	m.equityShares.WithMVP(m.lastMarketValueProxy)
}

func (m *Market) closeMarket(ctx context.Context, t time.Time) {
	m.mkt.State = types.Market_STATE_TRADING_TERMINATED
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	// market is closed, final settlement
	// call settlement and stuff
	positions, err := m.settlement.Settle(t, m.markPrice)
	if err != nil {
		m.log.Error("Failed to get settle positions on market close",
			logging.Error(err))
		return
	}

	transfers, err := m.collateral.FinalSettlement(ctx, m.GetID(), positions)
	if err != nil {
		m.log.Error("Failed to get ledger movements after settling closed market",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return
	}

	// @TODO pass in correct context -> Previous or next block?
	// Which is most appropriate here?
	// this will be next block
	m.broker.Send(events.NewTransferResponse(ctx, transfers))

	asset, _ := m.mkt.GetAsset()
	parties := make([]string, 0, len(m.parties))
	for k := range m.parties {
		parties = append(parties, k)
	}

	clearMarketTransfers, err := m.collateral.ClearMarket(ctx, m.GetID(), asset, parties)
	if err != nil {
		m.log.Error("Clear market error",
			logging.MarketID(m.GetID()),
			logging.Error(err))
		return
	}

	m.broker.Send(events.NewTransferResponse(ctx, clearMarketTransfers))
}

func (m *Market) unregisterAndReject(ctx context.Context, order *types.Order, err error) error {
	_ = m.position.UnregisterOrder(order)
	order.UpdatedAt = m.currentTime.UnixNano()
	order.Status = types.Order_STATUS_REJECTED
	if oerr, ok := types.IsOrderError(err); ok {
		// the order wasn't invalid, so stopped is a better status, rather than rejected.
		if oerr == types.OrderError_ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS {
			order.Status = types.Order_STATUS_STOPPED
		}
		order.Reason = oerr
	} else {
		// should not happened but still...
		order.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
	}
	m.broker.Send(events.NewOrderEvent(ctx, order))
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Failure after submitting order to matching engine",
			logging.Order(*order),
			logging.Error(err))
	}
	return err
}

func (m *Market) getNewPeggedPrice(order *types.Order) (uint64, error) {
	if m.as.InAuction() {
		return 0, ErrCannotRepriceDuringAuction
	}

	var (
		err   error
		price uint64
	)

	switch order.PeggedOrder.Reference {
	case types.PeggedReference_PEGGED_REFERENCE_MID:
		price, err = m.getStaticMidPrice(order.Side)
	case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
		price, err = m.getBestStaticBidPrice()
	case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
		price, err = m.getBestStaticAskPrice()
	}
	if err != nil {
		return 0, ErrUnableToReprice
	}

	if order.PeggedOrder.Offset >= 0 {
		return price + uint64(order.PeggedOrder.Offset), nil
	}

	// At this stage offset is negative so we change it's sign to cast it to an
	// unsigned type
	offset := uint64(-order.PeggedOrder.Offset)
	if price <= offset {
		return 0, ErrUnableToReprice
	}

	return price - offset, nil
}

// Reprice a pegged order. This only updates the price on the order
func (m *Market) repricePeggedOrder(ctx context.Context, order *types.Order) error {
	// Work out the new price of the order
	price, err := m.getNewPeggedPrice(order)
	if err != nil {
		return err
	}
	order.Price = price
	return nil
}

func (m *Market) parkAllPeggedOrders(ctx context.Context) []*types.Order {
	toPark := m.peggedOrders.GetAllActiveOrders()
	for _, order := range toPark {
		m.parkOrder(ctx, order)
	}
	return toPark
}

// EnterAuction : Prepare the order book to be run as an auction
func (m *Market) EnterAuction(ctx context.Context) {
	// Change market type to auction
	ordersToCancel, err := m.matching.EnterAuction()
	if err != nil {
		m.log.Error("Error entering auction: ", logging.Error(err))
	}

	// Move into auction mode to prevent pegged order repricing
	event := m.as.AuctionStarted(ctx)

	// this is at least the size of the orders to be cancelled
	updatedOrders := make([]*types.Order, 0, len(ordersToCancel))

	// Cancel all the orders that were invalid
	for _, order := range ordersToCancel {
		_, err := m.cancelOrder(ctx, order.PartyId, order.Id)
		if err != nil {
			m.log.Debug("error cancelling order when entering auction",
				logging.MarketID(m.GetID()),
				logging.OrderID(order.Id),
				logging.Error(err))
		}
		updatedOrders = append(updatedOrders, order)
	}

	// now update all special orders
	m.enterAuctionSpecialOrders(ctx, updatedOrders)

	// Send an event bus update
	m.broker.Send(event)
}

// LeaveAuction : Return the orderbook and market to continuous trading
func (m *Market) LeaveAuction(ctx context.Context, now time.Time) {
	// Change market type to continuous trading
	uncrossedOrders, ordersToCancel, err := m.matching.LeaveAuction(m.currentTime)
	if err != nil {
		m.log.Error("Error leaving auction", logging.Error(err))
	}

	// Process each confirmation & apply fee calculations to each trade
	evts := make([]events.Event, 0, len(uncrossedOrders))
	for _, uncrossedOrder := range uncrossedOrders {
		// handle fees first
		err := m.applyFees(ctx, uncrossedOrder.Order, uncrossedOrder.Trades)
		if err != nil {
			// @TODO this ought to be an event
			m.log.Error("Unable to apply fees to order",
				logging.String("OrderID", uncrossedOrder.Order.Id))
		}

		// then do the confirmation
		m.handleConfirmation(ctx, uncrossedOrder)

		if uncrossedOrder.Order.Remaining == 0 {
			uncrossedOrder.Order.Status = types.Order_STATUS_FILLED
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
		conf, err := m.cancelOrder(ctx, order.PartyId, order.Id)
		if err != nil {
			m.log.Panic("Failed to cancel order",
				logging.Error(err),
				logging.String("OrderID", order.Id))
		}

		updatedOrders = append(updatedOrders, conf.Order)
	}

	// now that we're left the auction, we can mark all positions
	// in case any trader is distressed (Which shouldn't be possible)
	// we'll fall back to the a network order at the new mark price (mid-price)
	m.confirmMTM(ctx, &types.Order{Price: m.markPrice})

	// keep var to see if we're leaving opening auction
	isOpening := m.as.IsOpeningAuction()
	// update auction state, so we know what the new tradeMode ought to be
	endEvt := m.as.Left(ctx, now)

	for _, uncrossedOrder := range uncrossedOrders {
		if !isOpening {
			// @TODO we should update this once
			for _, trade := range uncrossedOrder.Trades {
				err := m.pMonitor.CheckPrice(
					ctx, m.as, trade.Price, trade.Size, now, true,
				)
				if err != nil {
					m.log.Panic("unable to run check price with price monitor",
						logging.String("market-id", m.GetID()),
						logging.Error(err))
				}
			}
		}

		updatedOrders = append(updatedOrders, uncrossedOrder.Order)
		updatedOrders = append(
			updatedOrders, uncrossedOrder.PassiveOrdersAffected...)

	}

	// Send an event bus update
	m.broker.Send(endEvt)

	m.checkForReferenceMoves(ctx, updatedOrders, true)

	m.updateLiquidityFee(ctx)
}

func (m *Market) validatePeggedOrder(ctx context.Context, order *types.Order) types.OrderError {
	if order.Type != types.Order_TYPE_LIMIT {
		// All pegged orders must be LIMIT orders
		return types.ErrPeggedOrderMustBeLimitOrder
	}

	if order.TimeInForce != types.Order_TIME_IN_FORCE_GTT && order.TimeInForce != types.Order_TIME_IN_FORCE_GTC {
		// Pegged orders can only be GTC or GTT
		return types.ErrPeggedOrderMustBeGTTOrGTC
	}

	if order.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
		// We must specify a valid reference
		return types.ErrPeggedOrderWithoutReferencePrice
	}

	if order.Side == types.Side_SIDE_BUY {
		switch order.PeggedOrder.Reference {
		case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
			return types.ErrPeggedOrderBuyCannotReferenceBestAskPrice
		case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
			if order.PeggedOrder.Offset > 0 {
				return types.ErrPeggedOrderOffsetMustBeLessOrEqualToZero
			}
		case types.PeggedReference_PEGGED_REFERENCE_MID:
			if order.PeggedOrder.Offset >= 0 {
				return types.ErrPeggedOrderOffsetMustBeLessThanZero
			}
		}
	} else {
		switch order.PeggedOrder.Reference {
		case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
			if order.PeggedOrder.Offset < 0 {
				return types.ErrPeggedOrderOffsetMustBeGreaterOrEqualToZero
			}
		case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
			return types.ErrPeggedOrderSellCannotReferenceBestBidPrice
		case types.PeggedReference_PEGGED_REFERENCE_MID:
			if order.PeggedOrder.Offset <= 0 {
				return types.ErrPeggedOrderOffsetMustBeGreaterThanZero
			}
		}
	}
	return types.OrderError_ORDER_ERROR_UNSPECIFIED
}

func (m *Market) validateOrder(ctx context.Context, order *types.Order) error {
	// Check we are allowed to handle this order type with the current market status
	isAuction := m.as.InAuction()
	if isAuction && order.TimeInForce == types.Order_TIME_IN_FORCE_GFN {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_GFN_ORDER_DURING_AN_AUCTION
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrGFNOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.Order_TIME_IN_FORCE_IOC {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrIOCOrderReceivedAuctionTrading
	}

	if isAuction && order.TimeInForce == types.Order_TIME_IN_FORCE_FOK {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrFOKOrderReceivedAuctionTrading
	}

	if !isAuction && order.TimeInForce == types.Order_TIME_IN_FORCE_GFA {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_GFA_ORDER_DURING_CONTINUOUS_TRADING
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrGFAOrderReceivedDuringContinuousTrading
	}

	// Check the expiry time is valid
	if order.ExpiresAt > 0 && order.ExpiresAt < order.CreatedAt {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrInvalidExpiresAtTime
	}

	if m.closed {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_MARKET_CLOSED
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrMarketClosed
	}

	if order.Type == types.Order_TYPE_NETWORK {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_TYPE
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrInvalidOrderType
	}

	// Validate market
	if order.MarketId != m.mkt.Id {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INVALID_MARKET_ID
		m.broker.Send(events.NewOrderEvent(ctx, order))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.Order(*order),
				logging.String("market", m.mkt.Id))
		}
		return types.ErrInvalidMarketID
	}

	// Validate pegged orders
	if order.PeggedOrder != nil {
		reason := m.validatePeggedOrder(ctx, order)
		if reason != types.OrderError_ORDER_ERROR_UNSPECIFIED {
			order.Status = types.Order_STATUS_REJECTED
			order.Reason = reason

			m.broker.Send(events.NewOrderEvent(ctx, order))

			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failed to validate pegged order details",
					logging.Order(*order),
					logging.String("market", m.mkt.Id))
			}
			return reason
		}
	}
	return nil
}

func (m *Market) validateAccounts(ctx context.Context, order *types.Order) error {
	asset, _ := m.mkt.GetAsset()
	if !m.collateral.HasGeneralAccount(order.PartyId, asset) {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE
		m.broker.Send(events.NewOrderEvent(ctx, order))

		// trader should be created before even trying to post order
		return ErrTraderDoNotExists
	}

	// ensure party have a general account, and margin account is / can be created
	_, err := m.collateral.CreatePartyMarginAccount(ctx, order.PartyId, order.MarketId, asset)
	if err != nil {
		m.log.Error("Margin account verification failed",
			logging.String("party-id", order.PartyId),
			logging.String("market-id", m.GetID()),
			logging.String("asset", asset),
		)
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_MISSING_GENERAL_ACCOUNT
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrMissingGeneralAccountForParty
	}

	// from this point we know the party have a margin account
	// we had it to the list of parties.
	m.addParty(order.PartyId)
	return nil
}

func (m *Market) releaseMarginExcess(ctx context.Context, partyID string) {
	// if this position went 0
	pos, ok := m.position.GetPositionByPartyID(partyID)
	if !ok {
		// position was never created or party went distressed and don't exist
		// all good we can return
		return
	}

	// now check if all buy/sell/size are 0
	if pos.Buy() != 0 || pos.Sell() != 0 || pos.Size() != 0 || pos.VWBuy() != 0 || pos.VWSell() != 0 {
		// position is not 0, nothing to release surely
		return
	}

	asset, _ := m.mkt.GetAsset()
	transfers, err := m.collateral.ClearPartyMarginAccount(
		ctx, partyID, m.GetID(), asset)
	if err != nil {
		m.log.Error("unable to clear party margin account", logging.Error(err))
		return
	}
	evt := events.NewTransferResponse(
		ctx, []*types.TransferResponse{transfers})
	m.broker.Send(evt)
}

// SubmitOrder submits the given order
func (m *Market) SubmitOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	order.CreatedAt = m.currentTime.UnixNano()

	if !m.canTrade() {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_MARKET_CLOSED
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return nil, ErrTradingNotAllowed
	}

	conf, orderUpdates, err := m.submitOrder(ctx, order, true)
	if err != nil {
		return nil, err
	}

	allUpdatedOrders := append(
		[]*types.Order{conf.Order}, conf.PassiveOrdersAffected...)
	allUpdatedOrders = append(allUpdatedOrders, orderUpdates...)

	m.checkForReferenceMoves(
		ctx, allUpdatedOrders, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return conf, nil
}

func (m *Market) submitOrder(ctx context.Context, order *types.Order, setID bool) (*types.OrderConfirmation, []*types.Order, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.Id, orderValidity)
	}()

	// set those at the beginning as even rejected order get through the buffers
	if setID {
		m.idgen.SetID(order)
	}
	order.Version = InitialOrderVersion
	order.Status = types.Order_STATUS_ACTIVE

	if err := m.validateOrder(ctx, order); err != nil {
		return nil, nil, err
	}

	if err := m.validateAccounts(ctx, order); err != nil {
		return nil, nil, err
	}

	if order.PeggedOrder != nil {
		// Add pegged order to time sorted list
		m.peggedOrders.Add(order)
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
		m.expiringOrders.Insert(*order)
	}

	return orderConf, orderUpdates, err
}

func (m *Market) submitValidatedOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, []*types.Order, error) {
	isPegged := order.PeggedOrder != nil
	if isPegged {
		order.Status = types.Order_STATUS_PARKED
		order.Reason = types.OrderError_ORDER_ERROR_UNSPECIFIED

		if m.as.InAuction() {
			// If we are in an auction, we don't insert this order into the book
			// Maybe should return an orderConfirmation with order state PARKED
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil, nil
		} else {
			// Reprice
			err := m.repricePeggedOrder(ctx, order)
			if err != nil {
				m.broker.Send(events.NewOrderEvent(ctx, order))
				return &types.OrderConfirmation{Order: order}, nil, nil
			}
		}
	}

	oldPos, ok := m.position.GetPositionByPartyID(order.PartyId)
	// Register order as potential positions
	pos := m.position.RegisterOrder(order)
	checkMargin := true
	if !isPegged && ok {
		oldVol, newVol := pos.Size()+pos.Buy()-pos.Sell(), oldPos.Size()+pos.Buy()-pos.Sell()
		if oldVol < 0 {
			oldVol = -oldVol
		}
		if newVol < 0 {
			newVol = -newVol
		}
		// check margin if the new volume is greater, or the same (implying long to short, or short to long)
		checkMargin = oldVol <= newVol
	}

	// Perform check and allocate margin unless the order is (partially) closing the trader position
	if checkMargin {
		if err := m.checkMarginForOrder(ctx, pos, order); err != nil {
			if m.log.GetLevel() <= logging.DebugLevel {
				m.log.Debug("Unable to check/add margin for trader",
					logging.Order(*order), logging.Error(err))
			}
			_ = m.unregisterAndReject(
				ctx, order, types.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED)
			return nil, nil, ErrMarginCheckFailed
		}
	}

	// from here we may have assigned some margin.
	// we add the check to roll it back in case we have a 0 positions after this
	defer m.releaseMarginExcess(ctx, order.PartyId)

	// If we are not in an opening auction, apply fees
	var trades []*types.Trade
	// we're not in auction (not opening, not any other auction
	if !m.as.InAuction() {
		// first we call the order book to evaluate auction triggers and get the list of trades
		var err error
		trades, err = m.checkPriceAndGetTrades(ctx, order)
		if err != nil {
			return nil, nil, m.unregisterAndReject(ctx, order, err)
		}

		// try to apply fees on the trade
		err = m.applyFees(ctx, order, trades)
		if err != nil {
			return nil, nil, m.unregisterAndReject(ctx, order, err)
		}
	}

	// if an auction was trigger, and we are a pegged order
	// let's return now.
	if m.as.InAuction() && isPegged {
		return &types.OrderConfirmation{Order: order}, nil, nil
	}

	order.Status = types.Order_STATUS_ACTIVE

	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(order)
	if err != nil {
		return nil, nil, m.unregisterAndReject(ctx, order, err)
	}

	// if order was FOK or IOC some or all of it may have not be consumed, so we need to
	// remove them from the potential orders,
	// then we should be able to process the rest of the order properly.
	if ((order.TimeInForce == types.Order_TIME_IN_FORCE_FOK ||
		order.TimeInForce == types.Order_TIME_IN_FORCE_IOC ||
		order.Status == types.Order_STATUS_STOPPED) &&
		confirmation.Order.Remaining != 0) ||
		// Also do it if specifically we went against a wash trade
		(order.Status == types.Order_STATUS_REJECTED &&
			order.Reason == types.OrderError_ORDER_ERROR_SELF_TRADING) {
		_ = m.position.UnregisterOrder(order)
	}

	// we replace the trades in the confirmation with the one we got initially
	// the contains the fees information
	confirmation.Trades = trades

	// Send out the order update here as handling the confirmation message
	// below might trigger an action that can change the order details.
	m.broker.Send(events.NewOrderEvent(ctx, order))

	orderUpdates := m.handleConfirmation(ctx, confirmation)
	return confirmation, orderUpdates, nil
}

func (m *Market) checkPriceAndGetTrades(ctx context.Context, order *types.Order) ([]*types.Trade, error) {
	trades, err := m.matching.GetTrades(order)
	if err != nil {
		return nil, err
	}
	persistent := true
	switch order.TimeInForce {
	case types.Order_TIME_IN_FORCE_FOK, types.Order_TIME_IN_FORCE_GFN, types.Order_TIME_IN_FORCE_IOC:
		persistent = false
	}

	for _, t := range trades {
		if merr := m.pMonitor.CheckPrice(ctx, m.as, t.Price, t.Size, m.currentTime, persistent); merr != nil {
			// a specific order error
			if err, ok := merr.(types.OrderError); ok {
				return nil, err
			}
			m.log.Panic("unable to run check price with price monitor",
				logging.String("market-id", m.GetID()),
				logging.Error(merr))
		}
	}

	if evt := m.as.AuctionExtended(ctx); evt != nil {
		m.broker.Send(evt)
	}
	m.checkLiquidity(ctx, trades)

	// start the  monitoring auction if required?
	if m.as.AuctionStart() {
		m.EnterAuction(ctx)
		return nil, nil
	}

	return trades, nil
}

func (m *Market) addParty(party string) {
	if _, ok := m.parties[party]; !ok {
		m.parties[party] = struct{}{}
	}
}

func (m *Market) applyFees(ctx context.Context, order *types.Order, trades []*types.Trade) error {
	// if we have some trades, let's try to get the fees

	if len(trades) <= 0 || m.as.IsOpeningAuction() {
		return nil
	}

	// first we get the fees for these trades
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

	if err != nil {
		return err
	}

	var (
		transfers []*types.TransferResponse
		asset, _  = m.mkt.GetAsset()
	)

	if !m.as.InAuction() {
		transfers, err = m.collateral.TransferFeesContinuousTrading(ctx, m.GetID(), asset, fees)
	} else if m.as.IsMonitorAuction() {
		// @TODO handle this properly
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	} else if m.as.IsFBA() {
		// @TODO implement transfer for auction types
		transfers, err = m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	}

	if err != nil {
		m.log.Error("unable to transfer fees for trades",
			logging.String("order-id", order.Id),
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return types.OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES
	}

	// send transfers through the broker
	if err == nil && len(transfers) > 0 {
		evt := events.NewTransferResponse(ctx, transfers)
		m.broker.Send(evt)
	}

	return nil
}

func (m *Market) handleConfirmationPassiveOrders(
	ctx context.Context,
	conf *types.OrderConfirmation) {

	if conf.PassiveOrdersAffected != nil {
		var (
			evts        = make([]events.Event, 0, len(conf.PassiveOrdersAffected))
			currentTime = m.currentTime.UnixNano()
		)

		// Insert or update passive orders siting on the book
		for _, order := range conf.PassiveOrdersAffected {
			// set the `updatedAt` value as these orders have changed
			order.UpdatedAt = currentTime
			evts = append(evts, events.NewOrderEvent(ctx, order))

			// If the order is a pegged order and is complete we must remove it from the pegged list
			if order.PeggedOrder != nil {
				if order.Remaining == 0 || order.Status != types.Order_STATUS_ACTIVE {
					m.removePeggedOrder(order)
				}
			}

			// remove the order from the expiring list
			// if it was a GTT order
			if order.IsExpireable() && order.IsFinished() {
				m.expiringOrders.RemoveOrder(order.ExpiresAt, order.Id)
			}
		}

		m.broker.SendBatch(evts)

	}
}

func (m *Market) handleConfirmation(ctx context.Context, conf *types.OrderConfirmation) (orderUpdates []*types.Order) {
	// When re-submitting liquidity order, it happen that the pricing is putting
	// the order at a price which makes it uncross straight away.
	// then triggering this handleConfirmation flow, etc.
	// Although the order is considered aggressive, and we never expect in the flow
	// for an aggressive order to be pegged, so we never remove them from the pegged
	// list. All this impact the float of EnterAuction, which if triggered from there
	// will try to park all pegged orders, including this order which have never been
	// remove from the pegged list. We add this check to make sure  that if the
	// aggressive order is pegged, we then do remove it from the list.
	if conf.Order.PeggedOrder != nil {
		if conf.Order.Remaining == 0 || conf.Order.Status != types.Order_STATUS_ACTIVE {
			m.removePeggedOrder(conf.Order)
		}
	}

	m.handleConfirmationPassiveOrders(ctx, conf)
	end := m.as.CanLeave()

	if len(conf.Trades) > 0 {

		// Calculate and set current mark price
		m.setMarkPrice(conf.Trades[len(conf.Trades)-1])

		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(conf.Trades))
		for idx, trade := range conf.Trades {
			trade.SetIDs(conf.Order, conf.PassiveOrdersAffected[idx], idx)

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions (this communicates with settlement via channel)
			m.position.Update(trade)
			// Record open interest change
			if err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), m.currentTime); err != nil {
				m.log.Debug("unable record open interest",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			// add trade to settlement engine for correct MTM settlement of individual trades
			m.settlement.AddTrade(trade)
			m.feeSplitter.AddTradeValue(trade.Size * trade.Price)
		}
		m.broker.SendBatch(tradeEvts)

		if !end {
			orderUpdates = m.confirmMTM(ctx, conf.Order)
		}
	}
	return
}

func (m *Market) confirmMTM(
	ctx context.Context, order *types.Order) (orderUpdates []*types.Order) {
	// now let's get the transfers for MTM settlement
	evts := m.position.UpdateMarkPrice(m.markPrice)
	settle := m.settlement.SettleMTM(ctx, m.markPrice, evts)

	// Only process collateral and risk once per order, not for every trade
	margins := m.collateralAndRisk(ctx, settle)
	if len(margins) > 0 {
		transfers, closed, bondPenalties, err := m.collateral.MarginUpdate(ctx, m.GetID(), margins)
		if err == nil && len(transfers) > 0 {
			evt := events.NewTransferResponse(ctx, transfers)
			m.broker.Send(evt)
		}
		if len(bondPenalties) > 0 {
			transfers, err := m.bondSlashing(ctx, bondPenalties...)
			if err != nil {
				m.log.Error("Failed to perform bond slashing",
					logging.Error(err))
			}
			m.broker.Send(events.NewTransferResponse(ctx, transfers))
		}
		if len(closed) > 0 {
			orderUpdates, err = m.resolveClosedOutTraders(ctx, closed, order)
			if err != nil {
				m.log.Error("unable to close out traders",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
		}
		m.updateLiquidityFee(ctx)
	}
	return
}

// updateLiquidityFee computes the current LiquidityProvision fee and updates
// the fee engine.
func (m *Market) updateLiquidityFee(ctx context.Context) {
	stake := m.getTargetStake()
	fee := m.liquidity.ProvisionsPerParty().FeeForTarget(uint64(stake))
	if fee != m.getLiquidityFee() {
		m.fee.SetLiquidityFee(fee)
		m.setLiquidityFee(fee)
		m.broker.Send(
			events.NewMarketUpdatedEvent(ctx, *m.mkt),
		)
	}
}

func (m *Market) setLiquidityFee(fee string) {
	m.mkt.Fees.Factors.LiquidityFee = fee
}
func (m *Market) getLiquidityFee() string {
	return m.mkt.Fees.Factors.LiquidityFee
}

// resolveClosedOutTraders - the traders with the given market position who haven't got sufficient collateral
// need to be closed out -> the network buys/sells the open volume, and trades with the rest of the network
// this flow is similar to the SubmitOrder bit where trades are made, with fewer checks (e.g. no MTM settlement, no risk checks)
// pass in the order which caused traders to be distressed
func (m *Market) resolveClosedOutTraders(ctx context.Context, distressedMarginEvts []events.Margin, o *types.Order) ([]*types.Order, error) {
	if len(distressedMarginEvts) == 0 {
		return nil, nil
	}
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "resolveClosedOutTraders")
	defer timer.EngineTimeCounterAdd()

	// this is going to be run after the the close out routines
	// are finished, in order to notify the liquidity engine of
	// any changes in the book / orders owned by the lp providers
	orderUpdates := []*types.Order{}
	distressedParties := []string{}
	defer func() {
		// First we check for all distressed parties if they are liquidity
		// providers, and if yea cancel their commitments
		for _, party := range distressedParties {
			if m.liquidity.IsLiquidityProvider(party) {
				if err := m.cancelLiquidityProvision(ctx, party, true); err != nil {
					m.log.Debug("could not cancel liquidity provision",
						logging.MarketID(m.GetID()),
						logging.PartyID(party),
						logging.Error(err))
				}
			}
		}

	}()

	distressedPos := make([]events.MarketPosition, 0, len(distressedMarginEvts))
	for _, v := range distressedMarginEvts {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("closing out trader",
				logging.PartyID(v.Party()),
				logging.MarketID(m.GetID()))
		}
		distressedPos = append(distressedPos, v)
		distressedParties = append(distressedParties, v.Party())
	}
	// cancel pending orders for traders
	rmorders, err := m.matching.RemoveDistressedOrders(distressedPos)
	if err != nil {
		m.log.Panic("Failed to remove distressed traders from the orderbook",
			logging.Error(err),
		)
	}
	mktID := m.GetID()
	// push rm orders into buf
	// and remove the orders from the positions engine
	evts := []events.Event{}
	for _, o := range rmorders {
		if o.IsExpireable() {
			m.expiringOrders.RemoveOrder(o.ExpiresAt, o.Id)
		}
		if o.PeggedOrder != nil {
			m.removePeggedOrder(o)
		}
		o.UpdatedAt = m.currentTime.UnixNano()
		evts = append(evts, events.NewOrderEvent(ctx, o))
		_ = m.position.UnregisterOrder(o)
	}

	// add the orders remove from the book to the orders
	// to be sent to the liquidity engine
	orderUpdates = append(orderUpdates, rmorders...)

	// now we also remove ALL parked order for the different parties
	for _, v := range distressedPos {
		orders, oevts := m.peggedOrders.RemoveAllParkedForParty(
			ctx, v.Party(), types.Order_STATUS_STOPPED)

		for _, v := range orders {
			m.expiringOrders.RemoveOrder(v.ExpiresAt, v.Id)
		}

		// add all pegged orders too to the orderUpdates
		orderUpdates = append(orderUpdates, orders...)
		// add all events to evts list
		evts = append(evts, oevts...)

		if m.liquidity.IsLiquidityProvider(v.Party()) {
			if err := m.confiscateBondAccount(ctx, v.Party()); err != nil {
				m.log.Error("unable to confiscate liquidity provision for a distressed party",
					logging.String("party-id", o.PartyId),
					logging.String("market-id", mktID),
					logging.Error(err),
				)
			}
		}
	}

	// send all orders which got stopped through the event bus
	m.broker.SendBatch(evts)

	closed := distressedMarginEvts // default behaviour (ie if rmorders is empty) is to close out all distressed positions we started out with

	// we need to check margin requirements again, it's possible for traders to no longer be distressed now that their orders have been removed
	if len(rmorders) != 0 {
		var okPos []events.Margin // need to declare this because we want to reassign closed
		// now that we closed orders, let's run the risk engine again
		// so it'll separate the positions still in distress from the
		// which have acceptable margins
		okPos, closed = m.risk.ExpectMargins(distressedMarginEvts, m.markPrice)

		if m.log.GetLevel() == logging.DebugLevel {
			for _, v := range okPos {
				if m.log.GetLevel() == logging.DebugLevel {
					m.log.Debug("previously distressed party have now an acceptable margin",
						logging.String("market-id", mktID),
						logging.String("party-id", v.Party()))
				}
			}
		}
	}

	// if no position are meant to be closed, just return now.
	if len(closed) <= 0 {
		return orderUpdates, nil
	}

	// we only need the MarketPosition events here, and rather than changing all the calls
	// we can just keep the MarketPosition bit
	closedMPs := make([]events.MarketPosition, 0, len(closed))
	// get the actual position, so we can work out what the total position of the market is going to be
	var networkPos int64
	for _, pos := range closed {
		networkPos += pos.Size()
		closedMPs = append(closedMPs, pos)
	}
	if networkPos == 0 {
		m.log.Warn("Network positions is 0 after closing out traders, nothing more to do",
			logging.String("market-id", m.GetID()))
		m.finalizePartiesCloseOut(ctx, closed, closedMPs)
		return orderUpdates, nil
	}
	// network order
	// @TODO this order is more of a placeholder than an actual final version
	// of the network order we'll be using
	size := uint64(math.Abs(float64(networkPos)))
	no := types.Order{
		MarketId:    m.GetID(),
		Remaining:   size,
		Status:      types.Order_STATUS_ACTIVE,
		PartyId:     networkPartyID,       // network is not a party as such
		Side:        types.Side_SIDE_SELL, // assume sell, price is zero in that case anyway
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   fmt.Sprintf("LS-%s", o.Id),    // liquidity sourcing, reference the order which caused the problem
		TimeInForce: types.Order_TIME_IN_FORCE_FOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:        types.Order_TYPE_NETWORK,
	}
	no.Size = no.Remaining
	m.idgen.SetID(&no)
	// we need to buy, specify side + max price
	if networkPos < 0 {
		no.Side = types.Side_SIDE_BUY
	}
	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(&no)
	if err != nil {
		// we can safely panic here, only possibility of failure
		// with the orderbook is in case of order validation, it should
		// not be possible for us to submit an invalid order at this
		// point, and an invalid order would be a code error then.
		m.log.Panic("Failure after submitting order to matching engine",
			logging.Order(no),
			logging.Error(err))
	}

	// FIXME(j): this is a temporary measure for the case where we do not have enough orders
	// in the book to 0 out the positions.
	// in this case we will just return now, cutting off the position resolution
	// this means that trader still being distressed will stay distressed,
	// then when a new order is placed, the distressed traders will go again through positions resolution
	// and if the volume of the book is acceptable, we will then process positions resolutions
	if no.Remaining == no.Size {
		return orderUpdates, ErrNotEnoughVolumeToZeroOutNetworkOrder
	}

	// @NOTE: At this point, the network order was updated by the orderbook
	// the price field now contains the average trade price at which the order was fulfilled
	m.broker.Send(events.NewOrderEvent(ctx, &no))

	m.handleConfirmationPassiveOrders(ctx, confirmation)

	// also add the passive orders from the book into the list
	// of updated orders to send to liquidity engine
	orderUpdates = append(orderUpdates, confirmation.PassiveOrdersAffected...)

	asset, _ := m.mkt.GetAsset()

	// pay the fees now
	fees, distressedPartiesFees, err := m.fee.CalculateFeeForPositionResolution(
		confirmation.Trades, closedMPs)
	if err != nil {
		// this cannot fail, so we shall just panic for now, maybe remove
		// the error later on
		m.log.Panic("unable to calculate fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
	}

	tresps, err := m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	if err != nil {
		// FIXME(): we may figure a better error handling in here
		m.log.Error("unable to transfer fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return orderUpdates, err
	}
	// send transfer to buffer
	m.broker.Send(events.NewTransferResponse(ctx, tresps))

	if len(confirmation.Trades) > 0 {
		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(confirmation.Trades))
		for idx, trade := range confirmation.Trades {
			trade.SetIDs(&no, confirmation.PassiveOrdersAffected[idx], idx)

			// setup the type of the trade to network
			// this trade did happen with a GOOD trader to
			// 0 out the BAD trader position
			trade.Type = types.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions - this is a special trade involving the network as party
			// so rather than checking this every time we call Update, call special UpdateNetwork
			m.position.UpdateNetwork(trade)
			if err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), m.currentTime); err != nil {
				m.log.Debug("unable record open interest",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}

			m.settlement.AddTrade(trade)
			m.feeSplitter.AddTradeValue(trade.Size * trade.Price)
		}
		m.broker.SendBatch(tradeEvts)
	}

	if err = m.zeroOutNetwork(ctx, closedMPs, &no, o, distressedPartiesFees); err != nil {
		// FIXME(): the method always returns nil,
		// no change we get an error here.
		m.log.Panic(
			"Failed to create closing order with distressed traders",
			logging.Error(err),
		)
	}

	// swipe all accounts and stuff
	m.finalizePartiesCloseOut(ctx, closed, closedMPs)

	// get the updated positions
	evt := m.position.Positions()

	// settle MTM, the positions have changed
	settle := m.settlement.SettleMTM(ctx, m.markPrice, evt)
	// we're not interested in the events here, they're used for margin updates
	// we know the margin requirements will be met, and come the next block
	// margins will automatically be checked anyway

	_, responses, err := m.collateral.MarkToMarket(ctx, m.GetID(), settle, asset)
	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug(
			"ledger movements after MTM on traders who closed out distressed",
			logging.Int("response-count", len(responses)),
			logging.String("raw", fmt.Sprintf("%#v", responses)),
		)
	}
	// send transfer to buffer
	m.broker.Send(events.NewTransferResponse(ctx, responses))

	return orderUpdates, err
}

func (m *Market) finalizePartiesCloseOut(
	ctx context.Context,
	closed []events.Margin,
	closedMPs []events.MarketPosition,
) {
	// remove accounts, positions and return
	// from settlement engine first
	m.settlement.RemoveDistressed(ctx, closed)
	// then from positions
	closedMPs = m.position.RemoveDistressed(closedMPs)
	asset, _ := m.mkt.GetAsset()
	// finally remove from collateral (moving funds where needed)
	movements, err := m.collateral.RemoveDistressed(
		ctx, closedMPs, m.GetID(), asset)
	if err != nil {
		m.log.Panic(
			"Failed to remove distressed accounts cleanly",
			logging.Error(err))
	}

	if len(movements.Transfers) > 0 {
		m.broker.Send(
			events.NewTransferResponse(
				ctx, []*types.TransferResponse{movements}),
		)
	}
}

func (m *Market) confiscateBondAccount(ctx context.Context, partyID string) error {
	asset, err := m.mkt.GetAsset()
	if err != nil {
		return err
	}
	bacc, err := m.collateral.GetOrCreatePartyBondAccount(ctx, partyID, m.mkt.Id, asset)
	if err != nil {
		return err
	}

	// we may alreadu have confiscated all funds
	if bacc.Balance == 0 {
		return nil
	}

	transfer := &types.Transfer{
		Owner: partyID,
		Amount: &types.FinancialAmount{
			Amount: bacc.Balance,
			Asset:  asset,
		},
		Type:      types.TransferType_TRANSFER_TYPE_BOND_SLASHING,
		MinAmount: bacc.Balance,
	}
	tresp, err := m.collateral.BondUpdate(ctx, m.mkt.Id, partyID, transfer)
	if err != nil {
		return err
	}
	m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))

	return nil
}

func (m *Market) zeroOutNetwork(ctx context.Context, traders []events.MarketPosition, settleOrder, initial *types.Order, fees map[string]*types.Fee) error {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "zeroOutNetwork")
	defer timer.EngineTimeCounterAdd()

	marketID := m.GetID()
	order := types.Order{
		MarketId:    marketID,
		Status:      types.Order_STATUS_FILLED,
		PartyId:     networkPartyID,
		Price:       settleOrder.Price,
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   "close-out distressed",
		TimeInForce: types.Order_TIME_IN_FORCE_FOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
		Type:        types.Order_TYPE_NETWORK,
	}

	asset, _ := m.mkt.GetAsset()
	marginLevels := types.MarginLevels{
		MarketId:  m.mkt.GetId(),
		Asset:     asset,
		Timestamp: m.currentTime.UnixNano(),
	}

	tradeEvts := make([]events.Event, 0, len(traders))
	for i, trader := range traders {
		tSide, nSide := types.Side_SIDE_SELL, types.Side_SIDE_SELL // one of them will have to sell
		if trader.Size() < 0 {
			tSide = types.Side_SIDE_BUY
		} else {
			nSide = types.Side_SIDE_BUY
		}
		tSize := uint64(math.Abs(float64(trader.Size())))

		// set order fields (network order)
		order.Size = tSize
		order.Remaining = 0
		order.Side = nSide
		order.Status = types.Order_STATUS_FILLED // An order with no remaining must be filled
		m.idgen.SetID(&order)

		// this is the party order
		partyOrder := types.Order{
			MarketId:    marketID,
			Size:        tSize,
			Remaining:   0,
			Status:      types.Order_STATUS_FILLED,
			PartyId:     trader.Party(),
			Side:        tSide,             // assume sell, price is zero in that case anyway
			Price:       settleOrder.Price, // average price
			CreatedAt:   m.currentTime.UnixNano(),
			Reference:   fmt.Sprintf("distressed-%d-%s", i, initial.Id),
			TimeInForce: types.Order_TIME_IN_FORCE_FOK, // this is an all-or-nothing order, so TIME_IN_FORCE == FOK
			Type:        types.Order_TYPE_NETWORK,
		}
		m.idgen.SetID(&partyOrder)

		// store the trader order, too
		m.broker.Send(events.NewOrderEvent(ctx, &partyOrder))
		m.broker.Send(events.NewOrderEvent(ctx, &order))

		// now let's create the trade between the party and network
		var (
			buyOrder, sellOrder     *types.Order
			buySideFee, sellSideFee *types.Fee
		)
		if order.Side == types.Side_SIDE_BUY {
			buyOrder = &order
			sellOrder = &partyOrder
			sellSideFee = fees[trader.Party()]
		} else {
			sellOrder = &order
			buyOrder = &partyOrder
			buySideFee = fees[trader.Party()]
		}

		trade := types.Trade{
			Id:        fmt.Sprintf("%s-%010d", partyOrder.Id, 1),
			MarketId:  partyOrder.MarketId,
			Price:     partyOrder.Price,
			Size:      partyOrder.Size,
			Aggressor: order.Side, // we consider network to be aggressor
			BuyOrder:  buyOrder.Id,
			SellOrder: sellOrder.Id,
			Buyer:     buyOrder.PartyId,
			Seller:    sellOrder.PartyId,
			Timestamp: partyOrder.CreatedAt,
			Type:      types.Trade_TYPE_NETWORK_CLOSE_OUT_BAD,
			SellerFee: sellSideFee,
			BuyerFee:  buySideFee,
		}
		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, trade))

		// 0 out margins levels for this trader
		marginLevels.PartyId = trader.Party()
		m.broker.Send(events.NewMarginLevelsEvent(ctx, marginLevels))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("trader closed-out with success",
				logging.String("party-id", trader.Party()),
				logging.String("market-id", m.GetID()))
		}
	}
	if len(tradeEvts) > 0 {
		m.broker.SendBatch(tradeEvts)
	}
	return nil
}

func (m *Market) checkMarginForOrder(ctx context.Context, pos *positions.MarketPosition, order *types.Order) error {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "checkMarginForOrder")
	defer timer.EngineTimeCounterAdd()
	risk, closed, err := m.calcMargins(ctx, pos, order)
	// margin error
	if err != nil {
		return err
	}
	// margins calculated, set about tranferring funds. At this point, if closed is not empty, those traders are distressed
	// the risk slice are risk events, that we must use to transfer funds
	return m.transferMargins(ctx, risk, closed)
}

func (m *Market) setMarkPrice(trade *types.Trade) {
	// The current mark price calculation is simply the last trade
	// in the future this will use varying logic based on market config
	// the responsibility for calculation could be elsewhere for testability
	m.markPrice = trade.Price
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between trader accounts (ie not to/from margin accounts after risk)
func (m *Market) collateralAndRisk(ctx context.Context, settle []events.Transfer) []events.Risk {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "collateralAndRisk")
	defer timer.EngineTimeCounterAdd()
	asset, _ := m.mkt.GetAsset()
	evts, response, err := m.collateral.MarkToMarket(ctx, m.GetID(), settle, asset)
	if err != nil {
		m.log.Error(
			"Failed to process mark to market settlement (collateral)",
			logging.Error(err),
		)
		return nil
	}
	// sending response to buffer
	if response != nil {
		m.broker.Send(events.NewTransferResponse(ctx, response))
	}

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdates := m.risk.UpdateMarginsOnSettlement(ctx, evts, m.markPrice)
	if len(riskUpdates) == 0 {
		return nil
	}

	return riskUpdates
}

func (m *Market) CancelAllOrders(ctx context.Context, partyID string) ([]*types.OrderCancellationConfirmation, error) {
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// get all order for this party in the book
	orders := m.matching.GetOrdersPerParty(partyID)

	// add all orders being eventually parked
	orders = append(orders, m.peggedOrders.GetAllForParty(partyID)...)

	// just an early exit, there's just no orders...
	if len(orders) <= 0 {
		return nil, nil
	}

	// now we eventually dedup them
	uniq := map[string]*types.Order{}
	for _, v := range orders {
		uniq[v.Id] = v
	}

	// put them back in the slice, and sort them
	orders = make([]*types.Order, 0, len(uniq))
	for _, v := range uniq {
		orders = append(orders, v)
	}
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].Id < orders[j].Id
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
		if m.liquidity.IsLiquidityOrder(partyID, order.Id) {
			continue
		}
		okOrders = append(okOrders, order)
	}

	cancellations := make([]*types.OrderCancellationConfirmation, 0, len(orders))

	// now iterate over all orders and cancel one by one.
	cancelledOrders := make([]*types.Order, 0, len(okOrders))
	for _, order := range okOrders {
		cancellation, err := m.cancelOrder(ctx, partyID, order.Id)
		if err != nil {
			return nil, err
		}
		cancellations = append(cancellations, cancellation)
		cancelledOrders = append(cancelledOrders, cancellation.Order)
	}

	m.checkForReferenceMoves(ctx, cancelledOrders, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return cancellations, nil
}

func (m *Market) CancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// cancelling and amending an order that is part of the LP commitment isn't allowed
	if m.liquidity.IsLiquidityOrder(partyID, orderID) {
		return nil, types.ErrEditNotAllowed
	}

	conf, err := m.cancelOrder(ctx, partyID, orderID)
	if err != nil {
		return conf, err
	}

	m.checkForReferenceMoves(ctx, []*types.Order{conf.Order}, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return conf, nil
}

// CancelOrder cancels the given order
func (m *Market) cancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {

	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
	}

	order, foundOnBook, err := m.getOrderByID(orderID)
	if err != nil {
		return nil, err
	}

	// Only allow the original order creator to cancel their order
	if order.PartyId != partyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Party ID mismatch",
				logging.String("party-id", partyID),
				logging.String("order-id", orderID),
				logging.String("market", m.mkt.Id))
		}
		return nil, types.ErrInvalidPartyID
	}

	defer m.releaseMarginExcess(ctx, partyID)

	if foundOnBook {
		cancellation, err := m.matching.CancelOrder(order)
		if cancellation == nil || err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failure after cancel order from matching engine",
					logging.String("party-id", partyID),
					logging.String("order-id", orderID),
					logging.String("market", m.mkt.Id),
					logging.Error(err))
			}
			return nil, err
		}
		_ = m.position.UnregisterOrder(order)
	}

	if order.IsExpireable() {
		m.expiringOrders.RemoveOrder(order.ExpiresAt, order.Id)
	}

	// If this is a pegged order, remove from pegged and parked lists
	if order.PeggedOrder != nil {
		m.removePeggedOrder(order)
		order.Status = types.Order_STATUS_CANCELLED
	}

	// Publish the changed order details
	order.UpdatedAt = m.currentTime.UnixNano()
	m.broker.Send(events.NewOrderEvent(ctx, order))

	return &types.OrderCancellationConfirmation{Order: order}, nil
}

// parkOrder removes the given order from the orderbook
// parkOrder will panic if it encounters errors, which means that it reached an
// invalid state.
func (m *Market) parkOrder(ctx context.Context, order *types.Order) {
	defer m.releaseMarginExcess(ctx, order.PartyId)

	if err := m.matching.RemoveOrder(order); err != nil {
		m.log.Panic("Failure to remove order from matching engine",
			logging.Order(*order),
			logging.Error(err))
	}

	m.peggedOrders.Park(order)
	m.broker.Send(events.NewOrderEvent(ctx, order))
	_ = m.position.UnregisterOrder(order)
}

// AmendOrder amend an existing order from the order book
func (m *Market) AmendOrder(ctx context.Context, orderAmendment *commandspb.OrderAmendment, party string) (*types.OrderConfirmation, error) {
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// explicitly/directly ordering an LP commitment order is not allowed
	if m.liquidity.IsLiquidityOrder(party, orderAmendment.OrderId) {
		return nil, types.ErrEditNotAllowed
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
	m.checkForReferenceMoves(ctx, allUpdatedOrders, false)
	m.checkLiquidity(ctx, nil)
	m.commandLiquidityAuction(ctx)

	return conf, nil
}

func (m *Market) amendOrder(
	ctx context.Context,
	orderAmendment *commandspb.OrderAmendment,
	party string,
) (cnf *types.OrderConfirmation, orderUpdates []*types.Order, returnedErr error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, nil, ErrMarketClosed
	}

	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, foundOnBook, err := m.getOrderByID(orderAmendment.OrderId)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid order ID",
				logging.OrderID(orderAmendment.GetOrderId()),
				logging.PartyID(party),
				logging.MarketID(orderAmendment.GetMarketId()),
				logging.Error(err))
		}
		return nil, nil, types.ErrInvalidOrderID
	}

	// We can only amend this order if we created it
	if existingOrder.PartyId != party {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid party ID",
				logging.String("original party id:", existingOrder.PartyId),
				logging.PartyID(party))
		}
		return nil, nil, types.ErrInvalidPartyID
	}

	// Validate Market
	if existingOrder.MarketId != m.mkt.Id {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.MarketID(m.mkt.Id),
				logging.Order(*existingOrder))
		}
		return nil, nil, types.ErrInvalidMarketID
	}

	if err := m.validateOrderAmendment(existingOrder, orderAmendment); err != nil {
		return nil, nil, err
	}

	amendedOrder, err := m.applyOrderAmendment(ctx, existingOrder, orderAmendment)
	if err != nil {
		return nil, nil, err
	}

	// If we have a pegged order that is no longer expiring, we need to remove it
	var (
		needToRemoveExpiry       = false
		needToAddExpiry          = false
		expiresAt          int64 = 0
	)
	defer func() {
		// no errors, amend most likely happened properly
		if returnedErr == nil {
			if needToRemoveExpiry {
				m.expiringOrders.RemoveOrder(expiresAt, existingOrder.Id)
			}
			if needToAddExpiry {
				m.expiringOrders.Insert(*amendedOrder)
			}
		}
	}()

	// We do this first, just in case the party would also have
	// change the expiry, and that would have been catched by
	// the follow up checks, so we do not insert a non-existing
	// order in the expiring orders
	// if remaining is reduces <= 0, then order is cancelled
	if amendedOrder.Remaining <= 0 {
		confirm, err := m.cancelOrder(
			ctx, existingOrder.PartyId, existingOrder.Id)
		if err != nil {
			return nil, nil, err
		}
		return &types.OrderConfirmation{
			Order: confirm.Order,
		}, nil, nil
	}

	// if we are amending from GTT to GTC, flag ready to remove from expiry list
	if existingOrder.IsExpireable() &&
		!amendedOrder.IsExpireable() {
		// We no longer need to handle the expiry
		needToRemoveExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if we are amending from GTC to GTT, flag ready to add to expiry list
	if !existingOrder.IsExpireable() &&
		amendedOrder.IsExpireable() {
		// We need to handle the expiry
		needToAddExpiry = true
	}

	// if both where expireable but we changed the duration
	// then we need to remove, then reinsert...
	if existingOrder.IsExpireable() &&
		amendedOrder.IsExpireable() &&
		existingOrder.ExpiresAt != amendedOrder.ExpiresAt {
		// We no longer need to handle the expiry
		needToRemoveExpiry = true
		needToAddExpiry = true
		expiresAt = existingOrder.ExpiresAt
	}

	// if expiration has changed and is before the original creation time, reject this amend
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < existingOrder.CreatedAt {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Amended expiry before original creation time",
				logging.Int64("original order created at ts:", existingOrder.CreatedAt),
				logging.Int64("amended expiry ts:", amendedOrder.ExpiresAt),
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
			m.expiringOrders.RemoveOrder(existingOrder.ExpiresAt, amendedOrder.Id)
		}

		// Update the existing message in place before we cancel it
		if foundOnBook {
			m.orderAmendInPlace(existingOrder, amendedOrder)
			cancellation, err := m.matching.CancelOrder(amendedOrder)
			if cancellation == nil || err != nil {
				m.log.Panic("Failure to cancel order from matching engine",
					logging.String("party-id", amendedOrder.PartyId),
					logging.String("order-id", amendedOrder.Id),
					logging.String("market", m.mkt.Id),
					logging.Error(err))
				return nil, nil, err
			}

			_ = m.position.UnregisterOrder(cancellation.Order)
			amendedOrder = cancellation.Order
		}

		// Update the order in our stores (will be marked as cancelled)
		// set the proper status
		amendedOrder.Status = types.Order_STATUS_EXPIRED
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))

		m.removePeggedOrder(amendedOrder)

		// m.checkForReferenceMoves(ctx, []*types.Order{}, false)

		return &types.OrderConfirmation{
			Order: amendedOrder,
		}, nil, nil
	}

	// TODO: This can be simplified by:
	// - amending the order in the peggedList first
	// - applying the changed based on auction / repricing
	if existingOrder.PeggedOrder != nil {

		// Amend in place during an auction
		if m.as.InAuction() {
			ret := m.orderAmendWhenParked(existingOrder, amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		}
		err := m.repricePeggedOrder(ctx, amendedOrder)
		if err != nil {
			// Failed to reprice so we have to park the order
			if amendedOrder.Status != types.Order_STATUS_PARKED {
				// If we are live then park
				m.parkOrder(ctx, existingOrder)
			}
			ret := m.orderAmendWhenParked(existingOrder, amendedOrder)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			return ret, nil, nil
		} else {
			// We got a new valid price, if we are parked we need to unpark
			if amendedOrder.Status == types.Order_STATUS_PARKED {
				orderConf, orderUpdts, err := m.submitValidatedOrder(ctx, amendedOrder)
				if err != nil {
					// If we cannot submit a new order then the amend has failed, return the error
					return nil, orderUpdts, err
				}
				// Update pegged order with new amended version
				m.peggedOrders.Amend(amendedOrder)
				return orderConf, orderUpdts, err
			}
		}
	}

	// from here these are the normal amendment
	var priceIncrease, priceShift, sizeIncrease, sizeDecrease, expiryChange, timeInForceChange bool

	if amendedOrder.Price != existingOrder.Price {
		priceShift = true
		priceIncrease = existingOrder.Price < amendedOrder.Price
	}

	if amendedOrder.Size > existingOrder.Size {
		sizeIncrease = true
	}
	if amendedOrder.Size < existingOrder.Size {
		sizeDecrease = true
	}

	if amendedOrder.ExpiresAt != existingOrder.ExpiresAt {
		expiryChange = true
	}

	if amendedOrder.TimeInForce != existingOrder.TimeInForce {
		timeInForceChange = true
	}

	// If nothing changed, amend in place to update updatedAt and version number
	if !priceShift && !sizeIncrease && !sizeDecrease && !expiryChange && !timeInForceChange {
		ret, err := m.orderAmendInPlace(existingOrder, amendedOrder)
		if err == nil {
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			// m.checkForReferenceMoves(ctx, []*types.Order{}, false)
		}
		return ret, nil, err
	}

	// Update potential new position after the amend
	pos := m.position.AmendOrder(existingOrder, amendedOrder)

	// Perform check and allocate margin if price or order size is increased
	// ignore rollback return here, as if we amend it means the order
	// is already on the book, not rollback will be needed, the margin
	// will be updated later on for sure.

	if priceIncrease || sizeIncrease {
		if err = m.checkMarginForOrder(ctx, pos, amendedOrder); err != nil {
			// Undo the position registering
			_ = m.position.AmendOrder(amendedOrder, existingOrder)

			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Unable to check/add margin for trader",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			return nil, nil, ErrMarginCheckFailed
		}
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		confirmation, err := m.orderCancelReplace(ctx, existingOrder, amendedOrder)
		var orders []*types.Order
		if err == nil {
			orders = m.handleConfirmation(ctx, confirmation)
			m.broker.Send(events.NewOrderEvent(ctx, confirmation.Order))
		}
		return confirmation, orders, err
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease || timeInForceChange {
		if sizeDecrease && amendedOrder.Remaining >= existingOrder.Remaining {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Order amendment not allowed when reducing to a larger amount", logging.Order(*existingOrder))
			}
			return nil, nil, ErrInvalidAmendRemainQuantity
		}
		ret, err := m.orderAmendInPlace(existingOrder, amendedOrder)
		if err == nil {
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			// m.checkForReferenceMoves(ctx, []*types.Order{}, false)
		}
		return ret, nil, err
	}

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Order amendment not allowed", logging.Order(*existingOrder))
	}
	return nil, nil, types.ErrEditNotAllowed
}

func (m *Market) validateOrderAmendment(
	order *types.Order,
	amendment *commandspb.OrderAmendment,
) error {
	// check TIME_IN_FORCE and expiry
	if amendment.TimeInForce == types.Order_TIME_IN_FORCE_GTT {
		if amendment.ExpiresAt == nil {
			return types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT
		}
		// if expiresAt is before or equal to created at
		// we return an error
		if amendment.ExpiresAt.Value <= order.CreatedAt {
			return types.OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT
		}
	}

	if amendment.TimeInForce == types.Order_TIME_IN_FORCE_GTC {
		// this is cool, but we need to ensure and expiry is not set
		if amendment.ExpiresAt != nil {
			return types.OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT
		}
	}

	if amendment.TimeInForce == types.Order_TIME_IN_FORCE_FOK ||
		amendment.TimeInForce == types.Order_TIME_IN_FORCE_IOC {
		// IOC and FOK are not acceptable for amend order
		return types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC
	}

	if (amendment.TimeInForce == types.Order_TIME_IN_FORCE_GFN ||
		amendment.TimeInForce == types.Order_TIME_IN_FORCE_GFA) &&
		amendment.TimeInForce != order.TimeInForce {
		// We cannot amend to a GFA/GFN orders
		return types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN
	}

	if (order.TimeInForce == types.Order_TIME_IN_FORCE_GFN ||
		order.TimeInForce == types.Order_TIME_IN_FORCE_GFA) &&
		(amendment.TimeInForce != order.TimeInForce &&
			amendment.TimeInForce != types.Order_TIME_IN_FORCE_UNSPECIFIED) {
		// We cannot amend from a GFA/GFN orders
		return types.OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN
	}

	if order.PeggedOrder == nil {
		// We cannot change a pegged orders details on a non pegged order
		if amendment.PeggedOffset != nil ||
			amendment.PeggedReference != types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
			return types.OrderError_ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER
		}
	} else if order.PeggedOrder != nil {
		// We cannot change the price on a pegged order
		if amendment.Price != nil {
			return types.OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER
		}
	}
	return nil
}

// this function assume the amendment have been validated before
func (m *Market) applyOrderAmendment(
	ctx context.Context,
	existingOrder *types.Order,
	amendment *commandspb.OrderAmendment,
) (order *types.Order, err error) {
	m.mu.Lock()
	currentTime := m.currentTime
	m.mu.Unlock()

	// initialize order with the existing order data
	order = &types.Order{
		Type:        existingOrder.Type,
		Id:          existingOrder.Id,
		MarketId:    existingOrder.MarketId,
		PartyId:     existingOrder.PartyId,
		Side:        existingOrder.Side,
		Price:       existingOrder.Price,
		Size:        existingOrder.Size,
		Remaining:   existingOrder.Remaining,
		TimeInForce: existingOrder.TimeInForce,
		CreatedAt:   existingOrder.CreatedAt,
		Status:      existingOrder.Status,
		ExpiresAt:   existingOrder.ExpiresAt,
		Reference:   existingOrder.Reference,
		Version:     existingOrder.Version + 1,
		UpdatedAt:   currentTime.UnixNano(),
	}
	if existingOrder.PeggedOrder != nil {
		order.PeggedOrder = &types.PeggedOrder{
			Reference: existingOrder.PeggedOrder.Reference,
			Offset:    existingOrder.PeggedOrder.Offset,
		}
	}

	// apply price changes
	if amendment.Price != nil && existingOrder.Price != amendment.Price.Value {
		order.Price = amendment.Price.Value
	}

	// apply size changes
	if amendment.SizeDelta != 0 {
		order.Size += uint64(amendment.SizeDelta)
		newRemaining := int64(existingOrder.Remaining) + amendment.SizeDelta
		if newRemaining <= 0 {
			newRemaining = 0
		}
		order.Remaining = uint64(newRemaining)
	}

	// apply tif
	if amendment.TimeInForce != types.Order_TIME_IN_FORCE_UNSPECIFIED {
		order.TimeInForce = amendment.TimeInForce
		if amendment.TimeInForce != types.Order_TIME_IN_FORCE_GTT {
			order.ExpiresAt = 0
		}
	}
	if amendment.ExpiresAt != nil {
		order.ExpiresAt = amendment.ExpiresAt.Value
	}

	// apply pegged order values
	if order.PeggedOrder != nil {
		if amendment.PeggedOffset != nil {
			order.PeggedOrder.Offset = amendment.PeggedOffset.Value
		}

		if amendment.PeggedReference != types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
			order.PeggedOrder.Reference = amendment.PeggedReference
		}
		if verr := m.validatePeggedOrder(ctx, order); verr != types.OrderError_ORDER_ERROR_UNSPECIFIED {
			err = verr
		}
	}
	return
}

func (m *Market) orderCancelReplace(ctx context.Context, existingOrder, newOrder *types.Order) (conf *types.OrderConfirmation, err error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "orderCancelReplace")

	cancellation, err := m.matching.CancelOrder(existingOrder)
	if cancellation == nil {
		if err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Panic("Failed to cancel order from matching engine during CancelReplace",
					logging.OrderWithTag(*existingOrder, "existing-order"),
					logging.OrderWithTag(*newOrder, "new-order"),
					logging.Error(err))
			}
		} else {
			err = fmt.Errorf("order cancellation failed (no error given)")
		}
	} else {
		// first we call the order book to evaluate auction triggers and get the list of trades
		trades, err := m.checkPriceAndGetTrades(ctx, newOrder)
		if err != nil {
			return nil, m.unregisterAndReject(ctx, newOrder, err)
		}

		// try to apply fees on the trade
		if err := m.applyFees(ctx, newOrder, trades); err != nil {
			return nil, m.unregisterAndReject(ctx, newOrder, err)
		}

		// Because other collections might be pointing at the original order
		// use it's memory when inserting the new version
		*existingOrder = *newOrder
		conf, err = m.matching.SubmitOrder(existingOrder)
		if err != nil {
			m.log.Panic("unable to submit order", logging.Error(err))
		}
		// replace the trades in the confirmation to have
		// the ones with the fees embedded
		conf.Trades = trades
	}

	timer.EngineTimeCounterAdd()
	return
}

func (m *Market) orderAmendInPlace(originalOrder, amendOrder *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "orderAmendInPlace")
	defer timer.EngineTimeCounterAdd()

	err := m.matching.AmendOrder(originalOrder, amendOrder)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure after amend order from matching engine (amend-in-place)",
				logging.OrderWithTag(*amendOrder, "new-order"),
				logging.Error(err))
		}
		return nil, err
	}
	return &types.OrderConfirmation{
		Order: amendOrder,
	}, nil
}

func (m *Market) orderAmendWhenParked(originalOrder, amendOrder *types.Order) *types.OrderConfirmation {
	amendOrder.Status = types.Order_STATUS_PARKED
	amendOrder.Price = 0
	*originalOrder = *amendOrder

	return &types.OrderConfirmation{
		Order: amendOrder,
	}
}

// RemoveExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked
func (m *Market) RemoveExpiredOrders(
	ctx context.Context, timestamp int64) ([]types.Order, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "RemoveExpiredOrders")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
	}

	expired := []types.Order{}
	for _, order := range m.expiringOrders.Expire(timestamp) {
		// The pegged expiry orders are copies and do not reflect the
		// current state of the order, therefore we look it up
		originalOrder, foundOnBook, err := m.getOrderByID(order.Id)
		if err == nil {
			// assign to the order the order from the book
			// so we get the most recent version from the book
			// to continue with
			order = *originalOrder

			// if the order was on the book basically
			// either a pegged + non parked
			// or a non-pegged order
			if foundOnBook {
				m.position.UnregisterOrder(&order)
				m.matching.DeleteOrder(&order)
			}
		}

		// if this was a pegged order
		// remove from the pegged / parked list
		if order.PeggedOrder != nil {
			m.removePeggedOrder(&order)
		}

		// now we add to the list of expired orders
		// and assign the appropriate status
		order.UpdatedAt = m.currentTime.UnixNano()
		order.Status = types.Order_STATUS_EXPIRED
		expired = append(expired, order)
	}

	// If we have removed an expired order, do we need to reprice any
	// or maybe notify the liquidity engine
	if len(expired) > 0 {
		expiredPtrs := make([]*types.Order, len(expired))
		for i := range expired {
			expiredPtrs[i] = &expired[i]
		}

		m.checkForReferenceMoves(ctx, expiredPtrs, false)
		m.checkLiquidity(ctx, nil)
		m.commandLiquidityAuction(ctx)
	}

	return expired, nil
}

func (m *Market) getBestStaticAskPrice() (uint64, error) {
	return m.matching.GetBestStaticAskPrice()
}

func (m *Market) getBestStaticAskPriceAndVolume() (uint64, uint64, error) {
	return m.matching.GetBestStaticAskPriceAndVolume()
}

func (m *Market) getBestStaticBidPrice() (uint64, error) {
	return m.matching.GetBestStaticBidPrice()
}

func (m *Market) getBestStaticBidPriceAndVolume() (uint64, uint64, error) {
	return m.matching.GetBestStaticBidPriceAndVolume()
}

func (m *Market) getBestStaticPrices() (bid, ask uint64, err error) {
	bid, err = m.getBestStaticBidPrice()
	if err != nil {
		return
	}
	ask, err = m.getBestStaticAskPrice()
	return
}

func (m *Market) getStaticMidPrice(side types.Side) (uint64, error) {
	bid, err := m.matching.GetBestStaticBidPrice()
	if err != nil {
		return 0, err
	}
	ask, err := m.matching.GetBestStaticAskPrice()
	if err != nil {
		return 0, err
	}
	var mid uint64
	if side == types.Side_SIDE_BUY {
		mid = (bid + ask + 1) / 2
	} else {
		mid = (bid + ask) / 2
	}

	return mid, nil
}

// removePeggedOrder looks through the pegged and parked list
// and removes the matching order if found
func (m *Market) removePeggedOrder(order *types.Order) {
	// remove if order was expiring
	m.expiringOrders.RemoveOrder(order.ExpiresAt, order.Id)
	m.peggedOrders.Remove(order)
}

// getOrderBy looks for the order in the order book and in the list
// of pegged orders in the market. Returns the order if found, a bool
// representing if the order was found on the order book and any error code
func (m *Market) getOrderByID(orderID string) (*types.Order, bool, error) {
	order, err := m.matching.GetOrderByID(orderID)
	if err == nil {
		return order, true, nil
	}

	// The pegged order list contains all the pegged orders in the system
	// whether they are parked or live. Check this list of a matching order
	if o := m.peggedOrders.GetByID(orderID); o != nil {
		return o, false, nil
	}

	// We couldn't find it
	return nil, false, ErrOrderNotFound
}

// create an actual risk model, and calculate the risk factors
// if something goes wrong, return the hard-coded values of old
func getInitialFactors(log *logging.Logger, mkt *types.Market, asset string) *types.RiskResult {
	rm, err := risk.NewModel(log, mkt.TradableInstrument.RiskModel, asset)
	// @TODO log this error
	if err != nil {
		return nil
	}
	if ok, fact := rm.CalculateRiskFactors(nil); ok {
		return fact
	}
	// default to hard-coded risk factors
	return &types.RiskResult{
		RiskFactors: map[string]*types.RiskFactor{
			asset: {Long: 0.15, Short: 0.25},
		},
		PredictedNextRiskFactors: map[string]*types.RiskFactor{
			asset: {Long: 0.15, Short: 0.25},
		},
	}
}

func (m *Market) getRiskFactors() (*types.RiskFactor, error) {
	a, err := m.mkt.GetAsset()
	if err != nil {
		return nil, err
	}
	rf, err := m.risk.GetRiskFactors(a)
	if err != nil {
		return nil, err
	}
	return rf, nil
}

func (m *Market) getTargetStake() float64 {
	rf, err := m.getRiskFactors()
	if err != nil {
		logging.Error(err)
		m.log.Debug("unable to get risk factors, can't calculate target")
		return 0
	}
	return m.tsCalc.GetTargetStake(*rf, m.currentTime, m.markPrice)
}

func (m *Market) getSuppliedStake() uint64 {
	return m.liquidity.CalculateSuppliedStake()
}

func (m *Market) OnMarketMinProbabilityOfTradingLPOrdersUpdate(_ context.Context, f float64) {
	m.liquidity.OnMinProbabilityOfTradingLPOrdersUpdate(f)
}

func (m *Market) BondPenaltyFactorUpdate(ctx context.Context, v float64) {
	m.bondPenaltyFactor = v
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
	if err := m.fee.OnFeeFactorsMakerFeeUpdate(ctx, f); err != nil {
		return err
	}
	m.mkt.Fees.Factors.MakerFee = fmt.Sprintf("%f", f)
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) OnFeeFactorsInfrastructureFeeUpdate(ctx context.Context, f float64) error {
	if err := m.fee.OnFeeFactorsInfrastructureFeeUpdate(ctx, f); err != nil {
		return err
	}
	m.mkt.Fees.Factors.InfrastructureFee = fmt.Sprintf("%f", f)
	m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

	return nil
}

func (m *Market) OnSuppliedStakeToObligationFactorUpdate(v float64) {
	m.liquidity.OnSuppliedStakeToObligationFactorUpdate(v)
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
	return m.tsCalc.UpdateScalingFactor(v)
}

func (m *Market) OnMarketLiquidityProvisionShapesMaxSizeUpdate(v int64) error {
	return m.liquidity.OnMarketLiquidityProvisionShapesMaxSizeUpdate(v)
}

func (m *Market) OnMarketLiquidityMaximumLiquidityFeeFactorLevelUpdate(v float64) {
	m.liquidity.OnMaximumLiquidityFeeFactorLevelUpdate(v)
}

func (m *Market) OnMarketProbabilityOfTradingTauScalingUpdate(_ context.Context, v float64) {
	m.liquidity.OnProbabilityOfTradingTauScalingUpdate(v)
}

func (m *Market) OnMarketLiquidityTargetStakeTriggeringRatio(ctx context.Context, v float64) {
	m.lMonitor.UpdateTargetStakeTriggerRatio(ctx, v)
	//TODO: Send an event containing updated parameter
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

func (m *Market) checkLiquidity(ctx context.Context, trades []*types.Trade) {
	// before we check liquidity, ensure we've moved all funds that can go towards
	// provided stake to the bond accounts so we don't trigger liquidity auction for no reason
	m.checkBondBalance(ctx)
	_, vBid, _ := m.getBestStaticBidPriceAndVolume()
	_, vAsk, _ := m.getBestStaticAskPriceAndVolume()

	rf, err := m.getRiskFactors()
	if err != nil {
		m.log.Panic("unable to get risk factors, can't check liquidity",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
	}

	m.lMonitor.CheckLiquidity(
		m.as, m.currentTime,
		float64(m.getSuppliedStake()),
		trades,
		*rf,
		m.markPrice,
		vBid, vAsk)
	if evt := m.as.AuctionExtended(ctx); evt != nil {
		m.broker.Send(evt)
	}
}

// command liquidity auction checks if liquidity auction should be entered and if it can end
func (m *Market) commandLiquidityAuction(ctx context.Context) {
	// start the liquidity monitoring auction if required
	if !m.as.InAuction() && m.as.AuctionStart() {
		m.EnterAuction(ctx)
	}
	// end the liquidity monitoring auction if possible
	if m.as.InAuction() && m.as.CanLeave() && !m.as.IsOpeningAuction() {
		p, v, _ := m.matching.GetIndicativePriceAndVolume()
		if err := m.pMonitor.CheckPrice(ctx, m.as, p, v, m.currentTime, true); err != nil {
			m.log.Panic("unable to run check price with price monitor",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		// TODO: Need to also get indicative trades and check how they'd impact target stake,
		// see  https://github.com/vegaprotocol/vega/issues/3047
		// If price monitoring doesn't trigger auction than leave it
		if evt := m.as.AuctionExtended(ctx); evt != nil {
			m.broker.Send(evt)
		}
	}
}

func (m *Market) canTrade() bool {
	return m.mkt.State == types.Market_STATE_ACTIVE ||
		m.mkt.State == types.Market_STATE_PENDING ||
		m.mkt.State == types.Market_STATE_SUSPENDED
}

func (m *Market) canSubmitCommitment() bool {
	return m.canTrade() || m.mkt.State == types.Market_STATE_PROPOSED
}

// cleanupOnReject remove all resources created while the
// market was on PREPARED state.
// we'll need to remove all accounts related to the market
// all margin accounts for this market
// all bond accounts for this market too.
// at this point no fees would have been collected or anything
// like this.
func (m *Market) cleanupOnReject(ctx context.Context) {
	// get the list of all parties in this market
	parties := make([]string, 0, len(m.parties))
	for k := range m.parties {
		parties = append(parties, k)
	}

	err := m.stopAllLiquidityProvisionOnReject(ctx)
	if err != nil {
		m.log.Debug("could not stop all liquidity provision on market rejection",
			logging.MarketID(m.GetID()),
			logging.Error(err))
	}

	asset, _ := m.mkt.GetAsset()
	tresps, err := m.collateral.ClearMarket(ctx, m.GetID(), asset, parties)
	if err != nil {
		m.log.Panic("unable to cleanup a rejected market",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
		return
	}

	// then send the responses
	m.broker.Send(events.NewTransferResponse(ctx, tresps))
}

func (m *Market) stopAllLiquidityProvisionOnReject(ctx context.Context) error {
	for party := range m.liquidity.ProvisionsPerParty() {
		// here we ignore  the list of orders that could have been
		// created with this party liquidity provision. At this point
		// if we are calling this function, the market is in a PENDING
		// state, which means that liquidity provision can be submitted
		// but orders would never be able to be deployed, so it's safe
		// to ignorethe second return as it shall be an empty slice.
		_, err := m.liquidity.StopLiquidityProvision(ctx, party)
		if err != nil {
			return err
		}
	}

	return nil
}

func lpsToLiquidityProviderFeeShare(lps map[string]*lp) []*types.LiquidityProviderFeeShare {
	out := make([]*types.LiquidityProviderFeeShare, 0, len(lps))
	for k, v := range lps {
		out = append(out, &types.LiquidityProviderFeeShare{
			Party:                 k,
			EquityLikeShare:       v.share.String(),
			AverageEntryValuation: v.avg.String(),
		})
	}

	// sort then so we produce the same output on all nodes
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].Party < out[j].Party
	})

	return out
}

func (m *Market) distributeLiquidityFees(ctx context.Context) error {
	asset, err := m.mkt.GetAsset()
	if err != nil {
		return fmt.Errorf("failed to get asset: %w", err)
	}

	acc, err := m.collateral.GetMarketLiquidityFeeAccount(m.mkt.GetId(), asset)
	if err != nil {
		return fmt.Errorf("failed to get market liquidity fee account: %w", err)
	}

	// We can't distribute any share when no balance.
	if acc.Balance == 0 {
		return nil
	}

	shares := m.equityShares.Shares(m.liquidity.GetInactiveParties())
	if len(shares) == 0 {
		return nil
	}

	feeTransfer := m.fee.BuildLiquidityFeeDistributionTransfer(shares, acc)
	if feeTransfer == nil {
		return nil
	}

	resp, err := m.collateral.TransferFees(ctx, m.GetID(), asset, feeTransfer)
	if err != nil {
		return fmt.Errorf("failed to transfer fees: %w", err)
	}

	m.broker.Send(events.NewTransferResponse(ctx, resp))
	return nil
}
