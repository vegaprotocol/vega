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

package eth

import (
	"context"
	"errors"
	"math/big"
	"sync"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
)

var ErrMissingConfirmations = errors.New("not enough confirmations")

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
	cfg       Config
	ethClient EthereumClientConfirmations

	time Time

	mu                  sync.Mutex
	required            uint64
	curHeight           uint64
	curHeightLastUpdate time.Time
}

func NewEthereumConfirmations(
	cfg Config,
	ethClient EthereumClientConfirmations,
	time Time,
) *EthereumConfirmations {
	if time == nil {
		time = StdTime{}
	}
	return &EthereumConfirmations{
		cfg:       cfg,
		ethClient: ethClient,
		time:      time,
	}
}

func (e *EthereumConfirmations) UpdateConfirmations(confirmations uint64) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.required = confirmations
}

func (e *EthereumConfirmations) Check(block uint64) error {
	return e.CheckRequiredConfirmations(block, e.required)
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

func (e *EthereumConfirmations) currentHeight(
	_ context.Context,
) (uint64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// if last update of the heigh was more that 15 seconds
	// ago, we try to update, we assume an eth block takes
	// ~15 seconds
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if now := e.time.Now(); e.curHeightLastUpdate.Add(e.cfg.RetryDelay.Get()).Before(now) {
		// get the last block header
		h, err := e.ethClient.HeaderByNumber(ctx, nil)
		if err != nil {
			return e.curHeight, err
		}
		e.curHeightLastUpdate = now
		e.curHeight = h.Number.Uint64()
	}

	return e.curHeight, nil
}
