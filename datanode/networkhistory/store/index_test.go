package store_test

import (
	"sort"
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderingOfEntries(t *testing.T) {
	index, err := store.NewIndex(t.TempDir())
	require.NoError(t, err)
	defer func() { _ = index.Close() }()

	allEntries, err := index.ListAllEntriesOldestFirst()
	assert.Equal(t, 0, len(allEntries))
	require.NoError(t, err)

	var addedEntries []store.SegmentIndexEntry

	numEntries := int64(100)
	for i := int64(0); i < (numEntries/2)-2; i++ {
		entry := store.SegmentIndexEntry{
			SegmentMetaData: store.SegmentMetaData{
				HeightFrom: (i * 1000) + 1,
				HeightTo:   (i + 1) * 1000,
			},
		}

		index.Add(entry)
		addedEntries = append(addedEntries, entry)

		entry = store.SegmentIndexEntry{
			SegmentMetaData: store.SegmentMetaData{
				HeightFrom: (numEntries-i-1)*1000 + 1,
				HeightTo:   (numEntries - i) * 1000,
			},
		}

		index.Add(entry)
		addedEntries = append(addedEntries, entry)
	}

	allEntries, err = index.ListAllEntriesOldestFirst()

	require.NoError(t, err)

	// Sort oldest first
	sort.Slice(addedEntries, func(i, j int) bool {
		return addedEntries[i].HeightFrom < addedEntries[j].HeightFrom
	})

	assert.Equal(t, len(addedEntries), len(allEntries))
	assert.Equal(t, addedEntries, allEntries)
}

func TestGetHighestHeightEntry(t *testing.T) {
	index, err := store.NewIndex(t.TempDir())
	require.NoError(t, err)
	defer func() { _ = index.Close() }()

	entry, err := index.GetHighestBlockHeightEntry()
	require.Error(t, store.ErrIndexEntryNotFound, err)
	assert.Equal(t, int64(0), entry.HeightTo)
	assert.Equal(t, int64(0), entry.HeightFrom)

	index.Add(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom: 2001,
			HeightTo:   3000,
		},
	})

	index.Add(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom: 1,
			HeightTo:   1000,
		},
	})

	index.Add(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom: 1001,
			HeightTo:   2000,
		},
	})

	entry, err = index.GetHighestBlockHeightEntry()
	require.NoError(t, err)
	assert.Equal(t, int64(3000), entry.HeightTo)
}

func TestGetEntryByHeight(t *testing.T) {
	index, err := store.NewIndex(t.TempDir())
	require.NoError(t, err)
	defer func() { _ = index.Close() }()

	index.Add(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom: 2001,
			HeightTo:   3000,
		},
	})

	index.Add(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom: 1,
			HeightTo:   1000,
		},
	})

	index.Add(store.SegmentIndexEntry{
		SegmentMetaData: store.SegmentMetaData{
			HeightFrom: 1001,
			HeightTo:   2000,
		},
	})

	entry, err := index.Get(3000)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), entry.HeightTo)

	entry, err = index.Get(4300)
	require.Error(t, store.ErrIndexEntryNotFound, err)
	assert.Equal(t, int64(0), entry.HeightTo)
	assert.Equal(t, int64(0), entry.HeightFrom)
}
