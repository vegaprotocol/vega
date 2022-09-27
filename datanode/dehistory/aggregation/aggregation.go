package aggregation

import (
	"errors"
	"sort"

	"code.vegaprotocol.io/vega/datanode/dehistory/store"
	"code.vegaprotocol.io/vega/datanode/entities"
)

type AggregatedHistorySegment struct {
	HeightFrom              int64
	HeightTo                int64
	ChainID                 string
	FromCurrentDatanodeData bool
}

// GetContiguousHistoryIncludingDataNodeExistingData returns the contiguous history oldest segment first.
func GetContiguousHistoryIncludingDataNodeExistingData(histories []store.SegmentIndexEntry, datanodeOldestHistoryBlock *entities.Block,
	datanodeLastBlock *entities.Block,
) ([]AggregatedHistorySegment, error) {
	if datanodeOldestHistoryBlock == nil || datanodeLastBlock == nil {
		if datanodeOldestHistoryBlock != nil || datanodeLastBlock != nil {
			return nil, errors.New(" datanodeLastBlock and datanodeOldestHistoryBlock must both be either nil or not nil")
		}
	}

	if len(histories) == 0 {
		return nil, nil
	}

	chainID := histories[0].ChainID

	aggHistory := make([]AggregatedHistorySegment, 0, 10)
	for _, indexEntry := range histories {
		aggHistory = append(aggHistory, AggregatedHistorySegment{
			HeightFrom:              indexEntry.HeightFrom,
			HeightTo:                indexEntry.HeightTo,
			ChainID:                 indexEntry.ChainID,
			FromCurrentDatanodeData: false,
		})
	}

	dataNodeHistorySegment := getHistorySegmentForDataNodeExistingData(chainID, datanodeOldestHistoryBlock, datanodeLastBlock)

	var startFromHistorySegment AggregatedHistorySegment
	if dataNodeHistorySegment != nil {
		startFromHistorySegment = *dataNodeHistorySegment
	} else {
		startFromHistorySegment = getMostRecentHistorySegment(aggHistory)
	}

	contiguousHistory := getContiguousHistoryFromFirstHistorySegment(startFromHistorySegment, aggHistory)

	// Sort history oldest first
	sort.Slice(contiguousHistory, func(i, j int) bool {
		return contiguousHistory[i].HeightFrom < contiguousHistory[j].HeightFrom
	})

	return contiguousHistory, nil
}

func getHistorySegmentForDataNodeExistingData(chainID string, datanodeOldestHistoryBlock *entities.Block, datanodeLastBlock *entities.Block) *AggregatedHistorySegment {
	if datanodeOldestHistoryBlock == nil || datanodeLastBlock == nil {
		return nil
	}

	return &AggregatedHistorySegment{
		ChainID:                 chainID,
		HeightFrom:              datanodeOldestHistoryBlock.Height,
		HeightTo:                datanodeLastBlock.Height,
		FromCurrentDatanodeData: true,
	}
}

func getMostRecentHistorySegment(aggHistory []AggregatedHistorySegment) AggregatedHistorySegment {
	mostRecent := aggHistory[0]
	for i := 0; i < len(aggHistory); i++ {
		if aggHistory[i].HeightTo > mostRecent.HeightTo {
			mostRecent = aggHistory[i]
		}
	}
	return mostRecent
}

func getContiguousHistoryFromFirstHistorySegment(firstHistory AggregatedHistorySegment, histories []AggregatedHistorySegment) []AggregatedHistorySegment {
	var contiguousHistory []AggregatedHistorySegment
	toHeightToHistory := map[int64]AggregatedHistorySegment{}
	for _, history := range histories {
		toHeightToHistory[history.HeightTo] = history
	}

	startHistory := firstHistory
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
