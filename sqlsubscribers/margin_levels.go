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
	Add(*entities.MarginLevels) error
	OnTimeUpdateEvent(context.Context) error
}

type MarginLevels struct {
	store    MarginLevelsStore
	log      *logging.Logger
	vegaTime time.Time
	seqNum   uint64
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
		err := ml.store.OnTimeUpdateEvent(e.Context())
		if err != nil {
			ml.log.Error("inserting margin level events to Postgres failed", logging.Error(err))
		}
	case MarginLevelsEvent:
		ml.seqNum = e.Sequence()
		ml.consume(e)
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

	record.SyntheticTime = ml.vegaTime.Add(time.Duration(ml.seqNum) * time.Microsecond)
	record.SeqNum = ml.seqNum

	return errors.Wrap(ml.store.Add(record), "inserting margin levels to SQL store failed")
}
