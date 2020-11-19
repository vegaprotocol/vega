package execution

import (
	"context"
	"crypto/sha256"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"math"
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
	// ErrGFAOrderReceivedDuringContinuousTrading is returned is a gfa order hits the market when the market is in continous trading state
	ErrGFAOrderReceivedDuringContinuousTrading = errors.New("gfa order received during continuous trading")
	// ErrGFNOrderReceivedAuctionTrading is returned if a gfn order hits the market when in auction state
	ErrGFNOrderReceivedAuctionTrading = errors.New("gfn order received during auction trading")
	// ErrUnableToReprice we are unable to get a price required to reprice
	ErrUnableToReprice = errors.New("unable to reprice")
	// ErrOrderNotFound we cannot find the order in the market
	ErrOrderNotFound = errors.New("unable to find the order in the market")

	networkPartyID = "network"
)

// PriceMonitor interface to handle price monitoring/auction triggers
// @TODO the interface shouldn't be imported here
type PriceMonitor interface {
	CheckPrice(ctx context.Context, as price.AuctionState, p uint64, now time.Time) error
}

// TargetStakeCalculator interface
type TargetStakeCalculator interface {
	RecordOpenInterest(oi uint64, now time.Time) error
	GetTargetStake(rf types.RiskFactor, now time.Time) float64
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
	Mode() types.MarketState
	Trigger() types.AuctionTrigger
}

// Market represents an instance of a market in vega and is in charge of calling
// the engines in order to process all transctiona
type Market struct {
	log   *logging.Logger
	idgen *IDgenerator

	matchingConfig matching.Config

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
	peggedOrders         []*types.Order
	expiringPeggedOrders *matching.ExpiringOrders

	// A collection of pegged orders that have been parked
	parkedOrders []*types.Order

	// Store the previous price values so we can see what has changed
	lastBestBidPrice uint64
	lastBestAskPrice uint64
	lastMidPrice     uint64
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

	tsCalc := liquiditytarget.NewEngine(*mkt.TargetStake)
	liqEngine := liquidity.NewEngine(log, broker, idgen, tradableInstrument.RiskModel, pMonitor)

	market := &Market{
		log:                  log,
		idgen:                idgen,
		mkt:                  mkt,
		closingAt:            closingAt,
		currentTime:          now,
		markPrice:            tradableInstrument.Instrument.InitialMarkPrice,
		matching:             book,
		tradableInstrument:   tradableInstrument,
		risk:                 riskEngine,
		position:             positionEngine,
		settlement:           settleEngine,
		collateral:           collateralEngine,
		broker:               broker,
		fee:                  feeEngine,
		liquidity:            liqEngine,
		parties:              map[string]struct{}{},
		as:                   as,
		pMonitor:             pMonitor,
		tsCalc:               tsCalc,
		expiringPeggedOrders: matching.NewExpiringOrders(),
	}

	if market.as.AuctionStart() {
		market.EnterAuction(ctx)
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
	mId := logging.String("market-id", m.GetID())
	matchingHash := m.matching.Hash()
	m.log.Debug("orderbook state hash", logging.Hash(matchingHash), mId)

	positionHash := m.position.Hash()
	m.log.Debug("positions state hash", logging.Hash(positionHash), mId)

	accountsHash := m.collateral.Hash()
	m.log.Debug("accounts state hash", logging.Hash(accountsHash), mId)

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
		MarketState:           m.as.Mode(),
		Trigger:               m.as.Trigger(),
		TargetStake:           fmt.Sprintf("%.f", m.getTargetStake()),
		// FIXME(WITOLD): uncomment set real values here
		// SuppliedStake: getSuppliedStake(),
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

	m.risk.OnTimeUpdate(t)
	m.settlement.OnTick(t)
	m.liquidity.OnChainTimeUpdate(ctx, t)

	closed = t.After(m.closingAt)
	m.closed = closed
	m.currentTime = t

	// check price auction end
	if m.as.InAuction() {
		if m.as.IsOpeningAuction() {
			if endTS := m.as.ExpiresAt(); endTS != nil && endTS.Before(t) {
				// mark opening auction as ending
				m.as.EndAuction()
				m.LeaveAuction(ctx, t)
			}
		} else if m.as.IsPriceAuction() {
			p := m.matching.GetIndicativePrice()
			// ending auction now would result in no trades so feed the last mark price into pMonitor
			if p == 0 {
				p = m.markPrice
			}
			if err := m.pMonitor.CheckPrice(ctx, m.as, p, t); err != nil {
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

// repriceAllPeggedOrders runs through the slice of pegged orders and reprices all those
// which are using a reference that has moved. Returns the number of orders that were repriced.
func (m *Market) repriceAllPeggedOrders(ctx context.Context, changes uint8) uint64 {
	var repriceCount uint64
	for _, order := range m.peggedOrders {
		if (order.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_MID &&
			changes&PriceMoveMid > 0) ||
			(order.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_BEST_BID &&
				changes&PriceMoveBestBid > 0) ||
			(order.PeggedOrder.Reference == types.PeggedReference_PEGGED_REFERENCE_BEST_ASK &&
				changes&PriceMoveBestAsk > 0) {
			if order.Status != types.Order_STATUS_PARKED {
				price, err := m.getNewPeggedPrice(ctx, order)
				if err != nil {
					// We can't reprice so we should remove the order and park it
					m.parkOrderAndAdd(ctx, order)
				} else {
					// Amend the order on the orderbook
					m.amendPeggedOrder(ctx, order, price)
					repriceCount++
				}
			}
		}
	}
	return repriceCount
}

func (m *Market) getNewPeggedPrice(ctx context.Context, order *types.Order) (uint64, error) {
	var (
		err   error
		price uint64
	)

	switch order.PeggedOrder.Reference {
	case types.PeggedReference_PEGGED_REFERENCE_MID:
		price, err = m.getStaticMidPrice()
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
	price, err := m.getNewPeggedPrice(ctx, order)
	if err != nil {
		return err
	}
	order.Price = price
	return nil
}

// unparkAllPeggedOrders Attempt to place all pegged orders back onto the order book
func (m *Market) unparkAllPeggedOrders(ctx context.Context) {
	// Create slice to put any orders that we can't unpack
	failedToUnpark := make([]*types.Order, 0)
	for _, order := range m.peggedOrders {
		// Reprice the order and submit it
		err := m.repricePeggedOrder(ctx, order)
		if err != nil {
			// Failed to reprice
			failedToUnpark = append(failedToUnpark, order)
		} else {
			_, err := m.submitValidatedOrder(ctx, order)
			if err != nil {
				// Failed to place the order on the book
				failedToUnpark = append(failedToUnpark, order)
			}
		}
	}
	m.parkedOrders = failedToUnpark
}

// EnterAuction : Prepare the order book to be run as an auction
func (m *Market) EnterAuction(ctx context.Context) {
	// Change market type to auction
	ordersToCancel, ordersToPark, err := m.matching.EnterAuction()
	if err != nil {
		m.log.Error("Error entering auction: ", logging.Error(err))
	}

	// Cancel all the orders that were invalid
	for _, order := range ordersToCancel {
		m.CancelOrder(ctx, order.PartyID, order.Id)
	}

	// Send out events for all orders we park
	for _, order := range ordersToPark {
		m.parkOrder(ctx, order)
	}

	// Send an event bus update
	m.broker.Send(m.as.AuctionStarted(ctx))

	// At this point all pegged orders are parked but the pegged order list would be
	// identical to the parked order list so we save time by not updating the parked list
	m.parkedOrders = []*types.Order{}
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

	// Process each confirmation
	evts := make([]events.Event, 0, len(uncrossedOrders))
	for _, uncrossedOrder := range uncrossedOrders {
		m.handleConfirmation(ctx, uncrossedOrder.Order, uncrossedOrder)

		if uncrossedOrder.Order.Remaining == 0 {
			uncrossedOrder.Order.Status = types.Order_STATUS_FILLED
		}
		evts = append(evts, events.NewOrderEvent(ctx, uncrossedOrder.Order))
	}
	// send order events in a single batch, it's more efficient
	m.broker.SendBatch(evts)

	// Process each order we have to cancel
	for _, order := range ordersToCancel {
		_, err := m.CancelOrder(ctx, order.PartyID, order.Id)
		if err != nil {
			m.log.Error("Failed to cancel order", logging.String("OrderID", order.Id))
		}
	}

	// Apply fee calculations to each trade
	for _, uo := range uncrossedOrders {
		err := m.applyFees(ctx, uo.Order, uo.Trades)
		if err != nil {
			// @TODO this ought to be an event
			m.log.Error("Unable to apply fees to order", logging.String("OrderID", uo.Order.Id))
		}
	}

	// update auction state, so we know what the new tradeMode ought to be
	endEvt := m.as.AuctionEnded(ctx, now)

	// Send an event bus update
	m.broker.Send(endEvt)

	// We are moving to continuous trading so we have to unpark any pegged orders
	m.unparkAllPeggedOrders(ctx)
}

func (m *Market) validatePeggedOrder(ctx context.Context, order *types.Order) types.OrderError {
	if order.Type != types.Order_TYPE_LIMIT {
		// All pegged orders must be LIMIT orders
		return types.ErrPeggedOrderMustBeLimitOrder
	}

	if order.TimeInForce != types.Order_TIF_GTT && order.TimeInForce != types.Order_TIF_GTC {
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
	return types.OrderError_ORDER_ERROR_NONE
}

func (m *Market) validateOrder(ctx context.Context, order *types.Order) error {
	// Check we are allowed to handle this order type with the current market status
	isAuction := m.as.InAuction()
	if isAuction && order.TimeInForce == types.Order_TIF_GFN {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_GFN_ORDER_DURING_AN_AUCTION
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrGFAOrderReceivedDuringContinuousTrading
	}

	if isAuction && order.TimeInForce == types.Order_TIF_IOC {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrGFAOrderReceivedDuringContinuousTrading
	}

	if isAuction && order.TimeInForce == types.Order_TIF_FOK {
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION
		m.broker.Send(events.NewOrderEvent(ctx, order))
		return ErrGFAOrderReceivedDuringContinuousTrading
	}

	if !isAuction && order.TimeInForce == types.Order_TIF_GFA {
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
	if order.MarketID != m.mkt.Id {
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
		if reason != types.OrderError_ORDER_ERROR_NONE {
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
	if !m.collateral.HasGeneralAccount(order.PartyID, asset) {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE
		m.broker.Send(events.NewOrderEvent(ctx, order))

		// trader should be created before even trying to post order
		return ErrTraderDoNotExists
	}

	// ensure party have a general account, and margin account is / can be created
	_, err := m.collateral.CreatePartyMarginAccount(ctx, order.PartyID, order.MarketID, asset)
	if err != nil {
		m.log.Error("Margin account verification failed",
			logging.String("party-id", order.PartyID),
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
	m.addParty(order.PartyID)
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
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "SubmitOrder")
	orderValidity := "invalid"
	defer func() {
		timer.EngineTimeCounterAdd()
		metrics.OrderCounterInc(m.mkt.Id, orderValidity)
	}()

	// set those at the begining as even rejected order get through the buffers
	m.idgen.SetID(order)
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

	m.checkForReferenceMoves(ctx)

	return orderConf, err
}

func (m *Market) submitValidatedOrder(ctx context.Context, order *types.Order) (*types.OrderConfirmation, error) {
	if order.PeggedOrder != nil {
		order.Status = types.Order_STATUS_PARKED
		order.Reason = types.OrderError_ORDER_ERROR_NONE

		if m.as.InAuction() {
			// If we are in an auction, we don't insert this order into the book
			// Maybe should return an orderConfirmation with order state PARKED
			m.broker.Send(events.NewOrderEvent(ctx, order))
			return &types.OrderConfirmation{Order: order}, nil

		} else {
			// Reprice
			err := m.repricePeggedOrder(ctx, order)
			if err != nil {
				m.parkedOrders = append(m.parkedOrders, order)
				m.broker.Send(events.NewOrderEvent(ctx, order))
				return &types.OrderConfirmation{Order: order}, nil
			}
			order.Status = types.Order_STATUS_ACTIVE
		}
	}

	// Register order as potential positions
	pos, err := m.position.RegisterOrder(order)
	if err != nil {
		// adding order to the buffer first
		order.Status = types.Order_STATUS_REJECTED
		order.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
		m.broker.Send(events.NewOrderEvent(ctx, order))

		if m.log.GetLevel() <= logging.DebugLevel {
			m.log.Debug("Unable to register potential trader position",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		return nil, ErrMarginCheckFailed
	}

	// Perform check and allocate margin
	_, err = m.checkMarginForOrder(ctx, pos, order)
	if err != nil {
		_, err1 := m.position.UnregisterOrder(order)
		if err1 != nil {
			m.log.Error("Unable to unregister potential trader positions",
				logging.String("market-id", m.GetID()),
				logging.Error(err1))
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

	// from here we may have assigned some margin.
	// we add the check to roll it back in case we have a 0 positions after this
	defer m.releaseMarginExcess(ctx, order.PartyID)

	// If we are not in an opening auction, apply fees
	var trades []*types.Trade
	// we're not in auction (not opening, not any other auction
	if !m.as.InAuction() {

		// first we call the order book to evaluate auction triggers and get the list of trades
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
	if confirmation == nil || err != nil {
		_, err := m.position.UnregisterOrder(order)
		if err != nil {
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
	if ((order.TimeInForce == types.Order_TIF_FOK ||
		order.TimeInForce == types.Order_TIF_IOC ||
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

	m.handleConfirmation(ctx, order, confirmation)

	m.broker.Send(events.NewOrderEvent(ctx, order))

	return confirmation, nil
}

func (m *Market) checkPriceAndGetTrades(ctx context.Context, order *types.Order) ([]*types.Trade, error) {
	trades, err := m.matching.GetTrades(order)
	if err == nil && len(trades) > 0 {
		err = m.pMonitor.CheckPrice(ctx, m.as, trades[len(trades)-1].Price, m.currentTime)
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

func (m *Market) handleConfirmation(ctx context.Context, order *types.Order, confirmation *types.OrderConfirmation) {
	if confirmation.PassiveOrdersAffected != nil {
		// Insert or update passive orders siting on the book
		for _, order := range confirmation.PassiveOrdersAffected {
			// set the `updatedAt` value as these orders have changed
			order.UpdatedAt = m.currentTime.UnixNano()
			m.broker.Send(events.NewOrderEvent(ctx, order))

			// If the order is a pegged order and it complete we must remove it from the pegged lists
			if order.PeggedOrder != nil {
				if order.Remaining == 0 || order.Status != types.Order_STATUS_ACTIVE {
					m.removePeggedOrder(order)
				}
			}
		}
	}

	if len(confirmation.Trades) > 0 {

		// Calculate and set current mark price
		m.setMarkPrice(confirmation.Trades[len(confirmation.Trades)-1])

		// Insert all trades resulted from the executed order
		tradeEvts := make([]events.Event, 0, len(confirmation.Trades))
		for idx, trade := range confirmation.Trades {
			trade.Id = fmt.Sprintf("%s-%010d", order.Id, idx)
			if order.Side == types.Side_SIDE_BUY {
				trade.BuyOrder = order.Id
				trade.SellOrder = confirmation.PassiveOrdersAffected[idx].Id
			} else {
				trade.SellOrder = order.Id
				trade.BuyOrder = confirmation.PassiveOrdersAffected[idx].Id
			}

			tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, *trade))

			// Update positions (this communicates with settlement via channel)
			m.position.Update(trade)
			// Record open inteterest change
			err := m.tsCalc.RecordOpenInterest(m.position.GetOpenInterest(), m.currentTime)
			if err != nil {
				m.log.Debug("unable record open interest",
					logging.String("market-id", m.GetID()),
					logging.Error(err))
			}
			// add trade to settlement engine for correct MTM settlement of individual trades
			m.settlement.AddTrade(trade)
		}
		m.broker.SendBatch(tradeEvts)

		// now let's get the transfers for MTM settlement
		evts := m.position.UpdateMarkPrice(m.markPrice)
		settle := m.settlement.SettleMTM(ctx, m.markPrice, evts)

		// Only process collateral and risk once per order, not for every trade
		margins := m.collateralAndRisk(ctx, settle)
		if len(margins) > 0 {

			transfers, closed, err := m.collateral.MarginUpdate(ctx, m.GetID(), margins)
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
		}
	}
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
	for _, o := range rmorders {
		o.UpdatedAt = m.currentTime.UnixNano()
		m.broker.Send(events.NewOrderEvent(ctx, o))
		if _, err := m.position.UnregisterOrder(o); err != nil {
			m.log.Error("unable to unregister order for a distressed party",
				logging.String("party-id", o.PartyID),
				logging.String("market-id", mktID),
				logging.String("order-id", o.Id),
			)
		}
	}

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
		MarketID:    m.GetID(),
		Remaining:   size,
		Status:      types.Order_STATUS_ACTIVE,
		PartyID:     networkPartyID,       // network is not a party as such
		Side:        types.Side_SIDE_SELL, // assume sell, price is zero in that case anyway
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   fmt.Sprintf("LS-%s", o.Id), // liquidity sourcing, reference the order which caused the problem
		TimeInForce: types.Order_TIF_FOK,        // this is an all-or-nothing order, so TIF == FOK
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
		MarketID:    marketID,
		Status:      types.Order_STATUS_FILLED,
		PartyID:     networkPartyID,
		Price:       settleOrder.Price,
		CreatedAt:   m.currentTime.UnixNano(),
		Reference:   "close-out distressed",
		TimeInForce: types.Order_TIF_FOK, // this is an all-or-nothing order, so TIF == FOK
		Type:        types.Order_TYPE_NETWORK,
	}

	asset, _ := m.mkt.GetAsset()
	marginLevels := types.MarginLevels{
		MarketID:  m.mkt.GetId(),
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
			MarketID:    marketID,
			Size:        tSize,
			Remaining:   0,
			Status:      types.Order_STATUS_FILLED,
			PartyID:     trader.Party(),
			Side:        tSide,             // assume sell, price is zero in that case anyway
			Price:       settleOrder.Price, // average price
			CreatedAt:   m.currentTime.UnixNano(),
			Reference:   fmt.Sprintf("distressed-%d-%s", i, initial.Id),
			TimeInForce: types.Order_TIF_FOK, // this is an all-or-nothing order, so TIF == FOK
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
			MarketID:  partyOrder.MarketID,
			Price:     partyOrder.Price,
			Size:      partyOrder.Size,
			Aggressor: order.Side, // we consider network to be agressor
			BuyOrder:  buyOrder.Id,
			SellOrder: sellOrder.Id,
			Buyer:     buyOrder.PartyID,
			Seller:    sellOrder.PartyID,
			Timestamp: partyOrder.CreatedAt,
			Type:      types.Trade_TYPE_NETWORK_CLOSE_OUT_BAD,
			SellerFee: sellSideFee,
			BuyerFee:  buySideFee,
		}
		tradeEvts = append(tradeEvts, events.NewTradeEvent(ctx, trade))

		// 0 out margins levels for this trader
		marginLevels.PartyID = trader.Party()
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

	// this is a rollback transfer to be used in case the order do not
	// trade and do not stay in the book to prevent for margin being
	// locked in the margin account forever
	var riskRollback *types.Transfer

	asset, err := m.mkt.GetAsset()
	if err != nil {
		return nil, errors.Wrap(err, "unable to get risk updates")
	}

	e, err := m.collateral.GetPartyMargin(pos, asset, m.GetID())
	if err != nil {
		return nil, err
	}

	// @TODO replace markPrice with intidicative uncross price in auction mode if available
	price := m.markPrice
	if m.as.InAuction() {
		if ip := m.matching.GetIndicativePrice(); ip != 0 {
			price = ip
		}
	}
	riskUpdate, err := m.collateralAndRiskForOrder(ctx, e, price, pos)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("unable to top up margin on new order",
				logging.String("party-id", order.PartyID),
				logging.String("market-id", order.MarketID),
				logging.Error(err),
			)
		}
		return nil, ErrMarginCheckInsufficient
	} else if riskUpdate != nil {
		// this should always be a increase to the InitialMargin
		// if it does fail, we need to return an error straight away
		transfer, closePos, err := m.collateral.MarginUpdateOnOrder(ctx, m.GetID(), riskUpdate)
		if err != nil {
			return nil, errors.Wrap(err, "unable to get risk updates")
		}
		evt := events.NewTransferResponse(ctx, []*types.TransferResponse{transfer})
		m.broker.Send(evt)

		if closePos != nil {
			// if closePose is not nil then we return an error as well, it means the trader did not have enough
			// monies to reach the InitialMargin

			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("party did not have enough collateral to reach the InitialMargin",
					logging.Order(*order),
					logging.String("market-id", m.GetID()))
			}

			return nil, ErrMarginCheckInsufficient
		}

		if len(transfer.Transfers) > 0 {
			// we create the rollback transfer here, so it can be used in case of.
			riskRollback = &types.Transfer{
				Owner: riskUpdate.Party(),
				Amount: &types.FinancialAmount{
					Amount: int64(transfer.Transfers[0].Amount),
					Asset:  riskUpdate.Asset(),
				},
				Type:      types.TransferType_TRANSFER_TYPE_MARGIN_HIGH,
				MinAmount: int64(transfer.Transfers[0].Amount),
			}
		}
	}
	return riskRollback, nil
}

// this function handles moving money after settle MTM + risk margin updates
// but does not move the money between trader accounts (ie not to/from margin accounts after risk)
func (m *Market) collateralAndRiskForOrder(ctx context.Context, e events.Margin, price uint64, pos *positions.MarketPosition) (events.Risk, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "collateralAndRiskForOrder")
	defer timer.EngineTimeCounterAdd()

	// let risk engine do its thing here - it returns a slice of money that needs
	// to be moved to and from margin accounts
	riskUpdate, err := m.risk.UpdateMarginOnNewOrder(ctx, e, price)
	if err != nil {
		return nil, err
	}
	if riskUpdate == nil {
		return nil, nil
	}

	return riskUpdate, nil
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

	// Create a slive ready to store the generated events in
	evts := make([]events.Event, 0, len(m.parkedOrders)+len(cancellations))

	// Check the parked order list of any orders from that same party
	var parkedCancels []*types.OrderCancellationConfirmation
	for _, order := range m.parkedOrders {
		if order.PartyID == partyID {
			order.Status = types.Order_STATUS_CANCELLED
			m.removePeggedOrder(order)
			order.UpdatedAt = m.currentTime.UnixNano()
			evts = append(evts, events.NewOrderEvent(ctx, order))

			parkedCancel := &types.OrderCancellationConfirmation{
				Order: order,
			}
			parkedCancels = append(parkedCancels, parkedCancel)
		}
	}

	for _, cancellation := range cancellations {
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

// CancelOrder cancels the given order
func (m *Market) CancelOrder(ctx context.Context, partyID, orderID string) (*types.OrderCancellationConfirmation, error) {
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
	if order.PartyID != partyID {
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

	// If this is a pegged order, remove from pegged and parked lists
	if order.PeggedOrder != nil {
		m.removePeggedOrder(order)
		order.Status = types.Order_STATUS_CANCELLED
	}

	// Publish the changed order details
	order.UpdatedAt = m.currentTime.UnixNano()
	m.broker.Send(events.NewOrderEvent(ctx, order))

	m.checkForReferenceMoves(ctx)
	return &types.OrderCancellationConfirmation{Order: order}, nil
}

// parkOrderAndAdd removes the order from the orderbook and adds it to the parked list
func (m *Market) parkOrderAndAdd(ctx context.Context, order *types.Order) {
	m.parkOrder(ctx, order)
	m.parkedOrders = append(m.parkedOrders, order)
}

// parkOrder removes the given order from the orderbook
// parkOrder will panic if it encounters errors, which means that it reached an
// invalid state.
func (m *Market) parkOrder(ctx context.Context, order *types.Order) {
	defer m.releaseMarginExcess(ctx, order.PartyID)

	if err := m.matching.RemoveOrder(order); err != nil {
		m.log.Fatal("Failure to remove order from matching engine",
			logging.String("party-id", order.PartyID),
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
		m.log.Fatal("Failure unregistering order in positions engine (parking)",
			logging.Order(*order),
			logging.Error(err))
	}
}

// CancelOrderByID locates order by its Id and cancels it
// @TODO This function should not exist. Needs to be removed
func (m *Market) CancelOrderByID(orderID string) (*types.OrderCancellationConfirmation, error) {
	ctx := context.TODO()
	order, _, err := m.getOrderByID(orderID)
	if err != nil {
		return nil, err
	}
	return m.CancelOrder(ctx, order.PartyID, order.Id)
}

// AmendOrder amend an existing order from the order book
func (m *Market) AmendOrder(ctx context.Context, orderAmendment *types.OrderAmendment) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "AmendOrder")
	defer timer.EngineTimeCounterAdd()

	// Verify that the market is not closed
	if m.closed {
		return nil, ErrMarketClosed
	}

	// Try and locate the existing order specified on the
	// order book in the matching engine for this market
	existingOrder, _, err := m.getOrderByID(orderAmendment.OrderID)
	if err != nil {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid order ID",
				logging.String("id", orderAmendment.GetOrderID()),
				logging.String("party", orderAmendment.GetPartyID()),
				logging.String("market", orderAmendment.GetMarketID()),
				logging.Error(err))
		}
		return nil, types.ErrInvalidOrderID
	}

	// We can only amend this order if we created it
	if existingOrder.PartyID != orderAmendment.PartyID {
		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Invalid party ID",
				logging.String("original party id:", existingOrder.PartyID),
				logging.String("amend party id:", orderAmendment.PartyID))
		}
		return nil, types.ErrInvalidPartyID
	}

	// Validate Market
	if existingOrder.MarketID != m.mkt.Id {
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

	// if remaining is reduces <= 0, then order is cancelled
	if amendedOrder.Remaining <= 0 {
		confirm, err := m.CancelOrder(
			ctx, existingOrder.PartyID, existingOrder.Id)
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
		// Update the existing message in place before we cancel it
		m.orderAmendInPlace(existingOrder, amendedOrder)
		cancellation, err := m.matching.CancelOrder(amendedOrder)
		if cancellation == nil || err != nil {
			if m.log.GetLevel() == logging.DebugLevel {
				m.log.Debug("Failure to cancel order from matching engine",
					logging.String("party-id", amendedOrder.PartyID),
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
				// Remove from parked list
				for i, order := range m.parkedOrders {
					if order.Id == amendedOrder.Id {
						copy(m.parkedOrders[i:], m.parkedOrders[i+1:])
						m.parkedOrders[len(m.parkedOrders)-1] = nil
						m.parkedOrders = m.parkedOrders[:len(m.parkedOrders)-1]
						return orderConf, err
					}
				}
			}
		}
	}

	// from here these are the normal amendment
	var priceShift, sizeIncrease, sizeDecrease, expiryChange, timeInForceChange bool

	if amendedOrder.Price != existingOrder.Price {
		priceShift = true
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

	// Perform check and allocate margin
	// ignore rollback return here, as if we amend it means the order
	// is already on the book, not rollback will be needed, the margin
	// will be updated later on for sure.

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

	// if increase in size or change in price
	// ---> DO atomic cancel and submit
	if priceShift || sizeIncrease {
		confirmation, err := m.orderCancelReplace(ctx, existingOrder, amendedOrder)
		if err == nil {
			m.handleConfirmation(ctx, amendedOrder, confirmation)
			m.broker.Send(events.NewOrderEvent(ctx, amendedOrder))
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

// amendPeggedOrder amend an existing pegged order from the order book
// This does not need to perform all the checks of the full AmendOrder call
// as we know the order is valid already
func (m *Market) amendPeggedOrder(ctx context.Context, existingOrder *types.Order, price uint64) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(m.mkt.Id, "market", "amendPeggedOrder")
	defer timer.EngineTimeCounterAdd()

	amendedOrder := *existingOrder
	amendedOrder.Price = price

	// Update potential new position after the amend
	pos, err := m.position.AmendOrder(existingOrder, &amendedOrder)
	if err != nil {
		// adding order to the buffer first
		amendedOrder.Status = types.Order_STATUS_REJECTED
		amendedOrder.Reason = types.OrderError_ORDER_ERROR_INTERNAL_ERROR
		m.broker.Send(events.NewOrderEvent(ctx, &amendedOrder))

		if m.log.GetLevel() == logging.DebugLevel {
			m.log.Debug("Unable to amend potential trader position",
				logging.String("market-id", m.GetID()),
				logging.Error(err))
		}
		return nil, ErrMarginCheckFailed
	}

	// Perform check and allocate margin
	// ignore rollback return here, as if we amend it means the order
	// is already on the book, not rollback will be needed, the margin
	// will be updated later on for sure.
	if _, err = m.checkMarginForOrder(ctx, pos, &amendedOrder); err != nil {
		// Undo the position registering
		_, err1 := m.position.AmendOrder(&amendedOrder, existingOrder)
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

	var confirmation *types.OrderConfirmation
	if existingOrder.Status != types.Order_STATUS_PARKED {
		confirmation, err = m.orderCancelReplace(ctx, existingOrder, &amendedOrder)
		if err == nil {
			m.handleConfirmation(ctx, &amendedOrder, confirmation)
		}
	} else {
		confirmation = &types.OrderConfirmation{Order: existingOrder}

	}
	m.broker.Send(events.NewOrderEvent(ctx, &amendedOrder))
	*existingOrder = amendedOrder
	return confirmation, err
}

func (m *Market) validateOrderAmendment(
	order *types.Order,
	amendment *types.OrderAmendment,
) error {
	// check TIF and expiracy
	if amendment.TimeInForce == types.Order_TIF_GTT {
		if amendment.ExpiresAt == nil {
			return types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT
		}
		// if expiresAt is before or equal to created at
		// we return an error
		if amendment.ExpiresAt.Value <= order.CreatedAt {
			return types.OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT
		}
	} else if amendment.TimeInForce == types.Order_TIF_GTC {
		// this is cool, but we need to ensure and expiry is not set
		if amendment.ExpiresAt != nil {
			return types.OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT
		}
	} else if amendment.TimeInForce == types.Order_TIF_FOK ||
		amendment.TimeInForce == types.Order_TIF_IOC {
		// IOC and FOK are not acceptable for amend order
		return types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC
	} else if (amendment.TimeInForce == types.Order_TIF_GFN ||
		amendment.TimeInForce == types.Order_TIF_GFA) &&
		amendment.TimeInForce != order.TimeInForce {
		// We cannot amend to a GFA/GFN orders
		return types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN
	} else if (order.TimeInForce == types.Order_TIF_GFN ||
		order.TimeInForce == types.Order_TIF_GFA) &&
		(amendment.TimeInForce != order.TimeInForce &&
			amendment.TimeInForce != types.Order_TIF_UNSPECIFIED) {
		// We cannot amend from a GFA/GFN orders
		return types.OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN
	} else if order.PeggedOrder == nil {
		// We cannot change a pegged orders details on a non pegged order
		if amendment.PeggedOffset != nil ||
			amendment.PeggedReference != types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
			return types.OrderError_ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER
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
		MarketID:    existingOrder.MarketID,
		PartyID:     existingOrder.PartyID,
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
		order.PeggedOrder = &types.PeggedOrder{Reference: existingOrder.PeggedOrder.Reference,
			Offset: existingOrder.PeggedOrder.Offset}
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
	if amendment.TimeInForce != types.Order_TIF_UNSPECIFIED {
		order.TimeInForce = amendment.TimeInForce
		if amendment.TimeInForce != types.Order_TIF_GTT {
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
		if verr := m.validatePeggedOrder(ctx, order); verr != types.OrderError_ORDER_ERROR_NONE {
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
				m.log.Debug("Failed to cancel order from matching engine during CancelReplace",
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

		conf, err = m.matching.SubmitOrder(newOrder)
		// replace the trades in the confirmation to have
		// the ones with the fees embbeded
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

	expiredPegs := []types.Order{}
	for _, order := range m.expiringPeggedOrders.Expire(timestamp) {
		order := order

		// The pegged expiry orders are copies and do not reflect the
		// current state of the order, therefore we look it up
		originalOrder, _, err := m.getOrderByID(order.Id)
		if err == nil && originalOrder.Status != types.Order_STATUS_PARKED {
			m.unregisterOrder(&order)
		}
		m.removePeggedOrder(&order)
		originalOrder.Status = types.Order_STATUS_EXPIRED
		expiredPegs = append(expiredPegs, *originalOrder)
	}

	orderList := m.matching.RemoveExpiredOrders(timestamp)
	// need to remove the expired orders from the potentials positions
	for _, order := range orderList {
		order := order
		m.unregisterOrder(&order)
	}

	orderList = append(orderList, expiredPegs...)

	return orderList, nil
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

func (m *Market) getStaticMidPrice() (uint64, error) {
	bid, err := m.matching.GetBestStaticBidPrice()
	if err != nil {
		return 0, err
	}
	ask, err := m.matching.GetBestStaticAskPrice()
	if err != nil {
		return 0, err
	}
	return (bid + ask) / 2, nil
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
		newMid, _ := m.getStaticMidPrice()

		// Look for a move
		var changes uint8
		if newMid != m.lastMidPrice {
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
		m.lastMidPrice = newMid
		m.lastBestBidPrice = newBestBid
		m.lastBestAskPrice = newBestAsk

		// If we have any parked orders, see if we can get a
		// valid price for them and try to submit them
		if len(m.parkedOrders) > 0 {
			m.unparkAllPeggedOrders(ctx)
		}
	}
}

// GetPeggedOrderCount returns the number of pegged orders in the market
func (m *Market) GetPeggedOrderCount() int {
	return len(m.peggedOrders)
}

// GetParkedOrderCount returns hte number of parked orders in the market
func (m *Market) GetParkedOrderCount() int {
	return len(m.parkedOrders)
}

func (m *Market) addPeggedOrder(order *types.Order) {
	m.peggedOrders = append(m.peggedOrders, order)

	// expiring orders will be removed by RemoveExpiredOrders
	if order.IsPersistent() && order.ExpiresAt > 0 {
		m.expiringPeggedOrders.Insert(*order)
	}
}

// removePeggedOrder looks through the pegged and parked list
// and removes the matching order if found
func (m *Market) removePeggedOrder(order *types.Order) {
	for i, po := range m.peggedOrders {
		if po.Id == order.Id {
			// Remove item from slice
			copy(m.peggedOrders[i:], m.peggedOrders[i+1:])
			m.peggedOrders[len(m.peggedOrders)-1] = nil
			m.peggedOrders = m.peggedOrders[:len(m.peggedOrders)-1]
			break
		}
	}

	for i, po := range m.parkedOrders {
		if po.Id == order.Id {
			// Remove item from slice
			copy(m.parkedOrders[i:], m.parkedOrders[i+1:])
			m.parkedOrders[len(m.parkedOrders)-1] = nil
			m.parkedOrders = m.parkedOrders[:len(m.parkedOrders)-1]
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

func (m *Market) SubmitLiquidityProvision(ctx context.Context, sub *types.LiquidityProvisionSubmission, party, id string) error {
	return nil
}

func (m *Market) getTargetStake() float64 {
	rf, err := m.getRiskFactors()
	if err != nil {
		m.log.Debug("unable to get risk factors, can't calculate target")
		return 0
	}
	return m.tsCalc.GetTargetStake(*rf, m.currentTime)
}
