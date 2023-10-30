// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package store_test

import (
	"sort"
	"testing"

	"code.vegaprotocol.io/vega/datanode/networkhistory/segment"
	"code.vegaprotocol.io/vega/datanode/networkhistory/store"
	"code.vegaprotocol.io/vega/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeSegment(heightFrom, heightTo int64) segment.Full {
	return segment.Full{
		MetaData: segment.MetaData{
			Base: segment.Base{
				HeightFrom: heightFrom,
				HeightTo:   heightTo,
			},
		},
	}
}

func TestOrderingOfEntriesOldestFirst(t *testing.T) {
	index, err := store.NewIndex(t.TempDir(), logging.NewTestLogger())
	require.NoError(t, err)
	defer func() { _ = index.Close() }()

	allEntries, err := index.ListAllEntriesOldestFirst()
	assert.Equal(t, 0, len(allEntries))
	require.NoError(t, err)

	var addedEntries []segment.Full

	numEntries := int64(100)
	for i := int64(0); i < (numEntries/2)-2; i++ {
		entry := makeSegment((i*1000)+1, (i+1)*1000)
		index.Add(entry)
		addedEntries = append(addedEntries, entry)

		entry = makeSegment((numEntries-i-1)*1000+1, (numEntries-i)*1000)
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
	assert.Equal(t, addedEntries, []segment.Full(allEntries))
}

func TestOrderingOfEntriesMostRecentFirst(t *testing.T) {
	index, err := store.NewIndex(t.TempDir(), logging.NewTestLogger())
	require.NoError(t, err)
	defer func() { _ = index.Close() }()

	allEntries, err := index.ListAllEntriesOldestFirst()
	assert.Equal(t, 0, len(allEntries))
	require.NoError(t, err)

	var addedEntries []segment.Full

	numEntries := int64(100)
	for i := int64(0); i < (numEntries/2)-2; i++ {
		entry := makeSegment((i*1000)+1, (i+1)*1000)
		index.Add(entry)
		addedEntries = append(addedEntries, entry)

		entry = makeSegment((numEntries-i-1)*1000+1, (numEntries-i)*1000)
		index.Add(entry)
		addedEntries = append(addedEntries, entry)
	}

	allEntries, err = index.ListAllEntriesMostRecentFirst()

	require.NoError(t, err)

	// Sort most recent first
	sort.Slice(addedEntries, func(i, j int) bool {
		return addedEntries[i].HeightFrom > addedEntries[j].HeightFrom
	})

	assert.Equal(t, len(addedEntries), len(allEntries))
	assert.Equal(t, addedEntries, []segment.Full(allEntries))
}

func TestGetHighestHeightEntry(t *testing.T) {
	index, err := store.NewIndex(t.TempDir(), logging.NewTestLogger())
	require.NoError(t, err)
	defer func() { _ = index.Close() }()

	entry, err := index.GetHighestBlockHeightEntry()
	require.Error(t, store.ErrIndexEntryNotFound, err)
	assert.Equal(t, int64(0), entry.HeightTo)
	assert.Equal(t, int64(0), entry.HeightFrom)

	index.Add(makeSegment(2001, 3000))
	index.Add(makeSegment(1, 1000))
	index.Add(makeSegment(1001, 2000))

	entry, err = index.GetHighestBlockHeightEntry()
	require.NoError(t, err)
	assert.Equal(t, int64(3000), entry.HeightTo)
}

func TestGetEntryByHeight(t *testing.T) {
	index, err := store.NewIndex(t.TempDir(), logging.NewTestLogger())
	require.NoError(t, err)
	defer func() { _ = index.Close() }()

	index.Add(makeSegment(2001, 3000))
	index.Add(makeSegment(1, 1000))
	index.Add(makeSegment(1001, 2000))

	entry, err := index.Get(3000)
	require.NoError(t, err)
	assert.Equal(t, int64(3000), entry.HeightTo)

	entry, err = index.Get(4300)
	require.Error(t, store.ErrIndexEntryNotFound, err)
	assert.Equal(t, int64(0), entry.HeightTo)
	assert.Equal(t, int64(0), entry.HeightFrom)
}
