package segment_test

import (
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"

	"github.com/stretchr/testify/assert"
)

func TestContiguousHistory(t *testing.T) {
	var segments segment.Segments[segment.Base]

	_, err := segments.ContiguousHistoryInRange(0, 0)
	assert.NotNil(t, err)

	segments = []segment.Base{
		{HeightFrom: 0, HeightTo: 1000},
		{HeightFrom: 1001, HeightTo: 2000},
		{HeightFrom: 2001, HeightTo: 3000},
	}

	ch, err := segments.ContiguousHistoryInRange(0, 3000)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(ch.Segments))
	assert.Equal(t, int64(0), ch.HeightFrom)
	assert.Equal(t, int64(3000), ch.HeightTo)

	assert.Equal(t, int64(3000), ch.Segments[2].GetToHeight())
	assert.Equal(t, int64(2001), ch.Segments[2].GetFromHeight())
	assert.Equal(t, int64(2000), ch.Segments[1].GetToHeight())
	assert.Equal(t, int64(1001), ch.Segments[1].GetFromHeight())
	assert.Equal(t, int64(1000), ch.Segments[0].GetToHeight())
	assert.Equal(t, int64(0), ch.Segments[0].GetFromHeight())

	segments = []segment.Base{
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 0, HeightTo: 1000},
		{HeightFrom: 3001, HeightTo: 4000},
		{HeightFrom: 1001, HeightTo: 2000},
	}

	ch, err = segments.ContiguousHistoryInRange(0, 4000)
	assert.NoError(t, err)

	assert.Equal(t, 4, len(ch.Segments))
	assert.Equal(t, int64(0), ch.HeightFrom)
	assert.Equal(t, int64(4000), ch.HeightTo)

	assert.Equal(t, int64(4000), ch.Segments[3].GetToHeight())
	assert.Equal(t, int64(3001), ch.Segments[3].GetFromHeight())
	assert.Equal(t, int64(3000), ch.Segments[2].GetToHeight())
	assert.Equal(t, int64(2001), ch.Segments[2].GetFromHeight())
	assert.Equal(t, int64(2000), ch.Segments[1].GetToHeight())
	assert.Equal(t, int64(1001), ch.Segments[1].GetFromHeight())
	assert.Equal(t, int64(1000), ch.Segments[0].GetToHeight())
	assert.Equal(t, int64(0), ch.Segments[0].GetFromHeight())
}

func TestRequestingContiguousHistoryAcrossNonContiguousRangeFails(t *testing.T) {
	segments := segment.Segments[segment.Base]{
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 0, HeightTo: 1000},
		{HeightFrom: 3001, HeightTo: 4000},
	}

	_, err := segments.ContiguousHistoryInRange(0, 4000)
	assert.NotNil(t, err)
}

func TestContiguousHistoryInRange(t *testing.T) {
	segments := segment.Segments[segment.Base]{
		{HeightFrom: 0, HeightTo: 1000},
		{HeightFrom: 1001, HeightTo: 2000},
		{HeightFrom: 3001, HeightTo: 4000},
		{HeightFrom: 4001, HeightTo: 5000},
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 5001, HeightTo: 6000},
	}

	ch, err := segments.ContiguousHistoryInRange(2001, 5000)
	assert.NoError(t, err)

	assert.Equal(t, 3, len(ch.Segments))
	assert.Equal(t, int64(2001), ch.HeightFrom)
	assert.Equal(t, int64(5000), ch.HeightTo)

	assert.Equal(t, int64(5000), ch.Segments[2].GetToHeight())
	assert.Equal(t, int64(4001), ch.Segments[2].GetFromHeight())
	assert.Equal(t, int64(4000), ch.Segments[1].GetToHeight())
	assert.Equal(t, int64(3001), ch.Segments[1].GetFromHeight())
	assert.Equal(t, int64(3000), ch.Segments[0].GetToHeight())
	assert.Equal(t, int64(2001), ch.Segments[0].GetFromHeight())
}

func TestContiguousHistoryInRangeWithIncorrectFromAndToHeights(t *testing.T) {
	segments := segment.Segments[segment.Base]{
		{HeightFrom: 0, HeightTo: 1000},
		{HeightFrom: 1001, HeightTo: 2000},
		{HeightFrom: 2001, HeightTo: 3000},
	}

	_, err := segments.ContiguousHistoryInRange(1000, 3000)
	assert.NotNil(t, err)

	_, err = segments.ContiguousHistoryInRange(0, 2001)
	assert.NotNil(t, err)
}

func TestAllContigousHistories(t *testing.T) {
	segments := segment.Segments[segment.Base]{
		{HeightFrom: 6001, HeightTo: 7000},
		{HeightFrom: 2001, HeightTo: 3000},
		{HeightFrom: 11001, HeightTo: 12000},

		{HeightFrom: 0, HeightTo: 1000},
		{HeightFrom: 1001, HeightTo: 2000},

		{HeightFrom: 16001, HeightTo: 17000},

		{HeightFrom: 5001, HeightTo: 6000},

		{HeightFrom: 10001, HeightTo: 11000},

		{HeightFrom: 12001, HeightTo: 13000},

		{HeightFrom: 7001, HeightTo: 8000},
	}

	contiguousHistories := segments.AllContigousHistories()
	assert.Equal(t, 4, len(contiguousHistories))

	assert.Equal(t, 3, len(contiguousHistories[0].Segments))
	assert.Equal(t, int64(0), contiguousHistories[0].HeightFrom)
	assert.Equal(t, int64(3000), contiguousHistories[0].HeightTo)
	assert.Equal(t, int64(0), contiguousHistories[0].Segments[0].GetFromHeight())
	assert.Equal(t, int64(1000), contiguousHistories[0].Segments[0].GetToHeight())
	assert.Equal(t, int64(1001), contiguousHistories[0].Segments[1].GetFromHeight())
	assert.Equal(t, int64(2000), contiguousHistories[0].Segments[1].GetToHeight())
	assert.Equal(t, int64(2001), contiguousHistories[0].Segments[2].GetFromHeight())
	assert.Equal(t, int64(3000), contiguousHistories[0].Segments[2].GetToHeight())

	assert.Equal(t, 3, len(contiguousHistories[1].Segments))
	assert.Equal(t, int64(5001), contiguousHistories[1].HeightFrom)
	assert.Equal(t, int64(8000), contiguousHistories[1].HeightTo)
	assert.Equal(t, int64(5001), contiguousHistories[1].Segments[0].GetFromHeight())
	assert.Equal(t, int64(6000), contiguousHistories[1].Segments[0].GetToHeight())
	assert.Equal(t, int64(6001), contiguousHistories[1].Segments[1].GetFromHeight())
	assert.Equal(t, int64(7000), contiguousHistories[1].Segments[1].GetToHeight())
	assert.Equal(t, int64(7001), contiguousHistories[1].Segments[2].GetFromHeight())
	assert.Equal(t, int64(8000), contiguousHistories[1].Segments[2].GetToHeight())

	assert.Equal(t, 3, len(contiguousHistories[2].Segments))
	assert.Equal(t, int64(10001), contiguousHistories[2].HeightFrom)
	assert.Equal(t, int64(13000), contiguousHistories[2].HeightTo)
	assert.Equal(t, int64(10001), contiguousHistories[2].Segments[0].GetFromHeight())
	assert.Equal(t, int64(11000), contiguousHistories[2].Segments[0].GetToHeight())
	assert.Equal(t, int64(11001), contiguousHistories[2].Segments[1].GetFromHeight())
	assert.Equal(t, int64(12000), contiguousHistories[2].Segments[1].GetToHeight())
	assert.Equal(t, int64(12001), contiguousHistories[2].Segments[2].GetFromHeight())
	assert.Equal(t, int64(13000), contiguousHistories[2].Segments[2].GetToHeight())

	assert.Equal(t, 1, len(contiguousHistories[3].Segments))
	assert.Equal(t, int64(16001), contiguousHistories[3].HeightFrom)
	assert.Equal(t, int64(17000), contiguousHistories[3].HeightTo)
	assert.Equal(t, int64(16001), contiguousHistories[3].Segments[0].GetFromHeight())
	assert.Equal(t, int64(17000), contiguousHistories[3].Segments[0].GetToHeight())
}
