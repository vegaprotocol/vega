// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlstore_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestPosition(t *testing.T,
	ctx context.Context,
	ps *sqlstore.Positions,
	market entities.Market,
	party entities.Party,
	volume int64,
	block entities.Block,
	txHash entities.TxHash,
) entities.Position {
	t.Helper()
	pos := entities.NewEmptyPosition(market.ID, party.ID)
	pos.OpenVolume = volume
	pos.PendingOpenVolume = volume
	pos.VegaTime = block.VegaTime
	pos.RealisedPnl = decimal.New(0, 0)
	pos.PendingRealisedPnl = decimal.New(0, 0)
	pos.UnrealisedPnl = decimal.New(0, 0)
	pos.PendingUnrealisedPnl = decimal.New(0, 0)
	pos.AverageEntryPrice = decimal.New(0, 0)
	pos.PendingAverageEntryPrice = decimal.New(0, 0)
	pos.AverageEntryMarketPrice = decimal.New(0, 0)
	pos.PendingAverageEntryMarketPrice = decimal.New(0, 0)
	pos.Adjustment = decimal.New(0, 0)
	pos.Loss = decimal.New(0, 0)
	pos.TxHash = txHash
	err := ps.Add(ctx, pos)
	require.NoError(t, err)
	return pos
}

func positionLessThan(x, y entities.Position) bool {
	if x.MarketID != y.MarketID {
		return x.MarketID.String() < y.MarketID.String()
	}
	return x.PartyID.String() < y.PartyID.String()
}

func assertPositionsMatch(t *testing.T, expected, actual []entities.Position) {
	t.Helper()
	sortPositions := cmpopts.SortSlices(positionLessThan)
	assert.Empty(t, cmp.Diff(actual, expected, sortPositions))
}

func TestPosition(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	qs := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	block1 := addTestBlockForTime(t, ctx, bs, time.Now().Add((-26*time.Hour)-(2*time.Second)))
	block2 := addTestBlockForTime(t, ctx, bs, time.Now().Add((-26*time.Hour)-(1*time.Second)))
	block3 := addTestBlockForTime(t, ctx, bs, time.Now().Add(-26*time.Hour))

	market1 := entities.Market{ID: entities.MarketID("dead")}
	market2 := entities.Market{ID: entities.MarketID("beef")}
	party1 := addTestParty(t, ctx, qs, block1)
	party2 := addTestParty(t, ctx, qs, block1)

	pos1a := addTestPosition(t, ctx, ps, market1, party1, 100, block1, txHashFromString("pos_1a"))
	pos1b := addTestPosition(t, ctx, ps, market1, party1, 200, block1, txHashFromString("pos_1b"))

	pos2 := addTestPosition(t, ctx, ps, market1, party2, 300, block2, txHashFromString("pos_2"))
	pos3 := addTestPosition(t, ctx, ps, market2, party1, 400, block2, txHashFromString("pos_3"))

	_, err := ps.Flush(ctx)
	require.NoError(t, err)

	_, _ = pos1a, pos1b

	assert.NoError(t, err)

	// Add some new positions
	pos1c := addTestPosition(t, ctx, ps, market1, party1, 200, block3, txHashFromString("pos_1c"))
	pos4 := addTestPosition(t, ctx, ps, market2, party2, 500, block3, txHashFromString("pos_4"))
	ps.Flush(ctx)

	t.Run("GetAll", func(t *testing.T) {
		expected := []entities.Position{pos1c, pos2, pos3, pos4}
		actual, err := ps.GetAll(ctx)
		require.NoError(t, err)
		assertPositionsMatch(t, expected, actual)
	})

	t.Run("GetByParty", func(t *testing.T) {
		expected := []entities.Position{pos1c, pos3}
		actual, err := ps.GetByParty(ctx, party1.ID.String())
		require.NoError(t, err)
		assertPositionsMatch(t, expected, actual)
	})

	t.Run("GetByMarket", func(t *testing.T) {
		expected := []entities.Position{pos1c, pos2}
		actual, err := ps.GetByMarket(ctx, market1.ID.String())
		require.NoError(t, err)
		assertPositionsMatch(t, expected, actual)
	})

	t.Run("GetByMarketAndParty", func(t *testing.T) {
		expected := pos4
		actual, err := ps.GetByMarketAndParty(ctx, market2.ID.String(), party2.ID.String())
		require.NoError(t, err)
		assert.True(t, expected.Equal(actual))
	})

	t.Run("GetByTxHash", func(t *testing.T) {
		expected := pos4
		actual, err := ps.GetByTxHash(ctx, expected.TxHash)
		require.NoError(t, err)
		assert.True(t, expected.Equal(actual[0]))
	})

	t.Run("GetBadMarketAndParty", func(t *testing.T) {
		_, err := ps.GetByMarketAndParty(ctx, market2.ID.String(), "ffff")
		assert.ErrorIs(t, err, entities.ErrNotFound)
	})
}

func setupPositionPaginationData(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, ps *sqlstore.Positions, pts *sqlstore.Parties) []entities.Position {
	t.Helper()
	positions := make([]entities.Position, 0, 10)
	blockTime := time.Now()
	for i := 0; i < 10; i++ {
		market := entities.Market{ID: entities.MarketID(fmt.Sprintf("deadbeef%02d", i))}
		for j := 0; j < 10; j++ {
			block := addTestBlockForTime(t, ctx, bs, blockTime)
			party := entities.Party{ID: entities.PartyID(fmt.Sprintf("deadbeef%02d", j)), VegaTime: &block.VegaTime}
			err := pts.Add(ctx, party)
			require.NoError(t, err)
			position := addTestPosition(t, ctx, ps, market, party, int64(i), block, defaultTxHash)
			positions = append(positions, position)
			blockTime = blockTime.Add(time.Minute)
		}
		blockTime = blockTime.Add(time.Hour)
	}
	_, err := ps.Flush(ctx)
	require.NoError(t, err)

	return positions
}

func TestPositions_CursorPagination(t *testing.T) {
	t.Run("should return all positions for party when no cursor is provided", testPositionCursorPaginationPartyNoCursor)
	t.Run("should return first page of positions for party when first is provided", testPositionCursorPaginationPartyFirstCursor)
	t.Run("should return last page of positions for party when last is provided", testPositionCursorPaginationPartyLastCursor)
	t.Run("should return requested page of positions for party when first and after is provided", testPositionCursorPaginationPartyFirstAfterCursor)
	t.Run("should return requested page of positions for party when last and before is provided", testPositionCursorPaginationPartyLastBeforeCursor)
	t.Run("should return all positions for party and market when no cursor is provided", testPositionCursorPaginationPartyMarketNoCursor)

	t.Run("should return all positions for party when no cursor is provided - newest first", testPositionCursorPaginationPartyNoCursorNewestFirst)
	t.Run("should return first page of positions for party when first is provided - newest first", testPositionCursorPaginationPartyFirstCursorNewestFirst)
	t.Run("should return last page of positions for party when last is provided - newest first", testPositionCursorPaginationPartyLastCursorNewestFirst)
	t.Run("should return requested page of positions for party when first and after is provided - newest first", testPositionCursorPaginationPartyFirstAfterCursorNewestFirst)
	t.Run("should return requested page of positions for party when last and before is provided - newest first", testPositionCursorPaginationPartyLastBeforeCursorNewestFirst)
	t.Run("should return all positions for party and market when no cursor is provided - newest first", testPositionCursorPaginationPartyMarketNoCursorNewestFirst)
}

func testPositionCursorPaginationPartyNoCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[0],
		positions[10],
		positions[20],
		positions[30],
		positions[40],
		positions[50],
		positions[60],
		positions[70],
		positions[80],
		positions[90],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, want[i].Equal(g))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyFirstCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)
	first := int32(3)

	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[0],
		positions[10],
		positions[20],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, want[i].Equal(g))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyLastCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[70],
		positions[80],
		positions[90],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, want[i].Equal(g))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyFirstAfterCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	first := int32(3)
	after := positions[20].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[30],
		positions[40],
		positions[50],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, want[i].Equal(g))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyLastBeforeCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	last := int32(3)
	before := positions[70].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[40],
		positions[50],
		positions[60],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, want[i].Equal(g))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyMarketNoCursor(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("deadbeef00")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[0],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, want[i].Equal(g))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[0].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[0],
		positions[10],
		positions[20],
		positions[30],
		positions[40],
		positions[50],
		positions[60],
		positions[70],
		positions[80],
		positions[90],
	}

	want = entities.ReverseSlice(want)

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, g.Equal(want[i]))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyFirstCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)
	first := int32(3)

	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[90],
		positions[80],
		positions[70],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, g.Equal(want[i]))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyLastCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[20],
		positions[10],
		positions[0],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, g.Equal(want[i]))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyFirstAfterCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	first := int32(3)
	after := positions[70].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[60],
		positions[50],
		positions[40],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, g.Equal(want[i]))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyLastBeforeCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	last := int32(3)
	before := positions[20].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[50],
		positions[40],
		positions[30],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, g.Equal(want[i]))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testPositionCursorPaginationPartyMarketNoCursorNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	ps := sqlstore.NewPositions(connectionSource)
	pts := sqlstore.NewParties(connectionSource)
	bs := sqlstore.NewBlocks(connectionSource)

	emptyMarketID := entities.MarketID("deadbeef00")

	positions := setupPositionPaginationData(t, ctx, bs, ps, pts)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	party := entities.Party{ID: entities.PartyID("deadbeef00")}
	want := []entities.Position{
		positions[0],
	}

	got, pageInfo, err := ps.GetByPartyConnection(ctx, []string{party.ID.String()}, []string{emptyMarketID.String()}, pagination)
	require.NoError(t, err)
	for i, g := range got {
		assert.True(t, g.Equal(want[i]))
	}
	// assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[0].Cursor().Encode(),
	}, pageInfo)
}
