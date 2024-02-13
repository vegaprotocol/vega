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

package broker_test

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"os/exec"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/broker"
	"code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.nanomsg.org/mangos/v3/protocol/pair"
)

type brokerTst struct {
	*broker.Broker
	cfunc context.CancelFunc
	ctx   context.Context
	ctrl  *gomock.Controller
	stats *stats.Blockchain
}

type evt struct {
	t       events.Type
	ctx     context.Context
	sid     uint64
	id      string
	cid     string
	txHash  string
	blockNr int64
}

func getBroker(t *testing.T) *brokerTst {
	t.Helper()
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	statistics := stats.NewBlockchain()
	broker, _ := broker.New(ctx, logging.NewTestLogger(), broker.NewDefaultConfig(), statistics)
	return &brokerTst{
		Broker: broker,
		cfunc:  cfunc,
		ctx:    ctx,
		ctrl:   ctrl,
		stats:  statistics,
	}
}

func (b brokerTst) randomEvt() *evt {
	idString := "generic-id"
	if ctxV, ok := b.ctx.Value("traceID").(string); ok {
		idString = ctxV
	}
	return &evt{
		t:       events.All,
		ctx:     b.ctx,
		id:      idString,
		cid:     "testchain",
		txHash:  "testTxHash",
		blockNr: 0,
	}
}

func (b *brokerTst) Finish() {
	b.cfunc()
}

func TestSequenceIDGen(t *testing.T) {
	t.Run("Sequence ID is correctly - events dispatched per block (ordered)", testSequenceIDGenSeveralBlocksOrdered)
	t.Run("Sequence ID is correctly - events dispatched for several blocks at the same time", testSequenceIDGenSeveralBlocksUnordered)
}

func TestSubscribe(t *testing.T) {
	t.Run("Subscribe and unsubscribe required - success", testSubUnsubSuccess)
	t.Run("Subscribe reuses keys", testSubReuseKey)
	t.Run("Unsubscribe automatically if subscriber is closed", testUnsubscribeAutomaticallyIfSubscriberIsClosed)
}

func TestStream(t *testing.T) {
	t.Run("Streams events over socket", testStreamsOverSocket)
	t.Run("Stops process if can not send to socket", testStopsProcessOnStreamError)
}

func TestSendEvent(t *testing.T) {
	t.Run("Skip optional subscribers", testSkipOptional)
	t.Run("Skip optional subscribers in a batch send", testSendBatchChannel)
	t.Run("Send batch to ack subscriber", testSendBatch)
	t.Run("Stop sending if context is cancelled", testStopCtx)
	t.Run("Stop sending if context is cancelled, even new events", testStopCtxSendAgain)
	t.Run("Skip subscriber based on channel state", testSkipSubscriberBasedOnChannelState)
	t.Run("Send only to typed subscriber (also tests TxErrEvents are skipped)", testEventTypeSubscription)
}

func testSequenceIDGenSeveralBlocksOrdered(t *testing.T) {
	t.Parallel()
	tstBroker := getBroker(t)
	defer tstBroker.Finish()
	ctxH1, ctxH2 := vgcontext.WithTraceID(tstBroker.ctx, "hash-1"), vgcontext.WithTraceID(tstBroker.ctx, "hash-2")
	dataH1 := []events.Event{
		events.NewTime(ctxH1, time.Now()),
		events.NewPartyEvent(ctxH1, types.Party{Id: "test-party-h1"}),
	}
	dataH2 := []events.Event{
		events.NewTime(ctxH2, time.Now()),
		events.NewPartyEvent(ctxH2, types.Party{Id: "test-party-h2"}),
	}
	allData := make([]events.Event, 0, len(dataH1)+len(dataH2))
	done := make(chan struct{})
	mu := sync.Mutex{}
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	sub.EXPECT().Types().Times(2).Return(nil)
	sub.EXPECT().Ack().Times(1).Return(true)
	sub.EXPECT().Skip().AnyTimes().Return(tstBroker.ctx.Done())
	sub.EXPECT().Closed().AnyTimes().Return(tstBroker.ctx.Done())
	sub.EXPECT().Push(gomock.Any()).AnyTimes().Do(func(evts ...events.Event) {
		// race detector complains about appending here, because data comes from
		// different go routines, so we'll use a quick & dirty fix: mutex the slice
		mu.Lock()
		defer mu.Unlock()
		allData = append(allData, evts...)
		if len(allData) >= cap(allData) {
			close(done)
		}
	})
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	k := tstBroker.Subscribe(sub)
	// send batches for both events - hash 2 after hash 1
	tstBroker.SendBatch(dataH1)
	tstBroker.SendBatch(dataH2)
	seqH1 := []uint64{}
	seqH2 := []uint64{}
	for i := range dataH1 {
		seqH1 = append(seqH1, dataH1[i].Sequence())
		seqH2 = append(seqH2, dataH2[i].Sequence())
	}
	assert.Equal(t, seqH1, seqH2)
	<-done
	tstBroker.Unsubscribe(k)
	assert.NotEqual(t, seqH1[0], seqH2[1]) // the two are equal, we can compare X-slice
	assert.Equal(t, len(allData), len(dataH1)+len(dataH2))
}

func testSequenceIDGenSeveralBlocksUnordered(t *testing.T) {
	t.Parallel()
	tstBroker := getBroker(t)
	defer tstBroker.Finish()
	ctxH1, ctxH2 := vgcontext.WithTraceID(tstBroker.ctx, "hash-1"), vgcontext.WithTraceID(tstBroker.ctx, "hash-2")
	dataH1 := []events.Event{
		events.NewTime(ctxH1, time.Now()),
		events.NewPartyEvent(ctxH1, types.Party{Id: "test-party-h1"}),
	}
	dataH2 := []events.Event{
		events.NewTime(ctxH2, time.Now()),
		events.NewPartyEvent(ctxH2, types.Party{Id: "test-party-h2"}),
	}
	allData := make([]events.Event, 0, len(dataH1)+len(dataH2))
	mu := sync.Mutex{}
	done := make(chan struct{})
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	sub.EXPECT().Types().Times(2).Return(nil)
	sub.EXPECT().Ack().Times(1).Return(true)
	sub.EXPECT().Skip().AnyTimes().Return(tstBroker.ctx.Done())
	sub.EXPECT().Closed().AnyTimes().Return(tstBroker.ctx.Done())
	sub.EXPECT().Push(gomock.Any()).AnyTimes().Do(func(evts ...events.Event) {
		mu.Lock()
		defer mu.Unlock()
		allData = append(allData, evts...)
		if len(allData) >= cap(allData) {
			close(done)
		}
	})
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	k := tstBroker.Subscribe(sub)
	// We can't use sendBatch here: we use the traceID of the first event in the batch to determine
	// the hash (batch-sending events can only happen within a single block)
	for i := range dataH1 {
		tstBroker.Send(dataH1[i])
		tstBroker.Send(dataH2[i])
	}
	seqH1 := []uint64{}
	seqH2 := []uint64{}
	for i := range dataH1 {
		seqH1 = append(seqH1, dataH1[i].Sequence())
		seqH2 = append(seqH2, dataH2[i].Sequence())
	}
	assert.Equal(t, seqH1, seqH2)
	<-done
	tstBroker.Unsubscribe(k)
	assert.NotEqual(t, seqH1[0], seqH2[1]) // the two are equal, we can compare X-slice
	assert.Equal(t, len(allData), len(dataH1)+len(dataH2))
}

func testSubUnsubSuccess(t *testing.T) {
	t.Parallel()
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	reqSub := mocks.NewMockSubscriber(broker.ctrl)
	// subscribe + unsubscribe -> 2 calls
	sub.EXPECT().Types().Times(2).Return(nil)
	sub.EXPECT().Ack().Times(1).Return(false)
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	reqSub.EXPECT().Types().Times(2).Return(nil)
	reqSub.EXPECT().Ack().Times(1).Return(true)
	reqSub.EXPECT().SetID(gomock.Any()).Times(1)
	k1 := broker.Subscribe(sub)    // not required
	k2 := broker.Subscribe(reqSub) // required
	assert.NotZero(t, k1)
	assert.NotZero(t, k2)
	assert.NotEqual(t, k1, k2)
	broker.Unsubscribe(k1)
	broker.Unsubscribe(k2)
	// no calls to subs expected once they are unsubscribed
	broker.Send(broker.randomEvt())
}

func testSubReuseKey(t *testing.T) {
	t.Parallel()
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	sub.EXPECT().Types().Times(4).Return(nil)
	sub.EXPECT().Ack().Times(1).Return(false)
	sub.EXPECT().SetID(gomock.Any()).Times(2)
	k1 := broker.Subscribe(sub)
	sub.EXPECT().Ack().Times(1).Return(true)
	assert.NotZero(t, k1)
	broker.Unsubscribe(k1)
	k2 := broker.Subscribe(sub)
	assert.Equal(t, k1, k2)
	broker.Unsubscribe(k2)
	// second unsubscribe is a no-op
	broker.Unsubscribe(k1)
}

func testUnsubscribeAutomaticallyIfSubscriberIsClosed(t *testing.T) {
	t.Parallel()

	testedBroker := getBroker(t)
	defer testedBroker.Finish()

	// The Broker.Send() method launches a go routine. As a result, to control
	// its state, we will use an unbuffered (blocking) channel to wait until
	// we unblock it by sending a signal, or it timeouts.
	waiter := newWaiter()

	// First, we add the subscriber to the broker.
	sub := mocks.NewMockSubscriber(testedBroker.ctrl)

	// This tells the broker to treat the subscriber synchronously.
	sub.EXPECT().Ack().Times(1).Return(true)
	// This tells the broker the subscriber is listening to all even types.
	sub.EXPECT().Types().AnyTimes().Return(nil)
	sub.EXPECT().SetID(1).Times(1)

	subID := testedBroker.Subscribe(sub)
	defer testedBroker.Unsubscribe(subID)
	require.Equal(t, 1, subID)

	// Then, we send an event that should be pushed since the subscriber is not
	// closed and doesn't want this event to be skipped.
	event1 := testedBroker.randomEvt()

	notClosedCh := make(chan struct{})
	defer close(notClosedCh)
	dontSkipCh := make(chan struct{})
	defer close(dontSkipCh)

	// We configure the subscriber to tell the broker to push the event.
	sub.EXPECT().Closed().Times(1).Return(notClosedCh)
	sub.EXPECT().Skip().Times(1).Return(dontSkipCh)
	sub.EXPECT().Push(event1).Times(1).Do(func(_ *evt) {
		// When the event is pushed, we tell the test to continue.
		waiter.Unblock()
	})

	// We send the first event.
	testedBroker.Send(event1)

	// Let's wait for a signal.
	if err := waiter.Wait(); err != nil {
		t.Fatalf("Subscriber.Skip() was never called: %v", err)
	}

	// To finish, we send another event, but, this time, the subscriber is
	// closed, so the broker does not push the event.
	event2 := testedBroker.randomEvt()

	sub.EXPECT().Closed().Times(1).DoAndReturn(func() <-chan struct{} {
		iAmClosed := make(chan struct{})
		// Returning a closed channel is how the subscriber notifies the broker
		// it is closed.
		close(iAmClosed)

		// We warn the test the method we are expecting to be called has been
		// called.
		waiter.Unblock()
		return iAmClosed
	})
	sub.EXPECT().Skip().AnyTimes().Return(dontSkipCh)
	sub.EXPECT().Push(gomock.Any()).Times(0)

	// We send the second event. It should be pushed.
	testedBroker.Send(event2)

	// Let's wait for a signal.
	if err := waiter.Wait(); err != nil {
		t.Fatalf("Subscriber.Skip() was never called: %v", err)
	}
}

func testSendBatch(t *testing.T) {
	t.Parallel()
	tstBroker := getBroker(t)
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	cancelCh := make(chan struct{})
	defer func() {
		tstBroker.Finish()
		close(cancelCh)
	}()
	sub.EXPECT().Types().Times(1).Return(nil)
	sub.EXPECT().Ack().AnyTimes().Return(true)
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	k1 := tstBroker.Subscribe(sub)
	assert.NotZero(t, k1)
	data := []events.Event{
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
	}
	// ensure all 3 events are being sent (wait for routine to spawn)
	sub.EXPECT().Closed().AnyTimes().Return(cancelCh)
	sub.EXPECT().Skip().AnyTimes().Return(cancelCh)
	wg := sync.WaitGroup{}
	wg.Add(1)
	sub.EXPECT().Push(gomock.Any()).Times(1).Do(func(evts ...events.Event) {
		assert.Equal(t, len(data), len(evts))
		wg.Done()
	})

	// send events
	tstBroker.SendBatch(data)
	wg.Wait()
	require.Equal(t, uint64(3), tstBroker.stats.CurrentEventsInBatch())
	tstBroker.stats.NewBatch()
	require.Equal(t, uint64(3), tstBroker.stats.TotalEventsLastBatch())
}

func testSendBatchChannel(t *testing.T) {
	t.Parallel()
	tstBroker := getBroker(t)
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	skipCh, closedCh, cCh := make(chan struct{}), make(chan struct{}), make(chan []events.Event, 1)
	defer func() {
		tstBroker.Finish()
		close(closedCh)
		close(skipCh)
	}()
	twg := sync.WaitGroup{}
	twg.Add(2)
	sub.EXPECT().Types().Times(2).Return(nil).Do(func() {
		twg.Done()
	})
	sub.EXPECT().Ack().AnyTimes().Return(false)
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	k1 := tstBroker.Subscribe(sub)
	assert.NotZero(t, k1)
	batch2 := []events.Event{
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
	}
	evts := []events.Event{
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
	}
	// ensure both batches are sent
	wg := sync.WaitGroup{}
	// 3 calls, only the first batch will be sent
	// third call is routine that tries to send the second batch. This will of course fail
	wg.Add(3)
	sub.EXPECT().Closed().AnyTimes().Return(closedCh)
	sub.EXPECT().Skip().AnyTimes().Return(skipCh)
	// we try to get the channel 3 times, only 1 of the attempts will actually publish the events
	sub.EXPECT().C().Times(3).Return(cCh).Do(func() {
		// Done call each time we tried sending an event
		wg.Done()
	})

	// send events
	tstBroker.SendBatch(evts)
	tstBroker.SendBatch(batch2)
	wg.Wait()
	// we've tried to send 2 batches of events, subscriber could only accept one. Check state of all the things
	// we need to unsubscribe the subscriber, because we're closing the channels and race detector complains
	// because there's a loop calling functions that are returning the channels we're closing here
	tstBroker.Unsubscribe(k1)
	// ensure unsubscribe has returned
	twg.Wait()

	// get our batches
	batches := [][]events.Event{
		<-cCh, <-cCh,
	}

	// assert we have all events now.
	batchSizes := map[int]struct{}{}
	evtSeq := map[uint64]struct{}{}
	for _, batch := range batches {
		batchSizes[len(batch)] = struct{}{}
		for _, v := range batch {
			evtSeq[v.Sequence()] = struct{}{}
		}
	}

	// now ensure we have the batch with right sizes
	_, ok := batchSizes[len(batch2)]
	assert.True(t, ok, "missing batch of size ", len(batch2))
	_, ok = batchSizes[len(evts)]
	assert.True(t, ok, "missing batch of size ", len(evts))

	// now ensure we got all sequence IDs
	for _, v := range append(evts, batch2...) {
		_, ok := evtSeq[v.Sequence()]
		if !ok {
			t.Fatalf("missing event sequence from batches %v", v.Sequence())
		}
	}
}

func testSkipOptional(t *testing.T) {
	t.Parallel()
	tstBroker := getBroker(t)
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	skipCh, closedCh, cCh := make(chan struct{}), make(chan struct{}), make(chan []events.Event, 1)
	defer func() {
		tstBroker.Finish()
		close(closedCh)
		close(skipCh)
	}()
	twg := sync.WaitGroup{}
	twg.Add(2)
	sub.EXPECT().Types().Times(2).Return(nil).Do(func() {
		twg.Done()
	})
	sub.EXPECT().Ack().AnyTimes().Return(false)
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	k1 := tstBroker.Subscribe(sub)
	assert.NotZero(t, k1)

	evts := []*evt{
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
	}
	// ensure all 3 events are being sent (wait for routine to spawn)
	wg := sync.WaitGroup{}
	wg.Add(len(evts)*2 - 1)
	sub.EXPECT().Closed().AnyTimes().Return(closedCh)
	sub.EXPECT().Skip().AnyTimes().Return(skipCh)
	// we try to get the channel 3 times, only 1 of the attempts will actually publish the event
	// the other 2 attempts will run in a routine
	sub.EXPECT().C().Times(len(evts)*2 - 1).Return(cCh).Do(func() {
		// Done call each time we tried sending an event
		wg.Done()
	})

	// send events
	for _, e := range evts {
		tstBroker.Send(e)
	}
	wg.Wait()
	// we've tried to send 3 events, subscriber could only accept one. Check state of all the things
	// we need to unsubscribe the subscriber, because we're closing the channels and race detector complains
	// because there's a loop calling functions that are returning the channels we're closing here
	tstBroker.Unsubscribe(k1)
	// ensure unsubscribe has returned
	twg.Wait()
	require.Equal(t, uint64(len(evts)), tstBroker.stats.CurrentEventsInBatch())
	tstBroker.stats.NewBatch()
	require.Equal(t, uint64(len(evts)), tstBroker.stats.TotalEventsLastBatch())

	// make a map to check all sequences
	seq := map[uint64]struct{}{}
	for i := len(evts); i != 0; i-- {
		ev := <-cCh
		assert.NotEmpty(t, ev)
		for _, e := range ev {
			seq[e.Sequence()] = struct{}{}
		}
	}

	// no verify all ev sequence are received
	for _, ev := range evts {
		_, ok := seq[ev.Sequence()]
		if !ok {
			t.Fatalf("missing event sequence from received events %v", ev.Sequence())
		}
	}

	// make sure the channel is empty (no writes were pending)
	assert.Equal(t, 0, len(cCh))
}

func testStopCtx(t *testing.T) {
	t.Parallel()
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	ch := make(chan struct{})
	sub.EXPECT().Closed().AnyTimes().Return(ch)
	sub.EXPECT().Skip().AnyTimes().Return(ch)
	// no calls sub are expected, we cancelled the context
	broker.cfunc()
	sub.EXPECT().Types().Times(2).Return(nil)
	sub.EXPECT().Ack().AnyTimes().Return(true)
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	k1 := broker.Subscribe(sub) // required sub
	assert.NotZero(t, k1)
	broker.Send(broker.randomEvt())
	// calling unsubscribe acquires lock, so we can ensure the Send call has returned
	broker.Unsubscribe(k1)
	close(ch)
}

func testStopCtxSendAgain(t *testing.T) {
	t.Parallel()
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	ch := make(chan struct{})
	sub.EXPECT().Closed().AnyTimes().Return(ch)
	sub.EXPECT().Skip().AnyTimes().Return(ch)
	// no calls sub are expected, we cancelled the context
	broker.cfunc()
	sub.EXPECT().Types().Times(2).Return(nil)
	sub.EXPECT().Ack().AnyTimes().Return(true)
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	k1 := broker.Subscribe(sub) // required sub
	assert.NotZero(t, k1)
	broker.Send(broker.randomEvt())
	broker.cfunc()
	broker.Send(broker.randomEvt())
	// calling unsubscribe acquires lock, so we can ensure the Send call has returned
	broker.Unsubscribe(k1)
	close(ch)
}

func testSkipSubscriberBasedOnChannelState(t *testing.T) {
	t.Parallel()

	testedBroker := getBroker(t)
	defer testedBroker.Finish()

	// The Broker.Send() method launches a go routine. As a result, to control
	// its state, we will use an unbuffered (blocking) channel to wait until
	// we unblock it by sending a signal, or it timeouts.
	waiter := newWaiter()

	// First, we add the subscriber to the broker.
	sub := mocks.NewMockSubscriber(testedBroker.ctrl)

	// This tells the broker to treat the subscriber synchronously.
	sub.EXPECT().Ack().Times(1).Return(true)
	// This tells the broker the subscriber is listening to all even types.
	sub.EXPECT().Types().AnyTimes().Return(nil)
	sub.EXPECT().SetID(1).Times(1)

	subID := testedBroker.Subscribe(sub)
	defer testedBroker.Unsubscribe(subID)
	require.Equal(t, 1, subID)

	// Then, we send an event that should be skipped since the subscriber tell
	// us to do so.
	event1 := testedBroker.randomEvt()

	notClosedCh := make(chan struct{})
	defer close(notClosedCh)

	// We configure the subscriber to tell the broker to skip the sending.
	sub.EXPECT().Closed().AnyTimes().Return(notClosedCh)
	sub.EXPECT().Skip().Times(1).DoAndReturn(func() <-chan struct{} {
		skipMeCh := make(chan struct{})
		// Returning a closed channel is how the subscriber notifies the broker
		// it wants to skip next events.
		close(skipMeCh)

		// We warn the test the method we are expecting to be called has been
		// called.
		waiter.Unblock()
		return skipMeCh
	})
	sub.EXPECT().Push(event1).Times(0)

	// We send the first event. It should be ignored.
	testedBroker.Send(event1)

	// Let's wait for a signal.
	if err := waiter.Wait(); err != nil {
		t.Fatalf("Subscriber.Skip() was never called: %v", err)
	}

	// To finish, we send another event, but, this time, the subscriber doesn't
	// want to skip it. So, it should be pushed.
	event2 := testedBroker.randomEvt()

	dontSkipCh := make(chan struct{})
	defer close(dontSkipCh)

	sub.EXPECT().Closed().AnyTimes().Return(notClosedCh)
	sub.EXPECT().Skip().AnyTimes().Return(dontSkipCh)
	sub.EXPECT().Push(event2).Times(1).Do(func(_ ...events.Event) {
		// We warn the test the method we are expecting to be called has been
		// called.
		waiter.Unblock()
	})

	// We send the second event. It should be pushed.
	testedBroker.Send(event2)

	// Let's wait for a signal.
	if err := waiter.Wait(); err != nil {
		t.Fatalf("Subscriber.Skip() was never called: %v", err)
	}
}

// test making sure that events are sent only to subs that are interested in it.
func testEventTypeSubscription(t *testing.T) {
	t.Parallel()
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	allSub := mocks.NewMockSubscriber(broker.ctrl)
	diffSub := mocks.NewMockSubscriber(broker.ctrl)
	skipCh, closeCh := make(chan struct{}), make(chan struct{})
	event := broker.randomEvt()
	event.t = events.TimeUpdate
	wg := sync.WaitGroup{}
	wg.Add(2)
	// Closed check
	sub.EXPECT().Closed().AnyTimes().Return(closeCh)
	diffSub.EXPECT().Closed().AnyTimes().Return(closeCh) // can use the same channels, we're not closing them anyway
	allSub.EXPECT().Closed().AnyTimes().Return(closeCh)
	// skip check
	sub.EXPECT().Skip().AnyTimes().Return(skipCh)
	allSub.EXPECT().Skip().AnyTimes().Return(skipCh)
	diffSub.EXPECT().Skip().AnyTimes().Return(skipCh)
	// actually push the event - diffSub expects nothing
	sub.EXPECT().Push(gomock.Any()).Times(1).Do(func(_ interface{}) {
		wg.Done()
	})
	allSub.EXPECT().Push(gomock.Any()).Times(1).Do(func(_ interface{}) {
		wg.Done()
	})
	// the event types this subscriber is interested in
	sub.EXPECT().Types().Times(2).Return([]events.Type{events.TimeUpdate})
	allSub.EXPECT().Types().Times(2).Return(nil) // subscribed to ALL events
	// fake type:
	different := events.Type(int(events.All) + int(events.TimeUpdate) + 1 + int(events.TxErrEvent)) // this value cannot exist as an events.Type value
	diffSub.EXPECT().Types().Times(2).Return([]events.Type{different})
	// subscribe the subscriber
	sub.EXPECT().Ack().AnyTimes().Return(true)
	diffSub.EXPECT().Ack().AnyTimes().Return(true)
	allSub.EXPECT().Ack().AnyTimes().Return(true)
	sub.EXPECT().SetID(gomock.Any()).Times(1)
	diffSub.EXPECT().SetID(gomock.Any()).Times(1)
	allSub.EXPECT().SetID(gomock.Any()).Times(1)
	k1 := broker.Subscribe(sub)     // required sub
	k2 := broker.Subscribe(diffSub) // required sub, but won't be used anyway
	k3 := broker.Subscribe(allSub)
	assert.NotZero(t, k1)
	assert.NotZero(t, k2)
	assert.NotZero(t, k3)
	assert.NotEqual(t, k1, k2)
	// send a time event
	broker.Send(events.NewTime(context.Background(), time.Now()))
	// ensure the event was delivered
	wg.Wait()
	// unsubscribe the subscriber, now we're done
	broker.Unsubscribe(k1)
	broker.Unsubscribe(k2)
	broker.Unsubscribe(k3)
	close(skipCh)
	close(closeCh)
}

func testStreamsOverSocket(t *testing.T) {
	t.Parallel()
	ctx, cfunc := context.WithCancel(context.Background())
	config := broker.NewDefaultConfig()
	config.Socket.Enabled = true
	config.Socket.Transport = "inproc"

	sock, err := pair.NewSocket()
	assert.NoError(t, err)

	addr := fmt.Sprintf(
		"inproc://%s",
		net.JoinHostPort(config.Socket.Address, fmt.Sprintf("%d", config.Socket.Port)),
	)
	err = sock.Listen(addr)
	assert.NoError(t, err)

	broker, _ := broker.New(ctx, logging.NewTestLogger(), config, stats.NewBlockchain())

	defer func() {
		cfunc()
		sock.Close()
	}()

	sentEvent := events.NewTime(ctx, time.Date(2020, time.December, 25, 0o0, 0o1, 0o1, 0, time.UTC))

	broker.Send(sentEvent)

	receivedBytes, err := sock.Recv()
	assert.NoError(t, err)

	var receivedEvent eventspb.BusEvent
	err = proto.Unmarshal(receivedBytes, &receivedEvent)
	assert.NoError(t, err)
	assert.True(t, proto.Equal(sentEvent.StreamMessage(), &receivedEvent))
}

func testStopsProcessOnStreamError(t *testing.T) {
	t.Parallel()
	if os.Getenv("RUN_TEST") == "1" {
		ctx, cfunc := context.WithCancel(context.Background())
		config := broker.NewDefaultConfig()
		config.Socket.Enabled = true
		config.Socket.Transport = "inproc"

		// Having such a small buffers will make the process fail
		config.Socket.SocketChannelBufferSize = 0
		config.Socket.EventChannelBufferSize = 0

		sock, err := pair.NewSocket()
		assert.NoError(t, err)

		addr := fmt.Sprintf(
			"inproc://%s",
			net.JoinHostPort(config.Socket.Address, fmt.Sprintf("%d", config.Socket.Port)),
		)
		err = sock.Listen(addr)
		assert.NoError(t, err)

		broker, _ := broker.New(ctx, logging.NewTestLogger(), config, stats.NewBlockchain())

		defer func() {
			cfunc()
			sock.Close()
		}()

		sentEvent := events.NewTime(ctx, time.Date(2020, time.December, 25, 0o0, 0o1, 0o1, 0, time.UTC))

		broker.Send(sentEvent)
		// One of the next call should terminate the process
		broker.Send(sentEvent)
		broker.Send(sentEvent)
		broker.Send(sentEvent)
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run=TestStream/Stops_process_if_can_not_send_to_socket")
	cmd.Env = append(os.Environ(), "RUN_TEST=1")
	err := cmd.Run()
	if e, ok := err.(*exec.ExitError); ok && !e.Success() {
		return
	}
	t.Fatalf("process ran with err %v, want exit status 1", err)
}

func (e evt) Type() events.Type {
	return e.t
}

func (e evt) Context() context.Context {
	return e.ctx
}

func (e evt) Replace(context.Context) {}

func (e *evt) SetSequenceID(s uint64) {
	e.sid = s
}

func (e evt) Sequence() uint64 {
	return e.sid
}

func (e evt) TraceID() string {
	return e.id
}

func (e evt) ChainID() string {
	return e.cid
}

func (e evt) TxHash() string {
	return e.txHash
}

func (e evt) BlockNr() int64 {
	return e.blockNr
}

func (e evt) StreamMessage() *eventspb.BusEvent {
	return nil
}

func (e evt) CompositeCount() uint64 {
	return 1
}

type waiter struct {
	ch  chan struct{}
	ctx context.Context
}

func (c *waiter) Unblock() {
	c.ch <- struct{}{}
}

func (c *waiter) Wait() error {
	for {
		select {
		case <-c.ch:
			return nil
		case <-c.ctx.Done():
			return errors.New("waiter timed out")
		}
	}
}

// newWaiter waits until it's unblocked or after 30 seconds elapsed, so we
// don't block the tests.
func newWaiter() *waiter {
	ch := make(chan struct{}, 1)
	ctx, cancelFn := context.WithCancel(context.Background())
	ticker := time.NewTicker(30 * time.Second)

	go func() {
		<-ticker.C
		cancelFn()
		ticker.Stop()
		close(ch)
	}()

	return &waiter{
		ch:  ch,
		ctx: ctx,
	}
}
