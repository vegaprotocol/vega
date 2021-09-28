package matching

import (
	"fmt"
	"sort"

	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/protobuf/proto"
)

const (
	MatchingSnapshot = "matching engine"
)

func (ob *OrderBook) Keys() []string {
	return []string{MatchingSnapshot}
}

func (ob *OrderBook) Snapshot() (map[string][]byte, error) {
	data, err := ob.GetState(MatchingSnapshot)
	if err != nil {
		return nil, err
	}

	snapshot := map[string][]byte{}
	snapshot[MatchingSnapshot] = data
	return snapshot, nil
}

func (ob OrderBook) Namespace() types.SnapshotNamespace {
	payload := types.PayloadMatchingBook{}
	return payload.Namespace()
}

func (ob *OrderBook) GetHash(key string) ([]byte, error) {
	if key != MatchingSnapshot {
		return nil, fmt.Errorf("Unknown key for matching engine: %s", key)
	}

	b, e := ob.GetState(key)
	if e != nil {
		return nil, e
	}
	h := crypto.Hash(b)
	return h, nil
}

func (ob *OrderBook) GetState(key string) ([]byte, error) {
	if key != MatchingSnapshot {
		return nil, fmt.Errorf("Unknown key for matching engine: %s", key)
	}

	// Copy all the state into a domain object
	payload := ob.buildPayload()

	// Convert the domain object into a protobuf payload message
	p := payload.IntoProto()

	data, err := proto.Marshal(p)
	if err != nil {
		return nil, err
	}
	return data, nil
}

func (ob *OrderBook) buildPayload() *types.Payload {
	state := types.MatchingBook{}

	state.MarketID = ob.marketID
	state.Buy = ob.copyBuySide()
	state.Sell = ob.copySellSide()
	state.LastTradedPrice = ob.lastTradedPrice
	state.Auction = ob.auction
	state.BatchID = ob.batchID

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

func (ob *OrderBook) copyBuySide() []*types.Order {
	orders := make([]*types.Order, 0)
	pricelevels := ob.buy.getLevels()
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

func (ob *OrderBook) copySellSide() []*types.Order {
	orders := make([]*types.Order, 0)
	pricelevels := ob.sell.getLevels()
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

func (ob *OrderBook) LoadState(payload *types.PayloadMatchingBook) {
	mb := payload.MatchingBook

	ob.reset()
	ob.marketID = mb.MarketID
	ob.batchID = mb.BatchID
	ob.auction = mb.Auction
	ob.lastTradedPrice = mb.LastTradedPrice

	for _, o := range mb.Buy {
		_, err := ob.SubmitOrder(o)
		if err != nil {
			ob.log.Fatal("Error submitting buy order while loading snapshot")
		}
	}

	for _, o := range mb.Sell {
		_, err := ob.SubmitOrder(o)
		if err != nil {
			ob.log.Fatal("Error submitting sell order while loading snapshot")
		}
	}

	// If we are in an auction we need to build the IP&V structure
	if ob.auction {
		ob.indicativePriceAndVolume = NewIndicativePriceAndVolume(ob.log, ob.buy, ob.sell)
	}
}
