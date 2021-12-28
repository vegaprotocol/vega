package subscribers

import (
	"context"
	"time"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/events"
)

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

func (s *Service) ObserveEvents(ctx context.Context, retries int, eTypes []events.Type, batchSize int, filters ...EventFilter) (<-chan []*eventspb.BusEvent, chan<- int) {
	// one batch buffer for the out channel
	in, out := make(chan int), make(chan []*eventspb.BusEvent, 1)
	ctx, cfunc := context.WithCancel(ctx)
	// use stream subscriber
	// use buffer size of 0 for the time being
	sub := NewStreamSub(ctx, eTypes, batchSize, filters...)
	id := s.broker.Subscribe(sub)

	// makes the tick duration 2000ms max to wait basically
	tickDuration := 10 * time.Millisecond
	retries = 200

	go func() {
		data := []*eventspb.BusEvent{}
		defer func() {
			s.broker.Unsubscribe(id)
			close(out)
			cfunc()
		}()
		ret := retries

		trySend := func() {
			t := time.NewTicker(tickDuration)
			defer t.Stop()
			select {
			case <-ctx.Done():
				return
			case out <- data:
				data = []*eventspb.BusEvent{}
				ret = retries
			case <-t.C:
				if ret == 0 {
					return
				}
				ret--
			}
		}

		for {
			select {
			case <-ctx.Done():
				return
			case bs := <-in:
				// batch size changed: drain buffer and send data
				data = append(data, sub.UpdateBatchSize(ctx, bs)...)
				if len(data) > 0 {
					trySend()
				}
			default:
				// wait for actual changes
				data = append(data, sub.GetData(ctx)...)
				// this is a very rare thing, but it can happen
				if len(data) > 0 {
					trySend()
				}
			}
		}
	}()
	return out, in
}
