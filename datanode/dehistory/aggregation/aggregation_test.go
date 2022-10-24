package aggregation_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/dehistory/aggregation"
	"code.vegaprotocol.io/vega/datanode/dehistory/store"

	"code.vegaprotocol.io/vega/datanode/entities"
	"github.com/stretchr/testify/assert"
)

func TestGetHistoryIncludingDatanodeStateWhenDatanodeHasData(t *testing.T) {
	var datanodeOldestHistoryBlock *entities.Block
	var datanodeLastBlock *entities.Block

	datanodeOldestHistoryBlock = &entities.Block{Height: 0}
	datanodeLastBlock = &entities.Block{Height: 5000}

	historySnapshots := []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
	}

	histories, _ := aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)
	assert.Equal(t, len(histories), 1)
	assert.Equal(t, int64(0), histories[0].HeightFrom)
	assert.Equal(t, int64(5000), histories[0].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 2001}
	datanodeLastBlock = &entities.Block{Height: 5000}

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 4001, HeightTo: 5000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 5001, HeightTo: 6000}},
	}

	histories, _ = aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)

	assert.Equal(t, len(histories), 2)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(5000), histories[1].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 4001}
	datanodeLastBlock = &entities.Block{Height: 5000}

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 4001, HeightTo: 5000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 5001, HeightTo: 6000}},
	}

	histories, _ = aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)

	assert.Equal(t, len(histories), 4)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(3000), histories[1].HeightTo)
	assert.Equal(t, int64(3001), histories[2].HeightFrom)
	assert.Equal(t, int64(4000), histories[2].HeightTo)
	assert.Equal(t, int64(4001), histories[3].HeightFrom)
	assert.Equal(t, int64(5000), histories[3].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 4001}
	datanodeLastBlock = &entities.Block{Height: 5050}

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 4001, HeightTo: 5000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 5001, HeightTo: 6000}},
	}

	histories, _ = aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)

	assert.Equal(t, len(histories), 4)
	assert.Equal(t, int64(1001), histories[0].HeightFrom)
	assert.Equal(t, int64(2000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[1].HeightFrom)
	assert.Equal(t, int64(3000), histories[1].HeightTo)
	assert.Equal(t, int64(3001), histories[2].HeightFrom)
	assert.Equal(t, int64(4000), histories[2].HeightTo)
	assert.Equal(t, int64(4001), histories[3].HeightFrom)
	assert.Equal(t, int64(5050), histories[3].HeightTo)

	datanodeOldestHistoryBlock = &entities.Block{Height: 0}
	datanodeLastBlock = &entities.Block{Height: 5050}

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 3001, HeightTo: 4000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 4001, HeightTo: 5000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 5001, HeightTo: 6000}},
	}

	histories, _ = aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)

	assert.Equal(t, len(histories), 1)
	assert.Equal(t, int64(0), histories[0].HeightFrom)
	assert.Equal(t, int64(5050), histories[0].HeightTo)
}

func TestGetHistoryIncludingDatanodeStatWhenDatanodeIsEmpty(t *testing.T) {
	var datanodeOldestHistoryBlock *entities.Block
	var datanodeLastBlock *entities.Block

	var historySnapshots []store.SegmentIndexEntry

	histories, _ := aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)
	assert.Nil(t, histories)

	historySnapshots = []store.SegmentIndexEntry{
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 0, HeightTo: 1000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 1001, HeightTo: 2000}},
		{SegmentMetaData: store.SegmentMetaData{HeightFrom: 2001, HeightTo: 3000}},
	}

	histories, _ = aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)

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

	histories, _ = aggregation.GetContiguousHistoryIncludingDataNodeExistingData(historySnapshots, datanodeOldestHistoryBlock, datanodeLastBlock)

	assert.Equal(t, 2, len(histories))
	assert.Equal(t, int64(4000), histories[1].HeightTo)
	assert.Equal(t, int64(3001), histories[1].HeightFrom)
	assert.Equal(t, int64(3000), histories[0].HeightTo)
	assert.Equal(t, int64(2001), histories[0].HeightFrom)
}
