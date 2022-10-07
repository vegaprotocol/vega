package snapshot

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/service"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/logging"
)

type Service struct {
	log                 *logging.Logger
	config              Config
	brokerConfig        broker.Config
	connConfig          sqlstore.ConnectionConfig
	snapshotsPath       string
	blockStore          *sqlstore.Blocks
	getNetworkParameter func(ctx context.Context, key string) (entities.NetworkParameter, error)
	chainService        *service.Chain
	blockInterval       int64
	snapshotInProgress  atomic.Bool
}

func NewSnapshotService(log *logging.Logger, config Config, brokerConfig broker.Config, blockStore *sqlstore.Blocks,
	networkParameterService func(ctx context.Context, key string) (entities.NetworkParameter, error), chainService *service.Chain, connConfig sqlstore.ConnectionConfig,
	snapshotsPath string,
) (*Service, error) {
	service := &Service{
		log:                 log.Named("snapshot"),
		config:              config,
		brokerConfig:        brokerConfig,
		connConfig:          connConfig,
		snapshotsPath:       snapshotsPath,
		blockStore:          blockStore,
		getNetworkParameter: networkParameterService,
		chainService:        chainService,
	}

	if service.config.Enabled {
		_, err := os.Stat(service.snapshotsPath)
		if err != nil {
			if os.IsNotExist(err) {
				err = os.MkdirAll(service.snapshotsPath, fs.ModePerm)
				if err != nil {
					return nil, fmt.Errorf("failed to create the snapshots dir %s: %w", service.snapshotsPath, err)
				}
			} else {
				return nil, fmt.Errorf("failed to stat the snapshots dir %s: %w", service.snapshotsPath, err)
			}
		}

		if config.RemoveSnapshotsOnStartup {
			files, err := os.ReadDir(service.snapshotsPath)
			if err != nil {
				return nil, fmt.Errorf("failed to find read contents of snapshot dir  %s: %w", service.snapshotsPath, err)
			}

			for _, file := range files {
				fileToRemove := filepath.Join(service.snapshotsPath, file.Name())
				err = os.RemoveAll(fileToRemove)
				if err != nil {
					return nil, fmt.Errorf("failed to remove file from snapshot dir  %s: %w", fileToRemove, err)
				}
			}
		}
	}

	return service, nil
}

func (b *Service) OnBlockCommitted(ctx context.Context, chainID string, blockHeight int64) bool {
	if b.config.Enabled {
		// We poll for the correct interval on block commit to ensure that the correct interval
		// is always used, this is simpler than listening for the event and then having to handle
		// recovery scenarios.
		param, err := b.getNetworkParameter(ctx, netparams.SnapshotIntervalLength)
		if err != nil {
			if !errors.Is(err, sqlstore.ErrNoParameterFound) {
				b.log.Errorf("failed to get snapshot interval length network parameter:%w", err)
			}
		} else {
			blockInterval, err := strconv.ParseInt(param.Value, 10, 64)
			if err != nil {
				b.log.Errorf("failed to parse snapshot interval length network parameter:%w", err)
			} else {
				b.blockInterval = blockInterval

				// An interval less than 1000 when using a file source with no time between blocks results
				// in excessive snapshot data creation and should be avoided
				minBlockIntervalForZeroTimeBetweenBlock := int64(1000)
				if b.brokerConfig.UseEventFile &&
					b.brokerConfig.FileEventSourceConfig.TimeBetweenBlocks.Duration < time.Second {
					if blockInterval < minBlockIntervalForZeroTimeBetweenBlock {
						b.blockInterval = minBlockIntervalForZeroTimeBetweenBlock
					}
				}
			}
		}

		if b.snapshotRequiredAtBlockHeight(blockHeight) {
			fromHeight := GetFromHeight(blockHeight, b.blockInterval)
			_, err := b.CreateSnapshotAsync(ctx, chainID, fromHeight, blockHeight)
			if err != nil {
				b.log.Errorf("failed to create snapshot at height:%d, chain id:%s, error:%s", blockHeight, chainID, err)
			}
		}
	}

	return false
}

func (b *Service) Types() []events.Type {
	return []events.Type{events.NetworkParameterEvent}
}

func GetFromHeight(toHeight int64, snapshotInterval int64) int64 {
	fromHeight := int64(0)
	// if toHeight-snapshotInterval != 0 {
	fromHeight = toHeight - (snapshotInterval - 1)
	//}
	return fromHeight
}

// GetHistoryIncludingDatanodeState returns currentStateSnapshot of the youngest snapshot or nil if none is found. Return the contiguous
// history in oldest first order or nil if none is found.  If the datanode is populated with data, only snapshot data upto
// the datanodes current height will be returned.
func GetHistoryIncludingDatanodeState(datanodeOldestHistoryBlock *entities.Block, datanodeLastBlock *entities.Block, chainID string,
	currentStateData map[int64]CurrentStateSnapshot, histories []HistorySnapshot,
) (*CurrentStateSnapshot, []HistorySnapshot, error) {
	if datanodeOldestHistoryBlock != nil && datanodeLastBlock == nil {
		return nil, nil, errors.New("invalid arguments, datanode LastBlock cannot be nil if datanodeOldestHistoryBlock is not nil")
	}

	var currentStateSnapshot *CurrentStateSnapshot
	if datanodeLastBlock != nil {
		// Use the datanode's latest block as the current state snapshot
		currentStateSnapshot = &CurrentStateSnapshot{ChainID: chainID, Height: datanodeLastBlock.Height}
	} else {
		// Otherwise use the latest available current state snapshot
		currentStateSnapshot = getLatestCurrentStateSnapshot(currentStateData)
	}

	var contiguousHistory []HistorySnapshot
	if currentStateSnapshot != nil {
		var firstHistory *HistorySnapshot
		if datanodeOldestHistoryBlock != nil {
			// Use current datanode data as the first history
			firstHistory = &HistorySnapshot{
				ChainID:    chainID,
				HeightFrom: datanodeOldestHistoryBlock.Height,
				HeightTo:   datanodeLastBlock.Height,
			}
		} else {
			// If it exists use the history that corresponds to the current state snapshot
			firstHistory = getFirstHistoryForCurrentState(histories, currentStateSnapshot)
		}

		if firstHistory != nil {
			contiguousHistory = getContiguousHistoryFromFirstHistory(firstHistory, histories)
		}

		// Sort history oldest first
		sort.Slice(contiguousHistory, func(i, j int) bool {
			return contiguousHistory[i].HeightFrom < contiguousHistory[j].HeightFrom
		})
	}

	return currentStateSnapshot, contiguousHistory, nil
}

func getFirstHistoryForCurrentState(histories []HistorySnapshot, currentStateSnapshot *CurrentStateSnapshot) *HistorySnapshot {
	for _, history := range histories {
		if history.HeightTo == currentStateSnapshot.Height {
			return &history
		}
	}

	return nil
}

func getLatestCurrentStateSnapshot(currentStateSnapshots map[int64]CurrentStateSnapshot) *CurrentStateSnapshot {
	var currentStateSnapshot *CurrentStateSnapshot
	if len(currentStateSnapshots) > 0 {
		var sortedByHeight []CurrentStateSnapshot
		for _, csSnapshot := range currentStateSnapshots {
			sortedByHeight = append(sortedByHeight, csSnapshot)
		}

		sort.Slice(sortedByHeight, func(i, j int) bool {
			return sortedByHeight[i].Height > sortedByHeight[j].Height
		})

		currentStateSnapshot = &sortedByHeight[0]
	}
	return currentStateSnapshot
}

func getContiguousHistoryFromFirstHistory(firstHistory *HistorySnapshot, histories []HistorySnapshot) []HistorySnapshot {
	var contiguousHistory []HistorySnapshot
	toHeightToHistory := map[int64]HistorySnapshot{}
	for _, history := range histories {
		toHeightToHistory[history.HeightTo] = history
	}

	startHistory := *firstHistory
	contiguousHistory = append(contiguousHistory, startHistory)
	for {
		if history, ok := toHeightToHistory[startHistory.HeightFrom-1]; ok {
			contiguousHistory = append(contiguousHistory, history)
			startHistory = history
		} else {
			break
		}
	}
	return contiguousHistory
}

func (b *Service) snapshotRequiredAtBlockHeight(lastCommittedBlockHeight int64) bool {
	if b.blockInterval > 0 {
		return lastCommittedBlockHeight > 0 && lastCommittedBlockHeight%b.blockInterval == 0
	}

	return false
}
