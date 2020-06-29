package broker_test

import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/broker"
	"code.vegaprotocol.io/vega/broker/mocks"
	"code.vegaprotocol.io/vega/events"

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

func TestSubscribe(t *testing.T) {
	t.Run("Subscribe and unsubscribe required - success", testSubUnsubSuccess)
	t.Run("Subscribe reuses keys", testSubReuseKey)
	t.Run("Unsubscribe automatically if subscriber is closed", testAutoUnsubscribe)
}

func TestSendEvent(t *testing.T) {
	t.Run("Skip optional subscribers", testSkipOptional)
	t.Run("Stop sending if context is cancelled", testStopCtx)
	t.Run("Skip subscriber based on channel state", testSubscriberSkip)
	t.Run("Send only to typed subscriber", testEventTypeSubscription)
}

func testSubUnsubSuccess(t *testing.T) {
	broker := getBroker(t)
	defer broker.Finish()
	sub := mocks.NewMockSubscriber(broker.ctrl)
	reqSub := mocks.NewMockSubscriber(broker.ctrl)
	// subscribe + unsubscribe -> 2 calls
	sub.EXPECT().Types().Times(2).Return(nil)
	reqSub.EXPECT().Types().Times(2).Return(nil)
	k1 := broker.Subscribe(sub, false)   // not required
	k2 := broker.Subscribe(reqSub, true) // required
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
	k1 := broker.Subscribe(sub, false)
	assert.NotZero(t, k1)
	broker.Unsubscribe(k1)
	k2 := broker.Subscribe(sub, true)
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
	k1 := broker.Subscribe(sub, true)
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
	k2 := broker.Subscribe(sub, false)
	assert.Equal(t, k1, k2)
}

func testSkipOptional(t *testing.T) {
	tstBroker := getBroker(t)
	defer tstBroker.Finish()
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	skipCh, closedCh, cCh := make(chan struct{}), make(chan struct{}), make(chan events.Event, 1)
	twg := sync.WaitGroup{}
	twg.Add(2)
	sub.EXPECT().Types().Times(2).Return(nil).Do(func() {
		twg.Done()
	})
	k1 := tstBroker.Subscribe(sub, false)
	assert.NotZero(t, k1)

	events := []*evt{
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
		tstBroker.randomEvt(),
	}
	// ensure all 3 events are being sent (wait for routine to spawn)
	wg := sync.WaitGroup{}
	wg.Add(len(events))
	sub.EXPECT().Closed().Times(len(events)).Return(closedCh)
	sub.EXPECT().Skip().Times(len(events)).Return(skipCh)
	// we try to get the channel 3 times, only 1 of the attempts will actually publish the event
	sub.EXPECT().C().Times(len(events)).Return(cCh).Do(func() {
		// Done call each time we tried sending an event
		wg.Done()
	})

	// send events
	for _, e := range events {
		tstBroker.Send(e)
	}
	wg.Wait()
	// we've tried to send 3 events, subscriber could only accept one. Check state of all the things
	// we need to unsubscribe the subscriber, because we're closing the channels and race detector complains
	// because there's a loop calling functions that are returning the channels we're closing here
	tstBroker.Unsubscribe(k1)
	// ensure unsubscribe has returned
	twg.Wait()
	close(closedCh)
	close(skipCh)
	assert.Equal(t, events[0], <-cCh)
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
	k1 := broker.Subscribe(sub, true) // required sub
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
	skip := true
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
		if skip {
			// skip the first one
			skip = false
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
	k1 := broker.Subscribe(sub, true) // required sub
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
	different := events.Type(int(events.All) + int(events.TimeUpdate) + 1) // this value cannot exist as an events.Type value
	diffSub.EXPECT().Types().Times(2).Return([]events.Type{different})
	// subscribe the subscriberjk
	k1 := broker.Subscribe(sub, true)     // required sub
	k2 := broker.Subscribe(diffSub, true) // required sub, but won't be used anyway
	k3 := broker.Subscribe(allSub, true)
	assert.NotZero(t, k1)
	assert.NotZero(t, k2)
	assert.NotZero(t, k3)
	assert.NotEqual(t, k1, k2)
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

func (e evt) Type() events.Type {
	return e.t
}

func (e evt) Context() context.Context {
	return e.ctx
}

func (e evt) TraceID() string {
	return e.id
}
