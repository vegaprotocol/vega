package utils

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/data-node/contextutil"
	"code.vegaprotocol.io/data-node/logging"
)

type observable interface{}

type subscriber[T observable] struct {
	ch     chan []T
	filter func(T) bool
}

type Observer[T observable] struct {
	subCount    int32
	lastSubId   uint64
	name        string
	log         *logging.Logger
	subscribers map[uint64]subscriber[T]
	mut         sync.RWMutex
	inChSize    int
	outChSize   int
}

func NewObserver[T observable](name string, log *logging.Logger, inChSize, outChSize int) Observer[T] {
	return Observer[T]{
		name:        name,
		log:         log,
		subscribers: map[uint64]subscriber[T]{},
		inChSize:    inChSize,
		outChSize:   outChSize,
	}

}

func (o *Observer[T]) Subscribe(ctx context.Context, ch chan []T, filter func(T) bool) uint64 {
	o.mut.Lock()
	defer o.mut.Unlock()

	o.lastSubId++
	o.subscribers[o.lastSubId] = subscriber[T]{ch, filter}

	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
	o.logDebug(ip, o.lastSubId, "new subscription")
	return o.lastSubId
}

func (o *Observer[T]) Unsubscribe(ctx context.Context, ref uint64) error {
	o.mut.Lock()
	defer o.mut.Unlock()

	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)

	if len(o.subscribers) == 0 {
		o.logDebug(ip, ref, "un-subscribe called but, no subscribers connected")
		return nil
	}

	if _, exists := o.subscribers[ref]; exists {
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

	var ok bool
	for id, sub := range o.subscribers {
		select {
		case sub.ch <- values:
			ok = true
		default:
			ok = false
		}
		if ok {
			o.logDebug("", id, "channel updated successfully")
		} else {
			o.logDebug("", id, "channel could not be updated")
		}
	}
}

func (o *Observer[T]) BlockingNotify(values []T) {
	o.mut.Lock()
	defer o.mut.Unlock()

	for _, sub := range o.subscribers {
		sub.ch <- values
	}
}

func (o *Observer[T]) Observe(ctx context.Context, retries int, filter func(T) bool) (<-chan []T, uint64) {
	in := make(chan []T, o.inChSize)
	out := make(chan []T, o.outChSize)
	ref := o.Subscribe(ctx, in, filter)
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
				close(in)
				close(out)
				return

			case values := <-in:
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
