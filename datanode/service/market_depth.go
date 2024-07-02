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

package service

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/utils"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/protos/vega"
)

type OrderStore interface {
	GetLiveOrders(ctx context.Context) ([]entities.Order, error)
}

type AMMStore interface {
	ListByStatus(ctx context.Context, status entities.AMMStatus, pagination entities.CursorPagination) ([]entities.AMMPool, entities.PageInfo, error)
}

type Positions interface {
	GetByMarketAndParty(ctx context.Context, marketID string, partyID string) (entities.Position, error)
}

type MarketDepth struct {
	log            *logging.Logger
	marketDepths   map[string]*entities.MarketDepth
	orderStore     OrderStore
	ammStore       AMMStore
	marketData     MarketDataStore
	positions      Positions
	depthObserver  utils.Observer[*types.MarketDepth]
	updateObserver utils.Observer[*types.MarketDepthUpdate]
	mu             sync.RWMutex
	sequenceNumber uint64
	ammOrders      map[string][]*types.Order
}

func NewMarketDepth(orderStore OrderStore, ammStore AMMStore, marketData MarketDataStore, positions Positions, logger *logging.Logger) *MarketDepth {
	return &MarketDepth{
		log:            logger,
		marketDepths:   map[string]*entities.MarketDepth{},
		orderStore:     orderStore,
		ammStore:       ammStore,
		marketData:     marketData,
		positions:      positions,
		depthObserver:  utils.NewObserver[*types.MarketDepth]("market_depth", logger, 100, 100),
		updateObserver: utils.NewObserver[*types.MarketDepthUpdate]("market_depth_update", logger, 100, 100),
	}
}

func (m *MarketDepth) Initialise(ctx context.Context) error {
	liveOrders, err := m.orderStore.GetLiveOrders(ctx)
	if err != nil {
		return err
	}

	// process the live orders and initialize market depths from database data
	for _, liveOrder := range liveOrders {
		order, err := types.OrderFromProto(liveOrder.ToProto())
		if err != nil {
			panic(err)
		}
		m.AddOrder(order, liveOrder.VegaTime, liveOrder.SeqNum)
	}

	// TODO get and expand active AMM's
	ammOrders := m.ExpandAMMs(ctx)
	m.ammOrders = ammOrders

	for _, orders := range ammOrders {
		for _, o := range orders {
			// TODO what the hell are sequence numbers about
			m.AddOrder(o, time.Time{}, m.sequenceNumber)
		}
	}

	return nil
}

func (m *MarketDepth) PublishAtEndOfBlock() {
	m.publishChanges()
}

func (m *MarketDepth) publishChanges() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for marketID, md := range m.marketDepths {
		buyPtr := []*types.PriceLevel{}
		sellPtr := []*types.PriceLevel{}

		// No need to notify anyone if nothing has changed
		if len(md.Changes) == 0 {
			continue
		}

		// Send out market depth updates to any listeners
		for _, pl := range md.Changes {
			if pl.Side == types.SideBuy {
				buyPtr = append(buyPtr, &types.PriceLevel{
					Price:          pl.Price.Clone(),
					NumberOfOrders: pl.TotalOrders,
					Volume:         pl.TotalVolume,
				})
			} else {
				sellPtr = append(sellPtr, &types.PriceLevel{
					Price:          pl.Price.Clone(),
					NumberOfOrders: pl.TotalOrders,
					Volume:         pl.TotalVolume,
				})
			}
		}

		marketDepthUpdate := &types.MarketDepthUpdate{
			MarketId:               marketID,
			Buy:                    types.PriceLevels(buyPtr).IntoProto(),
			Sell:                   types.PriceLevels(sellPtr).IntoProto(),
			SequenceNumber:         md.SequenceNumber,
			PreviousSequenceNumber: md.PreviousSequenceNumber,
		}

		m.updateObserver.Notify([]*types.MarketDepthUpdate{marketDepthUpdate})
		m.depthObserver.Notify([]*types.MarketDepth{md.ToProto(0)})

		// Clear the list of changes
		md.Changes = make([]*entities.PriceLevel, 0, len(md.Changes))
		md.PreviousSequenceNumber = md.SequenceNumber
	}
}

func (m *MarketDepth) UpdateAMM(pool entities.AMMPool) {

	// remove any existing orders for this AMM and we'll generate some more given its new definition
	existing := m.ammOrders[string(pool.AmmPartyID)]
	for _, o := range existing {
		o.Status = vega.Order_STATUS_CANCELLED
		m.AddOrder(o, time.Time{}, m.sequenceNumber)
	}

	if pool.Status == entities.AMMStatusCancelled || pool.Status == entities.AMMStatusStopped {
		return
	}

	marketID := string(pool.MarketID)
	marketData, err := m.marketData.GetMarketDataByID(context.Background(), marketID)
	if err != nil {
		m.log.Warn("unable to get market-data for market",
			logging.String("market-id", marketID),
			logging.Error(err),
		)
		return
	}

	reference := marketData.MidPrice
	if !marketData.IndicativePrice.IsZero() {
		reference = marketData.IndicativePrice
	}

	if reference.IsZero() {
		m.log.Warn("cannot calculate market-depth for AMM, no reference point available",
			logging.String("mid-price", marketData.MidPrice.String()),
			logging.String("indicative-price", marketData.IndicativePrice.String()),
		)
		return
	}

	// an AMM has appeared, lets add its volume
	orders, _ := m.ExpandAMM(pool, reference)

	for _, o := range orders {
		m.AddOrder(o, time.Time{}, m.sequenceNumber)
	}

}

func (m *MarketDepth) HandleAMMOrder(order *types.Order) {

	md := m.getDepth(order.MarketID)
	md.AddOrderUpdate(order)

}

func (m *MarketDepth) AddOrder(order *types.Order, vegaTime time.Time, sequenceNumber uint64) {

	// TODO this lock probably means we should lock handleAMMOrder too?
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.ammOrders[order.Party]; ok {
		m.HandleAMMOrder(order)
		return
	}

	// Non persistent and network orders do not matter
	if order.Type == types.OrderTypeMarket ||
		order.TimeInForce == types.OrderTimeInForceFOK ||
		order.TimeInForce == types.OrderTimeInForceIOC {
		return
	}

	// Orders that where not valid are ignored
	if order.Status == types.OrderStatusUnspecified {
		return
	}

	// we truncate the vegaTime by microsecond because Postgres only supports microsecond
	// granularity for time. In order to be able to reproduce the same sequence numbers regardless
	// the source, we have to truncate the time to microsecond granularity
	seqNum := uint64(vegaTime.Truncate(time.Microsecond).UnixNano()) + sequenceNumber

	if m.sequenceNumber > seqNum {
		// This update is older than the current MarketDepth
		return
	}

	m.sequenceNumber = seqNum

	md := m.getDepth(order.MarketID)
	md.AddOrderUpdate(order)
	md.SequenceNumber = m.sequenceNumber
}

func (m *MarketDepth) getDepth(marketID string) *entities.MarketDepth {
	// See if we already have a MarketDepth item for this market
	if md := m.marketDepths[marketID]; md != nil {
		return md
	}

	// First time we have an update for this market
	// so we need to create a new MarketDepth
	md := &entities.MarketDepth{
		MarketID:   marketID,
		LiveOrders: map[string]*types.Order{},
	}
	md.SequenceNumber = m.sequenceNumber
	m.marketDepths[marketID] = md
	return md
}

// GetMarketDepth builds up the structure to be sent out to any market depth listeners.
func (m *MarketDepth) GetMarketDepth(market string, limit uint64) *types.MarketDepth {
	m.mu.RLock()
	defer m.mu.RUnlock()
	md, ok := m.marketDepths[market]
	if !ok || md == nil {
		// When a market is new with no orders there will not be any market depth/order book
		// so we do not need to try and calculate the depth cumulative volumes etc
		return &types.MarketDepth{
			MarketId: market,
			Buy:      []*vega.PriceLevel{},
			Sell:     []*vega.PriceLevel{},
		}
	}

	return md.ToProto(limit)
}

func (m *MarketDepth) ObserveDepth(ctx context.Context, retries int, marketIds []string) (<-chan []*types.MarketDepth, uint64) {
	markets := map[string]bool{}
	for _, id := range marketIds {
		markets[id] = true
	}

	ch, ref := m.depthObserver.Observe(ctx,
		retries,
		func(md *types.MarketDepth) bool { return markets[md.MarketId] })
	return ch, ref
}

func (m *MarketDepth) ObserveDepthUpdates(ctx context.Context, retries int, marketIds []string) (<-chan []*types.MarketDepthUpdate, uint64) {
	markets := map[string]bool{}
	for _, id := range marketIds {
		markets[id] = true
	}

	ch, ref := m.updateObserver.Observe(ctx,
		retries,
		func(md *types.MarketDepthUpdate) bool { return markets[md.MarketId] })
	return ch, ref
}

/*****************************************************************************/
/*                 FUNCTIONS TO HELP WITH UNIT TESTING                       */
/*****************************************************************************/

func (m *MarketDepth) GetAllOrders(market string) map[string]*types.Order {
	md := m.marketDepths[market]
	if md != nil {
		return md.LiveOrders
	}
	return nil
}

// GetOrderCount returns the number of live orders for the given market.
func (m *MarketDepth) GetOrderCount(market string) int64 {
	var liveOrders int64
	var bookOrders uint64
	md := m.marketDepths[market]
	if md != nil {
		liveOrders = int64(len(md.LiveOrders))

		for _, pl := range md.BuySide {
			bookOrders += pl.TotalOrders
		}

		for _, pl := range md.SellSide {
			bookOrders += pl.TotalOrders
		}

		if liveOrders != int64(bookOrders) {
			return -1
		}
		return liveOrders
	}
	return 0
}

// GetVolumeAtPrice returns the order volume at the given price level.
func (m *MarketDepth) GetVolumeAtPrice(market string, side types.Side, price uint64) uint64 {
	md := m.marketDepths[market]
	if md != nil {
		pl := md.GetPriceLevel(side, num.NewUint(price))
		if pl == nil {
			return 0
		}
		return pl.TotalVolume + pl.TotalAMMVolume
	}
	return 0
}

// GetTotalVolume returns the total volume in the order book.
func (m *MarketDepth) GetTotalVolume(market string) int64 {
	var volume int64
	md := m.marketDepths[market]
	if md != nil {
		for _, pl := range md.BuySide {
			volume += int64(pl.TotalVolume) + int64(pl.TotalAMMVolume)
		}

		for _, pl := range md.SellSide {
			volume += int64(pl.TotalVolume) + int64(pl.TotalAMMVolume)
		}
		return volume
	}
	return 0
}

// GetOrderCountAtPrice returns the number of orders at the given price level.
func (m *MarketDepth) GetOrderCountAtPrice(market string, side types.Side, price uint64) uint64 {
	md := m.marketDepths[market]
	if md != nil {
		pl := md.GetPriceLevel(side, num.NewUint(price))
		if pl == nil {
			return 0
		}
		return pl.TotalOrders
	}
	return 0
}

// GetPriceLevels returns the number of non empty price levels.
func (m *MarketDepth) GetPriceLevels(market string) int {
	return m.GetBuyPriceLevels(market) + m.GetSellPriceLevels(market)
}

// GetBestBidPrice returns the highest bid price in the book.
func (m *MarketDepth) GetBestBidPrice(market string) *num.Uint {
	md := m.marketDepths[market]
	if md != nil {
		if len(md.BuySide) > 0 {
			return md.BuySide[0].Price.Clone()
		}
	}
	return num.UintZero()
}

// GetBestAskPrice returns the highest bid price in the book.
func (m *MarketDepth) GetBestAskPrice(market string) *num.Uint {
	md := m.marketDepths[market]
	if md != nil {
		if len(md.SellSide) > 0 {
			return md.SellSide[0].Price.Clone()
		}
	}
	return num.UintZero()
}

// GetBuyPriceLevels returns the number of non empty buy price levels.
func (m *MarketDepth) GetBuyPriceLevels(market string) int {
	md := m.marketDepths[market]
	if md != nil {
		return len(md.BuySide)
	}
	return 0
}

// GetSellPriceLevels returns the number of non empty sell price levels.
func (m *MarketDepth) GetSellPriceLevels(market string) int {
	md := m.marketDepths[market]
	if md != nil {
		return len(md.SellSide)
	}
	return 0
}
