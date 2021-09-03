package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

type CPE interface {
	Proto() eventspb.CheckpointEvent
}

type CheckpointStore interface {
	Save(cp *eventspb.CheckpointEvent) error
}

type CheckpointSub struct {
	*Base
	store CheckpointStore
	log   *logging.Logger
	mu    *sync.Mutex
}

func NewCheckpointSub(ctx context.Context, log *logging.Logger, store CheckpointStore, ack bool) *CheckpointSub {
	c := &CheckpointSub{
		Base:  NewBase(ctx, 10, ack),
		store: store,
		log:   log,
		mu:    &sync.Mutex{},
	}
	if c.isRunning() {
		go c.loop(c.ctx)
	}
	return c
}

func (c *CheckpointSub) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			c.Halt()
			return
		case e := <-c.ch:
			if c.isRunning() {
				c.Push(e...)
			}
		}
	}
}

func (c *CheckpointSub) Push(evts ...events.Event) {
	if len(evts) == 0 {
		return
	}
	c.mu.Lock()
	for _, e := range evts {
		if et, ok := e.(CPE); ok {
			cp := et.Proto()
			if err := c.store.Save(&cp); err != nil {
				c.log.Error("Error storing checkpoint event",
					logging.String("checkpoint-hash", cp.Hash),
					logging.Error(err),
				)
			}
		}
	}
	c.mu.Unlock()
}

func (c *CheckpointSub) Types() []events.Type {
	return []events.Type{
		events.CheckpointEvent,
	}
}
