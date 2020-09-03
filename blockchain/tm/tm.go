package tm

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/logging"

	"github.com/pkg/errors"
)

type Stats interface {
	IncHeight()
	TotalTxLastBatch() uint64
	Height() uint64
	SetAverageTxPerBatch(uint64)
	SetTotalTxLastBatch(uint64)
	TotalTxCurrentBatch() uint64
	SetTotalTxCurrentBatch(uint64)
	IncTotalTxCurrentBatch()
	SetAverageTxSizeBytes(uint64)
}

type Processor interface {
	Validate([]byte) error
	Process(payload []byte) error
	ResetSeenPayloads()
}

// ApplicationService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_service_mock.go -package mocks code.vegaprotocol.io/vega/blockchain/tm ApplicationService
type ApplicationService interface {
	Begin() error
	Commit() error
}

// ApplicationTime ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_time_mock.go -package mocks code.vegaprotocol.io/vega/blockchain/tm ApplicationTime
type ApplicationTime interface {
	SetTimeNow(context.Context, time.Time)
}

type TMChain struct {
	log          *logging.Logger
	socketServer *Server
	app          *AbciApplication
	processor    Processor
	service      ApplicationService
	time         ApplicationTime
	cancel       func()
}

func New(
	log *logging.Logger,
	cfg Config,
	stats Stats,
	proc Processor,
	service ApplicationService,
	time ApplicationTime,
	cancel func(),
	ghandler GenesisHandler,
) (*TMChain, error) {
	app := NewApplication(log, cfg, stats, proc, service, time, cancel, ghandler)
	socketServer := NewServer(log, cfg, app)
	if err := socketServer.Start(); err != nil {
		return nil, errors.Wrap(err, "ABCI socket server error")
	}

	return &TMChain{
		log:          log,
		socketServer: socketServer,
		app:          app,
		processor:    proc,
		service:      service,
		time:         time,
		cancel:       cancel,
	}, nil
}

func (t *TMChain) Stop() error {
	t.socketServer.Stop()
	return nil
}

// ReloadConf update the internal configuration
func (t *TMChain) ReloadConf(cfg Config) {
	t.log.Info("reloading configuration")
	if t.log.GetLevel() != cfg.Level.Get() {
		t.log.Info("updating log level",
			logging.String("old", t.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		t.log.SetLevel(cfg.Level.Get())
	}
	t.socketServer.ReloadConf(cfg)
	t.app.ReloadConf(cfg)
}
