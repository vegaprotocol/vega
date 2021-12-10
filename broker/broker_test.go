//go:build !race
// +build !race

package broker_test

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/broker/mocks"
	"code.vegaprotocol.io/data-node/logging"
	mocksdn "code.vegaprotocol.io/data-node/mocks"
	types "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"go.nanomsg.org/mangos/v3/protocol"
	"go.nanomsg.org/mangos/v3/protocol/push"

	"github.com/golang/protobuf/proto"
	mangosErr "go.nanomsg.org/mangos/v3/errors"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"golang.org/x/sync/errgroup"
)

const testChainId = "test-chain"

type brokerTst struct {
	*broker.Broker
	cfunc    context.CancelFunc
	ctx      context.Context
	ctrl     *gomock.Controller
	sock     protocol.Socket
	dialAddr string
}

type evt struct {
	t   events.Type
	ctx context.Context
	sid uint64
	id  string
	cid string
}

func getBroker(t *testing.T) *brokerTst {
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)

	// Use in process transport for testing
	config := broker.NewDefaultConfig()
	config.SocketConfig.TransportType = "inproc"
	config.SocketConfig.IP = t.Name()
	config.SocketConfig.Port = 8085

	sock, err := push.NewSocket()
	assert.NoError(t, err)
	socketConfig := config.SocketConfig

	chainInfo := mocksdn.NewMockChainInfoI(ctrl)
	chainInfo.EXPECT().GetChainID().Return(testChainId, nil).AnyTimes()
	broker, _ := broker.New(ctx, logging.NewTestLogger(), config, chainInfo)
	return &brokerTst{
		Broker:   broker,
		cfunc:    cfunc,
		ctx:      ctx,
		ctrl:     ctrl,
		sock:     sock,
		dialAddr: fmt.Sprintf("%s://%s:%d", socketConfig.TransportType, socketConfig.IP, socketConfig.Port),
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
		cid: "testchain",
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
	t.Run("Send only to typed subscriber (also tests TxErrEvents are skipped)", testEventTypeSubscription)
}

func TestReceive(t *testing.T) {
	t.Run("Receives events and sends them to broker", testSendsReceivedEvents)
	t.Run("Returns an error on version mismatch", testErrorOnVersionMismatch)
}

func TestTxErrEvents(t *testing.T) {
	t.Run("Ensure TxErrEvents are hidden from ALL subscribers", testTxErrNotAll)
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

func testSendsReceivedEvents(t *testing.T) {
	tstBroker := getBroker(t)
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	skipCh, closedCh := make(chan struct{}), make(chan struct{})
	defer func() {
		tstBroker.Finish()
		close(closedCh)
		close(skipCh)
	}()

	sub.EXPECT().Types().AnyTimes().Return(nil)
	sub.EXPECT().Ack().AnyTimes().Return(true)

	k1 := tstBroker.Subscribe(sub)
	assert.NotZero(t, k1)

	busEvts := []eventspb.BusEvent{
		{
			Version: 1,
			Id:      "id-1",
			Block:   "1",
			ChainId: testChainId,
			Type:    eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE,
			Event: &eventspb.BusEvent_TimeUpdate{
				TimeUpdate: &eventspb.TimeUpdate{
					Timestamp: 1628173151,
				},
			},
		},
		{
			Version: 1,
			Id:      "id-2",
			Block:   "2",
			ChainId: testChainId,
			Type:    eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE,
			Event: &eventspb.BusEvent_TimeUpdate{
				TimeUpdate: &eventspb.TimeUpdate{
					Timestamp: 1628173152,
				},
			},
		},
		{
			Version: 1,
			Id:      "id-3",
			Block:   "3",
			ChainId: testChainId,
			Type:    eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE,
			Event: &eventspb.BusEvent_TimeUpdate{
				TimeUpdate: &eventspb.TimeUpdate{
					Timestamp: 1628173152,
				},
			},
		},
	}

	// ensure all 3 events are being sent
	wg := sync.WaitGroup{}
	wg.Add(3)
	sub.EXPECT().Closed().AnyTimes().Return(closedCh)
	sub.EXPECT().Skip().AnyTimes().Return(skipCh)
	sub.EXPECT().Push(gomock.Any()).Times(3).Do(func(events ...interface{}) {
		wg.Done()
	})

	ctx, cancel := context.WithCancel(context.Background())
	go tstBroker.Receive(ctx)

	var numOfRetries int
	for {
		err := tstBroker.sock.Dial(tstBroker.dialAddr)
		if err == nil {
			break
		}

		if err != mangosErr.ErrConnRefused {
			continue
		}

		if numOfRetries < 5 {
			numOfRetries++
			time.Sleep(time.Microsecond * 500)
			continue
		}

		t.Fatal(err)
	}

	for _, evnt := range busEvts {
		b, err := proto.Marshal(&evnt)
		assert.NoError(t, err)

		err = tstBroker.sock.Send(b)
		assert.NoError(t, err)
	}

	wg.Wait()
	cancel()
}

func testErrorOnVersionMismatch(t *testing.T) {
	tstBroker := getBroker(t)
	sub := mocks.NewMockSubscriber(tstBroker.ctrl)
	skipCh, closedCh := make(chan struct{}), make(chan struct{})
	defer func() {
		tstBroker.Finish()
		close(closedCh)
		close(skipCh)
	}()

	sub.EXPECT().Types().AnyTimes().Return(nil)
	sub.EXPECT().Ack().AnyTimes().Return(true)

	k1 := tstBroker.Subscribe(sub)
	assert.NotZero(t, k1)

	evnt := eventspb.BusEvent{
		Version: 2,
		Id:      "id-1",
		Block:   "1",
		Type:    eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE,
		Event: &eventspb.BusEvent_TimeUpdate{
			TimeUpdate: &eventspb.TimeUpdate{
				Timestamp: 1628173151,
			},
		},
	}

	sub.EXPECT().Closed().AnyTimes().Return(closedCh)
	sub.EXPECT().Skip().AnyTimes().Return(skipCh)

	eg, ctx := errgroup.WithContext(context.Background())
	ctx, cancel := context.WithCancel(ctx)
	eg.Go(func() error {
		return tstBroker.Receive(ctx)
	})

	var numOfRetries int
	for {
		err := tstBroker.sock.Dial(tstBroker.dialAddr)
		if err == nil {
			break
		}

		if err != mangosErr.ErrConnRefused {
			continue
		}

		if numOfRetries < 5 {
			numOfRetries++
			time.Sleep(time.Microsecond * 500)
			continue
		}

		t.Fatal(err)
	}

	b, err := proto.Marshal(&evnt)
	assert.NoError(t, err)

	err = tstBroker.sock.Send(b)
	assert.NoError(t, err)

	eg.Go(func() error {
		time.Sleep(time.Second * 2)
		return fmt.Errorf("test has timed out")
	})

	err = eg.Wait()
	assert.EqualError(t, err, "mismatched BusEvent version received: 2, want 1")

	cancel()
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
		PartyId:    "party-1",
		Value:      types.Vote_VALUE_YES,
		ProposalId: "prop-1",
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
		PartyId:    "party-1",
		Value:      types.Vote_VALUE_YES,
		ProposalId: "prop-1",
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
		PartyId:    "party-2",
		Value:      types.Vote_VALUE_NO,
		ProposalId: "prop-2",
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

func (e evt) StreamMessage() *eventspb.BusEvent {
	return nil
}

func (e evt) ChainID() string {
	return e.cid
}
