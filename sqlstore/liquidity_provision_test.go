// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/protos/vega"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLiquidityProvision(t *testing.T) {
	t.Run("Upsert should insert a liquidity provision record if the id doesn't exist in the current block", testInsertNewInCurrentBlock)
	t.Run("Upsert should update a liquidity provision record if the id already exists in the current block", testUpdateExistingInCurrentBlock)
	t.Run("Get should return all LP for a given party if no market is provided", testGetLPByPartyOnly)
	t.Run("Get should return all LP for a given party and market if both are provided", testGetLPByPartyAndMarket)
	t.Run("Get should error if no party and market are provided", testGetLPNoPartyAndMarketErrors)
	t.Run("Get should return all LP for a given market if no party id is provided", testGetLPNoPartyWithMarket)
	t.Run("Get should return LP with the corresponding reference", testGetLPByReferenceAndParty)
}

func TestLiquidityProvisionPagination(t *testing.T) {
	t.Run("should return all liquidity provisions if no pagination is specified", testLiquidityProvisionPaginationNoPagination)
	t.Run("should return the first page of results if first is provided", testLiquidityProvisionPaginationFirst)
	t.Run("should return the last page of results if last is provided", testLiquidityProvisionPaginationLast)
	t.Run("should return the specified page of results if first and after are provided", testLiquidityProvisionPaginationFirstAfter)
	t.Run("should return the specified page of results if last and before are provided", testLiquidityProvisionPaginationLastBefore)
}

func setupLPTests(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.LiquidityProvision, *pgx.Conn) {
	t.Helper()

	DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	lp := sqlstore.NewLiquidityProvision(connectionSource)

	config := NewTestConfig(testDBPort)
	conn, err := pgx.Connect(ctx, config.ConnectionConfig.GetConnectionString())
	require.NoError(t, err)

	return bs, lp, conn
}

func testInsertNewInCurrentBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, lp, conn := setupLPTests(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	lpProto := getTestLiquidityProvision()

	data, err := entities.LiquidityProvisionFromProto(lpProto[0], block.VegaTime)
	require.NoError(t, err)
	assert.NoError(t, lp.Upsert(context.Background(), data))
	err = lp.Flush(ctx)
	require.NoError(t, err)

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)
}

func testUpdateExistingInCurrentBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, lp, conn := setupLPTests(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	lpProto := getTestLiquidityProvision()

	data, err := entities.LiquidityProvisionFromProto(lpProto[0], block.VegaTime)
	require.NoError(t, err)
	assert.NoError(t, lp.Upsert(context.Background(), data))

	data.Reference = "Updated"
	assert.NoError(t, lp.Upsert(context.Background(), data))
	err = lp.Flush(ctx)
	require.NoError(t, err)

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)
}

func testGetLPByReferenceAndParty(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, lp, conn := setupLPTests(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	lpProto := getTestLiquidityProvision()

	want := make([]entities.LiquidityProvision, 0)

	for _, lpp := range lpProto {
		block := addTestBlock(t, bs)

		data, err := entities.LiquidityProvisionFromProto(lpp, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, lp.Upsert(context.Background(), data))
		err = lp.Flush(ctx)
		require.NoError(t, err)

		data.CreatedAt = data.CreatedAt.Truncate(time.Microsecond)
		data.UpdatedAt = data.UpdatedAt.Truncate(time.Microsecond)

		want = append(want, data)

		time.Sleep(100 * time.Millisecond)
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)

	partyID := entities.NewPartyID("deadbaad")
	marketID := entities.NewMarketID("")
	got, _, err := lp.Get(ctx, partyID, marketID, "TEST1", entities.OffsetPagination{})
	require.NoError(t, err)
	assert.Equal(t, 1, len(got))
	assert.Equal(t, got[0].Reference, "TEST1")
}

func testGetLPByPartyOnly(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, lp, conn := setupLPTests(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	lpProto := getTestLiquidityProvision()

	want := make([]entities.LiquidityProvision, 0)

	for _, lpp := range lpProto {
		block := addTestBlock(t, bs)

		data, err := entities.LiquidityProvisionFromProto(lpp, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, lp.Upsert(context.Background(), data))
		err = lp.Flush(ctx)
		require.NoError(t, err)

		data.CreatedAt = data.CreatedAt.Truncate(time.Microsecond)
		data.UpdatedAt = data.UpdatedAt.Truncate(time.Microsecond)

		want = append(want, data)

		time.Sleep(100 * time.Millisecond)
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)

	partyID := entities.NewPartyID("deadbaad")
	marketID := entities.NewMarketID("")
	got, _, err := lp.Get(ctx, partyID, marketID, "", entities.OffsetPagination{})
	require.NoError(t, err)
	assert.Equal(t, len(want), len(got))
	assert.ElementsMatch(t, want, got)
}

func testGetLPByPartyAndMarket(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, lp, conn := setupLPTests(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	lpProto := getTestLiquidityProvision()

	wantMarketID := "dabbad00"

	want := make([]entities.LiquidityProvision, 0)

	for _, lpp := range lpProto {
		block := addTestBlock(t, bs)

		data, err := entities.LiquidityProvisionFromProto(lpp, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, lp.Upsert(context.Background(), data))
		err = lp.Flush(ctx)
		require.NoError(t, err)

		data.CreatedAt = data.CreatedAt.Truncate(time.Microsecond)
		data.UpdatedAt = data.UpdatedAt.Truncate(time.Microsecond)

		if data.MarketID.String() == wantMarketID {
			want = append(want, data)
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)

	partyID := entities.NewPartyID("DEADBAAD")
	marketID := entities.NewMarketID(wantMarketID)
	got, _, err := lp.Get(ctx, partyID, marketID, "", entities.OffsetPagination{})
	require.NoError(t, err)
	assert.Equal(t, len(want), len(got))
	assert.ElementsMatch(t, want, got)
}

func testGetLPNoPartyAndMarketErrors(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	_, lp, _ := setupLPTests(t, ctx)
	partyID := entities.NewPartyID("")
	marketID := entities.NewMarketID("")
	_, _, err := lp.Get(ctx, partyID, marketID, "", entities.OffsetPagination{})
	assert.Error(t, err)
}

func testGetLPNoPartyWithMarket(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, lp, conn := setupLPTests(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	lpProto := getTestLiquidityProvision()
	wantMarketID := "dabbad00"
	want := make([]entities.LiquidityProvision, 0)

	for _, lpp := range lpProto {
		block := addTestBlock(t, bs)

		data, err := entities.LiquidityProvisionFromProto(lpp, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, lp.Upsert(context.Background(), data))
		err = lp.Flush(ctx)
		require.NoError(t, err)

		data.CreatedAt = data.CreatedAt.Truncate(time.Microsecond)
		data.UpdatedAt = data.UpdatedAt.Truncate(time.Microsecond)

		if data.MarketID.String() == wantMarketID {
			want = append(want, data)
		}

		time.Sleep(100 * time.Millisecond)
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from liquidity_provisions").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)
	partyID := entities.NewPartyID("")
	marketID := entities.NewMarketID(wantMarketID)
	got, _, err := lp.Get(ctx, partyID, marketID, "", entities.OffsetPagination{})
	require.NoError(t, err)
	assert.Equal(t, len(want), len(got))
	assert.ElementsMatch(t, want, got)
}

func getTestLiquidityProvision() []*vega.LiquidityProvision {
	return []*vega.LiquidityProvision{
		{
			Id:               "deadbeef",
			PartyId:          "deadbaad",
			CreatedAt:        time.Now().UnixNano(),
			UpdatedAt:        time.Now().UnixNano(),
			MarketId:         "cafed00d",
			CommitmentAmount: "100000",
			Fee:              "0.3",
			Sells:            nil,
			Buys:             nil,
			Version:          0,
			Status:           vega.LiquidityProvision_STATUS_ACTIVE,
			Reference:        "TEST1",
		},
		{
			Id:               "0d15ea5e",
			PartyId:          "deadbaad",
			CreatedAt:        time.Now().UnixNano(),
			UpdatedAt:        time.Now().UnixNano(),
			MarketId:         "dabbad00",
			CommitmentAmount: "100000",
			Fee:              "0.3",
			Sells:            nil,
			Buys:             nil,
			Version:          0,
			Status:           vega.LiquidityProvision_STATUS_ACTIVE,
			Reference:        "TEST",
		},
		{
			Id:               "deadc0de",
			PartyId:          "deadbaad",
			CreatedAt:        time.Now().UnixNano(),
			UpdatedAt:        time.Now().UnixNano(),
			MarketId:         "deadd00d",
			CommitmentAmount: "100000",
			Fee:              "0.3",
			Sells:            nil,
			Buys:             nil,
			Version:          0,
			Status:           vega.LiquidityProvision_STATUS_ACTIVE,
			Reference:        "TEST",
		},
	}
}

func testLiquidityProvisionPaginationNoPagination(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, lpStore, _ := setupLPTests(t, timeoutCtx)
	testLps := addLiquidityProvisions(timeoutCtx, t, bs, lpStore)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := lpStore.Get(timeoutCtx, entities.PartyID{ID: "deadbaad"}, entities.MarketID{ID: ""}, "", pagination)

	require.NoError(t, err)
	assert.Equal(t, testLps, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[0].VegaTime,
		ID:       testLps[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[9].VegaTime,
		ID:       testLps[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testLiquidityProvisionPaginationFirst(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, lpStore, _ := setupLPTests(t, timeoutCtx)
	testLps := addLiquidityProvisions(timeoutCtx, t, bs, lpStore)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := lpStore.Get(timeoutCtx, entities.PartyID{ID: "deadbaad"}, entities.MarketID{ID: ""}, "", pagination)

	require.NoError(t, err)
	want := testLps[:3]
	assert.Equal(t, want, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[0].VegaTime,
		ID:       testLps[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[2].VegaTime,
		ID:       testLps[2].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testLiquidityProvisionPaginationLast(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, lpStore, _ := setupLPTests(t, timeoutCtx)
	testLps := addLiquidityProvisions(timeoutCtx, t, bs, lpStore)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := lpStore.Get(timeoutCtx, entities.PartyID{ID: "deadbaad"}, entities.MarketID{ID: ""}, "", pagination)

	require.NoError(t, err)
	want := testLps[7:]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[7].VegaTime,
		ID:       testLps[7].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[9].VegaTime,
		ID:       testLps[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testLiquidityProvisionPaginationFirstAfter(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, lpStore, _ := setupLPTests(t, timeoutCtx)
	testLps := addLiquidityProvisions(timeoutCtx, t, bs, lpStore)

	first := int32(3)
	after := entities.NewCursor(entities.DepositCursor{
		VegaTime: testLps[2].VegaTime,
		ID:       testLps[2].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := lpStore.Get(timeoutCtx, entities.PartyID{ID: "deadbaad"}, entities.MarketID{ID: ""}, "", pagination)

	require.NoError(t, err)
	want := testLps[3:6]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[3].VegaTime,
		ID:       testLps[3].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[5].VegaTime,
		ID:       testLps[5].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testLiquidityProvisionPaginationLastBefore(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, lsStore, _ := setupLPTests(t, timeoutCtx)
	testLps := addLiquidityProvisions(timeoutCtx, t, bs, lsStore)

	last := int32(3)
	before := entities.NewCursor(entities.LiquidityProvisionCursor{
		VegaTime: testLps[7].VegaTime,
		ID:       testLps[7].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := lsStore.Get(timeoutCtx, entities.PartyID{ID: "deadbaad"}, entities.MarketID{ID: ""}, "", pagination)

	require.NoError(t, err)
	want := testLps[4:7]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[4].VegaTime,
		ID:       testLps[4].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testLps[6].VegaTime,
		ID:       testLps[6].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func addLiquidityProvisions(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, lpstore *sqlstore.LiquidityProvision) []entities.LiquidityProvision {

	vegaTime := time.Now().Truncate(time.Microsecond)
	amount := int64(1000)
	lps := make([]entities.LiquidityProvision, 0, 10)
	for i := 0; i < 10; i++ {
		addTestBlockForTime(t, bs, vegaTime)

		lp := &vega.LiquidityProvision{

			Id:               fmt.Sprintf("deadbeef%02d", i+1),
			PartyId:          "deadbaad",
			CreatedAt:        vegaTime.UnixNano(),
			UpdatedAt:        vegaTime.UnixNano(),
			MarketId:         "cafed00d",
			CommitmentAmount: "100000",
			Fee:              "0.3",
			Sells:            nil,
			Buys:             nil,
			Version:          0,
			Status:           vega.LiquidityProvision_STATUS_ACTIVE,
			Reference:        "TEST1",
		}

		withdrawal, err := entities.LiquidityProvisionFromProto(lp, vegaTime)
		require.NoError(t, err, "Converting withdrawal proto to database entity")
		err = lpstore.Upsert(ctx, withdrawal)
		require.NoError(t, err)
		require.NoError(t, err)
		lpstore.Flush(ctx)
		lps = append(lps, withdrawal)
		require.NoError(t, err)

		vegaTime = vegaTime.Add(time.Second)
		amount += 100
	}

	return lps
}
