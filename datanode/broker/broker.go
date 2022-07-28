// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
)

// Subscriber interface allows pushing values to subscribers, can be set to
// a Skip state (temporarily not receiving any events), or closed. Otherwise events are pushed
//go:generate go run github.com/golang/mock/mockgen -destination mocks/subscriber_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/broker Subscriber
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

// BrokerI interface (horribly named) is declared here to provide a drop-in replacement for broker mocks used throughout
// in addition to providing the classical mockgen functionality, this mock can be used to check the actual events that will be generated
// so we don't have to rely on test-only helper functions
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/broker BrokerI
type BrokerI interface {
	Send(event events.Event)
	Subscribe(s Subscriber) int
	SubscribeBatch(subs ...Subscriber)
	Unsubscribe(k int)
	Receive(ctx context.Context) error
}

type eventSource interface {
	Listen() error
	Receive(ctx context.Context) (<-chan events.Event, <-chan error)
}

type subscription struct {
	Subscriber
	required bool
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/chaininfo_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/broker ChainInfoI
type ChainInfoI interface {
	SetChainID(string) error
	GetChainID() (string, error)
}

type OrderEventWithVegaTime struct {
	events.Order
	vegaTime time.Time
}

func (oe *OrderEventWithVegaTime) VegaTime() time.Time {
	return oe.vegaTime
}

func (oe *OrderEventWithVegaTime) GetOrder() *vega.Order {
	return oe.Order.Order()
}

// Broker - the base broker type
// perhaps we can extend this to embed into type-specific brokers
type Broker struct {
	ctx   context.Context
	mu    sync.RWMutex
	tSubs map[events.Type]map[int]*subscription
	// these fields ensure a unique ID for all subscribers, regardless of what event types they subscribe to
	// once the broker context is cancelled, this map will be used to notify all subscribers, who can then
	// close their internal channels. We can then cleanly shut down (not having unclosed channels)
	subs   map[int]subscription
	keys   []int
	eChans map[events.Type]chan []events.Event
	smVer  int // version of subscriber map

	eventSource eventSource
	quit        chan struct{}
	chainInfo   ChainInfoI
	vegaTime    time.Time
}

// New creates a new base broker
func New(ctx context.Context, log *logging.Logger, config Config, chainInfo ChainInfoI,
	eventsource eventSource) (*Broker, error) {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	b := &Broker{
		ctx:         ctx,
		tSubs:       map[events.Type]map[int]*subscription{},
		subs:        map[int]subscription{},
		keys:        []int{},
		eChans:      map[events.Type]chan []events.Event{},
		eventSource: eventsource,
		quit:        make(chan struct{}),
		chainInfo:   chainInfo,
	}

	return b, nil
}

func (b *Broker) sendChannel(sub Subscriber, evts []events.Event) {
	// wait for a max of 1 second
	timeout := time.NewTimer(time.Second)
	defer func() {
		// drain the channel if we managed to leave the function before the timer expired
		if !timeout.Stop() {
			<-timeout.C
		}
	}()
	select {
	case <-b.ctx.Done():
		return
	case <-sub.Closed():
		return
	case sub.C() <- evts:
		return
	case <-timeout.C:
		return
	}
}

func (b *Broker) sendChannelSync(sub Subscriber, evts []events.Event) bool {
	select {
	case <-b.ctx.Done():
		return false
	case <-sub.Skip():
		return false
	case <-sub.Closed():
		return true
	case sub.C() <- evts:
		return false
	default:
		// @TODO perhaps log that we've encountered the channel buffer of a subscriber
		// this could help us find out what combination of event types + batch sizes are
		// problematic
		go b.sendChannel(sub, evts)
		return false
	}
}

func (b *Broker) startSending(t events.Type, evt events.Event) {
	var (
		subs map[int]*subscription
		ver  int
	)
	b.mu.Lock()
	ch, ok := b.eChans[t]
	if !ok {
		subs, ver = b.getSubsByType(t, 0)
		ln := len(subs) + 1                      // at least buffer 1
		ch = make(chan []events.Event, ln*20+20) // create a channel with buffer, min 40
		b.eChans[t] = ch                         // assign the newly created channel
	}
	b.mu.Unlock()

	if t == events.TimeUpdate {
		timeUpdate := evt.(entities.TimeUpdateEvent)
		b.vegaTime = timeUpdate.Time().Truncate(time.Microsecond)
	}

	if t == events.OrderEvent {
		orderEvent := evt.(*events.Order)
		evt = &OrderEventWithVegaTime{*orderEvent, b.vegaTime}
	}

	ch <- []events.Event{evt}
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
				// we're only reading here, so allow multiple routines to do traverse the map simultaneously
				b.mu.RLock()
				ns, nv := b.getSubsByType(t, ver)
				b.mu.RUnlock()
				// if nv == ver, the subs haven't changed
				if nv != ver {
					ver = nv
					subs = ns
				}
				unsub := make([]int, 0, len(subs))
				for k, sub := range subs {
					select {
					case <-sub.Skip():
						continue
					case <-sub.Closed():
						unsub = append(unsub, k)
					default:
						if sub.required {
							sub.Push(events...)
						} else if rm := b.sendChannelSync(sub, events); rm {
							unsub = append(unsub, k)
						}
					}
				}
				if len(unsub) != 0 {
					b.mu.Lock()
					b.rmSubs(unsub...)
					// we could update the sub map here, because we know subscribers have been removed
					// but that would hold a write lock for a longer time.
					b.mu.Unlock()
				}
			}
		}
	}(ch, t)
}

// Send sends an event to all subscribers
func (b *Broker) Send(event events.Event) {
	b.startSending(event.Type(), event)
}

// simplified version for better performance - unfortunately, we'll still need to copy the map
func (b *Broker) getSubsByType(t events.Type, sv int) (map[int]*subscription, int) {
	// we add the entire ALL map to type-specific maps, so if set, we can return this map directly
	if sv != 0 && sv == b.smVer {
		return nil, sv
	}
	subs, ok := b.tSubs[t]
	if !ok && t != events.TxErrEvent {
		// if a typed map isn't set (yet), and it's not the error event, we can return
		// ALL subscribers directly instead
		subs = b.tSubs[events.All]
	}
	// we still need to create a copy to keep the race detector happy
	cpy := make(map[int]*subscription, len(subs))
	for k, v := range subs {
		cpy[k] = v
	}
	return cpy, b.smVer
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
	// filter out weird types values like []events.Type{events.PartyEvent, events.All,}
	// those subscribers subscribe to all events no matter what, so treat them accordingly
	isAll := false
	if len(types) == 0 {
		isAll = true
		types = []events.Type{events.All}
	} else {
		for _, t := range types {
			if t == events.All {
				types = []events.Type{events.All}
				isAll = true
				break
			}
		}
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
			// Don't add ALL subs to the map they're already in, and don't add it to the
			// special TxErrEvent map, but we should add them to all other maps
			if t != events.All && t != events.TxErrEvent {
				b.tSubs[t][k] = &sub
			}
		}
	}
	b.smVer++
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
	if len(keys) == 0 {
		return
	}
	for _, k := range keys {
		// if the sub doesn't exist, this could be a duplicate call
		// we do not want the keys slice to contain duplicate values
		// and so we have to check this first
		s, ok := b.subs[k]
		if !ok {
			return
		}
		types := s.Types()
		for _, t := range types {
			if t == events.All {
				types = nil
				break
			}
		}
		if len(types) == 0 {
			// remove in all subscribers then
			for _, v := range b.tSubs {
				delete(v, k)
			}
		} else {
			for _, t := range types {
				delete(b.tSubs[t], k) // remove key from typed subs map
			}
		}
		delete(b.subs, k)
		b.keys = append(b.keys, k)
	}
	b.smVer++
}

func (b *Broker) Receive(ctx context.Context) error {
	if err := b.eventSource.Listen(); err != nil {
		return err
	}

	receiveCh, errCh := b.eventSource.Receive(ctx)

	for e := range receiveCh {
		if err := checkChainID(b.chainInfo, e.ChainID()); err != nil {
			return err
		}
		b.Send(e)
	}

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func checkChainID(expectedChainInfo ChainInfoI, chainID string) error {
	ourChainID, err := expectedChainInfo.GetChainID()
	if err != nil {
		return fmt.Errorf("Unable to get expected chain ID %w", err)
	}

	// An empty chain ID indicates this is our first run
	if ourChainID == "" {
		expectedChainInfo.SetChainID(chainID)
		return nil
	}

	if chainID != ourChainID {
		return fmt.Errorf("mismatched chain id received: %s, want %s", chainID, ourChainID)
	}

	return nil
}
