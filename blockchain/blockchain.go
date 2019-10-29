package blockchain

import (
	"errors"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/blockchain/noop"
	"code.vegaprotocol.io/vega/blockchain/tm"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	ErrInvalidChainProvider = errors.New("invalid chain provider")
)

type ExecutionEngine interface {
	SubmitOrder(order *types.Order) (*types.OrderConfirmation, error)
	CancelOrder(order *types.Order) (*types.OrderCancellationConfirmation, error)
	AmendOrder(order *types.OrderAmendment) (*types.OrderConfirmation, error)
	NotifyTraderAccount(notif *types.NotifyTraderAccount) error
	Withdraw(w *types.Withdraw) error
	Generate() error
}

type TimeService interface {
	SetTimeNow(time.Time)
	GetTimeNow() (time.Time, error)
	GetTimeLastBatch() (time.Time, error)
}

type chainImpl interface {
	Stop() error
}

type Blockchain struct {
	log        *logging.Logger
	clt        *Client
	chain      chainImpl
	execEngine ExecutionEngine
	time       TimeService
	processor  *Processor
	service    *abciService
	stats      *Stats
}

func New(
	log *logging.Logger,
	cfg Config,
	execEngine ExecutionEngine,
	time TimeService,
	stats *Stats,
	cancel func(),
) (*Blockchain, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	var (
		clt   chainClientImpl
		chain chainImpl
		err   error
	)

	service := newService(log, cfg, stats, execEngine, time)
	proc := NewProcessor(log, cfg, service)

	switch strings.ToLower(cfg.ChainProvider) {
	case "tendermint":
		chain, err = tm.New(log, cfg.Tendermint, stats, proc, service, time, cancel)
		if err == nil {
			clt, err = tm.NewClient(cfg.Tendermint)
		}
	case "noop":
		noopchain := noop.New(log, cfg.Noop, stats, time, proc, service)
		chain = noopchain
		clt = noopchain
	default:
		err = ErrInvalidChainProvider
	}
	if err != nil {
		return nil, err
	}

	log.Info("vega blockchain initialized", logging.String("chain-provider", cfg.ChainProvider))

	return &Blockchain{
		log:        log,
		clt:        newClient(clt),
		chain:      chain,
		execEngine: execEngine,
		time:       time,
		processor:  proc,
		stats:      stats,
	}, nil
}

// ReloadConf update the internal configuration of the processor
func (b *Blockchain) ReloadConf(cfg Config) {
	b.log.Info("reloading configuration")
	if b.log.GetLevel() != cfg.Level.Get() {
		b.log.Info("updating log level",
			logging.String("old", b.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		b.log.SetLevel(cfg.Level.Get())
	}
	b.processor.ReloadConf(cfg)
	b.service.ReloadConf(cfg)
	if chain, ok := b.chain.(*tm.TMChain); ok {
		chain.ReloadConf(cfg.Tendermint)
	}
}

func (b *Blockchain) Stop() error {
	return b.chain.Stop()
}

func (b *Blockchain) Client() *Client {
	return b.clt
}
