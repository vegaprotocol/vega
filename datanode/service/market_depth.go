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
	ListActive(ctx context.Context) ([]entities.AMMPool, error)
}

type Positions interface {
	GetByMarketAndParty(ctx context.Context, marketID string, partyID string) (entities.Position, error)
}

type AssetStore interface {
	GetByID(ctx context.Context, id string) (entities.Asset, error)
}

type ammCache struct {
	priceFactor    num.Decimal                 // the price factor for this market
	ammOrders      map[string][]*types.Order   // map amm id -> expanded orders, so we can remove them if amended
	activeAMMs     map[string]entities.AMMPool // map amm id -> amm definition, so we can refresh its expansion
	estimatedOrder map[string]struct{}         // order-id -> whether it was an estimated order

	// the lowest/highest bounds of all AMMs
	lowestBound  num.Decimal
	highestBound num.Decimal

	// reference -> calculation levels, if the reference point hasn't changed we can avoid the busy task
	// of recalculating them
	levels map[string][]*level
}

func (c *ammCache) addAMM(a entities.AMMPool) {
	c.activeAMMs[a.AmmPartyID.String()] = a

	low := a.ParametersBase
	if a.ParametersLowerBound != nil {
		low = *a.ParametersLowerBound
	}

	if c.lowestBound.IsZero() {
		c.lowestBound = low
	} else {
		c.lowestBound = num.MinD(c.lowestBound, low)
	}

	high := a.ParametersBase
	if a.ParametersUpperBound != nil {
		high = *a.ParametersUpperBound
	}
	c.highestBound = num.MaxD(c.highestBound, high)
}

func (c *ammCache) removeAMM(ammParty string) {
	delete(c.activeAMMs, ammParty)
	delete(c.ammOrders, ammParty)

	// now we need to recalculate the lowest/highest

	c.lowestBound = num.DecimalZero()
	c.highestBound = num.DecimalZero()
	for _, a := range c.activeAMMs {
		low := a.ParametersBase
		if a.ParametersLowerBound != nil {
			low = *a.ParametersLowerBound
		}
		if c.lowestBound.IsZero() {
			c.lowestBound = low
		} else {
			c.lowestBound = num.MinD(c.lowestBound, low)
		}

		high := a.ParametersBase
		if a.ParametersUpperBound != nil {
			high = *a.ParametersUpperBound
		}
		c.lowestBound = num.MaxD(c.highestBound, high)
	}
}

type MarketDepth struct {
	log            *logging.Logger
	cfg            MarketDepthConfig
	marketDepths   map[string]*entities.MarketDepth
	orderStore     OrderStore
	ammStore       AMMStore
	assetStore     AssetStore
	markets        MarketStore
	marketData     MarketDataStore
	positions      Positions
	depthObserver  utils.Observer[*types.MarketDepth]
	updateObserver utils.Observer[*types.MarketDepthUpdate]
	mu             sync.RWMutex
	sequenceNumber uint64

	ammCache map[string]*ammCache
}

func NewMarketDepth(
	cfg MarketDepthConfig,
	orderStore OrderStore,
	ammStore AMMStore,
	marketData MarketDataStore,
	positions Positions,
	assets AssetStore,
	markets MarketStore,
	logger *logging.Logger,
) *MarketDepth {
	return &MarketDepth{
		log:            logger,
		cfg:            cfg,
		marketDepths:   map[string]*entities.MarketDepth{},
		orderStore:     orderStore,
		ammStore:       ammStore,
		marketData:     marketData,
		positions:      positions,
		assetStore:     assets,
		markets:        markets,
		depthObserver:  utils.NewObserver[*types.MarketDepth]("market_depth", logger, 100, 100),
		updateObserver: utils.NewObserver[*types.MarketDepthUpdate]("market_depth_update", logger, 100, 100),
		ammCache:       map[string]*ammCache{},
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

	m.InitialiseAMMs(ctx)

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

func (m *MarketDepth) sequential(t time.Time, sequenceNumber uint64) bool {
	// we truncate the vegaTime by microsecond because Postgres only supports microsecond
	// granularity for time. In order to be able to reproduce the same sequence numbers regardless
	// the source, we have to truncate the time to microsecond granularity
	n := uint64(t.Truncate(time.Microsecond).UnixNano()) + sequenceNumber

	if m.sequenceNumber > n {
		// This update is older than the current MarketDepth
		return false
	}

	m.sequenceNumber = n
	return true
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

	if !m.sequential(vegaTime, sequenceNumber) {
		return
	}

	if m.isAMMOrder(order) {
		// this AMM order has come through the orders stream, it can only mean that it has traded so we need to refresh its depth
		m.onAMMTraded(order.Party, order.MarketID)
		return
	}

	md := m.getDepth(order.MarketID)
	md.AddOrderUpdate(order, false)
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

// GetVolumeAtPrice returns the order volume at the given price level.
func (m *MarketDepth) GetEstimatedVolumeAtPrice(market string, side types.Side, price uint64) uint64 {
	md := m.marketDepths[market]
	if md != nil {
		pl := md.GetPriceLevel(side, num.NewUint(price))
		if pl == nil {
			return 0
		}
		return pl.TotalEstimatedAMMVolume
	}
	return 0
}

// GetTotalVolume returns the total volume in the order book.
func (m *MarketDepth) GetTotalVolume(market string) int64 {
	var volume int64
	md := m.marketDepths[market]
	if md != nil {
		for _, pl := range md.BuySide {
			volume += int64(pl.TotalVolume) + int64(pl.TotalAMMVolume) + int64(pl.TotalEstimatedAMMVolume)
		}

		for _, pl := range md.SellSide {
			volume += int64(pl.TotalVolume) + int64(pl.TotalAMMVolume) + int64(pl.TotalEstimatedAMMVolume)
		}
		return volume
	}
	return 0
}

// GetAMMVolume returns the total volume in the order book.
func (m *MarketDepth) GetAMMVolume(market string, estimated bool) int64 {
	var volume int64
	md := m.marketDepths[market]
	if md != nil {
		for _, pl := range md.BuySide {
			if estimated {
				volume += int64(pl.TotalEstimatedAMMVolume)
				continue
			}
			volume += int64(pl.TotalAMMVolume)
		}

		for _, pl := range md.SellSide {
			if estimated {
				volume += int64(pl.TotalEstimatedAMMVolume)
				continue
			}
			volume += int64(pl.TotalAMMVolume)
		}
		return volume
	}
	return 0
}

// GetAMMVolume returns the total volume in the order book.
func (m *MarketDepth) GetTotalAMMVolume(market string) int64 {
	return m.GetAMMVolume(market, true) + m.GetAMMVolume(market, false)
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
