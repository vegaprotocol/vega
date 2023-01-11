package aggregation

import (
	"sort"

	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
)

type AggregatedHistorySegment struct {
	HeightFrom int64
	HeightTo   int64
	ChainID    string
}

// GetHighestContiguousHistoryFromSegmentIndexEntry returns the contiguous history oldest segment first.
func GetHighestContiguousHistoryFromSegmentIndexEntry(histories []store.SegmentIndexEntry) []AggregatedHistorySegment {
	aggHistory := make([]AggregatedHistorySegment, 0, 10)
	for _, indexEntry := range histories {
		aggHistory = append(aggHistory, AggregatedHistorySegment{
			HeightFrom: indexEntry.HeightFrom,
			HeightTo:   indexEntry.HeightTo,
			ChainID:    indexEntry.ChainID,
		})
	}

	return GetHighestContiguousHistory(aggHistory)
}

func GetHighestContiguousHistory(aggHistory []AggregatedHistorySegment) []AggregatedHistorySegment {
	if len(aggHistory) == 0 {
		return nil
	}

	startFromHistorySegment := getMostRecentHistorySegment(aggHistory)

	contiguousHistory := getContiguousHistoryFromFirstHistorySegment(startFromHistorySegment, aggHistory)

	// Sort history oldest first
	sort.Slice(contiguousHistory, func(i, j int) bool {
		return contiguousHistory[i].HeightFrom < contiguousHistory[j].HeightFrom
	})

	return contiguousHistory
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
