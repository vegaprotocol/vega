package matching

import (
	"context"
	"log"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

func (b *OrderBook) Keys() []string {
	return []string{b.snapshot.Key()}
}

func (b *OrderBook) Snapshot() (map[string][]byte, error) {
	payload, _, err := b.GetState(b.snapshot.Key())
	if err != nil {
		return nil, err
	}
	return map[string][]byte{b.snapshot.Key(): payload}, nil
}

func (b OrderBook) Namespace() types.SnapshotNamespace {
	return types.MatchingSnapshot
}

func (b *OrderBook) GetHash(key string) ([]byte, error) {
	if key != b.snapshot.Key() {
		return nil, types.ErrSnapshotKeyDoesNotExist
	}

	payload, _, e := b.GetState(key)
	if e != nil {
		return nil, e
	}

	return crypto.Hash(payload), nil
}

func (b *OrderBook) GetState(key string) ([]byte, []types.StateProvider, error) {
	if key != b.snapshot.Key() {
		return nil, nil, types.ErrSnapshotKeyDoesNotExist
	}

	// Copy all the state into a domain object
	payload := b.buildPayload()

	s, err := proto.Marshal(payload.IntoProto())
	return s, nil, err
}

func (b *OrderBook) buildPayload() *types.Payload {
	return &types.Payload{
		Data: &types.PayloadMatchingBook{
			MatchingBook: &types.MatchingBook{
				MarketID:        b.marketID,
				Buy:             b.copyOrders(b.buy),
				Sell:            b.copyOrders(b.sell),
				LastTradedPrice: b.lastTradedPrice,
				Auction:         b.auction,
				BatchID:         b.batchID,
			},
		},
	}
}

func (b *OrderBook) copyOrders(obs *OrderBookSide) []*types.Order {
	orders := make([]*types.Order, 0)
	pricelevels := obs.getLevels()
	for _, pl := range pricelevels {
		for _, order := range pl.orders {
			orders = append(orders, order.Clone())
		}
	}
	return orders
}

func (b *OrderBook) LoadState(_ context.Context, payload *types.Payload) ([]types.StateProvider, error) {
	if b.Namespace() != payload.Namespace() {
		return nil, types.ErrInvalidSnapshotNamespace
	}

	var mb *types.MatchingBook

	switch pl := payload.Data.(type) {
	case *types.PayloadMatchingBook:
		mb = pl.MatchingBook
	default:
		return nil, types.ErrUnknownSnapshotType
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
	return nil, nil
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
