package noop

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/logging"

	"github.com/tendermint/tendermint/p2p"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

type Processor interface {
	Validate([]byte) error
	Process(payload []byte) error
	ResetSeenPayloads()
}

type ApplicationService interface {
	Begin() error
	Commit() error
}

type ApplicationTime interface {
	SetTimeNow(context.Context, time.Time)
}

type NOOPChain struct {
	log         *logging.Logger
	ticker      *time.Ticker
	time        ApplicationTime
	proc        Processor
	service     ApplicationService
	genesisTime time.Time
	chainID     string
	txs         chan []byte

	totalTxLastBatch uint64
	blockHeight      uint64
}

func New(
	log *logging.Logger,
	cfg Config,
	timeService ApplicationTime,
	proc Processor,
	service ApplicationService,
) *NOOPChain {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	n := &NOOPChain{
		log:         log,
		ticker:      time.NewTicker(cfg.BlockDuration.Get()),
		blockHeight: 1,
		time:        timeService,
		proc:        proc,
		service:     service,
		genesisTime: time.Now(),
		txs:         make(chan []byte),
	}
	n.startTicker()
	return n
}

func (n *NOOPChain) startTicker() {
	go func() {
		n.log.Info("starting new block",
			logging.String("chain-provider", "noop"),
			logging.Uint64("block-height", n.blockHeight),
		)
		n.service.Begin()
		for {
			select {
			case tx := <-n.txs:
				n.totalTxLastBatch++
				n.proc.Process(tx)
			case <-n.ticker.C:
				n.log.Info("committing block",
					logging.String("chain-provider", "noop"),
					logging.Uint64("block-height", n.blockHeight),
				)
				n.service.Commit()
				n.blockHeight++
				n.totalTxLastBatch = 0
				n.time.SetTimeNow(context.Background(), time.Now())
				n.log.Info("starting new block",
					logging.String("chain-provider", "noop"),
					logging.Uint64("block-height", n.blockHeight),
				)
				n.service.Begin()
			}
		}
	}()
}

func (n *NOOPChain) Client() *NOOPChain {
	return n
}

func (n *NOOPChain) Stop() error {
	return nil
}

func (n *NOOPChain) GetGenesisTime(context.Context) (time.Time, error) {
	return n.genesisTime, nil
}

func (n *NOOPChain) GetChainID(context.Context) (string, error) {
	return n.chainID, nil
}

func (n *NOOPChain) GetStatus(context.Context) (*tmctypes.ResultStatus, error) {
	return &tmctypes.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			Version: "0.33.8",
		},
		SyncInfo: tmctypes.SyncInfo{
			CatchingUp: false,
		},
	}, nil
}

func (n *NOOPChain) GetNetworkInfo(context.Context) (*tmctypes.ResultNetInfo, error) {
	return &tmctypes.ResultNetInfo{
		Listening: true,
		Listeners: []string{},
		NPeers:    0,
	}, nil
}

func (n *NOOPChain) GetUnconfirmedTxCount(context.Context) (int, error) {
	return len(n.txs), nil
}

func (n *NOOPChain) Health() (*tmctypes.ResultHealth, error) {
	return &tmctypes.ResultHealth{}, nil
}

func (n *NOOPChain) SendTransaction(ctx context.Context, tx []byte) (bool, error) {
	n.txs <- tx
	return true, nil
}

func (n *NOOPChain) Validators() ([]*tmtypes.Validator, error) {
	return nil, nil
}

func (n *NOOPChain) GenesisValidators() ([]*tmtypes.Validator, error) {
	return nil, nil
}

func (n *NOOPChain) Subscribe(context.Context, func(tmctypes.ResultEvent) error, ...string) error {
	return nil
}
