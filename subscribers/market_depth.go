package subscribers

import (
	"context"
	"sync"

	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

type MarketDepthBuilder struct {
	*Base
	mu    sync.Mutex
	cfg   Config
	log   *logging.Logger
	store OrderStore
	buf   []types.Order
}

func NewMarketDepthBuilder(ctx context.Context, cfg Config, log *logging.Logger, ack bool) *MarketDepthBuilder {
	log = log.Named(namedLogger)
	log.SetLevel(cfg.OrderEventLogLevel.Level)

	o := MarketDepthBuilder{
		Base: NewBase(ctx, 10, ack),
		log:  log,
		buf:  []types.Order{},
		cfg:  cfg,
	}
	if o.isRunning() {
		go o.loop(o.ctx)
	}
	return &o
}

func (o *MarketDepthBuilder) loop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			o.Halt()
			return
		case e := <-o.ch:
			if o.isRunning() {
				o.Update(e)
			}
		}
	}
}

func (o *MarketDepthBuilder) Update(evts ...events.Event) {
	for _, e := range evts {
		switch te := e.(type) {
		case OE:
			o.updateMarketDepth(te.Order())
		}
	}
}

func (o *MarketDepthBuilder) Types() []events.Type {
	return []events.Type{
		events.OrderEvent,
	}
}

func (mdb *MarketDepthBuilder) updateMarketDepth(order *types.Order) {

}
