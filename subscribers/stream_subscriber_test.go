package subscribers_test

import (
	"context"
	"sync"
	"testing"

	"code.vegaprotocol.io/vega/events"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/subscribers"

	"github.com/stretchr/testify/assert"
)

type tstStreamSub struct {
	*subscribers.StreamSub
	ctx   context.Context
	cfunc context.CancelFunc
}

type accEvt interface {
	events.Event
	Account() types.Account
}

func getTestStreamSub(types []events.Type, bufSize int, filters ...subscribers.EventFilter) *tstStreamSub {
	ctx, cfunc := context.WithCancel(context.Background())
	return &tstStreamSub{
		StreamSub: subscribers.NewStreamSub(ctx, types, bufSize, filters...),
		ctx:       ctx,
		cfunc:     cfunc,
	}
}

func accMarketIDFilter(mID string) subscribers.EventFilter {
	return func(e events.Event) bool {
		ae, ok := e.(accEvt)
		if !ok {
			return false
		}
		if ae.Account().MarketID != mID {
			return false
		}
		return true
	}
}

func TestUnfilteredSubscription(t *testing.T) {
	t.Run("Stream subscriber without filters, no events", testUnfilteredNoEvents)
	t.Run("Stream subscriber without filters - with events", testUnfilteredWithEventsPush)
}

func TestFilteredSubscription(t *testing.T) {
	t.Run("Stream subscriber with filter - no valid events", testFilteredNoValidEvents)
	t.Run("Stream subscriber with filter - some valid events", testFilteredSomeValidEvents)
}

func TestSubscriberTypes(t *testing.T) {
	t.Run("Stream subscriber for all event types", testFilterAll)
}

func TestSubscriberBuffered(t *testing.T) {
	t.Run("Batched stream subscriber", testBatchedStreamSubscriber)
}

func TestMidChannelDone(t *testing.T) {
	t.Run("Stream subscriber stops mid event stream", testCloseChannelWrite)
}

func testUnfilteredNoEvents(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent}, 0)
	wg := sync.WaitGroup{}
	wg.Add(1)
	var data []*types.BusEvent
	go func() {
		data = sub.GetData(context.Background())
		wg.Done()
	}()
	sub.cfunc() // cancel ctx
	wg.Wait()
	// we expect to see no events
	assert.Equal(t, 0, len(data))
}

func testUnfilteredWithEventsPush(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent}, 0)
	defer sub.cfunc()
	set := []events.Event{
		events.NewAccountEvent(sub.ctx, types.Account{
			Id: "acc-1",
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id: "acc-2",
		}),
	}

	data := []*types.BusEvent{}
	done := make(chan struct{})
	getData := func() {
		done <- struct{}{}
		data = sub.GetData(context.Background())
		done <- struct{}{}
	}

	go getData()

	<-done
	sub.Push(set...)
	<-done
	// we expect to see no events
	assert.Equal(t, len(set), len(data))
	last := events.NewAccountEvent(sub.ctx, types.Account{
		Id: "acc-3",
	})

	go getData()

	<-done
	sub.Push(last)
	<-done
	assert.Equal(t, 1, len(data))
	rt, err := events.ProtoToInternal(data[0].Type)
	assert.NoError(t, err)
	assert.Equal(t, 1, len(rt))
	assert.Equal(t, events.AccountEvent, rt[0])
	acc := data[0].GetAccount()
	assert.NotNil(t, acc)
	assert.Equal(t, last.Account().Id, acc.Id)
}

func testFilteredNoValidEvents(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent}, 0, accMarketIDFilter("valid"))
	set := []events.Event{
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc-1",
			MarketID: "invalid",
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc-2",
			MarketID: "also-invalid",
		}),
	}
	sub.Push(set...)
	wg := sync.WaitGroup{}
	wg.Add(1)
	var data []*types.BusEvent
	go func() {
		data = sub.GetData(context.Background())
		wg.Done()
	}()
	sub.cfunc()
	wg.Wait()
	// we expect to see no events
	assert.Equal(t, 0, len(data))
}

func testFilteredSomeValidEvents(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent}, 0, accMarketIDFilter("valid"))
	defer sub.cfunc()
	set := []events.Event{
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc-1",
			MarketID: "invalid",
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc-2",
			MarketID: "valid",
		}),
	}

	data := []*types.BusEvent{}
	done := make(chan struct{})
	getData := func() {
		done <- struct{}{}
		data = sub.GetData(context.Background())
		done <- struct{}{}
	}
	go getData()

	<-done
	sub.Push(set...)
	<-done
	// we expect to see no events
	assert.Equal(t, 1, len(data))
}

func testFilterAll(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.All}, 0)
	assert.Nil(t, sub.Types())
}

func testBatchedStreamSubscriber(t *testing.T) {
	mID := "market-id"
	sub := getTestStreamSub([]events.Type{events.All}, 5)
	defer sub.cfunc()
	sent, rec := make(chan struct{}), make(chan struct{})
	set1 := []events.Event{
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc1",
			MarketID: mID,
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc2",
			MarketID: mID,
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc50",
			MarketID: "other-market",
		}),
	}
	sendRoutine := func(ch chan struct{}, sub *tstStreamSub, set []events.Event) {
		sub.C() <- set
		close(ch)
	}

	var data []*types.BusEvent
	go func() {
		rec <- struct{}{}
		data = sub.GetData(context.Background())
		close(rec)
	}()
	<-rec

	go sendRoutine(sent, sub, set1)
	// ensure all events were sent
	<-sent
	// now start receiving, this should not receive any events:
	// let's send a new batch, this ought to fill the buffer
	sent = make(chan struct{})
	go sendRoutine(sent, sub, set1)
	<-rec
	// buffer max reached, data sent
	assert.Equal(t, 5, len(data))
	// a total of 6 events were now sent to the subscriber, changing the buffer size ought to return 1 event
	<-sent
	data = sub.UpdateBatchSize(sub.ctx, len(set1)) // set batch size to match test-data set
	assert.Equal(t, 1, len(data))                  // we should have drained the buffer
	sent = make(chan struct{})
	go sendRoutine(sent, sub, set1)
	<-sent
	// we don't need the rec channel, the buffer is 3, and we sent 3 events
	data = sub.GetData(context.Background())
	assert.Equal(t, 3, len(data))
	// just in case -> this is with the rec channel, it ought to produce the exact same result
	sent = make(chan struct{})
	go sendRoutine(sent, sub, set1)
	<-sent
	rec = make(chan struct{})
	// buffer is 3, we sent 3 events, GetData ought to return
	go func() {
		data = sub.GetData(context.Background())
		close(rec)
	}()
	<-rec
	assert.Equal(t, 3, len(data))
}

// this test aims to replicate the crash when trying to write to a closed channel
func testCloseChannelWrite(t *testing.T) {
	mID := "tstMarket"
	sub := getTestStreamSub([]events.Type{events.AccountEvent}, 0, accMarketIDFilter(mID))
	set := []events.Event{
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc1",
			MarketID: mID,
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc2",
			MarketID: mID,
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc50",
			MarketID: "other-market",
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc3",
			MarketID: mID,
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc4",
			MarketID: mID,
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc51",
			MarketID: "other-market",
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc5",
			MarketID: "other-market",
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc6",
			MarketID: mID,
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id:       "acc7",
			MarketID: mID,
		}),
	}
	started := make(chan struct{})
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		first := false
		defer wg.Done()
		// keep iterating until the context was closed, ensuring
		// the context is cancelled mid-send
		for {
			select {
			case <-sub.Closed():
				return
			case <-sub.Skip():
				return
			case sub.C() <- set:
				// case ch <- e:
				if !first {
					first = true
					close(started)
				}
			}
		}
	}()
	<-started
	// wait for sub to be confirmed closed down
	data := sub.GetData(sub.ctx)
	sub.cfunc()
	wg.Wait()
	// we received at least the first event, which is valid (filtered)
	// so this slice ought not to be empty
	assert.NotEmpty(t, data)
}
