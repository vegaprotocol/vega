package nullchain

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/blockchain"
	"code.vegaprotocol.io/vega/logging"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/p2p"
	"github.com/tendermint/tendermint/proto/tendermint/crypto"
	"github.com/tendermint/tendermint/proto/tendermint/types"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

const namedLogger = "nullchain"

var (
	ErrNotImplemented      = errors.New("not implemented for nullblockchain")
	ErrGenesisFileRequired = errors.New("--blockchain.nullchain.genesis-file is required")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_service_mock.go -package mocks code.vegaprotocol.io/vega/blockchain/nullchain ApplicationService
type ApplicationService interface {
	InitChain(res abci.RequestInitChain) (resp abci.ResponseInitChain)
	BeginBlock(req abci.RequestBeginBlock) (resp abci.ResponseBeginBlock)
	EndBlock(req abci.RequestEndBlock) (resp abci.ResponseEndBlock)
	Commit() (resp abci.ResponseCommit)
	DeliverTx(req abci.RequestDeliverTx) (resp abci.ResponseDeliverTx)
}

type NullBlockchain struct {
	log                  *logging.Logger
	app                  ApplicationService
	srvAddress           string
	chainID              string
	genesisFile          string
	genesisTime          time.Time
	blockDuration        time.Duration
	transactionsPerBlock uint64

	now         time.Time
	blockHeight int64
	pending     []*abci.RequestDeliverTx

	srv *http.Server

	mu sync.Mutex
}

func NewClient(
	log *logging.Logger,
	cfg blockchain.NullChainConfig,
	app ApplicationService,
) *NullBlockchain {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	now := time.Now()
	n := &NullBlockchain{
		log:                  log,
		app:                  app,
		srvAddress:           net.JoinHostPort(cfg.IP, strconv.Itoa(cfg.Port)),
		chainID:              vgrand.RandomStr(12),
		transactionsPerBlock: cfg.TransactionsPerBlock,
		blockDuration:        cfg.BlockDuration.Duration,
		genesisFile:          cfg.GenesisFile,
		genesisTime:          now,
		now:                  now,
		blockHeight:          1,
		pending:              make([]*abci.RequestDeliverTx, 0),
	}

	return n
}

// ReloadConf update the internal configuration.
func (n *NullBlockchain) ReloadConf(cfg blockchain.Config) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.log.Info("reloading configuration")
	if n.log.GetLevel() != cfg.Level.Get() {
		n.log.Info("updating log level",
			logging.String("old", n.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		n.log.SetLevel(cfg.Level.Get())
	}

	n.blockDuration = cfg.Null.BlockDuration.Duration
	n.transactionsPerBlock = cfg.Null.TransactionsPerBlock
}

func (n *NullBlockchain) StartChain() error {
	err := n.InitChain(n.genesisFile)
	if err != nil {
		return err
	}

	// Start the first block
	n.BeginBlock()
	return nil
}

func (n *NullBlockchain) processBlock() {
	n.log.Debugf("processing block %d with %d transactions", n.blockHeight, len(n.pending))
	for _, tx := range n.pending {
		n.app.DeliverTx(*tx)
	}
	n.pending = n.pending[:0]

	n.EndBlock()
	n.app.Commit()

	// Increment time, blockheight, and start a new block
	n.blockHeight++
	n.now = n.now.Add(n.blockDuration)
	n.BeginBlock()
}

func (n *NullBlockchain) handleTransaction(tx []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.pending = append(n.pending, &abci.RequestDeliverTx{Tx: tx})

	n.log.Debugf("transaction added to block: %d of %d", len(n.pending), n.transactionsPerBlock)
	if len(n.pending) == int(n.transactionsPerBlock) {
		n.processBlock()
	}
}

// ForwardTime moves the chain time forward by the given duration, delivering any pending
// transaction and creating any extra empty blocks if time is stepped forward by more than
// a block duration.
func (n *NullBlockchain) ForwardTime(d time.Duration) {
	n.log.Debugf("time-forwarding by %s", d)

	nBlocks := d / n.blockDuration
	if nBlocks == 0 {
		n.log.Debugf("not a full block-duration, not moving time: %s < %s", d, n.blockDuration)
		return
	}

	n.mu.Lock()
	defer n.mu.Unlock()
	for i := 0; i < int(nBlocks); i++ {
		n.processBlock()
	}
}

// InitChain processes the given genesis file setting the chain's time, and passing the
// appstate through to the processors InitChain.
func (n *NullBlockchain) InitChain(genesisFile string) error {
	n.log.Debugf("creating chain",
		logging.String("genesisFile", genesisFile),
	)

	exists, err := vgfs.FileExists(genesisFile)
	if !exists || err != nil {
		return ErrGenesisFileRequired
	}

	b, err := vgfs.ReadFile(genesisFile)
	if err != nil {
		return err
	}

	// Parse the appstate of the genesis file, same layout as a TM genesis-file
	genesis := struct {
		GenesisTime *time.Time      `json:"genesis_time"`
		ChainID     string          `json:"chain_id"`
		Appstate    json.RawMessage `json:"app_state"`
	}{}

	err = json.Unmarshal(b, &genesis)
	if err != nil {
		return err
	}

	// Set genesis-time and chain-id from genesis file
	if genesis.GenesisTime != nil {
		n.genesisTime = *genesis.GenesisTime
		n.now = *genesis.GenesisTime
	}

	if len(genesis.ChainID) != 0 {
		n.chainID = genesis.ChainID
	}

	// read appstate so that we can set the validators
	appstate := struct {
		Validators map[string]struct{} `json:"validators"`
	}{}

	err = json.Unmarshal(genesis.Appstate, &appstate)
	if err != nil {
		return err
	}

	validators := make([]abci.ValidatorUpdate, 0, len(appstate.Validators))
	for k := range appstate.Validators {
		pubKey, _ := base64.StdEncoding.DecodeString(k)
		validators = append(validators,
			abci.ValidatorUpdate{
				PubKey: crypto.PublicKey{
					Sum: &crypto.PublicKey_Ed25519{
						Ed25519: pubKey,
					},
				},
			},
		)
	}

	n.log.Debug("sending InitChain into core",
		logging.String("chainID", n.chainID),
		logging.Int64("blockHeight", n.blockHeight),
		logging.String("time", n.now.String()),
		logging.Int("n_validators", len(validators)),
	)
	n.app.InitChain(
		abci.RequestInitChain{
			Time:          n.now,
			ChainId:       n.chainID,
			InitialHeight: n.blockHeight,
			AppStateBytes: genesis.Appstate,
			Validators:    validators,
		},
	)
	return nil
}

func (n *NullBlockchain) BeginBlock() *NullBlockchain {
	n.log.Debug("sending BeginBlock",
		logging.String("time", n.now.String()),
	)
	r := abci.RequestBeginBlock{
		Header: types.Header{
			Time:    n.now,
			Height:  n.blockHeight,
			ChainID: n.chainID,
		},
	}
	n.app.BeginBlock(r)
	return n
}

func (n *NullBlockchain) EndBlock() *NullBlockchain {
	n.log.Debug("sending EndBlock",
		logging.Int64("blockHeight", n.blockHeight),
	)
	r := abci.RequestEndBlock{
		Height: n.blockHeight,
	}
	n.app.EndBlock(r)
	return n
}

func (n *NullBlockchain) GetGenesisTime(context.Context) (time.Time, error) {
	return n.genesisTime, nil
}

func (n *NullBlockchain) GetChainID(context.Context) (string, error) {
	return n.chainID, nil
}

func (n *NullBlockchain) GetStatus(context.Context) (*tmctypes.ResultStatus, error) {
	return &tmctypes.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			Version: "0.34.12",
		},
		SyncInfo: tmctypes.SyncInfo{
			CatchingUp: false,
		},
	}, nil
}

func (n *NullBlockchain) GetNetworkInfo(context.Context) (*tmctypes.ResultNetInfo, error) {
	return &tmctypes.ResultNetInfo{
		Listening: true,
		Listeners: []string{},
		NPeers:    0,
	}, nil
}

func (n *NullBlockchain) GetUnconfirmedTxCount(context.Context) (int, error) {
	n.mu.Lock()
	defer n.mu.Unlock()
	return len(n.pending), nil
}

func (n *NullBlockchain) Health(_ context.Context) (*tmctypes.ResultHealth, error) {
	return &tmctypes.ResultHealth{}, nil
}

func (n *NullBlockchain) SendTransactionAsync(ctx context.Context, tx []byte) (string, error) {
	go func() {
		n.handleTransaction(tx)
	}()
	return vgrand.RandomStr(64), nil
}

func (n *NullBlockchain) SendTransactionSync(ctx context.Context, tx []byte) (string, error) {
	n.handleTransaction(tx)
	return vgrand.RandomStr(64), nil
}

func (n *NullBlockchain) SendTransactionCommit(ctx context.Context, tx []byte) (string, error) {
	// I think its worth only implementing this if needed. With time-forwarding we already have
	// control over when a block ends and gets committed, so I don't think its worth adding the
	// the complexity of trying to keep track of tx deliveries here.
	n.log.Error("not implemented")
	return "", ErrNotImplemented
}

func (n *NullBlockchain) Validators(_ context.Context, _ *int64) ([]*tmtypes.Validator, error) {
	n.log.Error("not implemented")
	return nil, ErrNotImplemented
}

func (n *NullBlockchain) GenesisValidators(_ context.Context) ([]*tmtypes.Validator, error) {
	n.log.Error("not implemented")
	return nil, ErrNotImplemented
}

func (n *NullBlockchain) Subscribe(context.Context, func(tmctypes.ResultEvent) error, ...string) error {
	n.log.Error("not implemented")
	return ErrNotImplemented
}
