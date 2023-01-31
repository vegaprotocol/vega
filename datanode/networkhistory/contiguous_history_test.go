package networkhistory_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory"

	"code.vegaprotocol.io/vega/datanode/networkhistory/store"

	"github.com/stretchr/testify/assert"
)

func TestGetContiguousHistory(t *testing.T) {
	var historySnapshots []store.SegmentIndexEntry

	_, err := networkhistory.GetContiguousHistory(historySnapshots, 0, 0)
	assert.NotNil(t, err)

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
	}

	contiguousHistory, err := networkhistory.GetContiguousHistory(historySnapshots, 0, 3000)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(contiguousHistory.SegmentsOldestFirst))
	assert.Equal(t, int64(0), contiguousHistory.HeightFrom)
	assert.Equal(t, int64(3000), contiguousHistory.HeightTo)

	assert.Equal(t, int64(3000), contiguousHistory.SegmentsOldestFirst[2].GetToHeight())
	assert.Equal(t, int64(2001), contiguousHistory.SegmentsOldestFirst[2].GetFromHeight())
	assert.Equal(t, int64(2000), contiguousHistory.SegmentsOldestFirst[1].GetToHeight())
	assert.Equal(t, int64(1001), contiguousHistory.SegmentsOldestFirst[1].GetFromHeight())
	assert.Equal(t, int64(1000), contiguousHistory.SegmentsOldestFirst[0].GetToHeight())
	assert.Equal(t, int64(0), contiguousHistory.SegmentsOldestFirst[0].GetFromHeight())

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
	}

	contiguousHistory, err = networkhistory.GetContiguousHistory(historySnapshots, 0, 4000)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(contiguousHistory.SegmentsOldestFirst))
	assert.Equal(t, int64(0), contiguousHistory.HeightFrom)
	assert.Equal(t, int64(4000), contiguousHistory.HeightTo)

	assert.Equal(t, int64(4000), contiguousHistory.SegmentsOldestFirst[3].GetToHeight())
	assert.Equal(t, int64(3001), contiguousHistory.SegmentsOldestFirst[3].GetFromHeight())
	assert.Equal(t, int64(3000), contiguousHistory.SegmentsOldestFirst[2].GetToHeight())
	assert.Equal(t, int64(2001), contiguousHistory.SegmentsOldestFirst[2].GetFromHeight())
	assert.Equal(t, int64(2000), contiguousHistory.SegmentsOldestFirst[1].GetToHeight())
	assert.Equal(t, int64(1001), contiguousHistory.SegmentsOldestFirst[1].GetFromHeight())
	assert.Equal(t, int64(1000), contiguousHistory.SegmentsOldestFirst[0].GetToHeight())
	assert.Equal(t, int64(0), contiguousHistory.SegmentsOldestFirst[0].GetFromHeight())
}

func TestRequestingContiguousHistoryAcrossNonContiguousRangeFails(t *testing.T) {
	historySnapshots := []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
	}

	_, err := networkhistory.GetContiguousHistory(historySnapshots, 0,
		4000)
	assert.NotNil(t, err)
}

func TestGetContiguousHistoryBetweenHeights(t *testing.T) {
	historySnapshots := []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 4001, HeightTo: 5000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 5001, HeightTo: 6000}},
	}

	contiguousHistory, err := networkhistory.GetContiguousHistory(historySnapshots, 2001, 5000)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(contiguousHistory.SegmentsOldestFirst))
	assert.Equal(t, int64(2001), contiguousHistory.HeightFrom)
	assert.Equal(t, int64(5000), contiguousHistory.HeightTo)

	assert.Equal(t, int64(5000), contiguousHistory.SegmentsOldestFirst[2].GetToHeight())
	assert.Equal(t, int64(4001), contiguousHistory.SegmentsOldestFirst[2].GetFromHeight())
	assert.Equal(t, int64(4000), contiguousHistory.SegmentsOldestFirst[1].GetToHeight())
	assert.Equal(t, int64(3001), contiguousHistory.SegmentsOldestFirst[1].GetFromHeight())
	assert.Equal(t, int64(3000), contiguousHistory.SegmentsOldestFirst[0].GetToHeight())
	assert.Equal(t, int64(2001), contiguousHistory.SegmentsOldestFirst[0].GetFromHeight())
}

func TestGetContiguousHistoryWithIncorrectFromAndToHeights(t *testing.T) {
	historySnapshots := []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
	}

	_, err := networkhistory.GetContiguousHistory(historySnapshots, 1000, 3000)
	assert.NotNil(t, err)

	_, err = networkhistory.GetContiguousHistory(historySnapshots, 0,
		2001)
	assert.NotNil(t, err)
}

func TestGetContiguousHistories(t *testing.T) {
	historySnapshots := []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 6001, HeightTo: 7000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 11001, HeightTo: 12000}},

		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},

		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 16001, HeightTo: 17000}},

		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 5001, HeightTo: 6000}},

		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 10001, HeightTo: 11000}},

		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 12001, HeightTo: 13000}},

		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 7001, HeightTo: 8000}},
	}

	contiguousHistories := networkhistory.GetContiguousHistories(historySnapshots)
	assert.Equal(t, 4, len(contiguousHistories))

	assert.Equal(t, 3, len(contiguousHistories[0].SegmentsOldestFirst))
	assert.Equal(t, int64(0), contiguousHistories[0].HeightFrom)
	assert.Equal(t, int64(3000), contiguousHistories[0].HeightTo)
	assert.Equal(t, int64(0), contiguousHistories[0].SegmentsOldestFirst[0].GetFromHeight())
	assert.Equal(t, int64(1000), contiguousHistories[0].SegmentsOldestFirst[0].GetToHeight())
	assert.Equal(t, int64(1001), contiguousHistories[0].SegmentsOldestFirst[1].GetFromHeight())
	assert.Equal(t, int64(2000), contiguousHistories[0].SegmentsOldestFirst[1].GetToHeight())
	assert.Equal(t, int64(2001), contiguousHistories[0].SegmentsOldestFirst[2].GetFromHeight())
	assert.Equal(t, int64(3000), contiguousHistories[0].SegmentsOldestFirst[2].GetToHeight())

	assert.Equal(t, 3, len(contiguousHistories[1].SegmentsOldestFirst))
	assert.Equal(t, int64(5001), contiguousHistories[1].HeightFrom)
	assert.Equal(t, int64(8000), contiguousHistories[1].HeightTo)
	assert.Equal(t, int64(5001), contiguousHistories[1].SegmentsOldestFirst[0].GetFromHeight())
	assert.Equal(t, int64(6000), contiguousHistories[1].SegmentsOldestFirst[0].GetToHeight())
	assert.Equal(t, int64(6001), contiguousHistories[1].SegmentsOldestFirst[1].GetFromHeight())
	assert.Equal(t, int64(7000), contiguousHistories[1].SegmentsOldestFirst[1].GetToHeight())
	assert.Equal(t, int64(7001), contiguousHistories[1].SegmentsOldestFirst[2].GetFromHeight())
	assert.Equal(t, int64(8000), contiguousHistories[1].SegmentsOldestFirst[2].GetToHeight())

	assert.Equal(t, 3, len(contiguousHistories[2].SegmentsOldestFirst))
	assert.Equal(t, int64(10001), contiguousHistories[2].HeightFrom)
	assert.Equal(t, int64(13000), contiguousHistories[2].HeightTo)
	assert.Equal(t, int64(10001), contiguousHistories[2].SegmentsOldestFirst[0].GetFromHeight())
	assert.Equal(t, int64(11000), contiguousHistories[2].SegmentsOldestFirst[0].GetToHeight())
	assert.Equal(t, int64(11001), contiguousHistories[2].SegmentsOldestFirst[1].GetFromHeight())
	assert.Equal(t, int64(12000), contiguousHistories[2].SegmentsOldestFirst[1].GetToHeight())
	assert.Equal(t, int64(12001), contiguousHistories[2].SegmentsOldestFirst[2].GetFromHeight())
	assert.Equal(t, int64(13000), contiguousHistories[2].SegmentsOldestFirst[2].GetToHeight())

	assert.Equal(t, 1, len(contiguousHistories[3].SegmentsOldestFirst))
	assert.Equal(t, int64(16001), contiguousHistories[3].HeightFrom)
	assert.Equal(t, int64(17000), contiguousHistories[3].HeightTo)
	assert.Equal(t, int64(16001), contiguousHistories[3].SegmentsOldestFirst[0].GetFromHeight())
	assert.Equal(t, int64(17000), contiguousHistories[3].SegmentsOldestFirst[0].GetToHeight())
}
