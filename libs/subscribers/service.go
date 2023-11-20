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

package subscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/libs/broker"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mocks.go -package mocks code.vegaprotocol.io/vega/libs/subscribers Broker
type Broker interface {
	Subscribe(s broker.Subscriber) int
	Unsubscribe(id int)
}

type Service struct {
	log           *logging.Logger
	broker        Broker
	maxBufferSize int
}

type StreamSubscription interface {
	Halt()
	Push(evts ...events.Event)
	UpdateBatchSize(ctx context.Context, size int) []*eventspb.BusEvent
	Types() []events.Type
	GetData(ctx context.Context) []*eventspb.BusEvent
	C() chan<- []events.Event
	Closed() <-chan struct{}
	Skip() <-chan struct{}
	SetID(id int)
	ID() int
	Ack() bool
}

const namedLogger = "subscribers"

func NewService(log *logging.Logger, broker Broker, maxBufferSize int) *Service {
	return &Service{
		log:           log.Named(namedLogger),
		broker:        broker,
		maxBufferSize: maxBufferSize,
	}
}

func (s *Service) ObserveEvents(ctx context.Context, retries int, eTypes []events.Type, batchSize int, filters ...EventFilter) (<-chan []*eventspb.BusEvent, chan<- int) {
	return s.ObserveEventsOnStream(ctx, retries, NewStreamSub(ctx, eTypes, batchSize, filters...))
}

func (s *Service) ObserveEventsOnStream(ctx context.Context, retries int,
	sub StreamSubscription,
) (<-chan []*eventspb.BusEvent, chan<- int) {
	// one batch buffer for the out channel
	in, out := make(chan int), make(chan []*eventspb.BusEvent, 1)
	ctx, cfunc := context.WithCancel(ctx)

	// use stream subscriber
	// use buffer size of 0 for the time being
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
				dataLength := len(data)

				if dataLength > s.maxBufferSize {
					s.log.Warningf("slow consumer detected, closing event observer")
					return
				}

				if dataLength > 0 {
					trySend()
				}
			default:
				// wait for actual changes
				data = append(data, sub.GetData(ctx)...)
				dataLength := len(data)

				if dataLength > s.maxBufferSize {
					s.log.Warningf("slow consumer detected, closing event observer")
					return
				}

				// this is a very rare thing, but it can happen
				if dataLength > 0 {
					trySend()
				}
			}
		}
	}()
	return out, in
}
