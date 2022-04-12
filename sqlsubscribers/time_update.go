package sqlsubscribers

import (
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
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

func (t *Time) Types() []events.Type {
	return []events.Type{events.TimeUpdate}
}

func (t *Time) Push(evt events.Event) error {
	switch et := evt.(type) {
	case TimeUpdateEvent:
		return t.consume(et)
	default:
		return errors.Errorf("unknown event type %s", et.Type().String())
	}
}

func (t *Time) consume(te TimeUpdateEvent) error {
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

	//@Todo figure out why still getting dup key violation on this at startup:return errors.Wrap(t.store.Add(block), "error adding block")
	err = t.store.Add(block)
	if err != nil {
		t.log.Errorf("error adding block:%s", err)
	}

	return nil
}
