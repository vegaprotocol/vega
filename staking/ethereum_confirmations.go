package staking

import (
	"context"
	"math/big"
	"sync"
	"time"

	vgproto "code.vegaprotocol.io/protos/vega"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/ethereum_client_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/staking EthereumClientConfirmations
type EthereumClientConfirmations interface {
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_mock.go -package mocks code.vegaprotocol.io/vega/staking Time
type Time interface {
	Now() time.Time
}

type StdTime struct{}

func (StdTime) Now() time.Time { return time.Now() }

type EthereumConfirmations struct {
	ethClient EthereumClientConfirmations

	time Time

	mu                  sync.Mutex
	required            uint64
	curHeight           uint64
	curHeightLastUpdate time.Time
}

func NewEthereumConfirmations(
	ethClient EthereumClientConfirmations,
	time Time,
) *EthereumConfirmations {
	if time == nil {
		time = StdTime{}
	}
	return &EthereumConfirmations{
		ethClient: ethClient,
		time:      time,
	}
}

func (e *EthereumConfirmations) OnEthereumConfigUpdate(rawcfg interface{}) error {
	cfg, ok := rawcfg.(*vgproto.EthereumConfig)
	if !ok {
		return ErrNotAnEthereumConfig
	}

	e.set(uint64(cfg.Confirmations))

	return nil
}

func (e *EthereumConfirmations) set(confirmations uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.required = confirmations
}

func (e *EthereumConfirmations) Check(block uint64) error {
	curBlock, err := e.currentHeight(context.Background())
	if err != nil {
		return err
	}

	if curBlock < block ||
		(curBlock-block) < uint64(e.required) {
		return ErrMissingConfirmations
	}

	return nil
}

func (e *EthereumConfirmations) currentHeight(
	ctx context.Context) (uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// if last update of the heigh was more that 15 seconds
	// ago, we try to update, we assume an eth block takes
	// ~15 seconds
	now := e.time.Now()
	if e.curHeightLastUpdate.Add(15 * time.Second).Before(now) {
		// get the last block header
		h, err := e.ethClient.HeaderByNumber(context.Background(), nil)
		if err != nil {
			return e.curHeight, err
		}
		e.curHeightLastUpdate = now
		e.curHeight = h.Number.Uint64()
	}

	return e.curHeight, nil
}
