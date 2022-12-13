package dehistory

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/logging"
)

type BlockCommitHandler struct {
	log                       *logging.Logger
	cfg                       Config
	snapshotData              func(ctx context.Context, chainID string, toHeight int64) error
	usingEventFile            bool
	eventFileTimeBetweenBlock time.Duration
}

func NewBlockCommitHandler(
	log *logging.Logger,
	cfg Config,
	snapshotData func(ctx context.Context, chainID string, toHeight int64) error,
	usingEventFile bool, eventFileTimeBetweenBlock time.Duration,
) *BlockCommitHandler {
	return &BlockCommitHandler{
		log:                       log.Named("block-commit-handler"),
		cfg:                       cfg,
		snapshotData:              snapshotData,
		usingEventFile:            usingEventFile,
		eventFileTimeBetweenBlock: eventFileTimeBetweenBlock,
	}
}

func (b *BlockCommitHandler) OnBlockCommitted(ctx context.Context, chainID string, blockHeight int64, snapshotTaken bool) {
	snapTaken := snapshotTaken
	if b.usingEventFile && b.eventFileTimeBetweenBlock < time.Second {
		snapTaken = blockHeight%1000 == 0
	}
	if blockHeight > 0 && bool(b.cfg.Publish) && snapTaken {
		err := b.snapshotData(ctx, chainID, blockHeight)
		if err != nil {
			b.log.Errorf("failed to snapshot data:%w", err)
		}
	}
}
