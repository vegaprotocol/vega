package sqlsubscribers

import (
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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

func (ml *MarginLevels) Type() events.Type {
	return events.MarginLevelsEvent
}

func (ml *MarginLevels) Push(evt events.Event) {
	switch e := evt.(type) {
	case TimeUpdateEvent:
		ml.vegaTime = e.Time()
	case MarginLevelsEvent:
		ml.consume(e)
	}
}

func (ml *MarginLevels) consume(event MarginLevelsEvent) {
	marginLevels := event.MarginLevels()
	record, err := entities.MarginLevelsFromProto(&marginLevels, ml.vegaTime)
	if err != nil {
		ml.log.Error("converting margin levels proto to database entity failed", logging.Error(err))
	}

	if err = ml.store.Upsert(record); err != nil {
		ml.log.Error("Inserting margin levels to SQL store failed", logging.Error(err))
	}
}
