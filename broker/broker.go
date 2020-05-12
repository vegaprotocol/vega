package broker

import (
	"context"
	"sync"
)

// Subscriber interface allows pushing values to subscribers, can be set to
// a Skip state (temporarily not receiving any events), or closed. Otherwise events are pushed
//go:generate go run github.com/golang/mock/mockgen -destination mocks/subscriber_mock.go -package mocks code.vegaprotocol.io/vega/broker Subscriber
type Subscriber interface {
	Push(val interface{})
	Skip() <-chan struct{}
	Closed() <-chan struct{}
	C() chan<- interface{}
}

type subscription struct {
	Subscriber
	required bool
}

// Broker - the base broker type
// perhaps we can extend this to embed into type-specific brokers
type Broker struct {
	ctx  context.Context
	mu   sync.Mutex
	subs map[int]subscription
	keys []int
}

// New creates a new base broker
func New(ctx context.Context) *Broker {
	return &Broker{
		ctx:  ctx,
		subs: map[int]subscription{},
		keys: []int{},
	}
}

// Send sends an event to all subscribers
func (b *Broker) Send(event interface{}) {
	b.mu.Lock()
	// push the event out in a routine
	// unlock the mutex once done
	go func() {
		unsub := make([]int, 0, len(b.subs))
		defer func() {
			b.rmSubs(unsub...)
			b.mu.Unlock()
		}()
		for k, sub := range b.subs {
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
				} else {
					select {
					case sub.C() <- event:
						continue
					default:
						// skip this event
						continue
					}
				}
			}
		}
	}()
}

// Subscribe registers a new subscriber, returning the key
func (b *Broker) Subscribe(s Subscriber, req bool) int {
	b.mu.Lock()
	k := b.getKey()
	b.subs[k] = subscription{
		Subscriber: s,
		required:   req,
	}
	b.mu.Unlock()
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
		if _, ok := b.subs[k]; !ok {
			return
		}
		delete(b.subs, k)
		b.keys = append(b.keys, k)
	}
}
