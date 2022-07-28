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

package utils

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/data-node/datanode/contextutil"
	"code.vegaprotocol.io/data-node/logging"
)

type subscriber[T any] struct {
	ch chan []T
}

type Observer[T any] struct {
	subCount    int32
	lastSubId   uint64
	name        string
	log         *logging.Logger
	subscribers map[uint64]subscriber[T]
	mut         sync.RWMutex
	inChSize    int
	outChSize   int
}

func NewObserver[T any](name string, log *logging.Logger, inChSize, outChSize int) Observer[T] {
	return Observer[T]{
		name:        name,
		log:         log,
		subscribers: map[uint64]subscriber[T]{},
		inChSize:    inChSize,
		outChSize:   outChSize,
	}
}

func (o *Observer[T]) Subscribe(ctx context.Context, filter func(T) bool) (chan []T, uint64) {
	o.mut.Lock()
	defer o.mut.Unlock()

	ch := make(chan []T, o.inChSize)
	o.lastSubId++
	o.subscribers[o.lastSubId] = subscriber[T]{ch}

	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
	o.logDebug(ip, o.lastSubId, "new subscription")
	return ch, o.lastSubId
}

func (o *Observer[T]) Unsubscribe(ctx context.Context, ref uint64) error {
	o.mut.Lock()
	defer o.mut.Unlock()

	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)

	if len(o.subscribers) == 0 {
		o.logDebug(ip, ref, "un-subscribe called but, no subscribers connected")
		return nil
	}

	if sub, exists := o.subscribers[ref]; exists {
		close(sub.ch)
		delete(o.subscribers, ref)
		return nil
	}

	return fmt.Errorf("no subscriber with id: %d", ref)
}

func (o *Observer[T]) GetSubscribersCount() int32 {
	return atomic.LoadInt32(&o.subCount)
}

func (o *Observer[T]) Notify(values []T) {
	o.mut.Lock()
	defer o.mut.Unlock()

	if len(o.subscribers) == 0 {
		return
	}

	if len(values) == 0 {
		return
	}

	for id, sub := range o.subscribers {
		select {
		case sub.ch <- values:
			o.logDebug("", id, "channel updated successfully")
		default:
			o.logWarning("", id, "channel could not be updated, closing")
			delete(o.subscribers, id) // safe to delete from map while iterating
			close(sub.ch)
		}
	}
}

func (o *Observer[T]) Observe(ctx context.Context, retries int, filter func(T) bool) (<-chan []T, uint64) {
	out := make(chan []T, o.outChSize)
	in, ref := o.Subscribe(ctx, filter)
	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)

	go func() {
		atomic.AddInt32(&o.subCount, 1)
		defer atomic.AddInt32(&o.subCount, -1)

		ctx, cancel := context.WithCancel(ctx)
		defer cancel()

		for {
			select {
			case <-ctx.Done():
				o.logDebug(ip, ref, "closed connection")
				if err := o.Unsubscribe(ctx, ref); err != nil {
					o.logError(ip, ref, "failure un-subscribing when context.Done()")
				}
				close(out)
				return

			case values, ok := <-in:
				if !ok {
					// 'in' channel may get closed because Notify() couldn't write to it
					close(out)
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
						o.logDebug(ip, ref, "sent successfully")
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
					cancel()
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
	o.log.Debug(
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
