// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package storage

import (
	"fmt"
	"os"

	"code.vegaprotocol.io/shared/paths"
)

const (
	// AccountsDataPath is the default path for the account store files
	AccountsDataPath = "accounts"
	// CandlesDataPath is the default path for the candle store files
	CandlesDataPath = "candles"
	// CheckpointsDataPath is the default path for the checkpoints store files
	CheckpointsDataPath = "checkpoints"
	// MarketsDataPath is the default path for the market store files
	MarketsDataPath = "markets"
	// OrdersDataPath is the default path for the order store files
	OrdersDataPath = "orders"
	// TradesDataPath is the default path for the trade store files
	TradesDataPath = "trades"
	// ChainInfoPath is the default path for the chain files
	ChainInfoPath = "chain_info"
)

type Storage struct {
	BaseDir         string
	AccountsHome    string
	OrdersHome      string
	TradesHome      string
	CandlesHome     string
	MarketsHome     string
	CheckpointsHome string
	ChainInfoHome   string
}

func InitialiseStorage(vegaPaths paths.Paths) (*Storage, error) {
	var err error

	storageHome, err := vegaPaths.CreateStateDirFor(paths.DataNodeStorageHome)
	if err != nil {
		return nil, fmt.Errorf("couldn't get storage directory: %w", err)
	}

	storage := &Storage{
		BaseDir: storageHome,
	}

	if storage.AccountsHome, err = vegaPaths.CreateStateDirFor(paths.JoinStatePath(paths.DataNodeStorageHome, AccountsDataPath)); err != nil {
		return nil, fmt.Errorf("couldn't get accounts storage directory: %w", err)
	}

	if storage.OrdersHome, err = vegaPaths.CreateStateDirFor(paths.JoinStatePath(paths.DataNodeStorageHome, OrdersDataPath)); err != nil {
		return nil, fmt.Errorf("couldn't get orders storage directory: %w", err)
	}

	if storage.TradesHome, err = vegaPaths.CreateStateDirFor(paths.JoinStatePath(paths.DataNodeStorageHome, TradesDataPath)); err != nil {
		return nil, fmt.Errorf("couldn't get trades storage directory: %w", err)
	}

	if storage.CandlesHome, err = vegaPaths.CreateStateDirFor(paths.JoinStatePath(paths.DataNodeStorageHome, CandlesDataPath)); err != nil {
		return nil, fmt.Errorf("couldn't get candles storage directory: %w", err)
	}

	if storage.MarketsHome, err = vegaPaths.CreateStateDirFor(paths.JoinStatePath(paths.DataNodeStorageHome, MarketsDataPath)); err != nil {
		return nil, fmt.Errorf("couldn't get accounts storage directory: %w", err)
	}

	if storage.CheckpointsHome, err = vegaPaths.CreateStateDirFor(paths.JoinStatePath(paths.DataNodeStorageHome, CheckpointsDataPath)); err != nil {
		return nil, fmt.Errorf("couldn't get checkpoints storage directory: %w", err)
	}

	if storage.ChainInfoHome, err = vegaPaths.CreateStateDirFor(paths.JoinStatePath(paths.DataNodeStorageHome, ChainInfoPath)); err != nil {
		return nil, fmt.Errorf("couldn't get chain info storage directory: %w", err)
	}

	return storage, nil
}

// Purge will remove/clear the badger key and value files (i.e. databases)
// from disk at the locations specified by the given storage.Config. This is
// currently used within unit and integration tests to clear between runs.
func (s *Storage) Purge() {
	_ = os.RemoveAll(s.BaseDir)
}
