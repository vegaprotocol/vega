package sqlsubscribers

import (
	"context"
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
	Add(context.Context, entities.Block) error
}

type Time struct {
	store     BlockStore
	log       *logging.Logger
	lastBlock *entities.Block
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

func (t *Time) Push(ctx context.Context, evt events.Event) error {
	switch et := evt.(type) {
	case TimeUpdateEvent:
		return t.consume(ctx, et)
	default:
		return errors.Errorf("unknown event type %s", et.Type().String())
	}
}

func (t *Time) consume(ctx context.Context, te TimeUpdateEvent) error {
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

	// At startup we get time updates that have the same time to microsecond precision which causes
	// a primary key restraint failure, this code is to handle this scenario
	if t.lastBlock == nil || !block.VegaTime.Equal(t.lastBlock.VegaTime) {
		t.lastBlock = &block
		err = t.store.Add(ctx, block)
		if err != nil {
			return errors.Wrap(err, "failed to add block")
		}
	}

	return nil
}
