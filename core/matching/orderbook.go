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

package matching

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

var (
	// ErrNotEnoughOrders signals that not enough orders were
	// in the book to achieve a given operation.
	ErrNotEnoughOrders   = errors.New("insufficient orders")
	ErrOrderDoesNotExist = errors.New("order does not exist")
	ErrInvalidVolume     = errors.New("invalid volume")
	ErrNoBestBid         = errors.New("no best bid")
	ErrNoBestAsk         = errors.New("no best ask")
	ErrNotCrossed        = errors.New("not crossed")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/matching OffbookSource
type OffbookSource interface {
	BestPricesAndVolumes() (*num.Uint, uint64, *num.Uint, uint64)
	SubmitOrder(agg *types.Order, inner, outer *num.Uint) []*types.Order
	NotifyFinished()
	OrderbookShape(st, nd *num.Uint, id *string) ([]*types.Order, []*types.Order)
}

// OrderBook represents the book holding all orders in the system.
type OrderBook struct {
	log *logging.Logger
	Config

	cfgMu                    *sync.Mutex
	marketID                 string
	buy                      *OrderBookSide
	sell                     *OrderBookSide
	lastTradedPrice          *num.Uint
	latestTimestamp          int64
	ordersByID               map[string]*types.Order
	ordersPerParty           map[string]map[string]struct{}
	auction                  bool
	batchID                  uint64
	indicativePriceAndVolume *IndicativePriceAndVolume
	snapshot                 *types.PayloadMatchingBook
	stopped                  bool // if true then we should stop creating snapshots

	// we keep track here of which type of orders are in the orderbook so we can quickly
	// find an order of a certain type. These get updated when orders are added or removed from the book.
	peggedOrders *peggedOrders

	peggedOrdersCount uint64
	peggedCountNotify func(int64)
}

// CumulativeVolumeLevel represents the cumulative volume at a price level for both bid and ask.
type CumulativeVolumeLevel struct {
	price               *num.Uint
	bidVolume           uint64
	askVolume           uint64
	cumulativeBidVolume uint64
	cumulativeAskVolume uint64
	maxTradableAmount   uint64

	// keep track of how much of the cumulative volume is from AMMs
	cumulativeBidOffbook uint64
	cumulativeAskOffbook uint64
}

func (b *OrderBook) Hash() []byte {
	return crypto.Hash(append(b.buy.Hash(), b.sell.Hash()...))
}

// NewOrderBook create an order book with a given name.
func NewOrderBook(log *logging.Logger, config Config, marketID string, auction bool, peggedCountNotify func(int64)) *OrderBook {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &OrderBook{
		log:             log,
		marketID:        marketID,
		cfgMu:           &sync.Mutex{},
		buy:             &OrderBookSide{log: log, side: types.SideBuy},
		sell:            &OrderBookSide{log: log, side: types.SideSell},
		Config:          config,
		ordersByID:      map[string]*types.Order{},
		auction:         auction,
		batchID:         0,
		ordersPerParty:  map[string]map[string]struct{}{},
		lastTradedPrice: num.UintZero(),
		snapshot: &types.PayloadMatchingBook{
			MatchingBook: &types.MatchingBook{
				MarketID: marketID,
			},
		},
		peggedOrders:      newPeggedOrders(),
		peggedOrdersCount: 0,
		peggedCountNotify: peggedCountNotify,
	}
}

// ReloadConf is used in order to reload the internal configuration of
// the OrderBook.
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

func (b *OrderBook) SetOffbookSource(obs OffbookSource) {
	b.buy.offbook = obs
	b.sell.offbook = obs
}

// GetOrderBookLevelCount returns the number of levels in the book.
func (b *OrderBook) GetOrderBookLevelCount() uint64 {
	return uint64(len(b.buy.levels) + len(b.sell.levels))
}

func (b *OrderBook) GetPeggedOrdersCount() uint64 {
	return b.peggedOrdersCount
}

// GetFillPrice returns the average price which would be achieved for a given
// volume and give side of the book.
func (b *OrderBook) GetFillPrice(volume uint64, side types.Side) (*num.Uint, error) {
	if b.auction {
		p := b.GetIndicativePrice()
		return p, nil
	}

	if volume == 0 {
		return num.UintZero(), ErrInvalidVolume
	}

	var (
		price  = num.UintZero()
		vol    = volume
		levels []*PriceLevel
	)
	if side == types.SideSell {
		levels = b.sell.getLevels()
	} else {
		levels = b.buy.getLevels()
	}

	for i := len(levels) - 1; i >= 0; i-- {
		lvl := levels[i]
		if lvl.volume >= vol {
			// price += lvl.price * vol
			price.Add(price, num.UintZero().Mul(lvl.price, num.NewUint(vol)))
			// return price / volume, nil
			return price.Div(price, num.NewUint(volume)), nil
		}
		price.Add(price, num.UintZero().Mul(lvl.price, num.NewUint(lvl.volume)))
		vol -= lvl.volume
	}
	// at this point, we should check vol, make sure it's 0, if not return an error to indicate something is wrong
	// still return the price for the volume we could close out, so the caller can make a decision on what to do
	if vol == volume {
		return b.lastTradedPrice.Clone(), ErrNotEnoughOrders
	}
	price.Div(price, num.NewUint(volume-vol))
	if vol != 0 {
		return price, ErrNotEnoughOrders
	}
	return price, nil
}

// EnterAuction Moves the order book into an auction state.
func (b *OrderBook) EnterAuction() []*types.Order {
	// Scan existing orders to see which ones can be kept or cancelled
	ordersToCancel := append(
		b.buy.getOrdersToCancel(true),
		b.sell.getOrdersToCancel(true)...,
	)

	// Set the market state
	b.auction = true
	b.indicativePriceAndVolume = NewIndicativePriceAndVolume(b.log, b.buy, b.sell)

	// Return all the orders that have been removed from the book and need to be cancelled
	return ordersToCancel
}

// LeaveAuction Moves the order book back into continuous trading state.
func (b *OrderBook) LeaveAuction(at time.Time) ([]*types.OrderConfirmation, []*types.Order, error) {
	// Update batchID
	b.batchID++

	ts := at.UnixNano()

	// Uncross the book
	uncrossedOrders, err := b.uncrossBook()
	if err != nil {
		return nil, nil, err
	}

	for _, uo := range uncrossedOrders {
		// refresh if its an iceberg, noop if not
		b.icebergRefresh(uo.Order)

		if uo.Order.Remaining == 0 {
			uo.Order.Status = types.OrderStatusFilled
			b.remove(uo.Order)
		}

		if uo.Order.GeneratedOffbook {
			uo.Order.CreatedAt = ts
		}

		uo.Order.UpdatedAt = ts
		for idx, po := range uo.PassiveOrdersAffected {
			po.UpdatedAt = ts

			// refresh if its an iceberg, noop if not
			b.icebergRefresh(po)

			// also remove the orders from lookup tables
			if uo.PassiveOrdersAffected[idx].Remaining == 0 {
				uo.PassiveOrdersAffected[idx].Status = types.OrderStatusFilled
				b.remove(po)
			}
		}
		for _, tr := range uo.Trades {
			tr.Timestamp = ts
			tr.Aggressor = types.SideUnspecified
		}
	}

	// Remove any orders that will not be valid in continuous trading
	// Return all the orders that have been cancelled from the book
	ordersToCancel := append(
		b.buy.getOrdersToCancel(false),
		b.sell.getOrdersToCancel(false)...,
	)

	for _, oc := range ordersToCancel {
		oc.UpdatedAt = ts
	}

	// Flip back to continuous
	b.auction = false
	b.indicativePriceAndVolume = nil

	return uncrossedOrders, ordersToCancel, nil
}

func (b OrderBook) InAuction() bool {
	return b.auction
}

// CanLeaveAuction calls canUncross with required trades and, if that returns false
// without required trades (which still permits leaving liquidity auction
// if canUncross with required trades returs true, both returns are true.
func (b *OrderBook) CanLeaveAuction() (withTrades, withoutTrades bool) {
	withTrades = b.canUncross(true)
	withoutTrades = withTrades
	if withTrades {
		return
	}
	withoutTrades = b.canUncross(false)
	return
}

// CanUncross - a clunky name for a somewhat clunky function: this checks if there will be LIMIT orders
// on the book after we uncross the book (at the end of an auction). If this returns false, the opening auction should be extended.
func (b *OrderBook) CanUncross() bool {
	return b.canUncross(true)
}

func (b *OrderBook) BidAndAskPresentAfterAuction() bool {
	return b.canUncross(false)
}

func (b *OrderBook) canUncross(requireTrades bool) bool {
	bb, err := b.GetBestBidPrice() // sell
	if err != nil {
		return false
	}
	ba, err := b.GetBestAskPrice() // buy
	if err != nil || bb.IsZero() || ba.IsZero() || (requireTrades && bb.LT(ba)) {
		return false
	}

	// check all buy price levels below ba, find limit orders
	buyMatch := false
	// iterate from the end, where best is
	for i := len(b.buy.levels) - 1; i >= 0; i-- {
		l := b.buy.levels[i]
		if l.price.LT(ba) {
			for _, o := range l.orders {
				// limit order && not just GFA found
				if o.Type == types.OrderTypeLimit && o.TimeInForce != types.OrderTimeInForceGFA {
					buyMatch = true
					break
				}
			}
		}
	}
	sellMatch := false
	for i := len(b.sell.levels) - 1; i >= 0; i-- {
		l := b.sell.levels[i]
		if l.price.GT(bb) {
			for _, o := range l.orders {
				if o.Type == types.OrderTypeLimit && o.TimeInForce != types.OrderTimeInForceGFA {
					sellMatch = true
					break
				}
			}
		}
	}
	// non-GFA orders outside the price range found on the book, we can uncross
	if buyMatch && sellMatch {
		return true
	}
	_, v, _, _ := b.GetIndicativePriceAndVolume()
	// no buy orders remaining on the book after uncrossing, it buyMatches exactly
	vol := uint64(0)
	if !buyMatch {
		for i := len(b.buy.levels) - 1; i >= 0; i-- {
			l := b.buy.levels[i]
			// buy orders are ordered ascending
			if l.price.LT(ba) {
				break
			}
			for _, o := range l.orders {
				vol += o.Remaining
				// we've filled the uncrossing volume, and found an order that is not GFA
				if vol > v && o.TimeInForce != types.OrderTimeInForceGFA {
					buyMatch = true
					break
				}
			}
		}
		if !buyMatch {
			return false
		}
	}
	// we've had to check buy side - sell side is fine
	if sellMatch {
		return true
	}

	vol = 0
	// for _, l := range b.sell.levels {
	// sell side is ordered descending
	for i := len(b.sell.levels) - 1; i >= 0; i-- {
		l := b.sell.levels[i]
		if l.price.GT(bb) {
			break
		}
		for _, o := range l.orders {
			vol += o.Remaining
			if vol > v && o.TimeInForce != types.OrderTimeInForceGFA {
				sellMatch = true
				break
			}
		}
	}

	return sellMatch
}

// GetIndicativePriceAndVolume Calculates the indicative price and volume of the order book without modifying the order book state.
func (b *OrderBook) GetIndicativePriceAndVolume() (retprice *num.Uint, retvol uint64, retside types.Side, offbookVolume uint64) {
	// Generate a set of price level pairs with their maximum tradable volumes
	cumulativeVolumes, maxTradableAmount, err := b.buildCumulativePriceLevels()
	if err != nil {
		if b.log.GetLevel() <= logging.DebugLevel {
			b.log.Debug("could not get cumulative price levels", logging.Error(err))
		}
		return num.UintZero(), 0, types.SideUnspecified, 0
	}

	// Pull out all prices that match that volume
	prices := make([]*num.Uint, 0, len(cumulativeVolumes))
	for _, value := range cumulativeVolumes {
		if value.maxTradableAmount == maxTradableAmount {
			prices = append(prices, value.price)
		}
	}

	// get the maximum volume price from the average of the maximum and minimum tradable price levels
	var (
		uncrossPrice = num.UintZero()
		uncrossSide  types.Side
	)
	if len(prices) > 0 {
		// uncrossPrice = (prices[len(prices)-1] + prices[0]) / 2
		uncrossPrice.Div(
			num.UintZero().Add(prices[len(prices)-1], prices[0]),
			num.NewUint(2),
		)
	}

	// See which side we should fully process when we uncross
	ordersToFill := int64(maxTradableAmount)
	for _, value := range cumulativeVolumes {
		ordersToFill -= int64(value.bidVolume)
		if ordersToFill == 0 {
			// Buys fill exactly, uncross from the buy side
			uncrossSide = types.SideBuy
			offbookVolume = value.cumulativeBidOffbook
			break
		} else if ordersToFill < 0 {
			// Buys are not exact, uncross from the sell side
			uncrossSide = types.SideSell
			offbookVolume = value.cumulativeAskOffbook
			break
		}
	}

	return uncrossPrice, maxTradableAmount, uncrossSide, offbookVolume
}

// GetIndicativePrice Calculates the indicative price of the order book without modifying the order book state.
func (b *OrderBook) GetIndicativePrice() (retprice *num.Uint) {
	// Generate a set of price level pairs with their maximum tradable volumes
	cumulativeVolumes, maxTradableAmount, err := b.buildCumulativePriceLevels()
	if err != nil {
		if b.log.GetLevel() <= logging.DebugLevel {
			b.log.Debug("could not get cumulative price levels", logging.Error(err))
		}
		return num.UintZero()
	}

	// Pull out all prices that match that volume
	prices := make([]*num.Uint, 0, len(cumulativeVolumes))
	for _, value := range cumulativeVolumes {
		if value.maxTradableAmount == maxTradableAmount {
			prices = append(prices, value.price)
		}
	}

	// get the maximum volume price from the average of the minimum and maximum tradable price levels
	if len(prices) > 0 {
		// return (prices[len(prices)-1] + prices[0]) / 2
		return num.UintZero().Div(
			num.UintZero().Add(prices[len(prices)-1], prices[0]),
			num.NewUint(2),
		)
	}
	return num.UintZero()
}

func (b *OrderBook) GetIndicativeTrades() ([]*types.Trade, error) {
	// Get the uncrossing price and which side has the most volume at that price
	price, volume, uncrossSide, offbookVolume := b.GetIndicativePriceAndVolume()

	// If we have no uncrossing price, we have nothing to do
	if price.IsZero() && volume == 0 {
		return nil, nil
	}

	var (
		uncrossOrders  []*types.Order
		uncrossingSide *OrderBookSide
		uncrossBound   *num.Uint
	)

	min, max := b.indicativePriceAndVolume.GetCrossedRegion()
	if uncrossSide == types.SideBuy {
		uncrossingSide = b.buy
		uncrossBound = min
	} else {
		uncrossingSide = b.sell
		uncrossBound = max
	}

	// extract uncrossing orders from all AMMs
	uncrossOrders = b.indicativePriceAndVolume.ExtractOffbookOrders(price, uncrossSide, offbookVolume)

	// the remaining volume should now come from the orderbook
	volume -= offbookVolume

	// Remove all the orders from that side of the book up to the given volume
	uncrossOrders = append(uncrossOrders, uncrossingSide.ExtractOrders(price, volume, false)...)
	opSide := b.getOppositeSide(uncrossSide)
	output := make([]*types.Trade, 0, len(uncrossOrders))
	trades, err := opSide.fakeUncrossAuction(uncrossOrders, uncrossBound)
	if err != nil {
		return nil, err
	}
	// Update all the trades to have the correct uncrossing price
	for _, t := range trades {
		t.Price = price.Clone()
	}
	output = append(output, trades...)

	return output, nil
}

// buildCumulativePriceLevels this returns a slice of all the price levels with the
// cumulative volume for each level. Also returns the max tradable size.
func (b *OrderBook) buildCumulativePriceLevels() ([]CumulativeVolumeLevel, uint64, error) {
	bestBid, err := b.GetBestBidPrice()
	if err != nil {
		return nil, 0, ErrNoBestBid
	}
	bestAsk, err := b.GetBestAskPrice()
	if err != nil {
		return nil, 0, ErrNoBestAsk
	}
	// Short circuit if the book is not crossed
	if bestBid.LT(bestAsk) || bestBid.IsZero() || bestAsk.IsZero() {
		return nil, 0, ErrNotCrossed
	}

	volume, maxTradableAmount := b.indicativePriceAndVolume.
		GetCumulativePriceLevels(bestBid, bestAsk)
	return volume, maxTradableAmount, nil
}

// Uncrosses the book to generate the maximum volume set of trades
// if removeOrders is set to true then matched orders get removed from the book.
func (b *OrderBook) uncrossBook() ([]*types.OrderConfirmation, error) {
	// Get the uncrossing price and which side has the most volume at that price
	price, volume, uncrossSide, offbookVolume := b.GetIndicativePriceAndVolume()

	// If we have no uncrossing price, we have nothing to do
	if price.IsZero() && volume == 0 {
		return nil, nil
	}

	var (
		uncrossingSide *OrderBookSide
		uncrossBound   *num.Uint
	)

	if uncrossSide == types.SideBuy {
		uncrossingSide = b.buy
	} else {
		uncrossingSide = b.sell
	}

	fmt.Println("WWW uncrossed bound", uncrossBound, uncrossingSide)
	//uncrossBound = nil

	// extract uncrossing orders from all AMMs
	uncrossOrders := b.indicativePriceAndVolume.ExtractOffbookOrders(price, uncrossSide, offbookVolume)

	// the remaining volume should now come from the orderbook
	volume -= offbookVolume
	fmt.Println("WWW crossing side", uncrossSide, "orderbook", volume, "offbook", offbookVolume, min, max)

	// Remove all the orders from that side of the book up to the given volume
	uncrossOrders = append(uncrossOrders, uncrossingSide.ExtractOrders(price, volume, true)...)

	pf, _ := num.UintFromDecimal(uncrossOrders[0].Price.ToDecimal().Div(uncrossOrders[0].OriginalPrice.ToDecimal()))
	oneTick := num.Max(num.UintOne(), pf)
	min, max := b.indicativePriceAndVolume.GetCrossedRegion()
	if uncrossSide == types.SideBuy {
		uncrossBound = num.UintZero().Sub(min, oneTick)
	} else {
		uncrossBound = num.UintZero().Add(max, oneTick)
	}
	fmt.Println("WWW new uncross bound", uncrossBound, "min/max", min, max)
	return b.uncrossBookSide(uncrossOrders, b.getOppositeSide(uncrossSide), price.Clone(), uncrossBound)
}

// Takes extracted order from a side of the book, and uncross them
// with the opposite side.
func (b *OrderBook) uncrossBookSide(
	uncrossOrders []*types.Order,
	opSide *OrderBookSide,
	price, uncrossBound *num.Uint,
) ([]*types.OrderConfirmation, error) {
	var (
		uncrossedOrder *types.OrderConfirmation
		allOrders      = make([]*types.OrderConfirmation, 0, len(uncrossOrders))
	)
	if len(uncrossOrders) == 0 {
		return nil, nil
	}

	defer b.buy.uncrossFinished()

	// get price factor, if price is 10,000, but market price is 100, this is 10,000/100 -> 100
	// so we can get the market price simply by doing price / (order.Price/ order.OriginalPrice)
	// as the asset decimals may be < market decimals, the calculation must be done in decimals.
	mPrice, _ := num.UintFromDecimal(price.ToDecimal().Div(uncrossOrders[0].Price.ToDecimal().Div(uncrossOrders[0].OriginalPrice.ToDecimal())))
	// Uncross each one
	for _, order := range uncrossOrders {
		// since all of uncrossOrders will be traded away and at the same uncrossing price
		// iceberg orders are sent in as their full value instead of refreshing at each step
		if order.IcebergOrder != nil {
			order.Remaining += order.IcebergOrder.ReservedRemaining
			order.IcebergOrder.ReservedRemaining = 0
		}

		// try to get the market price value from the order
		trades, affectedOrders, _, err := opSide.uncross(order, false, uncrossBound)
		if err != nil {
			return nil, err
		}
		// If the affected order is fully filled set the status
		for _, affectedOrder := range affectedOrders {
			if affectedOrder.Remaining == 0 {
				affectedOrder.Status = types.OrderStatusFilled
			}
		}

		fmt.Println("WWW order", order.Remaining, order.Size)
		// Update all the trades to have the correct uncrossing price
		for index := 0; index < len(trades); index++ {
			trades[index].Price = price.Clone()
			trades[index].MarketPrice = mPrice.Clone()
		}
		uncrossedOrder = &types.OrderConfirmation{Order: order, PassiveOrdersAffected: affectedOrders, Trades: trades}
		allOrders = append(allOrders, uncrossedOrder)
	}

	return allOrders, nil
}

func (b *OrderBook) GetOrdersPerParty(party string) []*types.Order {
	orderIDs := b.ordersPerParty[party]
	if len(orderIDs) <= 0 {
		return []*types.Order{}
	}

	orders := make([]*types.Order, 0, len(orderIDs))
	for oid := range orderIDs {
		orders = append(orders, b.ordersByID[oid])
	}

	// sort before returning since callers will assume they are in a deterministic order
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].ID < orders[j].ID
	})
	return orders
}

// BestBidPriceAndVolume : Return the best bid and volume for the buy side of the book.
func (b *OrderBook) BestBidPriceAndVolume() (*num.Uint, uint64, error) {
	price, volume, err := b.buy.BestPriceAndVolume()

	if b.buy.offbook != nil {
		oPrice, oVolume, _, _ := b.buy.offbook.BestPricesAndVolumes()

		// no off source volume, return the orderbook
		if oVolume == 0 {
			return price, volume, err
		}

		// no orderbook volume or AMM price is better
		if err != nil || oPrice.GT(price) {
			//nolint: nilerr
			return oPrice, oVolume, nil
		}

		// AMM price equals orderbook price, combined volumes
		if err == nil && oPrice.EQ(price) {
			oVolume += volume
			return oPrice, oVolume, nil
		}
	}
	return price, volume, err
}

// BestOfferPriceAndVolume : Return the best bid and volume for the sell side of the book.
func (b *OrderBook) BestOfferPriceAndVolume() (*num.Uint, uint64, error) {
	price, volume, err := b.sell.BestPriceAndVolume()

	if b.sell.offbook != nil {
		_, _, oPrice, oVolume := b.buy.offbook.BestPricesAndVolumes()

		// no off source volume, return the orderbook
		if oVolume == 0 {
			return price, volume, err
		}

		// no orderbook volume or AMM price is better
		if err != nil || oPrice.LT(price) {
			//nolint: nilerr
			return oPrice, oVolume, nil
		}

		// AMM price equals orderbook price, combined volumes
		if err == nil && oPrice.EQ(price) {
			oVolume += volume
			return oPrice, oVolume, nil
		}
	}
	return price, volume, err
}

func (b *OrderBook) CancelAllOrders(party string) ([]*types.OrderCancellationConfirmation, error) {
	var (
		orders = b.GetOrdersPerParty(party)
		confs  = []*types.OrderCancellationConfirmation{}
		conf   *types.OrderCancellationConfirmation
		err    error
	)

	for _, o := range orders {
		conf, err = b.CancelOrder(o)
		if err != nil {
			return nil, err
		}
		confs = append(confs, conf)
	}

	return confs, err
}

func (b *OrderBook) CheckBook() bool {
	checkOBB, checkOBS := false, false
	if len(b.buy.levels) > 0 {
		checkOBB = true
		for _, o := range b.buy.levels[len(b.buy.levels)-1].orders {
			if o.PeggedOrder == nil || o.PeggedOrder.Reference != types.PeggedReferenceBestBid {
				checkOBB = false
				break
			}
		}
	}
	if len(b.sell.levels) > 0 {
		checkOBS = true
		for _, o := range b.sell.levels[len(b.sell.levels)-1].orders {
			if o.PeggedOrder == nil || o.PeggedOrder.Reference != types.PeggedReferenceBestAsk {
				checkOBS = false
				break
			}
		}
	}
	// if either buy or sell side is lacking non-pegged orders, check AMM orders.
	if checkOBB || checkOBS {
		// get best bid/ask price and volumes.
		bb, bbv, bs, bsv := b.buy.offbook.BestPricesAndVolumes()
		// if the buy side is lacking non-pegged orders, check if there are off-book orders.
		if checkOBB && (bb == nil || bb.IsZero() || bbv == 0) {
			return false
		}
		// same, but for sell side.
		if checkOBS && (bs == nil || bs.IsZero() || bsv == 0) {
			return false
		}
	}
	return true
}

// CancelOrder cancel an order that is active on an order book. Market and Order ID are validated, however the order must match
// the order on the book with respect to side etc. The caller will typically validate this by using a store, we should
// not trust that the external world can provide these values reliably.
func (b *OrderBook) CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error) {
	// Validate Market
	if order.MarketID != b.marketID {
		b.log.Panic("Market ID mismatch",
			logging.Order(*order),
			logging.String("order-book", b.marketID))
	}

	// Validate Order ID must be present
	if err := validateOrderID(order.ID); err != nil {
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
	order.Status = types.OrderStatusCancelled

	result := &types.OrderCancellationConfirmation{
		Order: order,
	}
	return result, nil
}

// RemoveOrder takes the order off the order book.
func (b *OrderBook) RemoveOrder(id string) (*types.Order, error) {
	order, err := b.GetOrderByID(id)
	if err != nil {
		return nil, err
	}
	order, err = b.DeleteOrder(order)
	if err != nil {
		return nil, err
	}

	// Important to mark the order as parked (and no longer active)
	order.Status = types.OrderStatusParked

	return order, nil
}

// AmendOrder amends an order which is an active order on the book.
func (b *OrderBook) AmendOrder(originalOrder, amendedOrder *types.Order) error {
	if originalOrder == nil {
		if amendedOrder != nil {
			b.log.Panic("invalid input, orginalOrder is nil", logging.Order(*amendedOrder))
		}
	}

	// If the creation date for the 2 orders is different, something went wrong
	if originalOrder != nil && originalOrder.CreatedAt != amendedOrder.CreatedAt {
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

	var (
		volumeChange int64
		side         = b.sell
		err          error
	)
	if amendedOrder.Side == types.SideBuy {
		side = b.buy
	}

	if volumeChange, err = side.amendOrder(amendedOrder); err != nil {
		if b.log.GetLevel() == logging.DebugLevel {
			b.log.Debug("Failed to amend",
				logging.String("side", amendedOrder.Side.String()),
				logging.Order(*amendedOrder),
				logging.String("market", b.marketID),
				logging.Error(err),
			)
		}
		return err
	}

	// update the order by ids mapping
	b.ordersByID[amendedOrder.ID] = amendedOrder
	if !b.auction {
		return nil
	}

	if volumeChange < 0 {
		b.indicativePriceAndVolume.RemoveVolumeAtPrice(
			amendedOrder.Price, uint64(-volumeChange), amendedOrder.Side, false)
	}

	if volumeChange > 0 {
		b.indicativePriceAndVolume.AddVolumeAtPrice(
			amendedOrder.Price, uint64(volumeChange), amendedOrder.Side, false)
	}

	return nil
}

func (b *OrderBook) UpdateAMM(party string) {
	if !b.auction {
		return
	}

	ipv := b.indicativePriceAndVolume
	ipv.removeOffbookShape(party)

	min, max := ipv.lastMinPrice, ipv.lastMaxPrice
	if min.IsZero() || max.IsZero() || min.GT(max) {
		// region is not crossed so we won't expand just yet
		return
	}

	ipv.addOffbookShape(ptr.From(party), ipv.lastMinPrice, ipv.lastMaxPrice)
	ipv.needsUpdate = true
}

// GetTrades returns the trades a given order generates if we were to submit it now
// this is used to calculate fees, perform price monitoring, etc...
func (b *OrderBook) GetTrades(order *types.Order) ([]*types.Trade, error) {
	// this should always return straight away in an auction
	if b.auction {
		return nil, nil
	}

	if err := b.validateOrder(order); err != nil {
		return nil, err
	}
	if order.CreatedAt > b.latestTimestamp {
		b.latestTimestamp = order.CreatedAt
	}

	idealPrice := b.theoreticalBestTradePrice(order)
	trades, err := b.getOppositeSide(order.Side).fakeUncross(order, true, idealPrice)
	// it's fine for the error to be a wash trade here,
	// it's just be stopped when really uncrossing.
	if err != nil && err != ErrWashTrade {
		// some random error happened, return both trades and error
		// this is a case that isn't covered by the current SubmitOrder call
		return trades, err
	}

	// no error uncrossing, in all other cases, return trades (could be empty) without an error
	return trades, nil
}

func (b *OrderBook) ReplaceOrder(rm, rpl *types.Order) (*types.OrderConfirmation, error) {
	if _, err := b.CancelOrder(rm); err != nil {
		return nil, err
	}
	return b.SubmitOrder(rpl)
}

func (b *OrderBook) ReSubmitSpecialOrders(order *types.Order) {
	// not allowed to submit a normal order here
	if order.PeggedOrder == nil {
		b.log.Panic("only pegged orders allowed", logging.Order(order))
	}

	order.BatchID = b.batchID

	// check if order would trade, that should never happen as well.
	switch order.Side {
	case types.SideBuy:
		price, err := b.GetBestAskPrice()
		if err == nil && price.LTE(order.Price) {
			b.log.Panic("re submit special order would cross", logging.Order(order), logging.BigUint("best-ask", price))
		}
	case types.SideSell:
		price, err := b.GetBestBidPrice()
		if err == nil && price.GTE(order.Price) {
			b.log.Panic("re submit special order would cross", logging.Order(order), logging.BigUint("best-bid", price))
		}
	default:
		b.log.Panic("invalid order side", logging.Order(order))
	}

	// now we can nicely add the order to the book, no uncrossing needed
	b.getSide(order.Side).addOrder(order)
	b.add(order)
}

// theoreticalBestTradePrice returns the best possible price the incoming order could trade
// as if the spread were as small as possible. This will be used to construct the first
// interval to query offbook orders matching with the other side.
func (b *OrderBook) theoreticalBestTradePrice(order *types.Order) *num.Uint {
	bp, _, err := b.getSide(order.Side).BestPriceAndVolume()
	if err != nil {
		return nil
	}

	switch order.Side {
	case types.SideBuy:
		return bp.Add(bp, num.UintOne())
	case types.SideSell:
		return bp.Sub(bp, num.UintOne())
	default:
		panic("unexpected order side")
	}
}

// SubmitOrder Add an order and attempt to uncross the book, returns a TradeSet protobuf message object.
func (b *OrderBook) SubmitOrder(order *types.Order) (*types.OrderConfirmation, error) {
	if err := b.validateOrder(order); err != nil {
		return nil, err
	}

	if _, ok := b.ordersByID[order.ID]; ok {
		b.log.Panic("an order in the book already exists with the same ID", logging.Order(*order))
	}

	if order.CreatedAt > b.latestTimestamp {
		b.latestTimestamp = order.CreatedAt
	}

	var (
		trades          []*types.Trade
		impactedOrders  []*types.Order
		lastTradedPrice *num.Uint
		err             error
	)
	order.BatchID = b.batchID

	if !b.auction {
		// uncross with opposite

		defer b.buy.uncrossFinished()

		idealPrice := b.theoreticalBestTradePrice(order)
		trades, impactedOrders, lastTradedPrice, err = b.getOppositeSide(order.Side).uncross(order, true, idealPrice)
		if !lastTradedPrice.IsZero() {
			b.lastTradedPrice = lastTradedPrice
		}
	}

	// if order is persistent type add to order book to the correct side
	// and we did not hit a error / wash trade error
	if order.IsPersistent() && err == nil {
		if order.IcebergOrder != nil && order.Status == types.OrderStatusActive {
			// now trades have been generated for the aggressive iceberg based on the
			// full size, set the peak limits ready for it to be added to the book.
			order.SetIcebergPeaks()
		}

		b.getSide(order.Side).addOrder(order)
		// also add it to the indicative price and volume if in auction
		if b.auction {
			b.indicativePriceAndVolume.AddVolumeAtPrice(
				order.Price, order.TrueRemaining(), order.Side, false)
		}
	}

	// Was the aggressive order fully filled?
	if order.Remaining == 0 {
		order.Status = types.OrderStatusFilled
	}

	// What is an Immediate or Cancel Order?
	// An immediate or cancel order (IOC) is an order to buy or sell that executes all
	// or part immediately and cancels any unfilled portion of the order.
	if order.TimeInForce == types.OrderTimeInForceIOC && order.Remaining > 0 {
		// Stopped as not filled at all
		if order.Remaining == order.Size {
			order.Status = types.OrderStatusStopped
		} else {
			// IOC so we set status as Cancelled.
			order.Status = types.OrderStatusPartiallyFilled
		}
	}

	// What is Fill Or Kill?
	// Fill or kill (FOK) is a type of time-in-force designation used in trading that instructs
	// the protocol to execute an order immediately and completely or not at all.
	// The order must be filled in its entirety or cancelled (killed).
	if order.TimeInForce == types.OrderTimeInForceFOK && order.Remaining == order.Size {
		// FOK and didnt trade at all we set status as Stopped
		order.Status = types.OrderStatusStopped
	}

	for idx := range impactedOrders {
		// refresh if its an iceberg, noop if not
		b.icebergRefresh(impactedOrders[idx])

		if impactedOrders[idx].Remaining == 0 {
			impactedOrders[idx].Status = types.OrderStatusFilled

			// delete from lookup tables
			b.remove(impactedOrders[idx])
		}
	}

	// if we did hit a wash trade, set the status to STOPPED
	if err == ErrWashTrade {
		if order.Size > order.Remaining {
			order.Status = types.OrderStatusPartiallyFilled
		} else {
			order.Status = types.OrderStatusStopped
		}
		order.Reason = types.OrderErrorSelfTrading
	}

	if order.Status == types.OrderStatusActive {
		b.add(order)
	}

	orderConfirmation := makeResponse(order, trades, impactedOrders)
	return orderConfirmation, nil
}

// DeleteOrder remove a given order on a given side from the book.
func (b *OrderBook) DeleteOrder(
	order *types.Order,
) (*types.Order, error) {
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

	b.remove(order)

	// also remove it to the indicative price and volume if in auction
	// here we use specifically the order which was in the book, just in case
	// the order passed in would be wrong
	// TODO: refactor this better, we should never need to pass in more that IDs there
	// because by using the order passed in remain and price, we could use
	// values which have been amended previously... (e.g: amending an order which
	// cancel the order if it expires it
	if b.auction {
		b.indicativePriceAndVolume.RemoveVolumeAtPrice(
			dorder.Price, dorder.TrueRemaining(), dorder.Side, false)
	}
	return dorder, err
}

// GetOrderByID returns order by its ID (IDs are not expected to collide within same market).
func (b *OrderBook) GetOrderByID(orderID string) (*types.Order, error) {
	if err := validateOrderID(orderID); err != nil {
		if b.log.GetLevel() == logging.DebugLevel {
			b.log.Debug("Order ID missing or invalid",
				logging.String("order-id", orderID))
		}
		return nil, err
	}
	// First look for the order in the order book
	order, exists := b.ordersByID[orderID]
	if !exists {
		return nil, ErrOrderDoesNotExist
	}
	return order, nil
}

// RemoveDistressedOrders remove from the book all order holding distressed positions.
func (b *OrderBook) RemoveDistressedOrders(
	parties []events.MarketPosition,
) ([]*types.Order, error) {
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
				b.log.Panic(
					"Failed to cancel a given order for party",
					logging.Order(*o),
					logging.String("party", party.Party()),
					logging.Error(err))
			}
			// here we set the status of the order as stopped as the system triggered it as well.
			confirm.Order.Status = types.OrderStatusStopped
			rmorders = append(rmorders, confirm.Order)
		}
	}
	return rmorders, nil
}

func (b OrderBook) getSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.SideBuy {
		return b.buy
	}
	return b.sell
}

func (b *OrderBook) getOppositeSide(orderSide types.Side) *OrderBookSide {
	if orderSide == types.SideBuy {
		return b.sell
	}
	return b.buy
}

func makeResponse(order *types.Order, trades []*types.Trade, impactedOrders []*types.Order) *types.OrderConfirmation {
	return &types.OrderConfirmation{
		Order:                 order,
		PassiveOrdersAffected: impactedOrders,
		Trades:                trades,
	}
}

func (b *OrderBook) GetBestBidPrice() (*num.Uint, error) {
	price, _, err := b.buy.BestPriceAndVolume()

	if b.buy.offbook != nil {
		offbook, volume, _, _ := b.buy.offbook.BestPricesAndVolumes()
		if volume == 0 {
			return price, err
		}

		if err != nil || offbook.GT(price) {
			//nolint: nilerr
			return offbook, nil
		}
	}
	return price, err
}

func (b *OrderBook) GetBestStaticBidPrice() (*num.Uint, error) {
	price, err := b.buy.BestStaticPrice()
	if b.buy.offbook != nil {
		offbook, volume, _, _ := b.buy.offbook.BestPricesAndVolumes()
		if volume == 0 {
			return price, err
		}

		if err != nil || offbook.GT(price) {
			//nolint: nilerr
			return offbook, nil
		}
	}
	return price, err
}

func (b *OrderBook) GetBestStaticBidPriceAndVolume() (*num.Uint, uint64, error) {
	price, volume, err := b.buy.BestStaticPriceAndVolume()

	if b.buy.offbook != nil {
		oPrice, oVolume, _, _ := b.buy.offbook.BestPricesAndVolumes()

		// no off source volume, return the orderbook
		if oVolume == 0 {
			return price, volume, err
		}

		// no orderbook volume or AMM price is better
		if err != nil || oPrice.GT(price) {
			//nolint: nilerr
			return oPrice, oVolume, nil
		}

		// AMM price equals orderbook price, combined volumes
		if err == nil && oPrice.EQ(price) {
			oVolume += volume
			return oPrice, oVolume, nil
		}
	}
	return price, volume, err
}

func (b *OrderBook) GetBestAskPrice() (*num.Uint, error) {
	price, _, err := b.sell.BestPriceAndVolume()

	if b.sell.offbook != nil {
		_, _, offbook, volume := b.sell.offbook.BestPricesAndVolumes()
		if volume == 0 {
			return price, err
		}

		if err != nil || offbook.LT(price) {
			//nolint: nilerr
			return offbook, nil
		}
	}
	return price, err
}

func (b *OrderBook) GetBestStaticAskPrice() (*num.Uint, error) {
	price, err := b.sell.BestStaticPrice()
	if b.sell.offbook != nil {
		_, _, offbook, volume := b.sell.offbook.BestPricesAndVolumes()
		if volume == 0 {
			return price, err
		}

		if err != nil || offbook.LT(price) {
			//nolint: nilerr
			return offbook, nil
		}
	}
	return price, err
}

func (b *OrderBook) GetBestStaticAskPriceAndVolume() (*num.Uint, uint64, error) {
	price, volume, err := b.sell.BestStaticPriceAndVolume()

	if b.sell.offbook != nil {
		_, _, oPrice, oVolume := b.sell.offbook.BestPricesAndVolumes()

		// no off source volume, return the orderbook
		if oVolume == 0 {
			return price, volume, err
		}

		// no orderbook volume or AMM price is better
		if err != nil || oPrice.LT(price) {
			//nolint: nilerr
			return oPrice, oVolume, nil
		}

		// AMM price equals orderbook price, combined volumes
		if err == nil && oPrice.EQ(price) {
			oVolume += volume
			return oPrice, oVolume, nil
		}
	}
	return price, volume, err
}

func (b *OrderBook) GetLastTradedPrice() *num.Uint {
	return b.lastTradedPrice
}

// PrintState prints the actual state of the book.
// this should be use only in debug / non production environment as it
// rely a lot on logging.
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

// GetTotalNumberOfOrders is a debug/testing function to return the total number of orders in the book.
func (b *OrderBook) GetTotalNumberOfOrders() int64 {
	return b.buy.getOrderCount() + b.sell.getOrderCount()
}

// GetTotalVolume is a debug/testing function to return the total volume in the order book.
func (b *OrderBook) GetTotalVolume() int64 {
	return b.buy.getTotalVolume() + b.sell.getTotalVolume()
}

func (b *OrderBook) Settled() []*types.Order {
	orders := make([]*types.Order, 0, len(b.ordersByID))
	for _, v := range b.ordersByID {
		v.Status = types.OrderStatusStopped
		orders = append(orders, v)
	}

	sort.Slice(orders, func(i, j int) bool {
		return orders[i].ID < orders[j].ID
	})

	// reset all order stores
	b.cleanup()
	b.buy.cleanup()
	b.sell.cleanup()

	return orders
}

// GetActivePeggedOrderIDs returns the order identifiers of all pegged orders in the order book that are not parked.
func (b *OrderBook) GetActivePeggedOrderIDs() []string {
	pegged := make([]string, 0, b.peggedOrders.Len())
	for _, ID := range b.peggedOrders.Iter() {
		if o, ok := b.ordersByID[ID]; ok {
			if o.Status == vega.Order_STATUS_PARKED {
				b.log.Panic("unexpected parked pegged order in order book",
					logging.Order(o))
			}
			pegged = append(pegged, o.ID)
		}
	}
	return pegged
}

func (b *OrderBook) GetVolumeAtPrice(price *num.Uint, side types.Side) uint64 {
	lvls := b.getSide(side).getLevelsForPrice(price)
	vol := uint64(0)
	for _, lvl := range lvls {
		vol += lvl.volume
	}
	return vol
}

// icebergRefresh will restore the peaks of an iceberg order if they have drifted below the minimum value
// if not the order remains unchanged.
func (b *OrderBook) icebergRefresh(o *types.Order) {
	if !o.IcebergNeedsRefresh() {
		return
	}

	if _, err := b.DeleteOrder(o); err != nil {
		b.log.Panic("could not delete iceberg order during refresh", logging.Error(err), logging.Order(o))
	}

	// refresh peaks
	o.SetIcebergPeaks()

	// make sure its active again
	o.Status = types.OrderStatusActive

	// put it to the back of the line
	b.getSide(o.Side).addOrder(o)
	b.add(o)
}

// remove removes the given order from all the lookup map.
func (b *OrderBook) remove(o *types.Order) {
	if ok := b.peggedOrders.Exists(o.ID); ok {
		b.peggedOrdersCount--
		b.peggedCountNotify(-1)
		b.peggedOrders.Delete(o.ID)
	}
	delete(b.ordersByID, o.ID)
	delete(b.ordersPerParty[o.Party], o.ID)
}

// add adds the given order too all the lookup maps.
func (b *OrderBook) add(o *types.Order) {
	if o.GeneratedOffbook {
		b.log.Panic("Can not add offbook order to the orderbook", logging.Order(o))
	}

	b.ordersByID[o.ID] = o
	if orders, ok := b.ordersPerParty[o.Party]; !ok {
		b.ordersPerParty[o.Party] = map[string]struct{}{
			o.ID: {},
		}
	} else {
		orders[o.ID] = struct{}{}
	}

	if o.PeggedOrder != nil {
		b.peggedOrders.Add(o.ID)
		b.peggedOrdersCount++
		b.peggedCountNotify(1)
	}
}

// cleanup removes all orders and resets the the order lookup maps.
func (b *OrderBook) cleanup() {
	b.ordersByID = map[string]*types.Order{}
	b.ordersPerParty = map[string]map[string]struct{}{}
	b.indicativePriceAndVolume = nil
	b.peggedOrders.Clear()
	b.peggedCountNotify(-int64(b.peggedOrdersCount))
	b.peggedOrdersCount = 0
}

// VWAP returns an error if the total volume for the side of the book is less than the given volume or if there are no levels.
// Otherwise it returns the volume weighted average price for achieving the given volume.
func (b *OrderBook) VWAP(volume uint64, side types.Side) (*num.Uint, error) {
	sidePriceLevels := b.buy
	if side == types.SideSell {
		sidePriceLevels = b.sell
	}
	dVol := num.DecimalFromInt64(int64(volume))
	remaining := volume
	price := num.UintZero()
	i := len(sidePriceLevels.levels) - 1
	if i < 0 {
		return nil, fmt.Errorf("no orders in book for side")
	}

	if volume == 0 && i >= 0 {
		return sidePriceLevels.levels[i].price.Clone(), nil
	}

	for {
		if i < 0 || remaining == 0 {
			break
		}
		size := remaining
		if sidePriceLevels.levels[i].volume < remaining {
			size = sidePriceLevels.levels[i].volume
		}
		price.AddSum(num.UintZero().Mul(sidePriceLevels.levels[i].price, num.NewUint(size)))
		remaining -= size
		i -= 1
	}

	if remaining == 0 {
		res, _ := num.UintFromDecimal(price.ToDecimal().Div(dVol))
		return res, nil
	}
	return nil, fmt.Errorf("insufficient volume in order book")
}
