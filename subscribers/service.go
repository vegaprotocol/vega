package subscribers

import (
	"context"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_bus_mock.go -package mocks code.vegaprotocol.io/vega/subscribers Broker
type Broker interface {
	Subscribe(s broker.Subscriber) int
	Unsubscribe(id int)
}

type Service struct {
	broker Broker
}

func NewService(broker Broker) *Service {
	return &Service{
		broker: broker,
	}
}

func (s *Service) ObserveEvents(ctx context.Context, retries int, eTypes []events.Type, batchSize int, filters ...EventFilter) (<-chan []*types.BusEvent, chan<- int) {
	// one batch buffer for the out channel
	in, out := make(chan int), make(chan []*types.BusEvent, 1)
	ctx, cfunc := context.WithCancel(ctx)
	// use stream subscriber
	// use buffer size of 0 for the time being
	sub := NewStreamSub(ctx, eTypes, batchSize, filters...)
	id := s.broker.Subscribe(sub)
	go func() {
		defer func() {
			s.broker.Unsubscribe(id)
			close(out)
			close(in)
			cfunc()
		}()
		ret := retries
		for {
			select {
			case <-ctx.Done():
				return
			case bs := <-in:
				// batch size changed: drain buffer and send data
				data := sub.UpdateBatchSize(ctx, bs)
				if len(data) > 0 {
					out <- data
				}
				// reset retry count
				ret = retries
			default:
				// wait for actual changes
				data := sub.GetData(ctx)
				// this is a very rare thing, but it can happen
				if len(data) == 0 {
					continue
				}
				select {
				case <-ctx.Done():
					return
				case out <- data:
					ret = retries
				default:
					if ret == 0 {
						return
					}
					ret--
				}
			}
		}
	}()
	return out, in
}
