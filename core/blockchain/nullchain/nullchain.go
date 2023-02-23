// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package nullchain

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/blockchain"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	vgfs "code.vegaprotocol.io/vega/libs/fs"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"

	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/p2p"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	tmctypes "github.com/cometbft/cometbft/rpc/core/types"
	tmtypes "github.com/cometbft/cometbft/types"
)

const namedLogger = "nullchain"

var (
	ErrNotImplemented      = errors.New("not implemented for nullblockchain")
	ErrChainReplaying      = errors.New("nullblockchain is replaying")
	ErrGenesisFileRequired = errors.New("--blockchain.nullchain.genesis-file is required")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/mocks.go -package mocks code.vegaprotocol.io/vega/core/blockchain/nullchain TimeService,ApplicationService
type TimeService interface {
	GetTimeNow() time.Time
}

type ApplicationService interface {
	InitChain(res abci.RequestInitChain) (resp abci.ResponseInitChain)
	BeginBlock(req abci.RequestBeginBlock) (resp abci.ResponseBeginBlock)
	EndBlock(req abci.RequestEndBlock) (resp abci.ResponseEndBlock)
	Commit() (resp abci.ResponseCommit)
	DeliverTx(req abci.RequestDeliverTx) (resp abci.ResponseDeliverTx)
	Info(req abci.RequestInfo) (resp abci.ResponseInfo)
}

// nullGenesis is a subset of a tendermint genesis file, just the bits we need to run the nullblockchain.
type nullGenesis struct {
	GenesisTime *time.Time      `json:"genesis_time"`
	ChainID     string          `json:"chain_id"`
	Appstate    json.RawMessage `json:"app_state"`
}

type NullBlockchain struct {
	log                  *logging.Logger
	cfg                  blockchain.NullChainConfig
	app                  ApplicationService
	timeService          TimeService
	srv                  *http.Server
	genesis              nullGenesis
	blockDuration        time.Duration
	transactionsPerBlock uint64

	now         time.Time
	blockHeight int64
	pending     []*abci.RequestDeliverTx

	mu        sync.Mutex
	replaying atomic.Bool
	replayer  *Replayer
}

func NewClient(
	log *logging.Logger,
	cfg blockchain.NullChainConfig,
	timeService TimeService,
) *NullBlockchain {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())

	n := &NullBlockchain{
		log:                  log,
		cfg:                  cfg,
		timeService:          timeService,
		transactionsPerBlock: cfg.TransactionsPerBlock,
		blockDuration:        cfg.BlockDuration.Duration,
		blockHeight:          1,
		pending:              make([]*abci.RequestDeliverTx, 0),
	}

	return n
}

func (n *NullBlockchain) SetABCIApp(app ApplicationService) {
	n.app = app
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
	if err := n.parseGenesis(); err != nil {
		return err
	}

	if r := n.app.Info(abci.RequestInfo{}); r.LastBlockHeight > 0 {
		n.log.Info("protocol loaded from snapshot", logging.Int64("height", r.LastBlockHeight))
		n.blockHeight = r.LastBlockHeight + 1
		n.now = n.timeService.GetTimeNow().Add(n.blockDuration)
	} else {
		n.log.Info("initialising new chain", logging.String("chain-id", n.genesis.ChainID), logging.Time("chain-time", n.now))
		err := n.InitChain()
		if err != nil {
			return err
		}
	}

	// not replaying or recording, proceed as normal
	if !n.cfg.Replay.Record && !n.cfg.Replay.Replay {
		return nil
	}

	r, err := NewNullChainReplayer(n.app, n.cfg.Replay, n.log)
	if err != nil {
		return err
	}
	n.replayer = r

	if n.cfg.Replay.Replay {
		n.log.Info("nullchain is replaying chain", logging.String("replay-file", n.cfg.Replay.ReplayFile))
		n.replaying.Store(true)
		blockHeight, blockTime, err := r.replayChain(n.blockHeight, n.genesis.ChainID)
		if err != nil {
			return err
		}
		n.replaying.Store(false)

		n.log.Info("nullchain finished replaying chain", logging.Int64("block-height", blockHeight))
		if blockHeight != 0 {
			// set the next height to where we replayed to
			n.blockHeight = blockHeight + 1
			n.now = blockTime.Add(n.blockDuration)
		}
	}

	if n.cfg.Replay.Record {
		n.log.Info("nullchain is recording chain data", logging.String("replay-file", n.cfg.Replay.ReplayFile))
	}

	return nil
}

func (n *NullBlockchain) processBlock() {
	if n.log.GetLevel() <= logging.DebugLevel {
		n.log.Debugf("processing block %d with %d transactions", n.blockHeight, len(n.pending))
	}

	resp := abci.ResponseCommit{}
	if n.replayer != nil && n.cfg.Replay.Record {
		n.replayer.startBlock(n.blockHeight, n.now.UnixNano(), n.pending)
		defer func() { n.replayer.saveBlock(resp.Data) }()
	}

	n.BeginBlock()
	for _, tx := range n.pending {
		n.app.DeliverTx(*tx)
	}
	n.pending = n.pending[:0]

	n.EndBlock()
	resp = n.app.Commit()

	// Increment time, blockheight, ready to start a new block
	n.blockHeight++
	n.now = n.now.Add(n.blockDuration)
}

func (n *NullBlockchain) handleTransaction(tx []byte) {
	n.mu.Lock()
	defer n.mu.Unlock()

	n.pending = append(n.pending, &abci.RequestDeliverTx{Tx: tx})
	if n.log.GetLevel() <= logging.DebugLevel {
		n.log.Debugf("transaction added to block: %d of %d", len(n.pending), n.transactionsPerBlock)
	}
	if len(n.pending) == int(n.transactionsPerBlock) {
		n.processBlock()
	}
}

// parseGenesis reads the Tendermint genesis file defined in the cfg and saves values needed to run the chain.
func (n *NullBlockchain) parseGenesis() error {
	var ng nullGenesis
	exists, err := vgfs.FileExists(n.cfg.GenesisFile)
	if !exists || err != nil {
		return ErrGenesisFileRequired
	}

	b, err := vgfs.ReadFile(n.cfg.GenesisFile)
	if err != nil {
		return err
	}

	err = json.Unmarshal(b, &ng)
	if err != nil {
		return err
	}

	n.now = time.Now()
	if ng.GenesisTime != nil {
		n.now = *ng.GenesisTime
	} else {
		// genesisTime not provided, just use now
		ng.GenesisTime = &n.now
	}

	if len(ng.ChainID) == 0 {
		// chainID not provided we'll just make one up
		ng.ChainID = vgrand.RandomStr(12)
	}

	n.genesis = ng
	return nil
}

// ForwardTime moves the chain time forward by the given duration, delivering any pending
// transaction and creating any extra empty blocks if time is stepped forward by more than
// a block duration.
func (n *NullBlockchain) ForwardTime(d time.Duration) {
	n.log.Debugf("time-forwarding by %s", d)

	nBlocks := d / n.blockDuration
	if nBlocks == 0 {
		n.log.Errorf("not a full block-duration, not moving time: %s < %s", d, n.blockDuration)
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
func (n *NullBlockchain) InitChain() error {
	// read appstate so that we can set the validators
	appstate := struct {
		Validators map[string]struct{} `json:"validators"`
	}{}

	if err := json.Unmarshal(n.genesis.Appstate, &appstate); err != nil {
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
		logging.String("chainID", n.genesis.ChainID),
		logging.Int64("blockHeight", n.blockHeight),
		logging.String("time", n.now.String()),
		logging.Int("n_validators", len(validators)),
	)
	n.app.InitChain(
		abci.RequestInitChain{
			Time:          n.now,
			ChainId:       n.genesis.ChainID,
			InitialHeight: n.blockHeight,
			AppStateBytes: n.genesis.Appstate,
			Validators:    validators,
		},
	)
	return nil
}

func (n *NullBlockchain) BeginBlock() *NullBlockchain {
	if n.log.GetLevel() <= logging.DebugLevel {
		n.log.Debug("sending BeginBlock",
			logging.Int64("height", n.blockHeight),
			logging.String("time", n.now.String()),
		)
	}

	r := abci.RequestBeginBlock{
		Header: types.Header{
			Time:    n.now,
			Height:  n.blockHeight,
			ChainID: n.genesis.ChainID,
		},
		Hash: vgcrypto.Hash([]byte(strconv.FormatInt(n.blockHeight+n.now.UnixNano(), 10))),
	}
	n.app.BeginBlock(r)
	return n
}

func (n *NullBlockchain) EndBlock() *NullBlockchain {
	if n.log.GetLevel() <= logging.DebugLevel {
		n.log.Debug("sending EndBlock", logging.Int64("blockHeight", n.blockHeight))
	}
	n.app.EndBlock(
		abci.RequestEndBlock{
			Height: n.blockHeight,
		},
	)
	return n
}

func (n *NullBlockchain) GetGenesisTime(context.Context) (time.Time, error) {
	return *n.genesis.GenesisTime, nil
}

func (n *NullBlockchain) GetChainID(context.Context) (string, error) {
	return n.genesis.ChainID, nil
}

func (n *NullBlockchain) GetStatus(context.Context) (*tmctypes.ResultStatus, error) {
	return &tmctypes.ResultStatus{
		NodeInfo: p2p.DefaultNodeInfo{
			Version: "0.34.20",
		},
		SyncInfo: tmctypes.SyncInfo{
			CatchingUp: n.replaying.Load(),
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

func (n *NullBlockchain) SendTransactionAsync(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTx, error) {
	if n.replaying.Load() {
		return &tmctypes.ResultBroadcastTx{}, ErrChainReplaying
	}
	go func() {
		n.handleTransaction(tx)
	}()
	randHash := []byte(vgrand.RandomStr(64))
	return &tmctypes.ResultBroadcastTx{Hash: randHash}, nil
}

func (n *NullBlockchain) CheckTransaction(ctx context.Context, tx []byte) (*tmctypes.ResultCheckTx, error) {
	n.log.Error("not implemented")
	return &tmctypes.ResultCheckTx{}, ErrNotImplemented
}

func (n *NullBlockchain) SendTransactionSync(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTx, error) {
	if n.replaying.Load() {
		return &tmctypes.ResultBroadcastTx{}, ErrChainReplaying
	}
	n.handleTransaction(tx)
	randHash := []byte(vgrand.RandomStr(64))
	return &tmctypes.ResultBroadcastTx{Hash: randHash}, nil
}

func (n *NullBlockchain) SendTransactionCommit(ctx context.Context, tx []byte) (*tmctypes.ResultBroadcastTxCommit, error) {
	// I think its worth only implementing this if needed. With time-forwarding we already have
	// control over when a block ends and gets committed, so I don't think its worth adding the
	// the complexity of trying to keep track of tx deliveries here.
	n.log.Error("not implemented")
	randHash := []byte(vgrand.RandomStr(64))
	return &tmctypes.ResultBroadcastTxCommit{Hash: randHash}, ErrNotImplemented
}

func (n *NullBlockchain) Validators(_ context.Context, _ *int64) ([]*tmtypes.Validator, error) {
	// TODO: if we are feeling brave we, could pretend to have a validator set and open
	// up the nullblockchain to more code paths
	return []*tmtypes.Validator{}, nil
}

func (n *NullBlockchain) GenesisValidators(_ context.Context) ([]*tmtypes.Validator, error) {
	n.log.Error("not implemented")
	return nil, ErrNotImplemented
}

func (n *NullBlockchain) Subscribe(context.Context, func(tmctypes.ResultEvent) error, ...string) error {
	n.log.Error("not implemented")
	return ErrNotImplemented
}
