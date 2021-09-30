package matching

import (
	"log"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

func (b *OrderBook) Keys() []string {
	return []string{b.snapshot.Key()}
}

func (b *OrderBook) Snapshot() (map[string][]byte, error) {
	payload, err := b.GetState(b.snapshot.Key())
	if err != nil {
		return nil, err
	}

	snapshot := map[string][]byte{}
	snapshot[b.snapshot.Key()] = payload
	return snapshot, nil
}

func (b OrderBook) Namespace() types.SnapshotNamespace {
	return types.MatchingSnapshot
}

func (b *OrderBook) GetHash(key string) ([]byte, error) {
	if key != b.snapshot.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	payload, e := b.GetState(key)
	if e != nil {
		return nil, e
	}

	h := crypto.Hash(payload)
	return h, nil
}

func (b *OrderBook) GetState(key string) ([]byte, error) {
	if key != b.snapshot.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	// Copy all the state into a domain object
	payload := b.buildPayload()

	// Convert the domain object into a protobuf payload message
	p := payload.IntoProto()

	data, err := proto.Marshal(p)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (b *OrderBook) buildPayload() *types.Payload {
	state := types.MatchingBook{}

	state.MarketID = b.marketID
	state.Buy = b.copyOrders(b.buy)
	state.Sell = b.copyOrders(b.sell)
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

func (b *OrderBook) copyOrders(obs *OrderBookSide) []*types.Order {
	orders := make([]*types.Order, 0)
	pricelevels := obs.getLevels()
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

func (b *OrderBook) LoadState(payload *types.Payload) error {
	if b.Namespace() != payload.Namespace() {
		return types.ErrInvalidSnapshotNamespace
	}

	var mb *types.MatchingBook

	switch pl := payload.Data.(type) {
	case *types.PayloadMatchingBook:
		mb = pl.MatchingBook
	default:
		return types.ErrInvalidType
	}

	// Check we have an empty book here or else we should panic
	if len(b.buy.levels) > 0 || len(b.sell.levels) > 0 {
		log.Panic("orderbook is not empty so we should not be loading snapshot state")
	}

	b.marketID = mb.MarketID
	b.batchID = mb.BatchID
	b.auction = mb.Auction
	b.lastTradedPrice = mb.LastTradedPrice

	for _, o := range mb.Buy {
		b.buy.addOrder(o)
		b.addOrderToMaps(o)
	}

	for _, o := range mb.Sell {
		b.sell.addOrder(o)
		b.addOrderToMaps(o)
	}

	// If we are in an auction we need to build the IP&V structure
	if b.auction {
		b.indicativePriceAndVolume = NewIndicativePriceAndVolume(b.log, b.buy, b.sell)
	}
	return nil
}

func (b *OrderBook) addOrderToMaps(o *types.Order) {
	b.ordersByID[o.ID] = o

	if orders, ok := b.ordersPerParty[o.Party]; !ok {
		b.ordersPerParty[o.Party] = map[string]struct{}{
			o.ID: {},
		}
	} else {
		orders[o.ID] = struct{}{}
	}
}
