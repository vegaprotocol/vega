package sqlsubscribers

import (
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
	Upsert(*entities.MarginLevels) error
}

type MarginLevels struct {
	store    MarginLevelsStore
	log      *logging.Logger
	vegaTime time.Time
}

func NewMarginLevels(store MarginLevelsStore, log *logging.Logger) *MarginLevels {
	return &MarginLevels{
		store: store,
		log:   log,
	}
}

func (ml *MarginLevels) Types() []events.Type {
	return []events.Type{events.MarginLevelsEvent}
}

func (ml *MarginLevels) Push(evt events.Event) error {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		ml.vegaTime = e.Time()
	case MarginLevelsEvent:
		return ml.consume(e)
	default:
		return errors.Errorf("unknown event type %s", e.Type().String())
	}

	return nil
}

func (ml *MarginLevels) consume(event MarginLevelsEvent) error {
	marginLevels := event.MarginLevels()
	record, err := entities.MarginLevelsFromProto(&marginLevels, ml.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting margin levels proto to database entity failed")
	}

	return errors.Wrap(ml.store.Upsert(record), "inserting margin levels to SQL store failed")
}
