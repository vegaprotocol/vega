package sqlsubscribers

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type MarginLevelsEvent interface {
	events.Event
	MarginLevels() vega.MarginLevels
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/margin_levels_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers MarginLevelsStore
type MarginLevelsStore interface {
	Add(entities.MarginLevels) error
	Flush(context.Context) error
}

type MarginLevels struct {
	store             MarginLevelsStore
	accountSource     AccountSource
	log               *logging.Logger
	vegaTime          time.Time
	eventDeduplicator *eventDeduplicator[int64, *vega.MarginLevels]
}

func NewMarginLevels(store MarginLevelsStore, accountSource AccountSource, log *logging.Logger) *MarginLevels {
	return &MarginLevels{
		store:         store,
		accountSource: accountSource,
		log:           log,
		eventDeduplicator: NewEventDeduplicator[int64, *vega.MarginLevels](func(ctx context.Context,
			ml *vega.MarginLevels, vegaTime time.Time) (int64, error) {
			a, err := entities.GetAccountFromMarginLevel(ctx, ml, accountSource, vegaTime)
			if err != nil {
				return 0, err
			}

			return a.ID, nil
		}),
	}
}

func (ml *MarginLevels) Types() []events.Type {
	return []events.Type{events.MarginLevelsEvent}
}

func (ml *MarginLevels) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		err := ml.flush(ctx)
		if err != nil {
			return errors.Wrap(err, "flushing margin levels")
		}
		ml.vegaTime = e.Time()
		return nil
	case MarginLevelsEvent:
		return ml.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}
}

func (ml *MarginLevels) flush(ctx context.Context) error {

	updates := ml.eventDeduplicator.Flush()
	for _, update := range updates {
		entity, err := entities.MarginLevelsFromProto(ctx, update, ml.accountSource, ml.vegaTime)
		if err != nil {
			return errors.Wrap(err, "converting margin level to database entity failed")
		}
		err = ml.store.Add(entity)
		if err != nil {
			return errors.Wrap(err, "add margin level to store")
		}

	}

	err := ml.store.Flush(ctx)

	return errors.Wrap(err, "flushing margin levels")
}

func (ml *MarginLevels) consume(ctx context.Context, event MarginLevelsEvent) error {
	marginLevel := event.MarginLevels()
	marginLevel.Timestamp = 0
	ml.eventDeduplicator.AddEvent(ctx, &marginLevel, ml.vegaTime)

	return nil
}
