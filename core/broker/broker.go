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

package broker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/libs/broker"
	"code.vegaprotocol.io/vega/logging"
)

type Stats interface {
	IncrementEventCount(count uint64)
}

// Interface interface (horribly named) is declared here to provide a drop-in replacement for broker mocks used throughout
// in addition to providing the classical mockgen functionality, this mock can be used to check the actual events that will be generated
// so we don't have to rely on test-only helper functions
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/broker_mock.go -package mocks code.vegaprotocol.io/vega/core/broker Interface
type Interface interface {
	Stage(event events.Event)
	Send(event events.Event)
	SendBatch(events []events.Event)
	Subscribe(s broker.Subscriber) int
	SubscribeBatch(subs ...broker.Subscriber)
	Unsubscribe(k int)
	SetStreaming(on bool) bool
}

// SocketClient is an interface to send serialized events over a socket.
//
//go:generate go run github.com/golang/mock/mockgen -destination mocks/socket_client_mock.go -package mocks code.vegaprotocol.io/vega/core/broker SocketClient
type SocketClient interface {
	SendBatch(events []events.Event) error
	Receive(ctx context.Context) (<-chan events.Event, <-chan error)
}

type FileClientSend interface {
	SendBatch(events []events.Event) error
}

type subscription struct {
	broker.Subscriber
	required bool
}

// Broker - the base broker type
// perhaps we can extend this to embed into type-specific brokers.
type Broker struct {
	ctx context.Context

	mu    sync.Mutex
	tSubs map[events.Type]map[int]*subscription
	// these fields ensure a unique ID for all subscribers, regardless of what event types they subscribe to
	// once the broker context is cancelled, this map will be used to notify all subscribers, who can then
	// close their internal channels. We can then cleanly shut down (not having unclosed channels)
	subs   map[int]subscription
	keys   []int
	eChans map[events.Type]chan []events.Event

	seqGen *gen

	log *logging.Logger

	config       Config
	socketClient SocketClient
	fileClient   FileClientSend
	canStream    bool // whether not we should send events to the socketClient
	stats        Stats

	staged []events.Event
}

// New creates a new base broker.
func New(ctx context.Context, log *logging.Logger, config Config, stats Stats) (*Broker, error) {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	b := &Broker{
		ctx:       ctx,
		log:       log,
		tSubs:     map[events.Type]map[int]*subscription{},
		subs:      map[int]subscription{},
		keys:      []int{},
		eChans:    map[events.Type]chan []events.Event{},
		seqGen:    newGen(),
		config:    config,
		canStream: bool(config.Socket.Enabled),
		stats:     stats,
		staged:    []events.Event{},
	}

	if config.Socket.Enabled {
		sc, err := newSocketClient(ctx, log, &config.Socket)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize socket client: %w", err)
		}

		b.socketClient = sc
	}

	if config.File.Enabled {
		fc, err := NewFileClient(log, &config.File)
		if err != nil {
			return nil, fmt.Errorf("failed to initialize file client: %w", err)
		}
		b.fileClient = fc
	}

	return b, nil
}

func (b *Broker) OnTick(ctx context.Context, _ time.Time) {
	if len(b.staged) == 0 {
		return
	}
	for _, evt := range b.staged {
		evt.Replace(ctx)
		b.Send(evt)
	}
	b.staged = []events.Event{}
}

func (b *Broker) sendChannel(sub broker.Subscriber, evts []events.Event) {
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

func (b *Broker) sendChannelSync(sub broker.Subscriber, evts []events.Event) bool {
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

func (b *Broker) startSending(t events.Type, evts []events.Event) {
	if b.StreamingEnabled() {
		if err := b.socketClient.SendBatch(evts); err != nil {
			b.log.Fatal("Failed to send to socket client", logging.Error(err))
		}
	}

	if b.fileStreamEnabled() {
		if err := b.fileClient.SendBatch(evts); err != nil {
			b.log.Fatal("Failed to send to file client", logging.Error(err))
		}
	}

	b.mu.Lock()
	ch, ok := b.eChans[t]
	if !ok {
		subs := b.getSubsByType(t)
		ln := len(subs) + 1                      // at least buffer 1
		ch = make(chan []events.Event, ln*20+20) // create a channel with buffer, min 40
		b.eChans[t] = ch                         // assign the newly created channel
	}
	b.mu.Unlock()
	select {
	case <-b.ctx.Done():
		return
	default:
		ch <- evts
	}
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
			case evts := <-ch:
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
							sub.Push(evts...)
						} else if shouldUnsubscribe := b.sendChannelSync(sub, evts); shouldUnsubscribe {
							unsub = append(unsub, k)
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
// but this will slow down those subscribers, so avoid doing silly things.
func (b *Broker) SendBatch(events []events.Event) {
	if len(events) == 0 {
		return
	}
	b.stats.IncrementEventCount(uint64(len(events)))
	evts := b.seqGen.setSequence(events...)
	b.startSending(events[0].Type(), evts)
}

// Send sends an event to all subscribers.
func (b *Broker) Send(event events.Event) {
	b.stats.IncrementEventCount(1)
	b.startSending(event.Type(), b.seqGen.setSequence(event))
}

// Stages an event to be sent at the start of the next block.
func (b *Broker) Stage(event events.Event) {
	b.staged = append(b.staged, event)
}

// simplified version for better performance - unfortunately, we'll still need to copy the map.
func (b *Broker) getSubsByType(t events.Type) map[int]*subscription {
	// we add the entire ALL map to type-specific maps, so if set, we can return this map directly

	subs, ok := b.tSubs[t]
	if !ok {
		// if a typed map isn't set (yet), and it's not the error event, we can return
		// ALL subscribers directly instead
		subs = b.tSubs[events.All]
	}

	// we still need to create a copy to keep the race detector happy
	cpy := make(map[int]*subscription, len(subs))
	for k, v := range subs {
		cpy[k] = v
	}
	return cpy
}

// Subscribe registers a new subscriber, returning the key.
func (b *Broker) Subscribe(s broker.Subscriber) int {
	b.mu.Lock()
	k := b.subscribe(s)
	s.SetID(k)
	b.mu.Unlock()
	return k
}

func (b *Broker) SubscribeBatch(subs ...broker.Subscriber) {
	b.mu.Lock()
	for _, s := range subs {
		k := b.subscribe(s)
		s.SetID(k)
	}
	b.mu.Unlock()
}

func (b *Broker) subscribe(s broker.Subscriber) int {
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
			if t != events.All {
				b.tSubs[t][k] = &sub
			}
		}
	}
	return k
}

// Unsubscribe removes subscriber from broker
// this does not change the state of the subscriber.
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
}

// SetStreaming allows the ability to toggle on and off sending events to the socketClient.
func (b *Broker) SetStreaming(on bool) bool {
	old := b.canStream
	b.canStream = on
	return old
}

func (b *Broker) StreamingEnabled() bool {
	return b.canStream && b.socketClient != nil
}

func (b *Broker) SocketClient() SocketClient {
	return b.socketClient
}

func (b *Broker) fileStreamEnabled() bool {
	return b.fileClient != nil
}
