// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package eth

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var (
	ErrMissingConfirmations = errors.New("not enough confirmations")
	ErrBlockNotFinalized    = errors.New("block not finalized")
)

type FinalityState int

const (
	FinalityStateSafe FinalityState = iota
	FinalityStateFinalized
	FinalityStateLatest
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/ethereum_client_confirmations_mock.go -package mocks code.vegaprotocol.io/vega/core/staking EthereumClientConfirmations
type EthereumClientConfirmations interface {
	HeaderByNumber(context.Context, *big.Int) (*ethtypes.Header, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_mock.go -package mocks code.vegaprotocol.io/vega/core/client/eth Time
type Time interface {
	Now() time.Time
}

type StdTime struct{}

func (StdTime) Now() time.Time { return time.Now() }

type EthereumConfirmations struct {
	retryDelay time.Duration

	ethClient EthereumClientConfirmations

	time Time

	mu                  sync.Mutex
	required            uint64
	curHeight           uint64
	curHeightLastUpdate time.Time
	finHeight           uint64
	finHeightLastUpdate time.Time
	finState            *big.Int
}

func NewEthereumConfirmations(cfg Config, ethClient EthereumClientConfirmations, time Time, cs FinalityState) *EthereumConfirmations {
	if time == nil {
		time = StdTime{}
	}

	conf := &EthereumConfirmations{
		retryDelay: cfg.RetryDelay.Get(),
		ethClient:  ethClient,
		time:       time,
	}

	switch cs {
	case FinalityStateSafe:
		conf.finState = big.NewInt(-4)
	case FinalityStateFinalized:
		conf.finState = big.NewInt(-3)
	case FinalityStateLatest:
		conf.finState = nil
	default:
		panic("unexpected confirmation state")
	}
	return conf
}

func (e *EthereumConfirmations) GetConfirmations() uint64 {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.required
}

func (e *EthereumConfirmations) UpdateConfirmations(confirmations uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.required = confirmations
}

func (e *EthereumConfirmations) Check(block uint64) error {
	if err := e.CheckRequiredConfirmations(block, e.required); err != nil {
		return err
	}

	// if finality state is "latest" we do not need to check as this will already be done by the confirmations count
	if e.finState == nil {
		return nil
	}

	finalized, err := e.finalizedHeight(context.Background())
	if err != nil {
		return err
	}

	if finalized < block {
		return ErrBlockNotFinalized
	}

	return nil
}

func (e *EthereumConfirmations) CheckRequiredConfirmations(block uint64, required uint64) error {
	curBlock, err := e.currentHeight(context.Background())
	if err != nil {
		return err
	}

	if curBlock < block || (curBlock-block) < required {
		return ErrMissingConfirmations
	}

	return nil
}

func (e *EthereumConfirmations) finalizedHeight(ctx context.Context) (uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	h, lastUpdate, err := e.getHeight(ctx, e.finHeight, e.finHeightLastUpdate, e.finState)
	if err != nil {
		return e.finHeight, err
	}

	// update cache
	e.finHeightLastUpdate = lastUpdate
	e.finHeight = h
	return e.finHeight, err
}

func (e *EthereumConfirmations) currentHeight(ctx context.Context) (uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	h, lastUpdate, err := e.getHeight(ctx, e.curHeight, e.curHeightLastUpdate, nil)
	if err != nil {
		return e.curHeight, err
	}

	// update cache
	e.curHeightLastUpdate = lastUpdate
	e.curHeight = h
	return e.curHeight, err
}

func (e *EthereumConfirmations) getHeight(ctx context.Context, lastHeight uint64, lastUpdate time.Time, block *big.Int) (uint64, time.Time, error) {
	// if last update of the height was more that 15 seconds
	// ago, we try to update, we assume an eth block takes
	// ~15 seconds
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	if now := e.time.Now(); lastUpdate.Add(e.retryDelay).Before(now) {
		// get the last block header
		h, err := e.ethClient.HeaderByNumber(ctx, block)
		if err != nil {
			return lastHeight, lastUpdate, err
		}
		lastUpdate = now
		lastHeight = h.Number.Uint64()
	}

	return lastHeight, lastUpdate, nil
}
