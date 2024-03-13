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

package commands

import (
	"context"

	"code.vegaprotocol.io/vega/logging"

	db "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/state"
	tmstore "github.com/cometbft/cometbft/store"
	"github.com/jessevdk/go-flags"
	"github.com/syndtr/goleveldb/leveldb/opt"
)

type PruneCmd struct {
	CometBFTHome string `default:"$HOME/.cometbft" description:"Directory for cometbft config and data" long:"cometbft-home" required:"true"`
	Blocks       uint   `default:"10000"           description:"set the amount of blocks to keep"       long:"blocks"        required:"true"`
}

var pruneCmd PruneCmd

func (opts *PruneCmd) Execute(args []string) error {
	logger := logging.NewLoggerFromConfig(logging.NewDefaultConfig())
	defer logger.AtExit()

	dbDir := opts.CometBFTHome

	o := opt.Options{
		DisableSeeksCompaction: true,
	}

	// Get BlockStore
	blockStoreDB, err := db.NewGoLevelDBWithOpts("blockstore", dbDir, &o)
	if err != nil {
		return err
	}
	blockStore := tmstore.NewBlockStore(blockStoreDB)

	// Get StateStore
	stateDB, err := db.NewGoLevelDBWithOpts("state", dbDir, &o)
	if err != nil {
		return err
	}

	stateStore := state.NewStore(stateDB, state.StoreOptions{DiscardABCIResponses: false})

	base := blockStore.Base()

	pruneHeight := blockStore.Height() - int64(opts.Blocks)

	state, err := stateStore.Load()
	if err != nil {
		return err
	}

	logger.Info("pruning block store")
	_, pruneHeaderHeight, err := blockStore.PruneBlocks(pruneHeight, state)
	if err != nil {
		return err
	}

	logger.Info("compacting block store")
	if err := blockStoreDB.Compact(nil, nil); err != nil {
		return err
	}

	logger.Info("pruning state store")
	err = stateStore.PruneStates(base, pruneHeight, pruneHeaderHeight)
	if err != nil {
		return err
	}

	logger.Info("compacting state store")
	if err := stateDB.Compact(nil, nil); err != nil {
		return err
	}

	return nil
}

func Prune(ctx context.Context, parser *flags.Parser) error {
	pruneCmd = PruneCmd{}

	var (
		short = "Prune a vega node state"
		long  = "Prune will remove the cometbft chain state up to a given block"
	)
	_, err := parser.AddCommand("prune", short, long, &pruneCmd)
	return err
}
