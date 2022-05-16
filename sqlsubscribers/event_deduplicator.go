package sqlsubscribers

import (
	"context"
	"time"

	"github.com/pkg/errors"
	"google.golang.org/protobuf/proto"
)

// Used to get the set of events that have changed between flushes.  If an event is the same when flushed as it was
// on the previous Flush it will not be returned by the Flush method
type eventDeduplicator[K comparable, V proto.Message] struct {
	lastFlushedEvents map[K]V
	newEvents         map[K]V
	getId             func(context.Context, V, time.Time) (K, error)
}

func NewEventDeduplicator[K comparable, V proto.Message](getId func(context.Context, V, time.Time) (K, error)) *eventDeduplicator[K, V] {
	return &eventDeduplicator[K, V]{
		lastFlushedEvents: map[K]V{},
		newEvents:         map[K]V{},
		getId:             getId,
	}
}

func (e *eventDeduplicator[K, V]) AddEvent(ctx context.Context, event V, vegaTime time.Time) error {

	id, err := e.getId(ctx, event, vegaTime)
	if err != nil {
		return errors.Wrap(err, "failed to add event to deduplicator")
	}

	e.newEvents[id] = event

	return nil
}

func (e *eventDeduplicator[K, V]) Flush() map[K]V {

	updatedEvents := map[K]V{}

	for id, added := range e.newEvents {
		updatedOrNew := false
		if lastFlushed, exists := e.lastFlushedEvents[id]; exists {
			if !proto.Equal(added, lastFlushed) {
				updatedOrNew = true
			}
		} else {
			updatedOrNew = true
		}

		if updatedOrNew {
			e.lastFlushedEvents[id] = added
			updatedEvents[id] = added
		}

	}

	e.newEvents = map[K]V{}
	return updatedEvents
}
