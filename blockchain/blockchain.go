package blockchain

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/blockchain/tm"
	"code.vegaprotocol.io/vega/logging"

	tmtypes "github.com/tendermint/tendermint/abci/types"
)

var (
	ErrInvalidChainProvider = errors.New("invalid chain provider")
)

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
	log  *logging.Logger
	cfg  *Config
	clt  *Client
	abci tmtypes.Application
	srv  *tm.Server
}

func New(
	log *logging.Logger,
	cfg Config,
	abci tmtypes.Application,
) (*Blockchain, error) {
	clt, err := tm.NewClient(cfg.Tendermint)
	if err != nil {
		return nil, err
	}

	srv := tm.NewServer(log, cfg.Tendermint, abci)
	if err := srv.Start(); err != nil {
		return nil, err
	}

	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	return &Blockchain{
		log:  log,
		cfg:  &cfg,
		clt:  NewClient(clt),
		abci: abci,
		srv:  srv,
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

	// TODO(gchain): enable this
	/*
		b.processor.ReloadConf(cfg)
		if chain, ok := b.chain.(*tm.TMChain); ok {
			chain.ReloadConf(cfg.Tendermint)
		}
	*/
}

func (b *Blockchain) Stop() error {
	b.srv.Stop()
	return nil
}

func (b *Blockchain) Client() *Client {
	return b.clt
}
