package broker

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
)

// Subscriber interface allows pushing values to subscribers, can be set to
// a Skip state (temporarily not receiving any events), or closed. Otherwise events are pushed
//go:generate go run github.com/golang/mock/mockgen -destination mocks/subscriber_mock.go -package mocks code.vegaprotocol.io/vega/broker Subscriber
type Subscriber interface {
	Push(val ...events.Event)
	Skip() <-chan struct{}
	Closed() <-chan struct{}
	C() chan<- events.Event
	Types() []events.Type
	SetID(id int)
	ID() int
	Ack() bool
}

type subscription struct {
	Subscriber
	required bool
}

// Broker - the base broker type
// perhaps we can extend this to embed into type-specific brokers
type Broker struct {
	ctx   context.Context
	mu    sync.Mutex
	tSubs map[events.Type]map[int]*subscription
	// these fields ensure a unique ID for all subscribers, regardless of what event types they subscribe to
	// once the broker context is cancelled, this map will be used to notify all subscribers, who can then
	// close their internal channels. We can then cleanly shut down (not having unclosed channels)
	subs map[int]subscription
	keys []int
}

// New creates a new base broker
func New(ctx context.Context) *Broker {
	return &Broker{
		ctx:   ctx,
		tSubs: map[events.Type]map[int]*subscription{},
		subs:  map[int]subscription{},
		keys:  []int{},
	}
}

func (b *Broker) sendChannel(sub Subscriber, evts []events.Event) (unsub bool) {
	for _, e := range evts {
		select {
		case <-b.ctx.Done():
			return
		case <-sub.Skip():
			return
		case <-sub.Closed():
			unsub = true
			return
		case sub.C() <- e:
			continue
		default:
			continue
		}
	}
	return
}

// SendBatch sends a slice of events to subscribers that can handle the events in the slice
// the events don't have to be of the same type, and most subscribers will ignore unknown events
// but this will slow down those subscribers, so avoid doing silly things
func (b *Broker) SendBatch(events []events.Event) {
	if len(events) == 0 {
		return
	}
	// We could ensure that the events are all of the same type but when
	// pushing logging events in batch, this wouldn't work without additional
	// logic, so we're assuming the caller knows what they're doing
	// t := events[0].Type()
	// for i := 1; i < len(events); i++ {
	// 	if events[i].Type() != t {
	// 		events = append(events[:i], events[i+1:]...)
	// 		i--
	// 	}
	// }
	b.mu.Lock()
	subs := b.getSubsByType(events[0].Type())
	b.mu.Unlock()
	if len(subs) == 0 {
		return
	}
	// push the event out in a routine
	// unlock the mutex once done
	go func(subs map[int]*subscription) {
		unsub := make([]int, 0, len(subs))
		for k, sub := range subs {
			select {
			case <-b.ctx.Done():
				return
			case <-sub.Skip():
				continue
			case <-sub.Closed():
				unsub = append(unsub, k)
			default:
				if sub.required {
					sub.Push(events...)
				} else if rm := b.sendChannel(sub, events); rm {
					unsub = append(unsub, k)
				}
			}
		}
	}(subs)
}

// Send sends an event to all subscribers
func (b *Broker) Send(event events.Event) {
	b.mu.Lock()
	subs := b.getSubsByType(event.Type())
	b.mu.Unlock()
	if len(subs) == 0 {
		return
	}
	// push the event out in a routine
	// unlock the mutex once done
	go func(subs map[int]*subscription) {
		unsub := make([]int, 0, len(subs))
		defer func() {
			if len(unsub) > 0 {
				b.mu.Lock()
				b.rmSubs(unsub...)
				b.mu.Unlock()
			}
		}()
		for k, sub := range subs {
			select {
			case <-b.ctx.Done():
				// broker context cancelled, we're done
				return
			case <-sub.Skip():
				continue
			case <-sub.Closed():
				unsub = append(unsub, k)
			default:
				if sub.required {
					sub.Push(event)
				} else if rm := b.sendChannel(sub, []events.Event{event}); rm {
					unsub = append(unsub, k)
				}
			}
		}
	}(subs)
}

func (b *Broker) getSubsByType(t events.Type) map[int]*subscription {
	ret := map[int]*subscription{}
	keys := []events.Type{
		t,
		events.All,
	}
	for _, key := range keys {
		if subs, ok := b.tSubs[key]; ok {
			for k, s := range subs {
				ret[k] = s
			}
		}
	}
	return ret
}

// Subscribe registers a new subscriber, returning the key
func (b *Broker) Subscribe(s Subscriber) int {
	b.mu.Lock()
	k := b.subscribe(s)
	b.mu.Unlock()
	return k
}

func (b *Broker) SubscribeBatch(subs ...Subscriber) {
	b.mu.Lock()
	for _, s := range subs {
		k := b.subscribe(s)
		s.SetID(k)
	}
	b.mu.Unlock()
}

func (b *Broker) subscribe(s Subscriber) int {
	k := b.getKey()
	sub := subscription{
		Subscriber: s,
		required:   s.Ack(),
	}
	b.subs[k] = sub
	types := sub.Types()
	if len(types) == 0 {
		types = []events.Type{events.All}
	}
	for _, t := range types {
		if _, ok := b.tSubs[t]; !ok {
			b.tSubs[t] = map[int]*subscription{}
		}
		b.tSubs[t][k] = &sub
	}
	return k
}

func (b *Broker) sub(s Subscriber, req bool) int {
	k := b.getKey()
	sub := subscription{
		Subscriber: s,
		required:   req,
	}
	b.subs[k] = sub
	types := s.Types()
	if len(types) == 0 {
		types = []events.Type{events.All}
	}
	for _, t := range types {
		if _, ok := b.tSubs[t]; !ok {
			b.tSubs[t] = map[int]*subscription{}
		}
		b.tSubs[t][k] = &sub
	}
	return k
}

// Unsubscribe removes subscriber from broker
// this does not change the state of the subscriber
func (b *Broker) Unsubscribe(k int) {
	b.mu.Lock()
	b.rmSubs(k)
	b.mu.Unlock()
}

func (b *Broker) getKey() int {
	if len(b.keys) > 0 {
		k := b.keys[0]
		b.keys = b.keys[1:] // pop first element
		return k
	}
	return len(b.subs) + 1 // add  1 to avoid zero value
}

func (b *Broker) rmSubs(keys ...int) {
	for _, k := range keys {
		// if the sub doesn't exist, this could be a duplicate call
		// we do not want the keys slice to contain duplicate values
		// and so we have to check this first
		s, ok := b.subs[k]
		if !ok {
			return
		}
		types := s.Types()
		if len(types) == 0 {
			types = []events.Type{events.All}
		}
		for _, t := range types {
			delete(b.tSubs[t], k) // remove key from typed subs map
		}
		delete(b.subs, k)
		b.keys = append(b.keys, k)
	}
}
