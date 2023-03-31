package networkhistory

import (
	"fmt"
	"sort"
)

type ContiguousHistory struct {
	HeightFrom          int64
	HeightTo            int64
	SegmentsOldestFirst []Segment
}

// GetContiguousHistory returns the contiguous history for the given block span, oldest first.
func GetContiguousHistory[T Segment](segments []T, fromHeight int64, toHeight int64) (*ContiguousHistory, error) {
	contiguousHistory := GetContiguousHistoryForSpan(GetContiguousHistories(segments), fromHeight, toHeight)
	if contiguousHistory == nil {
		return nil, fmt.Errorf("no contiguous history found for with history segments from height %d to %d", fromHeight, toHeight)
	}

	return contiguousHistory, nil
}

func truncateContiguousHistoryToSpan(contiguousHistory *ContiguousHistory, fromHeight int64, toHeight int64) {
	var truncatedSegments []Segment

	for _, segment := range contiguousHistory.SegmentsOldestFirst {
		if segment.GetFromHeight() >= fromHeight && segment.GetToHeight() <= toHeight {
			truncatedSegments = append(truncatedSegments, segment)
		}
	}

	contiguousHistory.HeightFrom = fromHeight
	contiguousHistory.HeightTo = toHeight

	contiguousHistory.SegmentsOldestFirst = truncatedSegments
}

func GetMostRecentContiguousHistory[T Segment](historySegments []T) *ContiguousHistory {
	contiguousHistories := GetContiguousHistories(historySegments)
	if len(contiguousHistories) == 0 {
		return nil
	}

	return contiguousHistories[len(contiguousHistories)-1]
}

func GetContiguousHistories[T Segment](historySegments []T) []*ContiguousHistory {
	sort.Slice(historySegments, func(i, j int) bool {
		return historySegments[i].GetFromHeight() < historySegments[j].GetFromHeight()
	})

	var contiguousHistories []*ContiguousHistory
	for _, segment := range historySegments {
		addedToContiguousHistory := false
		for _, contiguousHistory := range contiguousHistories {
			if segment.GetToHeight() == contiguousHistory.HeightFrom-1 {
				contiguousHistory.SegmentsOldestFirst = append(contiguousHistory.SegmentsOldestFirst, segment)
				contiguousHistory.HeightFrom = segment.GetFromHeight()
				addedToContiguousHistory = true
			}

			if segment.GetFromHeight() == contiguousHistory.HeightTo+1 {
				contiguousHistory.SegmentsOldestFirst = append(contiguousHistory.SegmentsOldestFirst, segment)
				contiguousHistory.HeightTo = segment.GetToHeight()
				addedToContiguousHistory = true
			}

			if addedToContiguousHistory {
				break
			}
		}

		if !addedToContiguousHistory {
			contiguousHistories = append(contiguousHistories, &ContiguousHistory{
				HeightFrom:          segment.GetFromHeight(),
				HeightTo:            segment.GetToHeight(),
				SegmentsOldestFirst: []Segment{segment},
			})
		}
	}

	return contiguousHistories
}

func GetContiguousHistoryForSpan(contiguousHistories []*ContiguousHistory, fromHeight int64, toHeight int64) *ContiguousHistory {
	for _, contiguousHistory := range contiguousHistories {
		if contiguousHistory.HeightFrom <= fromHeight && contiguousHistory.HeightTo >= toHeight {
			fromSegmentFound := false
			toSegmentFound := false
			for _, segment := range contiguousHistory.SegmentsOldestFirst {
				if segment.GetFromHeight() == fromHeight {
					fromSegmentFound = true
				}

				if segment.GetToHeight() == toHeight {
					toSegmentFound = true
				}
			}

			if fromSegmentFound && toSegmentFound {
				truncateContiguousHistoryToSpan(contiguousHistory, fromHeight, toHeight)
				return contiguousHistory
			}
		}
	}

	return nil
}
