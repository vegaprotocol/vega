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
	got, err := lp.Get(ctx, partyID, marketID, entities.OffsetPagination{})
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
	got, err := lp.Get(ctx, partyID, marketID, entities.OffsetPagination{})
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
	_, err := lp.Get(ctx, partyID, marketID, entities.OffsetPagination{})
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
	got, err := lp.Get(ctx, partyID, marketID, entities.OffsetPagination{})
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
			Reference:        "TEST",
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
