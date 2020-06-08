package matching

import (
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrNotEnoughOrders signals that not enough orders were
	// in the book to achieve a given operation
	ErrNotEnoughOrders   = errors.New("insufficient orders")
	ErrOrderDoesNotExist = errors.New("order does not exist")
	ErrInvalidVolume     = errors.New("invalid volume")
)

// OrderBook represents the book holding all orders in the system.
type OrderBook struct {
	log *logging.Logger
	Config

	cfgMu           *sync.Mutex
	marketID        string
	buy             *OrderBookSide
	sell            *OrderBookSide
	lastTradedPrice uint64
	latestTimestamp int64
	expiringOrders  *ExpiringOrders
	ordersByID      map[string]*types.Order
}

func isPersistent(o *types.Order) bool {
	return o.GetType() == types.Order_LIMIT &&
		(o.GetTimeInForce() == types.Order_TIF_GTC || o.GetTimeInForce() == types.Order_TIF_GTT)
}

// NewOrderBook create an order book with a given name
// TODO(jeremy): At the moment it takes as a parameter the initialMarkPrice from the market
// framework. This is used in order to calculate the CloseoutPNL when there's no volume in the
// book. It's currently set to the lastTradedPrice, so once a trade happen it naturally get
// updated and the new markPrice will be used there.
func NewOrderBook(log *logging.Logger, config Config, marketID string,
	initialMarkPrice uint64) *OrderBook {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &OrderBook{
		log:             log,
		marketID:        marketID,
		cfgMu:           &sync.Mutex{},
		buy:             &OrderBookSide{log: log},
		sell:            &OrderBookSide{log: log},
		Config:          config,
		lastTradedPrice: initialMarkPrice,
		expiringOrders:  NewExpiringOrders(),
		ordersByID:      map[string]*types.Order{},
	}
}

// ReloadConf is used in order to reload the internal configuration of
// the OrderBook
func (b *OrderBook) ReloadConf(cfg Config) {
	b.log.Info("reloading configuration")
	if b.log.GetLevel() != cfg.Level.Get() {
		b.log.Info("updating log level",
			logging.String("old", b.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		b.log.SetLevel(cfg.Level.Get())
	}

	b.cfgMu.Lock()
	b.Config = cfg
	b.cfgMu.Unlock()
}

// GetCloseoutPrice returns the exit price which would be achieved for a given
// volume and give side of the book
func (b *OrderBook) GetCloseoutPrice(volume uint64, side types.Side) (uint64, error) {
	var (
		price uint64
		err   error
	)

	if volume == 0 {
		return 0, ErrInvalidVolume
	}
	vol := volume
	if side == types.Side_SIDE_SELL {
		levels := b.sell.getLevels()
		for i := len(levels) - 1; i >= 0; i-- {
			lvl := levels[i]
			if lvl.volume >= vol {
				price += lvl.price * vol
				return price / volume, err
			}
			price += lvl.price * lvl.volume
			vol -= lvl.volume
		}
		// at this point, we should check vol, make sure it's 0, if not return an error to indicate something is wrong
		// still return the price for the volume we could close out, so the caller can make a decision on what to do
		if vol != 0 {
			err = ErrNotEnoughOrders
			// TODO(jeremy): there's no orders in the book so return the markPrice
			// this is a temporary fix for nicenet and this behaviour will need
			// to be properaly specified and handled in the future.
			if vol == volume {
				return b.lastTradedPrice, err
			}
		}
		return price / (volume - vol), err
	} else {
		levels := b.buy.getLevels()
		for i := len(levels) - 1; i >= 0; i-- {
			lvl := levels[i]
			if lvl.volume >= vol {
				price += lvl.price * vol
				return price / volume, err
			}
			price += lvl.price * lvl.volume
			vol -= lvl.volume
		}
		// if we reach this point, chances are vol != 0, in which case we should return an error along with the price
		if vol != 0 {
			err = ErrNotEnoughOrders
			// TODO(jeremy): there's no orders in the book so return the markPrice
			// this is a temporary fix for nice-net and this behaviour will need
			// to be properly specified and handled in the future.
			if vol == volume {
				return b.lastTradedPrice, err
			}
		}
		return price / (volume - vol), err
	}
}

// BestBidPriceAndVolume : Return the best bid and volume for the buy side of the book
func (b *OrderBook) BestBidPriceAndVolume() (uint64, uint64) {
	return b.buy.BestPriceAndVolume(types.Side_SIDE_BUY)
}

// BestOfferPriceAndVolume : Return the best bid and volume for the sell side of the book
func (b *OrderBook) BestOfferPriceAndVolume() (uint64, uint64) {
	return b.sell.BestPriceAndVolume(types.Side_SIDE_SELL)
}

// CancelOrder cancel an order that is active on an order book. Market and Order ID are validated, however the order must match
// the order on the book with respect to side etc. The caller will typically validate this by using a store, we should
// not trust that the external world can provide these values reliably.
func (b *OrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	// Validate Market
	if order.MarketID != b.marketID {
		if b.log.GetLevel() == logging.DebugLevel {
			b.log.Debug("Market ID mismatch",
				logging.Order(*order),
				logging.String("order-book", b.marketID))
		}
		return nil, types.OrderError_INVALID_MARKET_ID
	}

	// Validate Order ID must be present
	if err := validateOrderID(order.Id); err != nil {
		if b.log.GetLevel() == logging.DebugLevel {
			b.log.Debug("Order ID missing or invalid",
				logging.Order(*order),
				logging.String("order-book", b.marketID))
		}
		return nil, err
	}

	order, err := b.DeleteOrder(order)
	if err != nil {
		return nil, err
	}

	// Important to mark the order as cancelled (and no longer active)
	order.Status = types.Order_STATUS_CANCELLED

	result := &types.OrderCancellationConfirmation{
		Order: order,
	}
	return result, nil
}

// AmendOrder amend an order which is an active order on the book
func (b *OrderBook) AmendOrder(originalOrder, amendedOrder *types.Order) error {
	if originalOrder == nil {
		return types.ErrOrderNotFound
	}

	// If the creation date for the 2 orders is different, something went wrong
	if originalOrder.CreatedAt != amendedOrder.CreatedAt {
		return types.ErrOrderOutOfSequence
	}

	if err := b.validateOrder(amendedOrder); err != nil {
		if b.log.GetLevel() == logging.DebugLevel {
			b.log.Debug("Order validation failure",
				logging.Order(*amendedOrder),
				logging.Error(err),
				logging.String("order-book", b.marketID))
		}
		return err
	}

	if amendedOrder.Side == types.Side_SIDE_BUY {
		if err := b.buy.amendOrder(amendedOrder); err != nil {
			if b.log.GetLevel() == logging.DebugLevel {
				b.log.Debug("Failed to amend (buy side)",
					logging.Order(*amendedOrder),
					logging.Error(err),
					logging.String("order-book", b.marketID))
			}
			return err
		}
	} else {
		if err := b.sell.amendOrder(amendedOrder); err != nil {
			if b.log.GetLevel() == logging.DebugLevel {
				b.log.Debug("Failed to amend (sell side)",
					logging.Order(*amendedOrder),
					logging.Error(err),
					logging.String("order-book", b.marketID))
			}
			return err
		}
	}

	// If we have changed the ExpiresAt or TIF then update Expiry table
	if originalOrder.ExpiresAt != amendedOrder.ExpiresAt ||
		originalOrder.TimeInForce != amendedOrder.TimeInForce {
		b.removePendingGttOrder(*originalOrder)
		if amendedOrder.TimeInForce == types.Order_TIF_GTT {
			b.insertExpiringOrder(*amendedOrder)
		}
	}
	return nil
}

// SubmitOrder Add an order and attempt to uncross the book, returns a TradeSet protobuf message object
func (b *OrderBook) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	timer := metrics.NewTimeCounter(b.marketID, "matching", "SubmitOrder")

	if err := b.validateOrder(order); err != nil {
		timer.EngineTimeCounterAdd()
		return nil, err
	}

	if order.CreatedAt > b.latestTimestamp {
		b.latestTimestamp = order.CreatedAt
	}

	if b.LogPriceLevelsDebug {
		b.PrintState("Entry state:")
	}

	// uncross with opposite
	trades, impactedOrders, lastTradedPrice, err := b.getOppositeSide(order.Side).uncross(order)
	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	// if state of the book changed show state
	if b.LogPriceLevelsDebug && len(trades) != 0 {
		b.PrintState("After uncross state:")
	}

	// if order is persistent type add to order book to the correct side
	// and we did not hit a error / wash trade error
	if isPersistent(order) && order.Remaining > 0 && err == nil {

		// GTT orders need to be added to the expiring orders table, these orders will be removed when expired.
		if order.TimeInForce == types.Order_TIF_GTT {
			b.insertExpiringOrder(*order)
		}

		b.getSide(order.Side).addOrder(order, order.Side)

		if b.LogPriceLevelsDebug {
			b.PrintState("After addOrder state:")
		}
	}

	// Was the aggressive order fully filled?
	if order.Remaining == 0 {
		order.Status = types.Order_STATUS_FILLED
	}

	// What is an Immediate or Cancel Order?
	// An immediate or cancel order (IOC) is an order to buy or sell that executes all
	// or part immediately and cancels any unfilled portion of the order.
	if order.TimeInForce == types.Order_TIF_IOC && order.Remaining > 0 {
		// Stopped as not filled at all
		if order.Remaining == order.Size {
			order.Status = types.Order_STATUS_STOPPED
		} else {
			// IOC so we set status as Cancelled.
			order.Status = types.Order_STATUS_PARTIALLY_FILLED
		}
	}

	// What is Fill Or Kill?
	// Fill or kill (FOK) is a type of time-in-force designation used in trading that instructs
	// the protocol to execute an order immediately and completely or not at all.
	// The order must be filled in its entirety or cancelled (killed).
	if order.TimeInForce == types.Order_TIF_FOK && order.Remaining == order.Size {
		// FOK and didnt trade at all we set status as Stopped
		order.Status = types.Order_STATUS_STOPPED
	}

	for idx := range impactedOrders {
		if impactedOrders[idx].Remaining == 0 {
			impactedOrders[idx].Status = types.Order_STATUS_FILLED

			// Ensure any fully filled impacted GTT orders are removed
			// from internal matching engine pending orders list
			if impactedOrders[idx].TimeInForce == types.Order_TIF_GTT {
				b.removePendingGttOrder(*impactedOrders[idx])
			}

			// delete from lookup table
			delete(b.ordersByID, impactedOrders[idx].Id)
		}
	}

	// if we did hit a wash trade, set the status to stopped
	if err != nil && err == ErrWashTrade {
		order.Status = types.Order_STATUS_STOPPED
	}

	if order.Status == types.Order_STATUS_ACTIVE {
		b.ordersByID[order.Id] = order
	}

	orderConfirmation := makeResponse(order, trades, impactedOrders)
	timer.EngineTimeCounterAdd()
	return orderConfirmation, nil
}

// DeleteOrder remove a given order on a given side from the book
func (b *OrderBook) DeleteOrder(
	order *types.Order) (*types.Order, error) {
	dorder, err := b.getSide(order.Side).RemoveOrder(order)
	if err != nil {
		if b.log.GetLevel() == logging.DebugLevel {
			b.log.Debug("Failed to remove order",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))
		}
		return nil, types.ErrOrderRemovalFailure
	}
	delete(b.ordersByID, order.Id)
	return dorder, err
}

// RemoveExpiredOrders removes any GTT orders that will expire on or before the expiration timestamp (epoch+nano).
// expirationTimestamp must be of the format unix epoch seconds with nanoseconds e.g. 1544010789803472469.
// RemoveExpiredOrders returns a slice of Orders that were removed, internally it will remove the orders from the
// matching engine price levels. The returned orders will have an Order_Expired status, ready to update in stores.
func (b *OrderBook) RemoveExpiredOrders(expirationTimestamp int64) []types.Order {
	expiredOrders := b.expiringOrders.Expire(expirationTimestamp)
	if len(expiredOrders) <= 0 {
		return nil
	}
	out := make([]types.Order, 0, len(expiredOrders))

	// delete the orders now
	for at := range expiredOrders {
		order, err := b.DeleteOrder(&expiredOrders[at])
		if err == nil {
			// this may be not nil because the expiring order was cancelled before.
			// so it was already deleted, we do not remove them from the expiringOrders
			// when they get cancelled as this would required unnecessary computation that
			// can be delayed for later.
			order.Status = types.Order_STATUS_EXPIRED
			out = append(out, *order)
		}
	}

	if b.LogRemovedOrdersDebug {
		b.log.Debug("Removed expired orders from order book",
			logging.String("order-book", b.marketID),
			logging.Int("expired-orders", len(expiredOrders)))
	}

	return out
}

// GetOrderByID returns order by its ID (IDs are not expected to collide within same market)
func (b *OrderBook) GetOrderByID(orderID string) (*types.Order, error) {
	if err := validateOrderID(orderID); err != nil {
		if b.log.GetLevel() == logging.DebugLevel {
			b.log.Debug("Order ID missing or invalid",
				logging.String("order-id", orderID))
		}
		return nil, err
	}
	order, exists := b.ordersByID[orderID]
	if !exists {
		return nil, ErrOrderDoesNotExist
	}
	return order, nil
}

// RemoveDistressedOrders remove from the book all order holding distressed positions
func (b *OrderBook) RemoveDistressedOrders(
	parties []events.MarketPosition) ([]*types.Order, error) {
	rmorders := []*types.Order{}

	for _, party := range parties {
		orders := []*types.Order{}
		for _, l := range b.buy.levels {
			rm := l.getOrdersByParty(party.Party())
			orders = append(orders, rm...)
		}
		for _, l := range b.sell.levels {
			rm := l.getOrdersByParty(party.Party())
			orders = append(orders, rm...)
		}
		for _, o := range orders {
			confirm, err := b.CancelOrder(o)
			if err != nil {
				if b.log.GetLevel() == logging.DebugLevel {
					b.log.Debug(
						"Failed to cancel a given order for party",
						logging.Order(*o),
						logging.String("party", party.Party()),
						logging.Error(err))
				}
				// let's see whether we need to handle this further down
				continue
			}
			// here we set the status of the order as stopped as the system triggered it as well.
			confirm.Order.Status = types.Order_STATUS_STOPPED
			rmorders = append(rmorders, confirm.Order)
		}
	}
	return rmorders, nil
}

func (b OrderBook) getSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.Side_SIDE_BUY {
		return b.buy
	}
	return b.sell
}

func (b *OrderBook) getOppositeSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.Side_SIDE_BUY {
		return b.sell
	}
	return b.buy
}

func (b *OrderBook) insertExpiringOrder(ord types.Order) {
	timer := metrics.NewTimeCounter(b.marketID, "matching", "insertExpiringOrder")
	b.expiringOrders.Insert(ord)
	timer.EngineTimeCounterAdd()
}

func (b OrderBook) removePendingGttOrder(order types.Order) bool {
	return b.expiringOrders.RemoveOrder(order)
}

func makeResponse(order *types.Order, trades []*types.Trade, impactedOrders []*types.Order) *types.OrderConfirmation {

	return &types.OrderConfirmation{
		Order:                 order,
		PassiveOrdersAffected: impactedOrders,
		Trades:                trades,
	}
}

// PrintState prints the actual state of the book.
// this should be use only in debug / non production environment as it
// rely a lot on logging
func (b *OrderBook) PrintState(types string) {
	b.log.Debug("PrintState",
		logging.String("types", types))
	b.log.Debug("------------------------------------------------------------")
	b.log.Debug("                        BUY SIDE                            ")
	for _, priceLevel := range b.buy.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print(b.log)
		}
	}
	b.log.Debug("------------------------------------------------------------")
	b.log.Debug("                        SELL SIDE                           ")
	for _, priceLevel := range b.sell.getLevels() {
		if len(priceLevel.orders) > 0 {
			priceLevel.print(b.log)
		}
	}
	b.log.Debug("------------------------------------------------------------")
}
