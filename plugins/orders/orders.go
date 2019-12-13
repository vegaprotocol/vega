package orders

import (
	"context"

	"code.vegaprotocol.io/vega/buffer"
	"code.vegaprotocol.io/vega/config"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins"
	"code.vegaprotocol.io/vega/plugins/orders/proto"

	"google.golang.org/grpc"
)

const pluginName = "orders-core"

type Plugin struct {
	log      *logging.Logger
	ctx      context.Context
	orderSub buffer.OrderSub
	store    *orderStore
	svc      *service
}

func (p *Plugin) New(log *logging.Logger, ctx context.Context, bufs *buffer.Buffers, srv *grpc.Server, rawcfg interface{}) (plugins.Plugin, error) {
	log = log.Named(pluginName)
	log.Info("initializing new plugin",
		logging.String("plugin-name", pluginName))

	// load configuration
	cfg := Config{}
	err := config.LoadPluginConfig(rawcfg, pluginName, &cfg)
	if err != nil {
		return nil, err
	}
	log.SetLevel(cfg.Level.Get())

	store := newStore()
	svc := newService(ctx, log, store)
	proto.RegisterOrdersCoreServer(srv, svc)
	return &Plugin{
		log:      log,
		ctx:      ctx,
		orderSub: bufs.Orders.Subscribe(100000),
		store:    store,
		svc:      svc,
	}, nil
}

func (p *Plugin) Start() error {
	p.log.Info("starting plugin", logging.String("plugin-name", pluginName))
	for {
		select {
		case orders := <-p.orderSub.Recv():
			// do stuff yo
			p.store.SaveBatch(orders)
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
	plugins.Register(pluginName, &Plugin{}, DefaultConfig())
}
