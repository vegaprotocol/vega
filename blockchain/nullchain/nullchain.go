package nullchain

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	vgfs "code.vegaprotocol.io/shared/libs/fs"
	vgrand "code.vegaprotocol.io/shared/libs/rand"
	"code.vegaprotocol.io/vega/logging"

	abci "github.com/tendermint/tendermint/abci/types"
	"github.com/tendermint/tendermint/p2p"
	tmctypes "github.com/tendermint/tendermint/rpc/core/types"
	tmtypes "github.com/tendermint/tendermint/types"
)

var ErrGenesisFileRequired = errors.New("--blockchain.nullchain.genesis-file  is required")

//go:generate go run github.com/golang/mock/mockgen -destination mocks/application_service_mock.go -package mocks code.vegaprotocol.io/vega/blockchain/nullchain ApplicationService
type ApplicationService interface {
	InitChain(res abci.RequestInitChain) (resp abci.ResponseInitChain)
}

type NullBlockchain struct {
	log         *logging.Logger
	service     ApplicationService
	chainID     string
	genesisTime time.Time

	blockHeight int64
	now         time.Time

	transactionsPerBlock uint64
	blockDuration        time.Duration
}

func NewClient(
	log *logging.Logger,
	cfg Config,
	service ApplicationService,
) (*NullBlockchain, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(cfg.Level.Get())
	log.Info("starting nullblockchain")

	n := &NullBlockchain{
		log:                  log,
		blockHeight:          1,
		service:              service,
		chainID:              vgrand.RandomStr(12),
		transactionsPerBlock: cfg.TransactionsPerBlock,
		blockDuration:        cfg.BlockDuration.Duration,
	}

	err := n.InitChain(cfg.GenesisFile)
	if err != nil {
		return nil, err
	}
	return n, nil
}

// Blockchain server calls -- when tendermint calls into core

func (n *NullBlockchain) InitChain(genesisFile string) error {
	exists, err := vgfs.FileExists(genesisFile)
	if !exists || err != nil {
		return ErrGenesisFileRequired
	}

	b, err := vgfs.ReadFile(genesisFile)
	if err != nil {
		return err
	}

	// Parse the appstate of the genesis file so that we can send the netparams to core
	// a tendermint genesis-file will do
	genesis := struct {
		Appstate json.RawMessage `json:"app_state"`
	}{}

	err = json.Unmarshal(b, &genesis)
	if err != nil {
		return err
	}

	n.service.InitChain(
		abci.RequestInitChain{
			Time:          n.now,
			ChainId:       n.chainID,
			InitialHeight: n.blockHeight,
			AppStateBytes: genesis.Appstate,
		})
	return nil
}

// Blockchain client calls -- when core sends in requests to tendermint
// Everything that isn't needed for starting out is currently just stubbed out

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
	return 0, nil
}

func (n *NullBlockchain) Health(_ context.Context) (*tmctypes.ResultHealth, error) {
	return &tmctypes.ResultHealth{}, nil
}

func (n *NullBlockchain) SendTransaction(ctx context.Context, tx []byte) (bool, error) {
	return true, nil
}

func (n *NullBlockchain) SendTransactionCommit(ctx context.Context, tx []byte) (string, error) {
	return "", nil
}

func (n *NullBlockchain) SendTransactionAsync(ctx context.Context, tx []byte) (string, error) {
	return "", nil
}

func (n *NullBlockchain) SendTransactionSync(ctx context.Context, tx []byte) (string, error) {
	return "", nil
}

func (n *NullBlockchain) Validators(_ context.Context) ([]*tmtypes.Validator, error) {
	return nil, nil
}

func (n *NullBlockchain) GenesisValidators(_ context.Context) ([]*tmtypes.Validator, error) {
	return nil, nil
}

func (n *NullBlockchain) Subscribe(context.Context, func(tmctypes.ResultEvent) error, ...string) error {
	return nil
}
