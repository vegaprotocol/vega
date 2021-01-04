package broker_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type brokerTst struct {
	*broker.Broker
	cfunc context.CancelFunc
	ctx   context.Context
	ctrl  *gomock.Controller
}

type evt struct {
	t   events.Type
	ctx context.Context
	sid uint64
	id  string
}

func getBroker(t *testing.T) *brokerTst {
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	return &brokerTst{
		Broker: broker.New(ctx),
		cfunc:  cfunc,
		ctx:    ctx,
		ctrl:   ctrl,
	}
}

func (b brokerTst) randomEvt() *evt {
	idString := "generic-id"
	if ctxV, ok := b.ctx.Value("traceID").(string); ok {
		idString = ctxV
	}
	return &evt{
		t:   events.All,
		ctx: b.ctx,
		id:  idString,
	}
}

func (b *brokerTst) Finish() {
	b.cfunc()
	b.ctrl.Finish()
}

func TestSequenceIDGen(t *testing.T) {
	t.Run("Sequence ID is correctly - events dispatched per block (ordered)", testSequenceIDGenSeveralBlocksOrdered)
	t.Run("Sequence ID is correctly - events dispatched for several blocks at the same time", testSequenceIDGenSeveralBlocksUnordered)
}

func TestSubscribe(t *testing.T) {
	t.Run("Subscribe and unsubscribe required - success", testSubUnsubSuccess)
	t.Run("Subscribe reuses keys", testSubReuseKey)
	t.Run("Unsubscribe automatically if subscriber is closed", testAutoUnsubscribe)
}

func TestSendEvent(t *testing.T) {
	t.Run("Skip optional subscribers", testSkipOptional)
	t.Run("Skip optional subscribers in a batch send", testSendBatchChannel)
	t.Run("Send batch to ack subscriber", testSendBatch)
	t.Run("Stop sending if context is cancelled", testStopCtx)
	t.Run("Skip subscriber based on channel state", testSubscriberSkip)
	t.Run("Send only to typed subscriber (also tests TxErrEvents are skipped)", testEventTypeSubscription)
}

func TestTxErrEvents(t *testing.T) {
	t.Run("Ensure TxErrEvents are hidden from ALL subscribers", testTxErrNotAll)
}

func testSequenceIDGenSeveralBlocksOrdered(t *testing.T) {
	tstBroker := getBroker(t)
	defer tstBroker.Finish()
	ctxH1, ctxH2 := contextutil.WithTraceID(tstBroker.ctx, "hash-1"), contextutil.WithTraceID(tstBroker.ctx, "hash-2")
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
	tstBroker := getBroker(t)
	defer tstBroker.Finish()
	ctxH1, ctxH2 := contextutil.WithTraceID(tstBroker.ctx, "hash-1"), contextutil.WithTraceID(tstBroker.ctx, "hash-2")
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
	k := tstBroker.Subscribe(sub)
	// We can't use sendBatch here: we use the traceID of the fisrt event in the batch to determine
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
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	reqSub := mocks.NewMockSubscriber(broker.ctrl)
	// subscribe + unsubscribe -> 2 calls
	sub.EXPECT().Types().Times(2).Return(nil)
	sub.EXPECT().Ack().Times(1).Return(false)
	reqSub.EXPECT().Types().Times(2).Return(nil)
	reqSub.EXPECT().Ack().Times(1).Return(true)
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
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	sub.EXPECT().Types().Times(4).Return(nil)
	sub.EXPECT().Ack().Times(1).Return(false)
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

func testAutoUnsubscribe(t *testing.T) {
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	// sub, auto-unsub, sub again
	sub.EXPECT().Types().Times(3).Return(nil)
	sub.EXPECT().Ack().Times(1).Return(true)
	k1 := broker.Subscribe(sub)
	assert.NotZero(t, k1)
	// set up sub to be closed
	skipCh := make(chan struct{})
	closedCh := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	defer func() {
		close(skipCh)
	}()
	close(closedCh) // close the closed channel, so the subscriber is marked as closed when we try to send an event
	sub.EXPECT().Skip().AnyTimes().Return(skipCh)
	sub.EXPECT().Closed().AnyTimes().Return(closedCh).Do(func() {
		// indicator this function has been called already
		wg.Done()
	})
	// send an event, the subscriber should be marked as closed, and automatically unsubscribed
	broker.Send(broker.randomEvt())
	// introduce some wait mechanism here, because the unsubscribe call acquires its own lock now
	// so it's possible we haven't unsubscribed yet... the waitgroup should introduce enough time
	wg.Wait()
	// now try and subscribe again, the key should be reused
	sub.EXPECT().Ack().Times(1).Return(false)
	k2 := broker.Subscribe(sub)
	assert.Equal(t, k1, k2)
}

func testSendBatch(t *testing.T) {
	tstBroker := getBroker(t)
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	cancelCh := make(chan struct{})
	defer func() {
		tstBroker.Finish()
		close(cancelCh)
	}()
	sub.EXPECT().Types().Times(1).Return(nil)
	sub.EXPECT().Ack().AnyTimes().Return(true)
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
}

func testSendBatchChannel(t *testing.T) {
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
	k1 := broker.Subscribe(sub) // required sub
	assert.NotZero(t, k1)
	broker.Send(broker.randomEvt())
	// calling unsubscribe acquires lock, so we can ensure the Send call has returned
	broker.Unsubscribe(k1)
	close(ch)
}

func testSubscriberSkip(t *testing.T) {
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	skipCh, closeCh := make(chan struct{}), make(chan struct{})
	skip := int64(0)
	events := []*evt{
		broker.randomEvt(),
		broker.randomEvt(),
	}
	wg := sync.WaitGroup{}
	wg.Add(len(events))
	sub.EXPECT().Closed().AnyTimes().Return(closeCh).Do(func() {
		wg.Done()
	})
	sub.EXPECT().Skip().AnyTimes().DoAndReturn(func() <-chan struct{} {
		// ensure at least all events + 1 skip are called
		if s := atomic.AddInt64(&skip, 1); s == 1 {
			// skip the first one
			ch := make(chan struct{})
			// return closed channel, so this subscriber is marked to skip events
			close(ch)
			return ch
		}
		return skipCh
	})
	// we expect this call once, and only for the SECOND call
	sub.EXPECT().Push(events[1]).Times(1)
	sub.EXPECT().Types().Times(2).Return(nil)
	sub.EXPECT().Ack().AnyTimes().Return(true)
	k1 := broker.Subscribe(sub) // required sub
	assert.NotZero(t, k1)
	for _, e := range events {
		broker.Send(e)
	}
	wg.Wait()
	// calling unsubscribe acquires lock, so we can ensure the Send call has returned
	broker.Unsubscribe(k1)
	close(skipCh)
	close(closeCh)
}

// test making sure that events are sent only to subs that are interested in it
func testEventTypeSubscription(t *testing.T) {
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
	k1 := broker.Subscribe(sub)     // required sub
	k2 := broker.Subscribe(diffSub) // required sub, but won't be used anyway
	k3 := broker.Subscribe(allSub)
	assert.NotZero(t, k1)
	assert.NotZero(t, k2)
	assert.NotZero(t, k3)
	assert.NotEqual(t, k1, k2)
	// send the TxErrEvent, a special case none of the subscribers ought to ever receive
	broker.Send(events.NewTxErrEvent(broker.ctx, errors.New("random err"), "party-1", types.Vote{
		PartyID:    "party-1",
		Value:      types.Vote_VALUE_YES,
		ProposalID: "prop-1",
	}))
	// send the correct event
	broker.Send(event)
	// ensure the event was delivered
	wg.Wait()
	// unsubscribe the subscriber, now we're done
	broker.Unsubscribe(k1)
	broker.Unsubscribe(k2)
	broker.Unsubscribe(k3)
	close(skipCh)
	close(closeCh)
}

func testTxErrNotAll(t *testing.T) {
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	allSub := mocks.NewMockSubscriber(broker.ctrl)
	skipCh, closeCh := make(chan struct{}), make(chan struct{})

	// we'll send error events. Once both have been sent to the subscriber
	// we're certain none of them were pushed to the ALL subscriber
	wg := sync.WaitGroup{}
	wg.Add(2)
	// Closed check
	sub.EXPECT().Closed().AnyTimes().Return(closeCh)
	allSub.EXPECT().Closed().AnyTimes().Return(closeCh)
	// skip check
	sub.EXPECT().Skip().AnyTimes().Return(skipCh)
	allSub.EXPECT().Skip().AnyTimes().Return(skipCh)

	// all subscriber ought to never receive anything, so don't specify an EXPECT
	// allSub.EXPECT().Push(gomock.Any()).Times(0)
	// actually push the event - diffSub expects nothing
	sub.EXPECT().Push(gomock.Any()).Times(2).Do(func(_ interface{}) {
		wg.Done()
	})
	// the event types this subscriber is interested in
	sub.EXPECT().Types().AnyTimes().Return([]events.Type{events.TxErrEvent})
	allSub.EXPECT().Types().AnyTimes().Return(nil) // subscribed to ALL events

	// both subscribers are ack'ing
	sub.EXPECT().Ack().AnyTimes().Return(true)
	allSub.EXPECT().Ack().AnyTimes().Return(true)

	// TxErrEvent
	evt := events.NewTxErrEvent(broker.ctx, errors.New("some error"), "party-1", types.Vote{
		PartyID:    "party-1",
		Value:      types.Vote_VALUE_YES,
		ProposalID: "prop-1",
	})
	k1 := broker.Subscribe(sub)
	k2 := broker.Subscribe(allSub)
	assert.NotZero(t, k1)
	assert.NotZero(t, k2)
	assert.NotEqual(t, k1, k2)
	// send the correct event
	broker.Send(evt)
	// send a second event
	broker.Send(events.NewTxErrEvent(broker.ctx, errors.New("some error 2"), "party-2", types.Vote{
		PartyID:    "party-2",
		Value:      types.Vote_VALUE_NO,
		ProposalID: "prop-2",
	}))
	// ensure the event was delivered
	wg.Wait()
	// unsubscribe the subscriber, now we're done
	broker.Unsubscribe(k1)
	broker.Unsubscribe(k2)
	close(skipCh)
	close(closeCh)
}

func (e evt) Type() events.Type {
	return e.t
}

func (e evt) Context() context.Context {
	return e.ctx
}

func (e *evt) SetSequenceID(s uint64) {
	e.sid = s
}

func (e evt) Sequence() uint64 {
	return e.sid
}

func (e evt) TraceID() string {
	return e.id
}
