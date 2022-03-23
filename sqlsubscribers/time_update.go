package sqlsubscribers

import (
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
)

type TimeUpdateEvent interface {
	events.Event
	Time() time.Time
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_update_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers BlockStore
type BlockStore interface {
	Add(entities.Block) error
}

type Time struct {
	store BlockStore
	log   *logging.Logger
}

func NewTimeSub(
	store BlockStore,
	log *logging.Logger,
) *Time {
	t := &Time{
		store: store,
		log:   log,
	}
	return t
}

func (t *Time) Type() events.Type {
	return events.TimeUpdate
}

func (t *Time) Push(evt events.Event) {
	switch et := evt.(type) {
	case TimeUpdateEvent:
		t.consume(et)
	default:
		t.log.Panic("Unknown event type in time subscriber",
			logging.String("Type", et.Type().String()))
	}
}

func (t *Time) consume(te TimeUpdateEvent) {
	hash, err := hex.DecodeString(te.TraceID())
	if err != nil {
		t.log.Panic("Trace ID is not valid hex string",
			logging.String("traceId", te.TraceID()))
	}

	// Postgres only stores timestamps in microsecond resolution
	block := entities.Block{
		VegaTime: te.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   te.BlockNr(),
	}

	err = t.store.Add(block)
	if err != nil {
		t.log.Error("Error adding block",
			logging.Error(err))
	}
}
