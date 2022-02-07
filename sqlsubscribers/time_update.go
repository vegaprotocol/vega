package sqlsubscribers

import (
	"context"
	"encoding/hex"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	"code.vegaprotocol.io/vega/events"
)

type TimeUpdateEvent interface {
	events.Event
	Time() time.Time
}

type BlockStore interface {
	Add(entities.Block) error
	WaitForBlockHeight(height int64) (entities.Block, error)
}

type Time struct {
	*subscribers.Base
	store BlockStore
	log   *logging.Logger
}

func NewTimeSub(
	ctx context.Context,
	store BlockStore,
	log *logging.Logger,
) *Time {
	t := &Time{
		Base:  subscribers.NewBase(ctx, 1, true),
		store: store,
		log:   log,
	}
	return t
}

func (t *Time) Types() []events.Type {
	return []events.Type{
		events.TimeUpdate,
	}
}

func (t *Time) Push(evts ...events.Event) {
	for _, e := range evts {
		switch et := e.(type) {

		case TimeUpdateEvent:
			t.consume(et)
		default:
			t.log.Panic("Unknown event type in time subscriber",
				logging.String("Type", et.Type().String()))
		}
	}
}

func (t *Time) consume(te TimeUpdateEvent) {
	t.log.Debug("TimeUpdate: ",
		logging.Int64("block", te.BlockNr()),
		logging.Time("time", te.Time()))

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
