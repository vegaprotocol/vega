package blockchain

import (
	"context"
	"errors"
	"strings"
	"time"

	"code.vegaprotocol.io/vega/blockchain/noop"
	"code.vegaprotocol.io/vega/blockchain/tm"
	"code.vegaprotocol.io/vega/logging"
)

var (
	ErrInvalidChainProvider = errors.New("invalid chain provider")
)

type TimeService interface {
	SetTimeNow(context.Context, time.Time)
	GetTimeNow() (time.Time, error)
	GetTimeLastBatch() (time.Time, error)
}

type ABCIEngine interface {
	Processor
	Commit() error
	Begin() error
}

type Commander interface {
	SetChain(*Client)
}

type chainImpl interface {
	Stop() error
}

type GenesisHandler interface {
	OnGenesis(genesisTime time.Time, appState []byte, validatorsPubkey [][]byte) error
}

type Blockchain struct {
	log        *logging.Logger
	clt        *Client
	chain      chainImpl
	abciEngine ABCIEngine
	time       TimeService
	processor  *codec
}

func New(
	log *logging.Logger,
	cfg Config,
	abciEngine ABCIEngine,
	time TimeService,
	commander Commander,
	cancel func(),
	ghandler GenesisHandler,
	top tm.ValidatorTopology,
) (*Blockchain, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	var (
		clt   chainClientImpl
		chain chainImpl
		err   error
	)

	proc := NewCodec(log, cfg, abciEngine)
	// proc := NewProcessor(log, cfg, service)

	switch strings.ToLower(cfg.ChainProvider) {
	case "tendermint":
		chain, err = tm.New(log, cfg.Tendermint, proc, abciEngine, time, cancel, ghandler, top)
		if err == nil {
			clt, err = tm.NewClient(cfg.Tendermint)
		}
	case "noop":
		noopchain := noop.New(log, cfg.Noop, time, proc, abciEngine)
		chain = noopchain
		clt = noopchain
	default:
		err = ErrInvalidChainProvider
	}
	if err != nil {
		return nil, err
	}
	client := newClient(clt)
	commander.SetChain(client)

	log.Info("vega blockchain initialised", logging.String("chain-provider", cfg.ChainProvider))

	return &Blockchain{
		log:        log,
		clt:        client,
		chain:      chain,
		abciEngine: abciEngine,
		time:       time,
		processor:  proc,
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
