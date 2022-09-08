// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package utils

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/datanode/contextutil"
	"code.vegaprotocol.io/vega/logging"
)

type subscriber[T any] struct {
	ch chan T
}

// Notifier is a simple pub/pub construct. Create one, and call Notify(...) to push values into it.
// Subscribers register their interest by calling Subscribe() and are returned a channel which
// will receive all the values that are sent by Notify() calls.
//
// Importantly, the notification is non-blocking. You can specify the buffer size for
// the channels created inSubscribe() with 'chSize'.
// If the channel is full when a notification is received the channel will be closed and the
// subscription automatically unregistered.
type Notifier[T any] struct {
	subCount    int32
	lastSubID   uint64
	name        string
	log         *logging.Logger
	subscribers map[uint64]subscriber[T]
	mu          sync.Mutex
	chSize      int
}

func NewNotifier[T any](name string, log *logging.Logger, chSize int) Notifier[T] {
	return Notifier[T]{
		name:        name,
		log:         log,
		subscribers: map[uint64]subscriber[T]{},
		chSize:      chSize,
	}
}

func (o *Notifier[T]) Subscribe() (chan T, uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	ch := make(chan T, o.chSize)
	return ch, o.register(ch)
}

// Subscribe and immediately push 'msg' onto the channel as the first notification; handy for
// sending a 'initial value' message to subscribers. The push onto the channel is non-blocking
// and if 'inChSize' is not at least 1, the subscription will fail and an error will be returned.
func (o *Notifier[T]) SubscribeAndNotify(msg T) (chan T, uint64, error) {
	o.mu.Lock()
	defer o.mu.Unlock()

	ch := make(chan T, o.chSize)
	select {
	case ch <- msg:
	default:
		close(ch)
		return ch, 0, fmt.Errorf("unable to push initial notification")
	}

	return ch, o.register(ch), nil
}

func (o *Notifier[T]) register(ch chan T) uint64 {
	o.lastSubID++
	o.subscribers[o.lastSubID] = subscriber[T]{ch}
	atomic.AddInt32(&o.subCount, 1)
	return o.lastSubID
}

func (o *Notifier[T]) Unsubscribe(ref uint64) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if sub, exists := o.subscribers[ref]; exists {
		o.unsubscribe(ref, sub)
		return nil
	}

	return fmt.Errorf("no subscriber with id: %d", ref)
}

func (o *Notifier[T]) UnsubscribeAll() {
	o.mu.Lock()
	defer o.mu.Unlock()

	for ref, sub := range o.subscribers {
		o.unsubscribe(ref, sub)
	}
}

func (o *Notifier[T]) unsubscribe(ref uint64, sub subscriber[T]) {
	// mutex must already be obtained before calling this function
	close(sub.ch)
	delete(o.subscribers, ref)
	atomic.AddInt32(&o.subCount, -1)
}

func (o *Notifier[T]) GetSubscribersCount() int32 {
	// Use atomic for GetSubscribersCount so we don't have to obtain the mutex to report it.
	return atomic.LoadInt32(&o.subCount)
}

func (o *Notifier[T]) Notify(value T) {
	o.mu.Lock()
	defer o.mu.Unlock()

	if len(o.subscribers) == 0 {
		return
	}

	for id, sub := range o.subscribers {
		select {
		case sub.ch <- value:
		default:
			o.log.Warn(
				fmt.Sprintf("%s notifier: %s", o.name, "channel could not be updated, closing"))
			o.unsubscribe(id, sub)
		}
	}
}

// Observer build on Notifier - adding a method called Observe() which starts a goroutine that
// - subscribes to the notifier
// - optionally filters values, outputs them on a separate channel.
// - retries a specified number of tries to output on the output channel before giving up.
//
// An Observer[T] uses a Notifier[]T to allow updates to be pushed through in batches.
type Observer[T any] struct {
	notifier  Notifier[[]T]
	name      string
	log       *logging.Logger
	outChSize int
}

func NewObserver[T any](name string, log *logging.Logger, inChSize, outChSize int) Observer[T] {
	return Observer[T]{
		notifier:  NewNotifier[[]T](name, log, inChSize),
		name:      name,
		log:       log,
		outChSize: outChSize,
	}
}

func (o *Observer[T]) Notify(values []T) {
	o.notifier.Notify(values)
}

func (o *Observer[T]) GetSubscribersCount() int32 {
	return o.notifier.GetSubscribersCount()
}

func (o *Observer[T]) Observe(ctx context.Context, retries int, filter func(T) bool) (<-chan []T, uint64) {
	out := make(chan []T, o.outChSize)
	in, ref := o.notifier.Subscribe()
	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)

	go func() {
		defer close(out)

		for {
			select {
			case <-ctx.Done():
				o.logDebug(ip, ref, "closed connection")
				if err := o.notifier.Unsubscribe(ref); err != nil {
					o.logError(ip, ref, "failure un-subscribing when context.Done()")
				}
				return

			case values, ok := <-in:
				if !ok {
					// if 'in' channel is closed, it's because Notify() couldn't write to it.
					// In that case we have already unsubscribed, so don't try again.
					return
				}
				filtered := make([]T, 0, len(values))
				for _, value := range values {
					if filter(value) {
						filtered = append(filtered, value)
					}
				}
				if len(filtered) == 0 {
					continue
				}
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case out <- filtered:
						retryCount = retries
						success = true
					default:
						retryCount--
						if retryCount > 0 {
							o.logDebug(ip, ref, "not sent, retrying")
						}
						time.Sleep(time.Duration(10) * time.Millisecond)
					}
				}
				if !success && retryCount <= 0 {
					o.logWarning(ip, ref, "hit the retry limit")
					if err := o.notifier.Unsubscribe(ref); err != nil {
						o.logError(ip, ref, "failure un-subscribing after send retry limit")
					}
					return
				}
			}
		}
	}()

	return out, ref
}

func (o *Observer[T]) logDebug(ip string, ref uint64, msg string) {
	o.log.Debug(
		fmt.Sprintf("%s subscriber: %s", o.name, msg),
		logging.Uint64("id", ref),
		logging.String("ip-address", ip),
	)
}

func (o *Observer[T]) logWarning(ip string, ref uint64, msg string) {
	o.log.Warn(
		fmt.Sprintf("%s subscriber: %s", o.name, msg),
		logging.Uint64("id", ref),
		logging.String("ip-address", ip),
	)
}

func (o *Observer[T]) logError(ip string, ref uint64, msg string) {
	o.log.Error(
		fmt.Sprintf("%s subscriber: %s", o.name, msg),
		logging.Uint64("id", ref),
		logging.String("ip-address", ip),
	)
}
