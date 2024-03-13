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
	Blocks       uint   `default:"10000" description:"set the amount of blocks to keep" long:"blocks" required:"true"`
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
