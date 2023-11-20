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
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	dtypes "code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/subscribers/mocks"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestSlowConsumerIsDisconnected(t *testing.T) {
	ctx, cancelFn := context.WithCancel(context.Background())
	defer cancelFn()

	ctrl := gomock.NewController(t)

	broker := mocks.NewMockBroker(ctrl)
	broker.EXPECT().Subscribe(gomock.Any())
	broker.EXPECT().Unsubscribe(gomock.Any())

	maxBufferSize := 3
	testStreamSubscription := TestStreamSubscription{events: make(chan []*eventspb.BusEvent)}

	s := NewService(logging.NewTestLogger(), broker, maxBufferSize)
	out, _ := s.ObserveEventsOnStream(ctx, 2, testStreamSubscription)

	testStreamSubscription.events <- []*eventspb.BusEvent{
		events.NewAccountEvent(ctx, dtypes.Account{
			ID: "acc-1",
		}).StreamMessage(),
		events.NewAccountEvent(ctx, dtypes.Account{
			ID: "acc-2",
		}).StreamMessage(),
	}

	events1 := <-out
	assert.Equal(t,
		[]*eventspb.BusEvent{
			events.NewAccountEvent(ctx, dtypes.Account{ID: "acc-1"}).StreamMessage(),
			events.NewAccountEvent(ctx, dtypes.Account{ID: "acc-2"}).StreamMessage(),
		},
		events1)

	testStreamSubscription.events <- []*eventspb.BusEvent{
		events.NewAccountEvent(ctx, dtypes.Account{
			ID: "acc-3",
		}).StreamMessage(),
		events.NewAccountEvent(ctx, dtypes.Account{
			ID: "acc-4",
		}).StreamMessage(),
		events.NewAccountEvent(ctx, dtypes.Account{
			ID: "acc-5",
		}).StreamMessage(),
		events.NewAccountEvent(ctx, dtypes.Account{
			ID: "acc-6",
		}).StreamMessage(),
	}

	// We expect this channel to close
	for range out {
	}
}

type TestStreamSubscription struct {
	events chan []*eventspb.BusEvent
}

func (t TestStreamSubscription) Halt() {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) Push(evts ...events.Event) {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) UpdateBatchSize(ctx context.Context, size int) []*eventspb.BusEvent {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) Types() []events.Type {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) GetData(ctx context.Context) []*eventspb.BusEvent {
	return <-t.events
}

func (t TestStreamSubscription) C() chan<- []events.Event {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) Closed() <-chan struct{} {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) Skip() <-chan struct{} {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) SetID(id int) {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) ID() int {
	// TODO implement me
	panic("implement me")
}

func (t TestStreamSubscription) Ack() bool {
	// TODO implement me
	panic("implement me")
}
