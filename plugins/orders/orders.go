package orders

import (
	"context"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins"
	types "code.vegaprotocol.io/vega/proto"
)

const pluginName = "orders-core"

type Plugin struct {
	log      *logging.Logger
	ctx      context.Context
	orderSub buffer.OrderSub
	// partyid -> marketid -> orderid
	store map[string]map[string]map[string]types.Order
}

func (p *Plugin) New(log *logging.Logger, ctx context.Context, bufs *buffer.Buffers) plugins.Plugin {
	log.Info("initializing new plugin", logging.String("plugin-name", pluginName))
	return &Plugin{
		log:      log,
		ctx:      ctx,
		orderSub: bufs.Orders.Subscribe(100000),
		store:    map[string]map[string]map[string]types.Order{},
	}
}

func (p *Plugin) Start() error {
	p.log.Info("starting plugin", logging.String("plugin-name", pluginName))
	for {
		select {
		case ord := <-p.orderSub.Recv():
			// do stuff yo
			_ = ord
		case <-p.orderSub.Done():
			p.log.Error("order subscription cancelled",
				logging.Error(p.orderSub.Err()),
				logging.String("plugin", pluginName),
			)
			return p.orderSub.Err()
		case <-p.ctx.Done():
			p.log.Error("order subscription context cancelled",
				logging.Error(p.ctx.Err()),
				logging.String("plugin", pluginName),
			)
			return p.ctx.Err()
		}
	}
}

func init() {
	plugins.Register(pluginName, &Plugin{})
}
