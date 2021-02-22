package execution

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
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
	"code.vegaprotocol.io/vega/monitor/price"
	"code.vegaprotocol.io/vega/positions"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/risk"
	"code.vegaprotocol.io/vega/settlement"

	"github.com/golang/protobuf/proto"
	"github.com/pkg/errors"
)

// InitialOrderVersion is set on `Version` field for every new order submission read from the network
const InitialOrderVersion = 1

// PriceMoveMid used to indicate that the mid price has moved
const PriceMoveMid = 1

// PriceMoveBestBid used to indicate that the best bid price has moved
const PriceMoveBestBid = 2

// PriceMoveBestAsk used to indicate that the best ask price has moved
const PriceMoveBestAsk = 4

// PriceMoveAll used to indicate everything has moved
const PriceMoveAll = PriceMoveMid + PriceMoveBestBid + PriceMoveBestAsk

var (
	// ErrMarketClosed signals that an action have been tried to be applied on a closed market
	ErrMarketClosed = errors.New("market closed")
	// ErrTraderDoNotExists signals that the trader used does not exists
	ErrTraderDoNotExists = errors.New("trader does not exist")
	// ErrMarginCheckFailed signals that a margin check for a position failed
	ErrMarginCheckFailed = errors.New("margin check failed")
	// ErrMarginCheckInsufficient signals that a margin had not enough funds
	ErrMarginCheckInsufficient = errors.New("insufficient margin")
	// ErrInvalidInitialMarkPrice signals that the initial mark price for a market is invalid
	ErrInvalidInitialMarkPrice = errors.New("invalid initial mark price (mkprice <= 0)")
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
	// ErrInvalidMarketType is returned if the order is not valid for the current market type (auction/continuous)
	ErrInvalidMarketType = errors.New("invalid market type")
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
	// ErrTradingNotAllowed no trading related functionnalities are allowed in the current state
	ErrTradingNotAllowed = errors.New("trading not allowed")
	// ErrCommitmentSubmissionNotAllowed no commitment submission are permitted in the current state
	ErrCommitmentSubmissionNotAllowed = errors.New("commitment submission not allowed")
	// ErrNotEnoughStake is returned when a LP update results in not enough commitment
	ErrNotEnoughStake = errors.New("commitment submission rejected, not enouth stake")

	// ErrCannotRejectMarketNotInProposedState
	ErrCannotRejectMarketNotInProposedState = errors.New("cannot reject a market not in proposed state")
	// ErrCannotStateOpeningAuctionForMarketNotInProposedState
	ErrCannotStartOpeningAuctionForMarketNotInProposedState = errors.New("cannot start the opening auction for a market not in proposed state")

	networkPartyID = "network"
)

// PriceMonitor interface to handle price monitoring/auction triggers
// @TODO the interface shouldn't be imported here
type PriceMonitor interface {
	CheckPrice(ctx context.Context, as price.AuctionState, p uint64, v uint64, now time.Time) error
	GetCurrentBounds() []*types.PriceMonitoringBounds
}

// TargetStakeCalculator interface
type TargetStakeCalculator interface {
	RecordOpenInterest(oi uint64, now time.Time) error
	GetTargetStake(rf types.RiskFactor, now time.Time, markPrice uint64) float64
	UpdateScalingFactor(sFactor float64) error
	UpdateTimeWindow(tWindow time.Duration)
}

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
	AuctionEnd() bool
	// when does the auction start/end
	ExpiresAt() *time.Time
	Start() time.Time
	// signal we've started/ended the auction
	AuctionStarted(ctx context.Context) *events.Auction
	AuctionEnded(ctx context.Context, now time.Time) *events.Auction
	// get some data
	Mode() types.Market_TradingMode
	Trigger() types.AuctionTrigger
}

// Market represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transctiona
type Market struct {
	log   *logging.Logger
	idgen *IDgenerator

	mkt         *types.Market
	closingAt   time.Time
	currentTime time.Time

	mu sync.Mutex

	markPrice uint64

	// own engines
	matching           *matching.OrderBook
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

	tsCalc TargetStakeCalculator

	as *monitor.AuctionState // @TODO this should be an interface

	// A collection of time sorted pegged orders
	peggedOrders   []*types.Order
	expiringOrders *ExpiringOrders

	// Store the previous price values so we can see what has changed
	lastBestBidPrice uint64
	lastBestAskPrice uint64
	lastMidBuyPrice  uint64
	lastMidSellPrice uint64

	lastMarketValueProxy    float64
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
	collateralEngine *collateral.Engine,
	mkt *types.Market,
	now time.Time,
	broker Broker,
	idgen *IDgenerator,
	as *monitor.AuctionState,
) (*Market, error) {

	if len(mkt.Id) == 0 {
		return nil, ErrEmptyMarketID
	}

	tradableInstrument, err := markets.NewTradableInstrument(log, mkt.TradableInstrument)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate a new market")
	}

	if tradableInstrument.Instrument.InitialMarkPrice == 0 {
		return nil, ErrInvalidInitialMarkPrice
	}

	closingAt, err := tradableInstrument.Instrument.GetMarketClosingTime()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get market closing time")
	}

	// @TODO -> the raw auctionstate shouldn't be something exposed to the matching engine
	// as far as matching goes: it's either an auction or not
	book := matching.NewOrderBook(log, matchingConfig, mkt.Id,
		tradableInstrument.Instrument.InitialMarkPrice, as.InAuction())
	asset := tradableInstrument.Instrument.Product.GetAsset()
	riskEngine := risk.NewEngine(
		log,
		riskConfig,
		tradableInstrument.MarginCalculator,
		tradableInstrument.RiskModel,
		getInitialFactors(log, mkt, asset),
		book,
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
		return nil, errors.Wrap(err, "unable to instantiate fee engine")
	}

	pMonitor, err := price.NewMonitor(tradableInstrument.RiskModel, *mkt.PriceMonitoringSettings)
	if err != nil {
		return nil, errors.Wrap(err, "unable to instantiate price monitoring engine")
	}

	tsCalc := liquiditytarget.NewEngine(*mkt.TargetStakeParameters)
	liqEngine := liquidity.NewEngine(log, broker, idgen, tradableInstrument.RiskModel, pMonitor, mkt.Id)

	// The market is initially create in a proposed state
	mkt.State = types.Market_STATE_PROPOSED
	mkt.TradingMode = types.Market_TRADING_MODE_CONTINUOUS

	market := &Market{
		log:                log,
		idgen:              idgen,
		mkt:                mkt,
		closingAt:          closingAt,
		currentTime:        now,
		markPrice:          tradableInstrument.Instrument.InitialMarkPrice,
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
		tsCalc:             tsCalc,
		expiringOrders:     NewExpiringOrders(),
		feeSplitter:        &FeeSplitter{},
		equityShares:       NewEquityShares(1),
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
		Market:                m.GetID(),
		BestBidPrice:          bestBidPrice,
		BestBidVolume:         bestBidVolume,
		BestOfferPrice:        bestOfferPrice,
		BestOfferVolume:       bestOfferVolume,
		BestStaticBidPrice:    bestStaticBidPrice,
		BestStaticBidVolume:   bestStaticBidVolume,
		BestStaticOfferPrice:  bestStaticOfferPrice,
		BestStaticOfferVolume: bestStaticOfferVolume,
		MidPrice:              midPrice,
		StaticMidPrice:        staticMidPrice,
		MarkPrice:             m.markPrice,
		Timestamp:             m.currentTime.UnixNano(),
		OpenInterest:          m.position.GetOpenInterest(),
		IndicativePrice:       indicativePrice,
		IndicativeVolume:      indicativeVolume,
		AuctionStart:          auctionStart,
		AuctionEnd:            auctionEnd,
		MarketTradingMode:     m.as.Mode(),
		Trigger:               m.as.Trigger(),
		TargetStake:           strconv.FormatFloat(m.getTargetStake(), 'f', -1, 64),
		SuppliedStake:         strconv.FormatUint(m.getSuppliedStake(), 10),
		PriceMonitoringBounds: m.pMonitor.GetCurrentBounds(),
		MarketValueProxy:      strconv.FormatFloat(m.lastMarketValueProxy, 'f', -1, 64),
		// TODO(): set this with actual value when implemented.
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
	m.liquidity.OnChainTimeUpdate(ctx, t)
	m.risk.OnTimeUpdate(t)
	m.settlement.OnTick(t)

	// TODO(): This also assume that the market is not
	// being closed before the market is leaving
	// the opening auction, but settlement at expiry is
	// not even specd or implemented as of now...
	// if the state of the market is just PROPOSED,
	// we will just skip everything there as nothing apply.
	if m.mkt.State == types.Market_STATE_PROPOSED {
		return false
	}

	// distribute liquidity fees each `m.lpFeeDistributionTimeStep`
	if t.Sub(m.lastEquityShareDistributed) > m.lpFeeDistributionTimeStep {
		m.lastEquityShareDistributed = t

		if err := m.distributeLiquidityFees(ctx); err != nil {
			m.log.Panic("Distributing Liquidity Fees", logging.Error(err))
		}
	}

	closed = t.After(m.closingAt)
	m.closed = closed

	// check price auction end
	if m.as.InAuction() {
		p, v, _ := m.matching.GetIndicativePriceAndVolume()
		if m.as.IsOpeningAuction() {
			if endTS := m.as.ExpiresAt(); endTS != nil && endTS.Before(t) {
				// mark opening auction as ending
				// Prime price monitoring engine with the uncrossing price of the opening auction
				if err := m.pMonitor.CheckPrice(ctx, m.as, p, v, t); err != nil {
					m.log.Error("Price monitoring error", logging.Error(err))
				}
				m.as.EndAuction()
				m.LeaveAuction(ctx, t)

				// the market is now in a ACTIVE state
				m.mkt.State = types.Market_STATE_ACTIVE
				m.broker.Send(events.NewMarketUpdatedEvent(ctx, *m.mkt))

				// start the market fee window
				m.feeSplitter.TimeWindowStart(t)
			}
		} else if m.as.IsPriceAuction() {
			if err := m.pMonitor.CheckPrice(ctx, m.as, p, v, t); err != nil {
				m.log.Error("Price monitoring error", logging.Error(err))
				// @TODO handle or panic? (panic is last resort)
			}
			// price monitoring engine indicated auction can end
			if m.as.AuctionEnd() {
				m.LeaveAuction(ctx, t)
			}
		}
		// This is where ending liquidity auctions and FBA's will be handled
	}

	// TODO(): handle market start time

	m.risk.CalculateFactors(ctx, t)

	timer.EngineTimeCounterAdd()

	if !closed {
		m.broker.Send(events.NewMarketTick(ctx, m.mkt.Id, t))
		return
	}
	// market is closed, final settlement
	// call settlement and stuff
	positions, err := m.settlement.Settle(t, m.markPrice)
	if err != nil {
		m.log.Error(
			"Failed to get settle positions on market close",
			logging.Error(err),
		)
	} else {
		transfers, err := m.collateral.FinalSettlement(ctx, m.GetID(), positions)
		if err != nil {
			m.log.Error(
				"Failed to get ledger movements after settling closed market",
				logging.String("market-id", m.GetID()),
				logging.Error(err),
			)
		} else {
			// @TODO pass in correct context -> Previous or next block? Which is most appropriate here?
			// this will be next block
			evt := events.NewTransferResponse(ctx, transfers)
			m.broker.Send(evt)

			asset, _ := m.mkt.GetAsset()
			parties := make([]string, 0, len(m.parties))
			for k := range m.parties {
				parties = append(parties, k)
			}

			clearMarketTransfers, err := m.collateral.ClearMarket(ctx, m.GetID(), asset, parties)
			if err != nil {
				m.log.Error("Clear market error",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			} else {
				evt := events.NewTransferResponse(ctx, clearMarketTransfers)
				m.broker.Send(evt)
			}
		}
	}

	if mvwl := m.marketValueWindowLength; m.feeSplitter.Elapsed() > mvwl {
		ts := m.liquidity.ProvisionsPerParty().TotalStake()
		m.lastMarketValueProxy = m.feeSplitter.MarketValueProxy(mvwl, float64(ts))
		m.equityShares.WithMVP(m.lastMarketValueProxy)

		m.feeSplitter.TimeWindowStart(t)
	}

	return
}

func (m *Market) unregisterAndReject(ctx context.Context, order *types.Order, err error) error {
	_, perr := m.position.UnregisterOrder(order)
	if perr != nil {
		m.log.Error("Unable to unregister potential trader positions",
			logging.String("market-id", m.GetID()),
			logging.Error(err))
	}
	order.UpdatedAt = m.currentTime.UnixNano()
	order.Status = types.Order_STATUS_REJECTED
	if oerr, ok := types.IsOrderError(err); ok {
		order.Reason = oerr
	} else {
		// should not happend but still...
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

func HasReferenceMoved(order *types.Order, changes uint8) bool {
	if (order.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_MID &&
		changes&PriceMoveMid > 0) ||
		(order.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_BEST_BID &&
			changes&PriceMoveBestBid > 0) ||
		(order.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_BEST_ASK &&
			changes&PriceMoveBestAsk > 0) {
		return true
	}
	return false
}

// repriceAllPeggedOrders runs through the slice of pegged orders and reprices all those
// which are using a reference that has moved. Returns the number of orders that were repriced.
func (m *Market) repriceAllPeggedOrders(ctx context.Context, changes uint8) uint64 {
	var (
		repriceCount uint64
		toRemove     []*types.Order
	)

	// Go through all the pegged orders and remove from the order book
	for _, order := range m.peggedOrders {
		if HasReferenceMoved(order, changes) {
			if order.Status != types.Order_STATUS_PARKED {
				// Remove order if any volume remains, otherwise it's already been popped by the matching engine.

				cancellation, err := m.matching.CancelOrder(order)
				if cancellation == nil || err != nil {
					m.log.Panic("Failure after cancel order from matching engine",
						logging.Order(*order),
						logging.Error(err))
				}

				// Remove it from the trader position
				if _, err := m.position.UnregisterOrder(order); err != nil {
					m.log.Panic("Failure unregistering order in positions engine (cancel)",
						logging.Order(*order),
						logging.Error(err))
				}
			}
		}
	}

	// Reprice all the pegged order
	for _, order := range m.peggedOrders {
		if HasReferenceMoved(order, changes) {
			if price, err := m.getNewPeggedPrice(order); err != nil {
				// Failed to reprice, if we are parked we do nothing, if not parked we need to park
				if order.Status != types.Order_STATUS_PARKED {
					order.UpdatedAt = m.currentTime.UnixNano()
					order.Status = types.Order_STATUS_PARKED
					order.Price = 0
					m.broker.Send(events.NewOrderEvent(ctx, order))
				}
			} else {
				// Repriced so all good make sure status is correct
				order.Status = types.Order_STATUS_CANCELLED
				order.Price = price
			}
		}
	}

	// Reinsert all the orders
	for _, order := range m.peggedOrders {
		if HasReferenceMoved(order, changes) {
			if order.Status == types.Order_STATUS_CANCELLED {
				// try to submit the order
				if _, err := m.submitValidatedOrder(ctx, order); err != nil {
					// order could not be submitted, it's then been rejected
					// we just completely remove it.
					toRemove = append(toRemove, order)
					continue
				}
			}
		}
	}

	for _, o := range toRemove {
		m.removePeggedOrder(o)
	}

	return repriceCount
}

func (m *Market) getNewPeggedPrice(order *types.Order) (uint64, error) {
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

// EnterAuction : Prepare the order book to be run as an auction
func (m *Market) EnterAuction(ctx context.Context) {
	// Change market type to auction
	ordersToCancel, err := m.matching.EnterAuction()
	if err != nil {
		m.log.Error("Error entering auction: ", logging.Error(err))
	}

	// Move into auction mode to prevent pegged order repricing
	event := m.as.AuctionStarted(ctx)

	// Park all pegged orders
	for _, order := range m.peggedOrders {
		if order.Status != types.Order_STATUS_PARKED {
			m.parkOrder(ctx, order)
		}
	}

	// Cancel all the orders that were invalid
	for _, order := range ordersToCancel {
		m.cancelOrder(ctx, order.PartyId, order.Id)
	}

	// Send an event bus update
	m.broker.Send(event)
}

// LeaveAuction : Return the orderbook and market to continuous trading
func (m *Market) LeaveAuction(ctx context.Context, now time.Time) {
	// If we were an opening auction, clear it
	if m.as.IsOpeningAuction() {
		m.mkt.OpeningAuction = nil
	}

	// Change market type to continuous trading
	uncrossedOrders, ordersToCancel, err := m.matching.LeaveAuction(m.currentTime)
	if err != nil {
		m.log.Error("Error leaving auction", logging.Error(err))
	}

	// Process each confirmation & apply fee calculations to each trade
	evts := make([]events.Event, 0, len(uncrossedOrders))
	for _, uncrossedOrder := range uncrossedOrders {
		m.handleConfirmation(ctx, uncrossedOrder)

		if uncrossedOrder.Order.Remaining == 0 {
			uncrossedOrder.Order.Status = types.Order_STATUS_FILLED
		}
		evts = append(evts, events.NewOrderEvent(ctx, uncrossedOrder.Order))
		if err := m.applyFees(ctx, uncrossedOrder.Order, uncrossedOrder.Trades); err != nil {
			// @TODO this ought to be an event
			m.log.Error("Unable to apply fees to order", logging.String("OrderID", uncrossedOrder.Order.Id))
		}
	}
	// send order events in a single batch, it's more efficient
	m.broker.SendBatch(evts)

	// Process each order we have to cancel
	for _, order := range ordersToCancel {
		_, err := m.cancelOrder(ctx, order.PartyId, order.Id)
		if err != nil {
			m.log.Error("Failed to cancel order", logging.String("OrderID", order.Id))
		}
	}

	// now that we're left the auction, we can mark all positions
	// in case any trader is distressed (Which shouldn't be possible)
	// we'll fall back to the a network order at the new mark price (mid-price)
	m.confirmMTM(ctx, &types.Order{Price: m.markPrice})

	// update auction state, so we know what the new tradeMode ought to be
	endEvt := m.as.AuctionEnded(ctx, now)

	// Send an event bus update
	m.broker.Send(endEvt)

	// We are moving to continuous trading so we have to unpark any pegged orders
	m.repriceAllPeggedOrders(ctx, PriceMoveAll)

	// Store the lastest prices so we can see if anything moves
	m.lastMidBuyPrice, _ = m.getStaticMidPrice(types.Side_SIDE_BUY)
	m.lastMidSellPrice, _ = m.getStaticMidPrice(types.Side_SIDE_SELL)
	m.lastBestBidPrice, _ = m.getBestStaticBidPrice()
	m.lastBestAskPrice, _ = m.getBestStaticAskPrice()
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

	// now chec if all  buy/sell/size are 0
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
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}
	conf, err := m.submitOrder(ctx, order, true)
	if err != nil {
		return nil, err
	}

	if err := m.liquidityUpdate(ctx, append(conf.PassiveOrdersAffected, conf.Order)); err != nil {
		return nil, err
	}

	return conf, nil
}

func (m *Market) submitOrder(ctx context.Context, order *types.Order, setID bool) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.Id, orderValidity)
	}()

	// set those at the begining as even rejected order get through the buffers
	if setID {
		m.idgen.SetID(order)
	}
	order.Version = InitialOrderVersion
	order.Status = types.Order_STATUS_ACTIVE

	if err := m.validateOrder(ctx, order); err != nil {
		return nil, err
	}

	if err := m.validateAccounts(ctx, order); err != nil {
		return nil, err
	}

	if order.PeggedOrder != nil {
		// Add pegged order to time sorted list
		m.addPeggedOrder(order)
	}

	// Now that validation is handled, call the code to place the order
	orderConf, err := m.submitValidatedOrder(ctx, order)
	if err == nil {
		orderValidity = "valid"
	}

	if order.PeggedOrder != nil && order.Status != types.Order_STATUS_ACTIVE && order.Status != types.Order_STATUS_PARKED {
		// remove the pegged order from anywhere
		m.removePeggedOrder(order)
	}

	// insert an expiring order if it's either in the book
	// or in the parked list
	if order.IsExpireable() && !order.IsFinished() {
		m.expiringOrders.Insert(*order)
	}

	m.checkForReferenceMoves(ctx)

	return orderConf, err
}

func (m *Market) submitValidatedOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	isPegged := (order.PeggedOrder != nil)
	if isPegged {
		order.Status = types.Order_STATUS_PARKED
		order.Reason = types.OrderError_ORDER_ERROR_UNSPECIFIED

		if m.as.InAuction() {
			// If we are in an auction, we don't insert this order into the book
			// Maybe should return an orderConfirmation with order state PARKED
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil
		} else {
			// Reprice
			err := m.repricePeggedOrder(ctx, order)
			if err != nil {
				m.broker.Send(events.NewOrderEvent(ctx, order))
				return &types.OrderConfirmation{Order: order}, nil
			}
			order.Status = types.Order_STATUS_ACTIVE
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
		checkMargin = (oldVol <= newVol)
	}

	// Perform check and allocate margin unless the order is (partially) closing the trader position
	if checkMargin {
		if _, err := m.checkMarginForOrder(ctx, pos, order); err != nil {
			if _, err := m.position.UnregisterOrder(order); err != nil {
				m.log.Error("Unable to unregister potential trader positions",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}

			// adding order to the buffer first
			order.Status = types.Order_STATUS_REJECTED
			order.Reason = types.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED
			m.broker.Send(events.NewOrderEvent(ctx, order))

			if m.log.GetLevel() <= logging.DebugLevel {
				m.log.Debug("Unable to check/add margin for trader",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			return nil, ErrMarginCheckFailed
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
			return nil, m.unregisterAndReject(ctx, order, err)
		}

		// try to apply fees on the trade
		err = m.applyFees(ctx, order, trades)
		if err != nil {
			return nil, err
		}
	}
	// Send the aggressive order into matching engine
	confirmation, err := m.matching.SubmitOrder(order)
	if err != nil {
		if _, err := m.position.UnregisterOrder(order); err != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		order.Status = types.Order_STATUS_REJECTED
		if oerr, ok := types.IsOrderError(err); ok {
			order.Reason = oerr
		} else {
			// should not happend but still...
			order.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
		}
		m.broker.Send(events.NewOrderEvent(ctx, order))
		if m.log.GetLevel() <= logging.DebugLevel {
			m.log.Debug("Failure after submitting order to matching engine",
				logging.Order(*order),
				logging.Error(err))
		}
		return nil, err
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
		_, err := m.position.UnregisterOrder(order)
		if err != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
	}

	// we replace the trades in the confirmation with the one we got initially
	// the contains the fees informations
	confirmation.Trades = trades

	m.handleConfirmation(ctx, confirmation)

	m.broker.Send(events.NewOrderEvent(ctx, order))

	return confirmation, nil
}

func (m *Market) checkPriceAndGetTrades(ctx context.Context, order *types.Order) ([]*types.Trade, error) {
	trades, err := m.matching.GetTrades(order)
	if err == nil {
		for _, t := range trades {
			if err := m.pMonitor.CheckPrice(ctx, m.as, t.Price, t.Size, m.currentTime); err != nil {
				m.log.Error("Price monitoring error", logging.Error(err))
				// @TODO handle or panic? (panic is last resort)
			}
		}
		if m.as.AuctionStart() {
			m.EnterAuction(ctx)
			return nil, err
		}
	}
	return trades, err
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
		return m.unregisterAndReject(ctx, order, err)
	}
	_ = fees

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
		return m.unregisterAndReject(ctx,
			order, types.OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES)
	}

	// send transfers through the broker
	if err == nil && len(transfers) > 0 {
		evt := events.NewTransferResponse(ctx, transfers)
		m.broker.Send(evt)
	}

	return nil
}

func (m *Market) handleConfirmation(ctx context.Context, conf *types.OrderConfirmation) {
	if conf.PassiveOrdersAffected != nil {
		// Insert or update passive orders siting on the book
		for _, order := range conf.PassiveOrdersAffected {
			// set the `updatedAt` value as these orders have changed
			order.UpdatedAt = m.currentTime.UnixNano()
			m.broker.Send(events.NewOrderEvent(ctx, order))

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
	}
	end := m.as.AuctionEnd()

	if len(conf.Trades) > 0 {

		// Calculate and set current mark price
		m.setMarkPrice(conf.Trades[len(conf.Trades)-1])

		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(conf.Trades))
		for idx, trade := range conf.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", conf.Order.Id, idx)
			if conf.Order.Side == types.Side_SIDE_BUY {
				trade.BuyOrder = conf.Order.Id
				trade.SellOrder = conf.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = conf.Order.Id
				trade.BuyOrder = conf.PassiveOrdersAffected[idx].Id
			}

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions (this communicates with settlement via channel)
			m.position.Update(trade)
			// Record open inteterest change
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
			m.confirmMTM(ctx, conf.Order)
		}
	}
}

func (m *Market) confirmMTM(ctx context.Context, order *types.Order) {
	// now let's get the transfers for MTM settlement
	evts := m.position.UpdateMarkPrice(m.markPrice)
	settle := m.settlement.SettleMTM(ctx, m.markPrice, evts)

	// Only process collateral and risk once per order, not for every trade
	margins := m.collateralAndRisk(ctx, settle)
	if len(margins) > 0 {
		// TODO(): handle market makers penalties
		transfers, closed, _, err := m.collateral.MarginUpdate(ctx, m.GetID(), margins)
		if err == nil && len(transfers) > 0 {
			evt := events.NewTransferResponse(ctx, transfers)
			m.broker.Send(evt)
		}
		if len(closed) > 0 {
			err = m.resolveClosedOutTraders(ctx, closed, order)
			if err != nil {
				m.log.Error("unable to close out traders",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
		}
		m.updateLiquidityFee(ctx)
	}
}

// updateLiquidityFee computes the current LiquidityProvision fee and updates
// the fee engine.
func (m *Market) updateLiquidityFee(ctx context.Context) {
	stake := m.getTargetStake()
	fee := m.liquidity.ProvisionsPerParty().FeeForTarget(uint64(stake))
	_ = fee

	// TODO(jeremy): this need to be uncommented later
	// we do not set the fee for now so system-test
	// can keep running with the static fee setted up in the
	// genesis block.
	// m.fee.SetLiquidityFee(fee)
	//
	// m.mkt.Fees.Factors.LiquidityFee = fee
	// m.broker.Send(
	// 	events.NewMarketEvent(ctx, *m.mkt),
	// )
}

// resolveClosedOutTraders - the traders with the given market position who haven't got sufficient collateral
// need to be closed out -> the network buys/sells the open volume, and trades with the rest of the network
// this flow is similar to the SubmitOrder bit where trades are made, with fewer checks (e.g. no MTM settlement, no risk checks)
// pass in the order which caused traders to be distressed
func (m *Market) resolveClosedOutTraders(ctx context.Context, distressedMarginEvts []events.Margin, o *types.Order) error {
	if len(distressedMarginEvts) == 0 {
		return nil
	}
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "resolveClosedOutTraders")
	defer timer.EngineTimeCounterAdd()

	distressedPos := make([]events.MarketPosition, 0, len(distressedMarginEvts))
	for _, v := range distressedMarginEvts {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("closing out trader",
				logging.String("party-id", v.Party()),
				logging.String("market-id", m.GetID()))
		}
		distressedPos = append(distressedPos, v)
	}
	// cancel pending orders for traders
	rmorders, err := m.matching.RemoveDistressedOrders(distressedPos)
	if err != nil {
		m.log.Error(
			"Failed to remove distressed traders from the orderbook",
			logging.Error(err),
		)
		return err
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
		if _, err := m.position.UnregisterOrder(o); err != nil {
			m.log.Error("unable to unregister order for a distressed party",
				logging.String("party-id", o.PartyId),
				logging.String("market-id", mktID),
				logging.String("order-id", o.Id),
			)
		}
	}

	// now we also remove ALL parked order for the different parties
	for _, v := range distressedPos {
		orders := m.getAllParkedOrdersForParty(v.Party())
		for _, o := range orders {
			m.removePeggedOrder(o)
			o.UpdatedAt = m.currentTime.UnixNano()
			o.Status = types.Order_STATUS_STOPPED // closing out = status STOPPED
			evts = append(evts, events.NewOrderEvent(ctx, o))
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
		return nil
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

		// remove accounts, positions and return
		// from settlement engine first
		m.settlement.RemoveDistressed(ctx, closed)
		// then from positions
		closedMPs = m.position.RemoveDistressed(closedMPs)
		asset, _ := m.mkt.GetAsset()
		// finally remove from collateral (moving funds where needed)
		var movements *types.TransferResponse
		movements, err = m.collateral.RemoveDistressed(ctx, closedMPs, m.GetID(), asset)
		if err != nil {
			m.log.Error(
				"Failed to remove distressed accounts cleanly",
				logging.Error(err),
			)
			return err
		}
		if len(movements.Transfers) > 0 {
			evt := events.NewTransferResponse(ctx, []*types.TransferResponse{movements})
			m.broker.Send(evt)
		}
		return nil
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
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure after submitting order to matching engine",
				logging.Order(no),
				logging.Error(err))
		}
		return err
	}
	// @NOTE: At this point, the network order was updated by the orderbook
	// the price field now contains the average trade price at which the order was fulfilled
	m.broker.Send(events.NewOrderEvent(ctx, &no))

	// FIXME(j): this is a temporary measure for the case where we do not have enough orders
	// in the book to 0 out the positions.
	// in this case we will just return now, cutting off the position resolution
	// this means that trader still being distressed will stay distressed,
	// then when a new order is placed, the distressed traders will go again through positions resolution
	// and if the volume of the book is acceptable, we will then process positions resolutions
	if no.Remaining == no.Size {
		return ErrNotEnoughVolumeToZeroOutNetworkOrder
	}

	if confirmation.PassiveOrdersAffected != nil {
		// Insert or update passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			order.UpdatedAt = m.currentTime.UnixNano()
			m.broker.Send(events.NewOrderEvent(ctx, order))

			// If the order is a pegged order and is complete we must remove it from the pegged list
			if order.PeggedOrder != nil {
				if order.Remaining == 0 || order.Status != types.Order_STATUS_ACTIVE {
					m.removePeggedOrder(order)
				}
			}

			// remove expiring order
			if order.IsExpireable() && order.IsFinished() {
				m.expiringOrders.RemoveOrder(order.ExpiresAt, order.Id)
			}
		}
	}

	asset, _ := m.mkt.GetAsset()

	// pay the fees now
	fees, distressedPartiesFees, err := m.fee.CalculateFeeForPositionResolution(
		confirmation.Trades, closedMPs)
	if err != nil {
		m.log.Error("unable to calculate fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return err
	}
	tresps, err := m.collateral.TransferFees(ctx, m.GetID(), asset, fees)
	if err != nil {
		m.log.Error("unable to transfer fees for positions resolutions",
			logging.Error(err),
			logging.String("market-id", m.GetID()))
		return err
	}
	// send transfer to buffer
	m.broker.Send(events.NewTransferResponse(ctx, tresps))

	if len(confirmation.Trades) > 0 {
		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(confirmation.Trades))
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", no.Id, idx)
			if no.Side == types.Side_SIDE_BUY {
				trade.BuyOrder = no.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = no.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			// setup the type of the trade to network
			// this trade did happen with a GOOD trader to
			// 0 out the BAD trader position
			trade.Type = types.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions - this is a special trade involving the network as party
			// so rather than checking this every time we call Update, call special UpdateNetwork
			m.position.UpdateNetwork(trade)
			m.settlement.AddTrade(trade)
		}
		m.broker.SendBatch(tradeEvts)
	}

	if err = m.zeroOutNetwork(ctx, closedMPs, &no, o, distressedPartiesFees); err != nil {
		m.log.Error(
			"Failed to create closing order with distressed traders",
			logging.Error(err),
		)
		return err
	}
	// remove accounts, positions, any funds left on the distressed accounts will be moved to the
	// insurance pool, which needs to happen before we settle the non-distressed traders
	m.settlement.RemoveDistressed(ctx, closed)
	closedMPs = m.position.RemoveDistressed(closedMPs)
	movements, err := m.collateral.RemoveDistressed(ctx, closedMPs, m.GetID(), asset)
	if err != nil {
		m.log.Error(
			"Failed to remove distressed accounts cleanly",
			logging.Error(err),
		)
		return err
	}
	if len(movements.Transfers) > 0 {
		evt := events.NewTransferResponse(ctx, []*types.TransferResponse{movements})
		m.broker.Send(evt)
	}
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
	return err
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
			Aggressor: order.Side, // we consider network to be agressor
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

func (m *Market) checkMarginForOrder(ctx context.Context, pos *positions.MarketPosition, order *types.Order) (*types.Transfer, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "checkMarginForOrder")
	defer timer.EngineTimeCounterAdd()
	return m.calcMargins(ctx, pos, order)
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
	m.broker.Send(events.NewTransferResponse(ctx, response))

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

	cancellations, err := m.matching.CancelAllOrders(partyID)
	if cancellations == nil || err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure after cancelling all orders from matching engine",
				logging.String("party-id", partyID),
				logging.String("market", m.mkt.Id),
				logging.Error(err))
		}
		return nil, err
	}

	var (
		// Create a slice ready to store the generated events in
		evts = make([]events.Event, 0, len(cancellations))
		// Check the parked order list of any orders from that same party
		parkedCancels []*types.OrderCancellationConfirmation
		// orders from park list to be removed
		toRemove []*types.Order
	)

	for _, order := range m.peggedOrders {
		if order.PartyId == partyID && order.Status == types.Order_STATUS_PARKED {
			toRemove = append(toRemove, order)
		}
	}
	for _, order := range toRemove {
		order.Status = types.Order_STATUS_CANCELLED
		m.removePeggedOrder(order)
		order.UpdatedAt = m.currentTime.UnixNano()
		evts = append(evts, events.NewOrderEvent(ctx, order))

		parkedCancel := &types.OrderCancellationConfirmation{
			Order: order,
		}
		parkedCancels = append(parkedCancels, parkedCancel)
	}

	for _, cancellation := range cancellations {
		if cancellation.Order.IsExpireable() {
			m.expiringOrders.RemoveOrder(cancellation.Order.ExpiresAt, cancellation.Order.Id)
		}
		// if the order was a pegged order, remove from pegged list
		if cancellation.Order.PeggedOrder != nil {
			m.removePeggedOrder(cancellation.Order)
		}

		// Update the order in our stores (will be marked as cancelled)
		cancellation.Order.UpdatedAt = m.currentTime.UnixNano()
		evts = append(evts, events.NewOrderEvent(ctx, cancellation.Order))
		_, err = m.position.UnregisterOrder(cancellation.Order)
		if err != nil {
			m.log.Error("Failure unregistering order in positions engine (cancel)",
				logging.Order(*cancellation.Order),
				logging.Error(err))
		}
	}

	// Send off all the events in one big batch
	m.broker.SendBatch(evts)

	m.checkForReferenceMoves(ctx)

	cancellations = append(cancellations, parkedCancels...)
	return cancellations, nil
}

func (m *Market) CancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	return m.cancelOrder(ctx, partyID, orderID)
}

// CancelOrder cancels the given order
func (m *Market) cancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {

	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "CancelOrder")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
	}

	// cancelling and amending an order that is part of the LP commitment isn't allowed
	if m.liquidity.IsLiquidityOrder(partyID, orderID) {
		return nil, types.ErrEditNotAllowed
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
		_, err = m.position.UnregisterOrder(order)
		if err != nil {
			m.log.Error("Failure unregistering order in positions engine (cancel)",
				logging.Order(*order),
				logging.Error(err))
		}
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

	m.checkForReferenceMoves(ctx)

	if foundOnBook {
		if err := m.liquidityUpdate(ctx, []*types.Order{order}); err != nil {
			return nil, err
		}
	}

	return &types.OrderCancellationConfirmation{Order: order}, nil
}

// parkOrderAndAdd removes the order from the orderbook and adds it to the parked list
func (m *Market) parkOrderAndAdd(ctx context.Context, order *types.Order) {
	m.parkOrder(ctx, order)
}

// parkOrder removes the given order from the orderbook
// parkOrder will panic if it encounters errors, which means that it reached an
// invalid state.
func (m *Market) parkOrder(ctx context.Context, order *types.Order) {
	defer m.releaseMarginExcess(ctx, order.PartyId)

	if err := m.matching.RemoveOrder(order); err != nil {
		m.log.Panic("Failure to remove order from matching engine",
			logging.String("party-id", order.PartyId),
			logging.String("order-id", order.Id),
			logging.String("market", m.mkt.Id),
			logging.Error(err))
	}

	// Update the order in our stores (will be marked as parked)
	order.UpdatedAt = m.currentTime.UnixNano()
	order.Status = types.Order_STATUS_PARKED
	order.Price = 0
	m.broker.Send(events.NewOrderEvent(ctx, order))
	if _, err := m.position.UnregisterOrder(order); err != nil {
		m.log.Panic("Failure un-registering order in positions engine (parking)",
			logging.Order(*order),
			logging.Error(err))
	}
}

// AmendOrder amend an existing order from the order book
func (m *Market) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	if !m.canTrade() {
		return nil, ErrTradingNotAllowed
	}

	// explicitly/directly ordering an LP commitment order is not allowed
	if m.liquidity.IsLiquidityOrder(orderAmendment.PartyId, orderAmendment.OrderId) {
		return nil, types.ErrEditNotAllowed
	}
	conf, err := m.amendOrder(ctx, orderAmendment)
	if err != nil {
		return nil, err
	}

	if err := m.liquidityUpdate(ctx, append(conf.PassiveOrdersAffected, conf.Order)); err != nil {
		return nil, err
	}

	return conf, nil
}

func (m *Market) amendOrder(ctx context.Context, orderAmendment *types.OrderAmendment) (cnf *types.OrderConfirmation, returnedErr error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, ErrMarketClosed
	}

	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, _, err := m.getOrderByID(orderAmendment.OrderId)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid order ID",
				logging.String("id", orderAmendment.GetOrderId()),
				logging.String("party", orderAmendment.GetPartyId()),
				logging.String("market", orderAmendment.GetMarketId()),
				logging.Error(err))
		}
		return nil, types.ErrInvalidOrderID
	}

	// We can only amend this order if we created it
	if existingOrder.PartyId != orderAmendment.PartyId {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid party ID",
				logging.String("original party id:", existingOrder.PartyId),
				logging.String("amend party id:", orderAmendment.PartyId))
		}
		return nil, types.ErrInvalidPartyID
	}

	// Validate Market
	if existingOrder.MarketId != m.mkt.Id {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Market ID mismatch",
				logging.String("market-id", m.mkt.Id),
				logging.Order(*existingOrder))
		}
		return nil, types.ErrInvalidMarketID
	}

	if err := m.validateOrderAmendment(existingOrder, orderAmendment); err != nil {
		return nil, err
	}

	amendedOrder, err := m.applyOrderAmendment(ctx, existingOrder, orderAmendment)
	if err != nil {
		return nil, err
	}

	// If we have a pegged order that is no longer expiring, we need to remove it
	var (
		needToRemoveExpiry bool  = false
		needToAddExpiry    bool  = false
		expiresAt          int64 = 0
	)
	defer func() {
		// no errors, amend most likely happened properly
		if returnedErr == nil {
			if needToRemoveExpiry {
				m.expiringOrders.RemoveOrder(expiresAt, existingOrder.Id)

			}
			if needToAddExpiry {
				m.expiringOrders.Insert(*existingOrder)
			}
		}
	}()

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

	// if remaining is reduces <= 0, then order is cancelled
	if amendedOrder.Remaining <= 0 {
		confirm, err := m.cancelOrder(
			ctx, existingOrder.PartyId, existingOrder.Id)
		if err != nil {
			return nil, err
		}
		return &types.OrderConfirmation{
			Order: confirm.Order,
		}, nil
	}

	// if expiration has changed and is before the original creation time, reject this amend
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < existingOrder.CreatedAt {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Amended expiry before original creation time",
				logging.Int64("original order created at ts:", existingOrder.CreatedAt),
				logging.Int64("amended expiry ts:", amendedOrder.ExpiresAt),
				logging.Order(*existingOrder))
		}
		return nil, types.ErrInvalidExpirationDatetime
	}

	// if expiration has changed and is not 0, and is before currentTime
	// then we expire the order
	if amendedOrder.ExpiresAt != 0 && amendedOrder.ExpiresAt < amendedOrder.UpdatedAt {
		// remove the order from the expiring
		m.expiringOrders.RemoveOrder(amendedOrder.ExpiresAt, amendedOrder.Id)

		// Update the existing message in place before we cancel it
		m.orderAmendInPlace(existingOrder, amendedOrder)
		cancellation, err := m.matching.CancelOrder(amendedOrder)
		if cancellation == nil || err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failure to cancel order from matching engine",
					logging.String("party-id", amendedOrder.PartyId),
					logging.String("order-id", amendedOrder.Id),
					logging.String("market", m.mkt.Id),
					logging.Error(err))
			}
			return nil, err
		}

		// Update the order in our stores (will be marked as cancelled)
		// set the proper status
		cancellation.Order.Status = types.Order_STATUS_EXPIRED
		m.broker.Send(events.NewOrderEvent(ctx, cancellation.Order))
		_, err = m.position.UnregisterOrder(cancellation.Order)
		if err != nil {
			m.log.Error("Failure unregistering order in positions engine (amendOrder)",
				logging.Order(*amendedOrder),
				logging.Error(err))
		}

		m.checkForReferenceMoves(ctx)

		return &types.OrderConfirmation{
			Order: cancellation.Order,
		}, nil
	}

	if existingOrder.PeggedOrder != nil {

		// Amend in place during an auction
		if m.as.InAuction() {
			ret, err := m.orderAmendWhenParked(existingOrder, amendedOrder)
			if err == nil {
				m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			}
			return ret, err
		}
		err := m.repricePeggedOrder(ctx, amendedOrder)
		if err != nil {
			// Failed to reprice so we have to park the order
			if amendedOrder.Status != types.Order_STATUS_PARKED {
				// If we are live then park
				m.parkOrderAndAdd(ctx, existingOrder)
			}
			ret, err := m.orderAmendWhenParked(existingOrder, amendedOrder)
			if err == nil {
				m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			}
			return ret, err
		} else {
			// We got a new valid price, if we are parked we need to unpark
			if amendedOrder.Status == types.Order_STATUS_PARKED {
				orderConf, err := m.submitValidatedOrder(ctx, amendedOrder)
				if err != nil {
					// If we cannot submit a new order then the amend has failed, return the error
					return nil, err
				}
				// Update pegged order with new amended version
				for i, o := range m.peggedOrders {
					if o.Id == amendedOrder.Id {
						m.peggedOrders[i] = amendedOrder
						break
					}
				}
				return orderConf, err
			}
		}
	}

	// from here these are the normal amendment
	var priceIncrease, priceShift, sizeIncrease, sizeDecrease, expiryChange, timeInForceChange bool

	if amendedOrder.Price != existingOrder.Price {
		priceShift = true
		priceIncrease = (existingOrder.Price < amendedOrder.Price)
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
			m.checkForReferenceMoves(ctx)
		}
		return ret, err
	}

	// Update potential new position after the amend
	pos, err := m.position.AmendOrder(existingOrder, amendedOrder)
	if err != nil {
		// adding order to the buffer first
		amendedOrder.Status = types.Order_STATUS_REJECTED
		amendedOrder.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
		m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Unable to amend potential trader position",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		return nil, ErrMarginCheckFailed
	}

	// Perform check and allocate margin if price or order size is increased
	// ignore rollback return here, as if we amend it means the order
	// is already on the book, not rollback will be needed, the margin
	// will be updated later on for sure.

	if priceIncrease || sizeIncrease {
		if _, err = m.checkMarginForOrder(ctx, pos, amendedOrder); err != nil {
			// Undo the position registering
			_, err1 := m.position.AmendOrder(amendedOrder, existingOrder)
			if err1 != nil {
				m.log.Error("Unable to unregister potential amended trader position",
					logging.String("market-id", m.GetID()),
					logging.Error(err1))
			}

			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Unable to check/add margin for trader",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			return nil, ErrMarginCheckFailed
		}
	}

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		confirmation, err := m.orderCancelReplace(ctx, existingOrder, amendedOrder)
		if err == nil {
			m.handleConfirmation(ctx, confirmation)
			m.broker.Send(events.NewOrderEvent(ctx, confirmation.Order))
			m.checkForReferenceMoves(ctx)
		}
		return confirmation, err
	}

	// if decrease in size or change in expiration date
	// ---> DO amend in place in matching engine
	if expiryChange || sizeDecrease || timeInForceChange {
		if sizeDecrease && amendedOrder.Remaining >= existingOrder.Remaining {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Order amendment not allowed when reducing to a larger amount", logging.Order(*existingOrder))
			}
			return nil, ErrInvalidAmendRemainQuantity
		}
		ret, err := m.orderAmendInPlace(existingOrder, amendedOrder)
		if err == nil {
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
			m.checkForReferenceMoves(ctx)
		}
		return ret, err
	}

	if m.log.GetLevel() == logging.DebugLevel {
		m.log.Debug("Order amendment not allowed", logging.Order(*existingOrder))
	}
	return nil, types.ErrEditNotAllowed
}

func (m *Market) validateOrderAmendment(
	order *types.Order,
	amendment *types.OrderAmendment,
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
	amendment *types.OrderAmendment,
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
		err = m.applyFees(ctx, newOrder, trades)
		if err != nil {
			return nil, err
		}

		// Because other collections might be pointing at the original order
		// use it's memory when inserting the new version
		*existingOrder = *newOrder
		conf, err = m.matching.SubmitOrder(existingOrder) //lint:ignore SA4006 this value might be overwriter, careful!
		if err != nil {
			m.log.Panic("unable to submit order", logging.Error(err))
		}
		// replace thetrades in the confirmation to have
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

func (m *Market) orderAmendWhenParked(originalOrder, amendOrder *types.Order) (*types.OrderConfirmation, error) {
	amendOrder.Status = types.Order_STATUS_PARKED
	amendOrder.Price = 0
	*originalOrder = *amendOrder

	return &types.OrderConfirmation{
		Order: amendOrder,
	}, nil
}

// RemoveExpiredOrders remove all expired orders from the order book
// and also any pegged orders that are parked
func (m *Market) RemoveExpiredOrders(timestamp int64) ([]types.Order, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "RemoveExpiredOrders")
	defer timer.EngineTimeCounterAdd()

	if m.closed {
		return nil, ErrMarketClosed
	}

	expired := []types.Order{}
	for _, order := range m.expiringOrders.Expire(timestamp) {
		// The pegged expiry orders are copies and do not reflect the
		// current state of the order, therefore we look it up
		originalOrder, _, err := m.getOrderByID(order.Id)
		if err == nil {
			// assign to the order the order from the book
			// so we get the most recent version from the book
			// to continue with
			order = *originalOrder

			// if the order was on the book basically
			// either a pegged + non parked
			// or a non-pegged order
			if (order.PeggedOrder != nil && order.Status != types.Order_STATUS_PARKED) ||
				order.PeggedOrder == nil {
				m.unregisterOrder(&order)
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
	if len(expired) > 0 {
		m.checkForReferenceMoves(context.Background())
	}

	return expired, nil
}

func (m *Market) unregisterOrder(order *types.Order) {
	if _, err := m.position.UnregisterOrder(order); err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Failure unregistering order in positions engine (cancel)",
				logging.Order(*order),
				logging.Error(err))
		}
	}
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

// checkForReferenceMoves looks to see if the reference prices have moved since the
// last transaction was processed.
func (m *Market) checkForReferenceMoves(ctx context.Context) {
	if m.as.InAuction() {
		return
	}
	var repricedCount uint64
	for repricedCount = 1; repricedCount > 0; {
		// Get the current reference values and compare them to the last saved set
		newBestBid, _ := m.getBestStaticBidPrice()
		newBestAsk, _ := m.getBestStaticAskPrice()
		newMidBuy, _ := m.getStaticMidPrice(types.Side_SIDE_BUY)
		newMidSell, _ := m.getStaticMidPrice(types.Side_SIDE_SELL)

		// Look for a move
		var changes uint8
		if newMidBuy != m.lastMidBuyPrice ||
			newMidSell != m.lastMidSellPrice {
			changes |= PriceMoveMid
		}
		if newBestBid != m.lastBestBidPrice {
			changes |= PriceMoveBestBid
		}
		if newBestAsk != m.lastBestAskPrice {
			changes |= PriceMoveBestAsk
		}

		// If we have a reference price move, update any pegged orders that reference it
		if changes != 0 {
			repricedCount = m.repriceAllPeggedOrders(ctx, changes)
		} else {
			repricedCount = 0
		}

		// Update the last price values
		m.lastMidBuyPrice = newMidBuy
		m.lastMidSellPrice = newMidSell
		m.lastBestBidPrice = newBestBid
		m.lastBestAskPrice = newBestAsk
	}
}

func (m *Market) addPeggedOrder(order *types.Order) {
	m.peggedOrders = append(m.peggedOrders, order)
}

func (m *Market) getAllParkedOrdersForParty(party string) (orders []*types.Order) {
	for _, order := range m.peggedOrders {
		if order.PartyId == party && order.Status == types.Order_STATUS_PARKED {
			orders = append(orders, order)
		}
	}
	return
}

// removePeggedOrder looks through the pegged and parked list
// and removes the matching order if found
func (m *Market) removePeggedOrder(order *types.Order) {
	// remove if order was expiring
	m.expiringOrders.RemoveOrder(order.ExpiresAt, order.Id)

	for i, po := range m.peggedOrders {
		if po.Id == order.Id {
			// Remove item from slice
			copy(m.peggedOrders[i:], m.peggedOrders[i+1:])
			m.peggedOrders[len(m.peggedOrders)-1] = nil
			m.peggedOrders = m.peggedOrders[:len(m.peggedOrders)-1]
			break
		}
	}
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
	for _, order := range m.peggedOrders {
		if order.Id == orderID {
			return order, false, nil
		}
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

// repriceFuncW is an adapter for getNewPeggedPrice.
func (m *Market) repriceFuncW(po *types.PeggedOrder) (uint64, error) {
	return m.getNewPeggedPrice(
		&types.Order{PeggedOrder: po},
	)
}

// SubmitLiquidityProvision forwards a LiquidityProvisionSubmission to the Liquidity Engine.
func (m *Market) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, id string) (err error) {
	if !m.canSubmitCommitment() {
		return ErrCommitmentSubmissionNotAllowed
	}

	// Increasing the commitment should always be allowed, but decreasing is
	// only valid if the resulting amount still allows the market as a whole
	// to reach it's commitment level. Otherwise the commitment reduction is
	// rejected.
	if lp := m.liquidity.LiquidityProvisionByPartyID(party); lp != nil {
		if sub.CommitmentAmount < lp.CommitmentAmount {
			// this is the amount of stake surplus
			surplus := uint64(m.getTargetStake()) - m.getSuppliedStake()
			diff := lp.CommitmentAmount - sub.CommitmentAmount
			if diff > surplus {
				return ErrNotEnoughStake
			}
		}
	}

	if err := m.liquidity.SubmitLiquidityProvision(ctx, sub, party, id); err != nil {
		return err
	}

	defer func() {
		if err == nil {
			return
		}
		if newerr := m.liquidity.CancelLiquidityProvision(ctx, party); newerr != nil {
			m.log.Debug("unable to submit cancel liquidity provision submission",
				logging.String("party", party),
				logging.String("id", id),
				logging.Error(newerr))
			err = fmt.Errorf("%v, %w", err, newerr)
		}
	}()

	// WE WANT TO APPLY THECOMMITMENT IN BOND ACCOUNT
	asset, _ := m.mkt.GetAsset()
	bondAcc, err := m.collateral.GetOrCreatePartyBondAccount(ctx, party, m.GetID(), asset)
	if err != nil {
		return err
	}

	// now we calculate the amount that needs to be moved into the account
	amount := int64(sub.CommitmentAmount - bondAcc.Balance)
	ty := types.TransferType_TRANSFER_TYPE_BOND_LOW
	if amount < 0 {
		ty = types.TransferType_TRANSFER_TYPE_BOND_HIGH
		amount = -amount
	}
	transfer := &types.Transfer{
		Owner: party,
		Amount: &types.FinancialAmount{
			Amount: amount,
			Asset:  asset,
		},
		Type:      ty,
		MinAmount: amount,
	}

	tresp, err := m.collateral.BondUpdate(ctx, m.GetID(), party, transfer)
	if err != nil {
		return err
	}
	m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))

	// if something happen, rollback the transfer
	defer func() {
		if err == nil {
			return
		}
		if transfer.Type == types.TransferType_TRANSFER_TYPE_BOND_HIGH {
			transfer.Type = types.TransferType_TRANSFER_TYPE_BOND_LOW
		} else {
			transfer.Type = types.TransferType_TRANSFER_TYPE_BOND_HIGH
		}

		tresp, newerr := m.collateral.BondUpdate(ctx, m.GetID(), party, transfer)
		if newerr != nil {
			m.log.Debug("unable to rollback bon account topup",
				logging.String("party", party),
				logging.Int64("amount", amount),
				logging.Error(err))
			err = fmt.Errorf("%v, %w", err, newerr)
		}
		m.broker.Send(events.NewTransferResponse(ctx, []*types.TransferResponse{tresp}))
	}()

	existingOrders := m.matching.GetOrdersPerParty(party)
	newOrders, amendments, err := m.liquidity.CreateInitialOrders(m.markPrice, party, existingOrders, m.repriceFuncW)
	if err != nil {
		return err
	}

	if err := m.createAndUpdateOrders(ctx, newOrders, amendments); err != nil {
		return err
	}

	m.updateLiquidityFee(ctx)
	m.equityShares.SetPartyStake(party, float64(sub.CommitmentAmount))
	return nil
}

func (m *Market) liquidityUpdate(ctx context.Context, orders []*types.Order) error {
	newOrders, amendments, err := m.liquidity.Update(ctx, m.markPrice, m.repriceFuncW, orders)
	if err != nil {
		return err
	}

	return m.createAndUpdateOrders(ctx, newOrders, amendments)
}

func (m *Market) createAndUpdateOrders(ctx context.Context, newOrders []*types.Order, amendments []*types.OrderAmendment) (err error) {
	submittedIDs := []string{}
	// submitted order rollback
	defer func() {
		if err == nil || len(newOrders) <= 0 {
			return
		}
		party := newOrders[0].PartyId
		for _, v := range submittedIDs {
			_, newerr := m.cancelOrder(ctx, party, v)
			if newerr != nil {
				m.log.Error("unable to rollback order via cancel",
					logging.Error(newerr),
					logging.String("party", party),
					logging.String("order-id", v))
				err = fmt.Errorf("%v, %w", err, newerr)
			}
		}
	}()

	for _, order := range newOrders {
		if _, err := m.submitOrder(ctx, order, false); err != nil {
			return err
		}
		submittedIDs = append(submittedIDs, order.Id)
	}

	// amendment rollback
	amendmentsRollBack := []*types.OrderAmendment{}
	// submitted order rollback
	defer func() {
		if err == nil || len(amendmentsRollBack) <= 0 {
			return
		}
		for _, v := range amendmentsRollBack {
			_, newerr := m.amendOrder(ctx, v)
			if newerr != nil {
				m.log.Error("unable to rollback order via cancel",
					logging.Error(newerr),
					logging.String("party", v.PartyId),
					logging.String("order-id", v.OrderId))
				err = fmt.Errorf("%v, %w", err, newerr)
			}
		}
	}()

	for _, order := range amendments {
		if _, err := m.amendOrder(ctx, order); err != nil {
			return err
		}

		arb := *order
		arb.SizeDelta = -arb.SizeDelta
		amendmentsRollBack = append(amendmentsRollBack, &arb)
	}

	return nil
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

func lpsToLiquidityProviderFeeShare(lps map[string]*lp) []*types.LiquidityProviderFeeShare {
	out := make([]*types.LiquidityProviderFeeShare, 0, len(lps))
	for k, v := range lps {
		out = append(out, &types.LiquidityProviderFeeShare{
			Party:                 k,
			EquityLikeShare:       strconv.FormatFloat(v.share, 'f', -1, 64),
			AverageEntryValuation: strconv.FormatFloat(v.avg, 'f', -1, 64),
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
		return err
	}

	acc, err := m.collateral.GetMarketLiquidityFeeAccount(m.mkt.GetId(), asset)
	if err != nil {
		return err
	}

	// We can't distribute any share when no balance.
	if acc.Balance == 0 {
		return nil
	}

	shares := m.equityShares.Shares()
	if len(shares) == 0 {
		return nil
	}

	feeTransfer := m.fee.BuildLiquidityFeeDistributionTransfer(shares, acc)
	if feeTransfer == nil {
		return nil
	}

	resp, err := m.collateral.TransferFees(ctx, m.GetID(), asset, feeTransfer)
	if err != nil {
		return err
	}

	m.broker.Send(events.NewTransferResponse(ctx, resp))
	return nil
}
