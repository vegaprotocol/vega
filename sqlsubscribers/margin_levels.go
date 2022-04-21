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
	store         MarginLevelsStore
	accountSource AccountSource
	log           *logging.Logger
	vegaTime      time.Time
}

func NewMarginLevels(store MarginLevelsStore, accountSource AccountSource, log *logging.Logger) *MarginLevels {
	return &MarginLevels{
		store:         store,
		accountSource: accountSource,
		log:           log,
	}
}

func (ml *MarginLevels) Types() []events.Type {
	return []events.Type{events.MarginLevelsEvent}
}

func (ml *MarginLevels) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		ml.vegaTime = e.Time()
		err := ml.store.Flush(ctx)
		if err != nil {
			ml.log.Error("inserting margin level events to Postgres failed", logging.Error(err))
		}
	case MarginLevelsEvent:
		ml.consume(ctx, e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (ml *MarginLevels) consume(ctx context.Context, event MarginLevelsEvent) error {
	marginLevels := event.MarginLevels()
	record, err := entities.MarginLevelsFromProto(ctx, &marginLevels, ml.accountSource, ml.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting margin levels proto to database entity failed")
	}

	return errors.Wrap(ml.store.Add(record), "inserting margin levels to SQL store failed")
}
