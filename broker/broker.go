package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/events"
)

// Subscriber interface allows pushing values to subscribers, can be set to
// a Skip state (temporarily not receiving any events), or closed. Otherwise events are pushed
//go:generate go run github.com/golang/mock/mockgen -destination mocks/subscriber_mock.go -package mocks code.vegaprotocol.io/vega/broker Subscriber
type Subscriber interface {
	Push(val ...events.Event)
	Skip() <-chan struct{}
	Closed() <-chan struct{}
	C() chan<- []events.Event
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
	subs   map[int]subscription
	keys   []int
	eChans map[events.Type]chan []events.Event

	seqGen *gen
}

// New creates a new base broker
func New(ctx context.Context) *Broker {
	return &Broker{
		ctx:    ctx,
		tSubs:  map[events.Type]map[int]*subscription{},
		subs:   map[int]subscription{},
		keys:   []int{},
		eChans: map[events.Type]chan []events.Event{},
		seqGen: newGen(),
	}
}

func (b *Broker) sendChannel(sub Subscriber, evts []events.Event) {
	ctx, cfunc := context.WithTimeout(b.ctx, time.Second)
	defer cfunc()
	select {
	case <-ctx.Done():
		return
	case <-sub.Closed():
		return
	case sub.C() <- evts:
		return
	}
}

func (b *Broker) startSending(t events.Type, evts []events.Event) {
	b.mu.Lock()
	ch, ok := b.eChans[t]
	if !ok {
		subs := b.getSubsByType(t)
		ch = make(chan []events.Event, len(subs)*10+20) // create a channel with buffer
		b.eChans[t] = ch                                // assign the newly created channel
	}
	b.mu.Unlock()
	ch <- evts
	if ok {
		// we already started the routine to consume the channel
		// we can return here
		return
	}
	go func(ch chan []events.Event, t events.Type) {
		defer func() {
			b.mu.Lock()
			delete(b.eChans, t)
			close(ch)
			b.mu.Unlock()
		}()
		for {
			select {
			case <-b.ctx.Done():
				return
			case events := <-ch:
				b.mu.Lock()
				subs := b.getSubsByType(t)
				b.mu.Unlock()
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
						} else {
							go b.sendChannel(sub, events)
						}
					}
				}
				if len(unsub) != 0 {
					b.mu.Lock()
					b.rmSubs(unsub...)
					b.mu.Unlock()
				}
			}
		}
	}(ch, t)
}

// SendBatch sends a slice of events to subscribers that can handle the events in the slice
// the events don't have to be of the same type, and most subscribers will ignore unknown events
// but this will slow down those subscribers, so avoid doing silly things
func (b *Broker) SendBatch(events []events.Event) {
	if len(events) == 0 {
		return
	}
	evts := b.seqGen.setSequence(events...)
	b.startSending(events[0].Type(), evts)
}

// Send sends an event to all subscribers
func (b *Broker) Send(event events.Event) {
	b.startSending(event.Type(), b.seqGen.setSequence(event))
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
	isAll := false
	if len(types) == 0 {
		isAll = true
		types = []events.Type{events.All}
	}
	for _, t := range types {
		if _, ok := b.tSubs[t]; !ok {
			b.tSubs[t] = map[int]*subscription{}
			if !isAll {
				// not the ALL event, so can be added to the map, and as the "all" subscribers should be
				for ak, as := range b.tSubs[events.All] {
					b.tSubs[t][ak] = as
				}
			}
		}
		b.tSubs[t][k] = &sub
	}
	if isAll {
		for t := range b.tSubs {
			if t != events.All {
				b.tSubs[t][k] = &sub
			}
		}
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
		fmt.Printf("REMOVING TYPES: %v\n", types)
		if len(types) == 0 {
			types = []events.Type{events.All}
		}
		fmt.Printf("\ntsubs: %#v\n", b.tSubs)
		if len(types) == 0 || len(types) == 1 && types[0] == events.All {
			// remove in all subscribers then
			for _, v := range b.tSubs {
				delete(v, k)
			}
		} else {
			for _, t := range types {
				delete(b.tSubs[t], k) // remove key from typed subs map
			}
		}
		fmt.Printf("\ntsubs: %#v\n", b.tSubs)
		fmt.Printf("\nsubs: %#v\n", b.subs)
		delete(b.subs, k)
		fmt.Printf("\nsubs: %#v\n", b.subs)
		fmt.Printf("\nkeys: %#v\n", b.keys)
		b.keys = append(b.keys, k)
		fmt.Printf("\nkeys: %#v\n", b.keys)
	}
}
