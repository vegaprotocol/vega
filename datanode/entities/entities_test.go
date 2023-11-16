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

package entities_test

import (
	"crypto/sha256"
	"encoding/hex"
	"math/rand"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func generateTxHash() entities.TxHash {
	randomString := strconv.FormatInt(rand.Int63(), 10)
	hash := sha256.Sum256([]byte(randomString))
	return entities.TxHash(hex.EncodeToString(hash[:]))
}

func TestPageEntities(t *testing.T) {
	t.Run("Number of results is 2 more then the page limit", func(t *testing.T) {
		t.Run("The results are returned in order and we have next and previous when we are moving forward", testPageEntitiesForwardHasNextAndPrevious)
		t.Run("The results are returned in order and we have next and previous when we are moving backward", testPageEntitiesBackwardHasNextAndPrevious)
	})

	t.Run("Number of results is 1 more than the page limit", func(t *testing.T) {
		t.Run("When moving forward, we have a previous page, but no next page", testPagedEntitiesForwardHasPreviousButNoNext)
		t.Run("When moving backward, we have a next page, but no previous page", testPagedEntitiesBackwardHasNextButNoPrevious)
	})

	t.Run("Number of results is equal to the page limit", func(t *testing.T) {
		t.Run("When moving forward, we have no previous or next page", testPagedEntitiesForwardNoNextOrPreviousEqualLimit)
		t.Run("When moving backward, we have no previous or next page", testPagedEntitiesBackwardNoNextOrPreviousEqualLimit)
	})

	t.Run("Number of results is less than the page limit", func(t *testing.T) {
		t.Run("When moving forward, we have no previous or next page", testPagedEntitiesForwardNoNextOrPreviousLessThanLimit)
		t.Run("When moving backward, we have no previous or next page", testPagedEntitiesBackwardNoNextOrPreviousLessThanLimit)
	})
}

func testPageEntitiesForwardHasNextAndPrevious(t *testing.T) {
	trades := getTradesForward(t, 0, 0) // 0, 0 return all entries
	first := int32(5)
	afterTs := time.Unix(0, 1000000000000).UTC()
	after := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: afterTs,
	}.String()).Encode()
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       &first,
			After:       &after,
			Last:        nil,
			Before:      nil,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000001000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000005000000).UTC(),
	}.String()).Encode()

	wantPaged := trades[1:6]
	wantInfo := entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func testPageEntitiesBackwardHasNextAndPrevious(t *testing.T) {
	trades := getTradesBackward(t, 0, 0) // 0, 0 return all entries
	last := int32(5)
	beforeTs := time.Unix(0, 1000006000000).UTC()
	before := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: beforeTs,
	}.String()).Encode()
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       nil,
			After:       nil,
			Last:        &last,
			Before:      &before,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000001000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000005000000).UTC(),
	}.String()).Encode()

	wantPaged := getTradesForward(t, 1, 6)
	wantInfo := entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func testPagedEntitiesForwardHasPreviousButNoNext(t *testing.T) {
	trades := getTradesForward(t, 1, 0) // 0, 0 return all entries
	first := int32(5)
	afterTs := time.Unix(0, 1000001000000).UTC()
	after := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: afterTs,
	}.String()).Encode()
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       &first,
			After:       &after,
			Last:        nil,
			Before:      nil,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000002000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000006000000).UTC(),
	}.String()).Encode()

	wantPaged := trades[1:6]
	wantInfo := entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func testPagedEntitiesBackwardHasNextButNoPrevious(t *testing.T) {
	trades := getTradesBackward(t, 1, 0) // 0, 0 return all entries
	last := int32(5)
	beforeTs := time.Unix(0, 1000005000000).UTC()
	before := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: beforeTs,
	}.String()).Encode()
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       nil,
			After:       nil,
			Last:        &last,
			Before:      &before,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000000000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000004000000).UTC(),
	}.String()).Encode()

	wantPaged := getTradesForward(t, 0, 5)
	wantInfo := entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func testPagedEntitiesForwardNoNextOrPreviousEqualLimit(t *testing.T) {
	trades := getTradesForward(t, 0, 5) // 0, 0 return all entries
	first := int32(5)
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       &first,
			After:       nil,
			Last:        nil,
			Before:      nil,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000000000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000004000000).UTC(),
	}.String()).Encode()

	wantPaged := trades
	wantInfo := entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func testPagedEntitiesBackwardNoNextOrPreviousEqualLimit(t *testing.T) {
	trades := getTradesBackward(t, 0, 5) // 0, 0 return all entries
	last := int32(5)
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       nil,
			After:       nil,
			Last:        &last,
			Before:      nil,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000002000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000006000000).UTC(),
	}.String()).Encode()

	wantPaged := getTradesForward(t, 2, 0)
	wantInfo := entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func testPagedEntitiesForwardNoNextOrPreviousLessThanLimit(t *testing.T) {
	trades := getTradesForward(t, 0, 3) // 0, 0 return all entries
	first := int32(5)
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       &first,
			After:       nil,
			Last:        nil,
			Before:      nil,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000000000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000002000000).UTC(),
	}.String()).Encode()

	wantPaged := trades
	wantInfo := entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func testPagedEntitiesBackwardNoNextOrPreviousLessThanLimit(t *testing.T) {
	trades := getTradesBackward(t, 0, 3) // 0, 0 return all entries
	last := int32(5)
	newestFirst := false
	cursor, err := entities.CursorPaginationFromProto(
		&v2.Pagination{
			First:       nil,
			After:       nil,
			Last:        &last,
			Before:      nil,
			NewestFirst: &newestFirst,
		})
	require.NoError(t, err)
	gotPaged, gotInfo := entities.PageEntities[*v2.TradeEdge](trades, cursor)

	startCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000004000000).UTC(),
	}.String()).Encode()

	endCursor := entities.NewCursor(entities.TradeCursor{
		SyntheticTime: time.Unix(0, 1000006000000).UTC(),
	}.String()).Encode()

	wantPaged := getTradesForward(t, 4, 0)
	wantInfo := entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     startCursor,
		EndCursor:       endCursor,
	}
	assert.Equal(t, wantPaged, gotPaged)
	assert.Equal(t, wantInfo, gotInfo)
}

func getTradesForward(t *testing.T, start, end int) []entities.Trade {
	t.Helper()
	trades := []entities.Trade{
		{
			SyntheticTime: time.Unix(0, 1000000000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000001000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000002000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000003000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000004000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000005000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000006000000).UTC(),
		},
	}

	if end == 0 {
		end = len(trades)
	}

	if end < start {
		end = start
	}

	return trades[start:end]
}

func getTradesBackward(t *testing.T, start, end int) []entities.Trade {
	t.Helper()
	trades := []entities.Trade{
		{
			SyntheticTime: time.Unix(0, 1000006000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000005000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000004000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000003000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000002000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000001000000).UTC(),
		},
		{
			SyntheticTime: time.Unix(0, 1000000000000).UTC(),
		},
	}

	if end == 0 {
		end = len(trades)
	}

	if end < start {
		end = start
	}

	return trades[start:end]
}
