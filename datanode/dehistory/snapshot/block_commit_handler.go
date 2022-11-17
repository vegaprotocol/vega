package snapshot

import (
	"context"
	"errors"
	"strconv"
	"time"

	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"
)

type BlockCommitHandler struct {
	log                 *logging.Logger
	snapshotData        func(ctx context.Context, chainID string, toHeight int64, fromHeight int64) error
	getNetworkParameter func(ctx context.Context, key string) (entities.NetworkParameter, error)
	blockInterval       int64
	brokerConfig        broker.Config
}

func NewBlockCommitHandler(
	log *logging.Logger,
	snapshotData func(ctx context.Context, chainID string, toHeight int64, fromHeight int64) error,
	getNetworkParameter func(ctx context.Context, key string) (entities.NetworkParameter, error),
	brokerConfig broker.Config,
) *BlockCommitHandler {
	return &BlockCommitHandler{
		log:                 log.Named("block-commit-handler"),
		snapshotData:        snapshotData,
		getNetworkParameter: getNetworkParameter,
		brokerConfig:        brokerConfig,
	}
}

func (b *BlockCommitHandler) OnBlockCommitted(ctx context.Context, chainID string, blockHeight int64) {
	// We poll for the snapshot interval on block commit to ensure that the correct interval
	// is always used, this is simpler than listening for the network parameter event and then
	// having to handle recovery scenarios.
	param, err := b.getNetworkParameter(ctx, netparams.SnapshotIntervalLength)
	if err != nil {
		if !errors.Is(err, entities.ErrNotFound) {
			b.log.Errorf("failed to get snapshot interval length network parameter:%w", err)
		}
	} else {
		blockInterval, err := strconv.ParseInt(param.Value, 10, 64)
		if err != nil {
			b.log.Errorf("failed to parse snapshot interval length network parameter:%w", err)
		} else {
			b.blockInterval = blockInterval

			// An interval less than 1000 when using a file source with no time between blocks results
			// in excessive snapshot data creation and should be avoided, 1000 is a reasonable default
			if b.brokerConfig.UseEventFile &&
				b.brokerConfig.FileEventSourceConfig.TimeBetweenBlocks.Duration < time.Second {
				if blockInterval < 1000 {
					b.blockInterval = 1000
				}
			}
		}
	}

	if b.snapshotRequiredAtBlockHeight(blockHeight) {
		fromHeight := GetFromHeight(blockHeight, b.blockInterval)
		err = b.snapshotData(ctx, chainID, blockHeight, fromHeight)
		if err != nil {
			b.log.Errorf("failed to snapshot data:%w", err)
		}
	}
}

func (b *BlockCommitHandler) snapshotRequiredAtBlockHeight(lastCommittedBlockHeight int64) bool {
	if b.blockInterval > 0 {
		return lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%b.blockInterval == 0
	}

	return false
}

func GetFromHeight(toHeight int64, snapshotInterval int64) int64 {
	return toHeight - (snapshotInterval - 1)
}
