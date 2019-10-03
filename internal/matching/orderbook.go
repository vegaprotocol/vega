package matching

import (
	"fmt"
	"sort"
	"sync"

	"code.vegaprotocol.io/vega/internal/events"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/metrics"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	// ErrNotEnoughOrders signals that not enough orders were
	// in the book to achieve a given operation
	ErrNotEnoughOrders = errors.New("insufficient orders")
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
	expiringOrders  []types.Order // keep a list of all expiring trades, these will be in timestamp ascending order.
}

// NewOrderBook create an order book with a given name
// TODO(jeremy): At the moment it takes as a parameter the initialMarkPrice from the market
// framework. This is used in order to calculate the CloseoutPNL when there's no volume in the
// book. It's currently set to the lastTradedPrice, so once a trade happen it naturally get
// updated and the new markPrice will be used there.
func NewOrderBook(log *logging.Logger, config Config, marketID string,
	initialMarkPrice uint64, proRataMode bool) *OrderBook {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &OrderBook{
		log:             log,
		marketID:        marketID,
		cfgMu:           &sync.Mutex{},
		buy:             &OrderBookSide{log: log, proRataMode: proRataMode},
		sell:            &OrderBookSide{log: log, proRataMode: proRataMode},
		Config:          config,
		expiringOrders:  make([]types.Order, 0),
		lastTradedPrice: initialMarkPrice,
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
	vol := volume
	if side == types.Side_Buy {
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
	}
	for _, lvl := range b.sell.getLevels() {
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

// MarketOrderPrice return the price that would be applied for a market
// order based on the specified side.
// In the case of a Buy side, the highest sell price will be returned
// In the case of a Sell side, the lowest buy price will be returned
// TODO(jeremy): we won't be able to place an order if there is no order available
// in the meantime to make this function not failing we will return the
// initialMarkPrice / lastTradePrice in this case, this will fail later on when trying to
// place the order, by doing this we do not implement any more logic at this level
func (b *OrderBook) MarketOrderPrice(s types.Side) uint64 {
	if s == types.Side_Buy {
		p, err := b.sell.getHighestOrderPrice(types.Side_Sell)
		if err != nil {
			return b.lastTradedPrice
		}
		return p
	}
	p, err := b.buy.getLowestOrderPrice(types.Side_Buy)
	if err != nil {
		return b.lastTradedPrice
	}
	return p
}

// CancelOrder cancel an order that is active on an order book. Market and Order ID are validated, however the order must match
// the order on the book with respect to side etc. The caller will typically validate this by using a store, we should
// not trust that the external world can provide these values reliably.
func (b *OrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	// Validate Market
	if order.MarketID != b.marketID {
		b.log.Error("Market ID mismatch",
			logging.Order(*order),
			logging.String("order-book", b.marketID))
	}

	// Validate Order ID must be present
	if order.Id == "" || len(order.Id) < 4 {
		b.log.Error("Order ID missing or invalid",
			logging.Order(*order),
			logging.String("order-book", b.marketID))

		return nil, types.ErrInvalidOrderID
	}

	if order.Side == types.Side_Buy {
		if err := b.buy.RemoveOrder(order); err != nil {
			b.log.Error("Failed to remove order (buy side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return nil, types.ErrOrderRemovalFailure
		}
	} else {
		if err := b.sell.RemoveOrder(order); err != nil {
			b.log.Error("Failed to remove order (sell side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return nil, types.ErrOrderRemovalFailure
		}
	}

	// Important to mark the order as cancelled (and no longer active)
	order.Status = types.Order_Cancelled

	result := &types.OrderCancellationConfirmation{
		Order: order,
	}
	return result, nil
}

// AmendOrder amend an order which is an active order on the book
func (b *OrderBook) AmendOrder(order *types.Order) error {
	if err := b.validateOrder(order); err != nil {
		b.log.Error("Order validation failure",
			logging.Order(*order),
			logging.Error(err),
			logging.String("order-book", b.marketID))

		return err
	}

	if order.Side == types.Side_Buy {
		if err := b.buy.amendOrder(order); err != nil {
			b.log.Error("Failed to amend (buy side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return err
		}
	} else {
		if err := b.sell.amendOrder(order); err != nil {
			b.log.Error("Failed to amend (sell side)",
				logging.Order(*order),
				logging.Error(err),
				logging.String("order-book", b.marketID))

			return err
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
	trades, impactedOrders, lastTradedPrice := b.getOppositeSide(order.Side).uncross(order)
	if lastTradedPrice != 0 {
		b.lastTradedPrice = lastTradedPrice
	}

	// if state of the book changed show state
	if b.LogPriceLevelsDebug && len(trades) != 0 {
		b.PrintState("After uncross state:")
	}

	// if order is persistent type add to order book to the correct side
	if (order.TimeInForce == types.Order_GTC || order.TimeInForce == types.Order_GTT) && order.Remaining > 0 {

		// GTT orders need to be added to the expiring orders table, these orders will be removed when expired.
		if order.TimeInForce == types.Order_GTT {
			b.insertExpiringOrder(*order)
		}

		b.getSide(order.Side).addOrder(order, order.Side)

		if b.LogPriceLevelsDebug {
			b.PrintState("After addOrder state:")
		}
	}

	// did we fully fill the originating order?
	if order.Remaining == 0 {
		order.Status = types.Order_Filled
	}

	// update order statuses based on the order types if they didn't trade
	if (order.TimeInForce == types.Order_FOK || order.TimeInForce == types.Order_IOC) && order.Remaining == order.Size {
		order.Status = types.Order_Stopped
	}

	for idx := range impactedOrders {
		if impactedOrders[idx].Remaining == 0 {
			impactedOrders[idx].Status = types.Order_Filled

			// Ensure any fully filled impacted GTT orders are removed
			// from internal matching engine pending orders list
			if impactedOrders[idx].TimeInForce == types.Order_GTT {
				b.removePendingGttOrder(*impactedOrders[idx])
			}
		}
	}

	orderConfirmation := makeResponse(order, trades, impactedOrders)
	timer.EngineTimeCounterAdd()
	return orderConfirmation, nil
}

// DeleteOrder remove a given order on a given side from the book
func (b *OrderBook) DeleteOrder(order *types.Order) error {
	err := b.getSide(order.Side).RemoveOrder(order)
	return err
}

// RemoveExpiredOrders removes any GTT orders that will expire on or before the expiration timestamp (epoch+nano).
// expirationTimestamp must be of the format unix epoch seconds with nanoseconds e.g. 1544010789803472469.
// RemoveExpiredOrders returns a slice of Orders that were removed, internally it will remove the orders from the
// matching engine price levels. The returned orders will have an Order_Expired status, ready to update in stores.
func (b *OrderBook) RemoveExpiredOrders(expirationTimestamp int64) []types.Order {
	// expiringOrders are ordered, so it the first one ExpiresAt is bigger then the
	// expirationtimestamp, this means than no order is expired
	if len(b.expiringOrders) > 0 && b.expiringOrders[0].ExpiresAt > expirationTimestamp {
		return []types.Order{}
	}

	// expiring orders are ordered by expiration time.
	// so we'll search for the position where the expirationTimestamp would be in the slice
	// e.g: if our timestamp is 4
	// []int{1, 2, 3, 4, 4, 4, 4, 6, 7, 8}
	// ~~~~~~~~~~~~~~~~~~~~~~~~^
	// Also we add + 1 to the timestamp so it would find the last in the list,
	// if not the previous example would return
	// []int{1, 2, 3, 4, 4, 4, 4, 6, 7, 8}
	// ~~~~~~~~~~~~~~~^
	// by adding + 1 we actuall get everything which is stricly before the expirationTimestamp
	i := sort.Search(len(b.expiringOrders), func(i int) bool { return b.expiringOrders[i].ExpiresAt >= expirationTimestamp+1 })

	// make slice with the right size for the expired orders
	// then copy them
	expiredOrders := make([]types.Order, i)
	copy(expiredOrders, b.expiringOrders[:i])

	// delete the orders now
	for at := range expiredOrders {
		b.DeleteOrder(&expiredOrders[at])
		expiredOrders[at].Status = types.Order_Expired
	}

	// mem move all orders to be expired at the beginning of the slice, so
	// we do not need to reallocate in the future and we'll just reuse the actual slice.
	b.expiringOrders = b.expiringOrders[:copy(b.expiringOrders[0:], b.expiringOrders[i:])]

	if b.LogRemovedOrdersDebug {
		b.log.Debug("Removed expired orders from order book",
			logging.String("order-book", b.marketID),
			logging.Int("expired-orders", len(expiredOrders)),
			logging.Int("remaining-orders", len(b.expiringOrders)))
	}

	return expiredOrders
}

// RemoveDistressedOrders remove from the book all order holding distressed positions
func (b *OrderBook) RemoveDistressedOrders(traders []events.MarketPosition) error {
	for _, trader := range traders {
		total := trader.Buy() + trader.Sell()
		if total == 0 {
			continue
		}
		orders := make([]*types.Order, 0, int(total))
		if trader.Buy() > 0 {
			i := trader.Buy()
			for _, l := range b.buy.levels {
				rm := l.getOrdersByTrader(trader.Party())
				i -= int64(len(rm))
				orders = append(orders, rm...)
				if i == 0 {
					break
				}
			}
		}
		if trader.Sell() > 0 {
			i := trader.Sell()
			for _, l := range b.sell.levels {
				rm := l.getOrdersByTrader(trader.Party())
				i -= int64(len(rm))
				orders = append(orders, rm...)
				if i == 0 {
					break
				}
			}
		}
		for _, o := range orders {
			confirm, err := b.CancelOrder(o)
			if err != nil {
				b.log.Error(
					"Failed to cancel a given order for trader",
					logging.Order(*o),
					logging.String("trader", trader.Party()),
					logging.Error(err),
				)
				// let's see whether we need to handle this further down
				continue
			}
			if err := b.DeleteOrder(confirm.Order); err != nil {
				b.log.Error(
					"Failed to remove cancelled order",
					logging.Order(*confirm.Order),
					logging.Error(err),
				)
			}
		}
	}
	return nil
}

func (b OrderBook) getSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.Side_Buy {
		return b.buy
	}
	return b.sell
}

func (b *OrderBook) getOppositeSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.Side_Buy {
		return b.sell
	}
	return b.buy
}

func (b *OrderBook) insertExpiringOrder(ord types.Order) {
	timer := metrics.NewTimeCounter(b.marketID, "matching", "insertExpiringOrder")
	if len(b.expiringOrders) <= 0 {
		b.expiringOrders = append(b.expiringOrders, ord)
		timer.EngineTimeCounterAdd()
		return
	}

	// first find the position where this should be inserted
	i := sort.Search(len(b.expiringOrders), func(i int) bool { return b.expiringOrders[i].ExpiresAt >= ord.ExpiresAt })

	// append new elem first to make sure we have enough place
	// this would reallocate sufficiently then
	// no risk of this being a empty order, as it's overwritten just next with
	// the slice insert
	b.expiringOrders = append(b.expiringOrders, types.Order{})
	copy(b.expiringOrders[i+1:], b.expiringOrders[i:])
	b.expiringOrders[i] = ord
	timer.EngineTimeCounterAdd()
}

func (b OrderBook) removePendingGttOrder(order types.Order) bool {
	// this will return the index of the first order with an expiry matching the order expiry
	// e.g: []int{1, 2, 3, 4, 4, 4, 5, 6, 7, 8, 9}
	//                     ^ this will return index 3
	i := sort.Search(len(b.expiringOrders), func(i int) bool { return b.expiringOrders[i].ExpiresAt >= order.ExpiresAt })
	if i < len(b.expiringOrders) {
		// orders with the same expiry found, now we need to iterate over the result to find
		// an order with the same expiry and may the order ID
		for i <= len(b.expiringOrders) && b.expiringOrders[i].ExpiresAt == order.ExpiresAt {
			if b.expiringOrders[i].ExpiresAt == order.ExpiresAt {
				// we found our order, let's remove it
				b.expiringOrders = b.expiringOrders[:i+copy(b.expiringOrders[i:], b.expiringOrders[i+1:])]
				return true
			}
		}
	}
	return false
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
	b.log.Debug(fmt.Sprintf("%s", types))
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
