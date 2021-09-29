package matching

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

func (b *OrderBook) Keys() []string {
	return []string{b.snapshot.Key()}
}

func (b *OrderBook) Snapshot() (map[string]*types.Payload, error) {
	payload, err := b.GetState(b.snapshot.Key())
	if err != nil {
		return nil, err
	}

	snapshot := map[string]*types.Payload{}
	snapshot[b.snapshot.Key()] = payload
	return snapshot, nil
}

func (b OrderBook) Namespace() types.SnapshotNamespace {
	payload := types.PayloadMatchingBook{}
	return payload.Namespace()
}

func (b *OrderBook) GetHash(key string) ([]byte, error) {
	if key != b.snapshot.Key() {
		return nil, fmt.Errorf("unknown key for matching engine: %s", key)
	}

	payload, e := b.GetState(key)
	if e != nil {
		return nil, e
	}

	// Convert the domain object into a protobuf payload message
	p := payload.IntoProto()

	data, err := proto.Marshal(p)
	if err != nil {
		return nil, err
	}

	h := crypto.Hash(data)
	return h, nil
}

func (b *OrderBook) GetState(key string) (*types.Payload, error) {
	if key != b.snapshot.Key() {
		return nil, fmt.Errorf("unknown key for matching engine: %s", key)
	}

	// Copy all the state into a domain object
	payload := b.buildPayload()

	return payload, nil
}

func (b *OrderBook) buildPayload() *types.Payload {
	state := types.MatchingBook{}

	state.MarketID = b.marketID
	state.Buy = b.copyBuySide()
	state.Sell = b.copySellSide()
	state.LastTradedPrice = b.lastTradedPrice
	state.Auction = b.auction
	state.BatchID = b.batchID

	// Wrap it in a payload
	payload := &types.PayloadMatchingBook{
		MatchingBook: &state,
	}

	// Wrap that in a payload wrapper
	payloadWrapper := &types.Payload{
		Data: payload,
	}

	return payloadWrapper
}

func (b *OrderBook) copyBuySide() []*types.Order {
	orders := make([]*types.Order, 0)
	pricelevels := b.buy.getLevels()
	for _, pl := range pricelevels {
		for _, order := range pl.orders {
			orders = append(orders, order.Clone())
		}
	}

	// Sort the orders into creation time order
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt < orders[j].CreatedAt
	})

	return orders
}

func (b *OrderBook) copySellSide() []*types.Order {
	orders := make([]*types.Order, 0)
	pricelevels := b.sell.getLevels()
	for _, pl := range pricelevels {
		for _, order := range pl.orders {
			orders = append(orders, order.Clone())
		}
	}

	// Sort the orders into creation time order
	sort.Slice(orders, func(i, j int) bool {
		return orders[i].CreatedAt < orders[j].CreatedAt
	})

	return orders
}

func (b *OrderBook) LoadState(payload *types.PayloadMatchingBook) {
	mb := payload.MatchingBook

	b.reset()
	b.marketID = mb.MarketID
	b.batchID = mb.BatchID
	b.auction = mb.Auction
	b.lastTradedPrice = mb.LastTradedPrice

	for _, o := range mb.Buy {
		b.buy.addOrder(o)
		b.ordersByID[o.ID] = o
		b.addOrderToPartyMap(o)
	}

	for _, o := range mb.Sell {
		b.sell.addOrder(o)
		b.ordersByID[o.ID] = o
		b.addOrderToPartyMap(o)
	}

	// If we are in an auction we need to build the IP&V structure
	if b.auction {
		b.indicativePriceAndVolume = NewIndicativePriceAndVolume(b.log, b.buy, b.sell)
	}
}

func (b *OrderBook) addOrderToPartyMap(o *types.Order) {
	if orders, ok := b.ordersPerParty[o.Party]; !ok {
		b.ordersPerParty[o.Party] = map[string]struct{}{
			o.ID: {},
		}
	} else {
		orders[o.ID] = struct{}{}
	}
}
