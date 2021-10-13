package api

import (
	"context"
	"net"
	"strconv"
	"time"

	protoapi "code.vegaprotocol.io/protos/vega/api/v1"
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/monitoring"
	"code.vegaprotocol.io/vega/stats"
	"code.vegaprotocol.io/vega/subscribers"
	"code.vegaprotocol.io/vega/vegatime"

	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/event_service_mock.go -package mocks code.vegaprotocol.io/vega/api EventService
type EventService interface {
	ObserveEvents(ctx context.Context, retries int, eTypes []events.Type, batchSize int, filters ...subscribers.EventFilter) (<-chan []*eventspb.BusEvent, chan<- int)
}

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/api TimeService
type TimeService interface {
	GetTimeNow() time.Time
}

// EvtForwarder
//go:generate go run github.com/golang/mock/mockgen -destination mocks/evt_forwarder_mock.go -package mocks code.vegaprotocol.io/vega/api  EvtForwarder
type EvtForwarder interface {
	Forward(ctx context.Context, e *commandspb.ChainEvent, pk string) error
}

// Blockchain ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/api  Blockchain
type Blockchain interface {
	SubmitTransactionV2(ctx context.Context, tx *commandspb.Transaction, ty protoapi.SubmitTransactionRequest_Type) error
	GetGenesisTime(ctx context.Context) (genesisTime time.Time, err error)
	GetChainID(ctx context.Context) (chainID string, err error)
	GetNetworkInfo(ctx context.Context) (netInfo *tmctypes.ResultNetInfo, err error)
	GetStatus(ctx context.Context) (status *tmctypes.ResultStatus, err error)
	GetUnconfirmedTxCount(ctx context.Context) (count int, err error)
	Health() (*tmctypes.ResultHealth, error)
}

// GRPCServer represent the grpc api provided by the vega node
type GRPC struct {
	Config

	client        Blockchain
	log           *logging.Logger
	srv           *grpc.Server
	stats         *stats.Stats
	statusChecker *monitoring.Status
	timesvc       *vegatime.Svc
	evtfwd        EvtForwarder
	evtService    EventService

	// used in order to gracefully close streams
	ctx   context.Context
	cfunc context.CancelFunc

	core *coreService

	services []func(*grpc.Server)
}

// NewGRPC create a new instance of the GPRC api for the vega node
func NewGRPC(
	log *logging.Logger,
	config Config,
	stats *stats.Stats,
	client Blockchain,
	evtfwd EvtForwarder,
	timeService *vegatime.Svc,
	eventService *subscribers.Service,
	statusChecker *monitoring.Status,
) *GRPC {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	ctx, cfunc := context.WithCancel(context.Background())

	return &GRPC{
		log:           log,
		Config:        config,
		stats:         stats,
		client:        client,
		timesvc:       timeService,
		statusChecker: statusChecker,
		ctx:           ctx,
		cfunc:         cfunc,
		evtfwd:        evtfwd,
		evtService:    eventService,
	}
}

func (g *GRPC) RegisterService(f func(*grpc.Server)) {
	g.services = append(g.services, f)
}

// ReloadConf update the internal configuration of the GRPC server
func (g *GRPC) ReloadConf(cfg Config) {
	g.log.Info("reloading configuration")
	if g.log.GetLevel() != cfg.Level.Get() {
		g.log.Info("updating log level",
			logging.String("old", g.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		g.log.SetLevel(cfg.Level.Get())
	}

	// TODO(): not updating the the actual server for now, may need to look at this later
	// e.g restart the http server on another port or whatever
	g.Config = cfg
	g.core.updateConfig(cfg)
}

func remoteAddrInterceptor(log *logging.Logger) grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (resp interface{}, err error) {
		// first check if the request is forwarded from our restproxy
		// get the metadata
		var ip string
		md, ok := metadata.FromIncomingContext(ctx)
		if ok {
			forwardedFor, ok := md["x-forwarded-for"]
			if ok && len(forwardedFor) > 0 {
				log.Debug("grpc request x-forwarded-for",
					logging.String("method", info.FullMethod),
					logging.String("remote-ip-addr", forwardedFor[0]),
				)
				ip = forwardedFor[0]
			}
		}

		// if the request is not forwarded let's get it from the peer infos
		if len(ip) <= 0 {
			p, ok := peer.FromContext(ctx)
			if ok && p != nil {
				log.Debug("grpc peer client request",
					logging.String("method", info.FullMethod),
					logging.String("remote-ip-addr", p.Addr.String()))
				ip = p.Addr.String()
			}
		}

		ctx = vgcontext.WithRemoteIPAddr(ctx, ip)

		// Calls the handler
		h, err := handler(ctx, req)

		log.Debug("Invoked RPC call",
			logging.String("method", info.FullMethod),
			logging.Error(err),
		)

		return h, err
	}
}

// Start start the grpc server
func (g *GRPC) Start() {
	ip := g.IP
	port := strconv.Itoa(g.Port)

	g.log.Info("Starting gRPC based API", logging.String("addr", ip), logging.String("port", port))

	lis, err := net.Listen("tcp", net.JoinHostPort(ip, port))
	if err != nil {
		g.log.Panic("Failure listening on gRPC port", logging.String("port", port), logging.Error(err))
	}

	intercept := grpc.UnaryInterceptor(remoteAddrInterceptor(g.log))
	g.srv = grpc.NewServer(intercept)

	coreSvc := &coreService{
		log:           g.log,
		conf:          g.Config,
		blockchain:    g.client,
		statusChecker: g.statusChecker,
		timesvc:       g.timesvc,
		stats:         g.stats,
		evtForwarder:  g.evtfwd,
		eventService:  g.evtService,
	}
	g.core = coreSvc
	protoapi.RegisterCoreServiceServer(g.srv, coreSvc)

	for _, f := range g.services {
		f(g.srv)
	}

	go g.core.updateNetInfo(g.ctx)

	err = g.srv.Serve(lis)
	if err != nil {
		g.log.Panic("Failure serving gRPC API", logging.Error(err))
	}
}

// Stop stops the GRPC server
func (g *GRPC) Stop() {
	if g.srv == nil {
		return
	}

	done := make(chan struct{})
	go func() {
		g.log.Info("Gracefully stopping gRPC based API")
		g.srv.GracefulStop()
		done <- struct{}{}
	}()

	select {
	case <-done:
	case <-time.After(10 * time.Second):
		g.log.Info("Force stopping gRPC based API")
		g.srv.Stop()
	}
}
