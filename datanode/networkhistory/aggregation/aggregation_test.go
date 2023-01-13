package aggregation_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory/aggregation"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"

	"github.com/stretchr/testify/assert"
)

func TestGetContiguousHistory(t *testing.T) {
	var historySnapshots []store.SegmentIndexEntry

	histories := aggregation.GetHighestContiguousHistoryFromSegmentIndexEntry(historySnapshots)
	assert.Nil(t, histories)

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
	}

	histories = aggregation.GetHighestContiguousHistoryFromSegmentIndexEntry(historySnapshots)

	assert.Equal(t, 3, len(histories))
	assert.Equal(t, int64(3000), histories[2].HeightTo)
	assert.Equal(t, int64(2001), histories[2].HeightFrom)
	assert.Equal(t, int64(2000), histories[1].HeightTo)
	assert.Equal(t, int64(1001), histories[1].HeightFrom)
	assert.Equal(t, int64(1000), histories[0].HeightTo)
	assert.Equal(t, int64(0), histories[0].HeightFrom)

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
	}

	histories = aggregation.GetHighestContiguousHistoryFromSegmentIndexEntry(historySnapshots)

	assert.Equal(t, 2, len(histories))
	assert.Equal(t, int64(4000), histories[1].HeightTo)
	assert.Equal(t, int64(3001), histories[1].HeightFrom)
	assert.Equal(t, int64(3000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[0].HeightFrom)
}
