package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
)

type MarketDepthService interface {
	AddOrder(order *types.Order, vegaTime time.Time, sequenceNumber uint64)
}

type MarketDepth struct {
	subscriber
	depthService MarketDepthService
}

// NewMarketDepthBuilder constructor to create a market depth subscriber
func NewMarketDepth(depthService MarketDepthService) *MarketDepth {
	m := MarketDepth{
		depthService: depthService,
	}

	return &m
}

func (m *MarketDepth) Types() []events.Type {
	return []events.Type{events.OrderEvent}
}

func (m *MarketDepth) Push(ctx context.Context, evt events.Event) error {
	m.consume(ctx, evt.(OrderEvent))
	return nil
}

func (m *MarketDepth) consume(ctx context.Context, event OrderEvent) {
	order, err := types.OrderFromProto(event.Order())
	if err != nil {
		panic(err)
	}
	m.depthService.AddOrder(order, m.vegaTime, event.Sequence())
}
