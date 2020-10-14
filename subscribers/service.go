package subscribers

import (
	"context"
	"fmt"

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
	in, out := make(chan int), make(chan []*types.BusEvent)
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
			fmt.Printf("STOP LISTENING TO EVENTS IN SERVICE\n")
		}()
		ret := retries
		for {
			select {
			case bs := <-in:
				fmt.Printf("batch size changed!\n")
				// batch size changed: drain buffer and send data
				data := sub.UpdateBatchSize(ctx, bs)
				if len(data) > 0 {
					out <- data
				}
				// reset retry count
				ret = retries
			default:
				fmt.Printf("default then!\n")
				// wait for actual changes
				data := sub.GetData(ctx)
				// this is a very rare thing, but it can happen
				if len(data) == 0 && ctx.Err() == nil {
					continue
				}
				select {
				case <-ctx.Done():
					fmt.Printf("context is doneeee!!\n")
					return
				case out <- data:
					fmt.Printf("reducing retries baby! %v\n", retries)
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
