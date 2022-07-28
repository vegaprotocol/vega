// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package service

import (
	"context"
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/utils"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/market_depth_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/service OrderStore
type OrderStore interface {
	GetLiveOrders(ctx context.Context) ([]entities.Order, error)
}

type MarketDepth struct {
	marketDepths   map[string]*entities.MarketDepth
	orderStore     OrderStore
	log            *logging.Logger
	depthObserver  utils.Observer[*types.MarketDepth]
	updateObserver utils.Observer[*types.MarketDepthUpdate]
	mu             sync.RWMutex
	sequenceNumber uint64
}

func NewMarketDepth(orderStore OrderStore, logger *logging.Logger) *MarketDepth {
	return &MarketDepth{
		marketDepths:   map[string]*entities.MarketDepth{},
		orderStore:     orderStore,
		log:            logger,
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

	m.startPublishingChanges(ctx)
	return nil
}

func (m *MarketDepth) startPublishingChanges(ctx context.Context) {
	ticker := time.NewTicker(100 * time.Millisecond)
	go func() {
		for {
			select {
			case <-ticker.C:
				m.publishChanges()
			case <-ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()
}

func (m *MarketDepth) publishChanges() {
	m.mu.Lock()
	defer m.mu.Unlock()
	for marketID, md := range m.marketDepths {
		buyPtr := []*types.PriceLevel{}
		sellPtr := []*types.PriceLevel{}

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

func (m *MarketDepth) AddOrder(order *types.Order, vegaTime time.Time, sequenceNumber uint64) {
	m.mu.Lock()
	defer m.mu.Unlock()

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

	// See if we already have a MarketDepth item for this market
	md := m.marketDepths[order.MarketID]
	if md == nil {
		// First time we have an update for this market
		// so we need to create a new MarketDepth
		md = &entities.MarketDepth{
			MarketID:   order.MarketID,
			LiveOrders: map[string]*types.Order{},
		}
		md.SequenceNumber = m.sequenceNumber
		m.marketDepths[order.MarketID] = md
	}

	md.AddOrderUpdate(order)

	md.SequenceNumber = m.sequenceNumber
}

// GetMarketDepth builds up the structure to be sent out to any market depth listeners
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

// GetOrderCount returns the number of live orders for the given market
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

// GetVolumeAtPrice returns the order volume at the given price level
func (m *MarketDepth) GetVolumeAtPrice(market string, side types.Side, price uint64) uint64 {
	md := m.marketDepths[market]
	if md != nil {
		pl := md.GetPriceLevel(side, num.NewUint(price))
		if pl == nil {
			return 0
		}
		return pl.TotalVolume
	}
	return 0
}

// GetTotalVolume returns the total volume in the order book
func (m *MarketDepth) GetTotalVolume(market string) int64 {
	var volume int64
	md := m.marketDepths[market]
	if md != nil {
		for _, pl := range md.BuySide {
			volume += int64(pl.TotalVolume)
		}

		for _, pl := range md.SellSide {
			volume += int64(pl.TotalVolume)
		}
		return volume
	}
	return 0
}

// GetOrderCountAtPrice returns the number of orders at the given price level
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

// GetPriceLevels returns the number of non empty price levels
func (m *MarketDepth) GetPriceLevels(market string) int {
	return m.GetBuyPriceLevels(market) + m.GetSellPriceLevels(market)
}

// GetBestBidPrice returns the highest bid price in the book
func (m *MarketDepth) GetBestBidPrice(market string) *num.Uint {
	md := m.marketDepths[market]
	if md != nil {
		if len(md.BuySide) > 0 {
			return md.BuySide[0].Price.Clone()
		}
	}
	return num.Zero()
}

// GetBestAskPrice returns the highest bid price in the book
func (m *MarketDepth) GetBestAskPrice(market string) *num.Uint {
	md := m.marketDepths[market]
	if md != nil {
		if len(md.SellSide) > 0 {
			return md.SellSide[0].Price.Clone()
		}
	}
	return num.Zero()
}

// GetBuyPriceLevels returns the number of non empty buy price levels
func (m *MarketDepth) GetBuyPriceLevels(market string) int {
	md := m.marketDepths[market]
	if md != nil {
		return len(md.BuySide)
	}
	return 0
}

// GetSellPriceLevels returns the number of non empty sell price levels
func (m *MarketDepth) GetSellPriceLevels(market string) int {
	md := m.marketDepths[market]
	if md != nil {
		return len(md.SellSide)
	}
	return 0
}
