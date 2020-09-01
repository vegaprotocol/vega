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

func getTestStreamSub(types []events.Type, filters ...subscribers.EventFilter) *tstStreamSub {
	ctx, cfunc := context.WithCancel(context.Background())
	return &tstStreamSub{
		StreamSub: subscribers.NewStreamSub(ctx, types, filters...),
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

func testUnfilteredNoEvents(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent})
	wg := sync.WaitGroup{}
	wg.Add(1)
	var data []events.Event
	go func() {
		data = sub.GetData()
		wg.Done()
	}()
	sub.cfunc() // cancel ctx
	wg.Wait()
	// we expect to see no events
	assert.Equal(t, 0, len(data))
}

func testUnfilteredWithEventsPush(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent})
	defer sub.cfunc()
	set := []events.Event{
		events.NewAccountEvent(sub.ctx, types.Account{
			Id: "acc-1",
		}),
		events.NewAccountEvent(sub.ctx, types.Account{
			Id: "acc-2",
		}),
	}
	sub.Push(set...)
	data := sub.GetData()
	// we expect to see no events
	assert.Equal(t, len(set), len(data))
	last := events.NewAccountEvent(sub.ctx, types.Account{
		Id: "acc-3",
	})
	sub.Push(last)
	data = sub.GetData()
	assert.Equal(t, 1, len(data))
	assert.Equal(t, events.AccountEvent, data[0].Type())
	ae, ok := data[0].(accEvt)
	assert.True(t, ok)
	assert.Equal(t, last.Account().Id, ae.Account().Id)
}

func testFilteredNoValidEvents(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent}, accMarketIDFilter("valid"))
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
	var data []events.Event
	go func() {
		data = sub.GetData()
		wg.Done()
	}()
	sub.cfunc()
	wg.Wait()
	// we expect to see no events
	assert.Equal(t, 0, len(data))
}

func testFilteredSomeValidEvents(t *testing.T) {
	sub := getTestStreamSub([]events.Type{events.AccountEvent}, accMarketIDFilter("valid"))
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
	sub.Push(set...)
	data := sub.GetData()
	// we expect to see no events
	assert.Equal(t, 1, len(data))
}
