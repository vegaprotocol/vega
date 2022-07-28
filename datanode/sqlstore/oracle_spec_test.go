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
	"encoding/hex"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleSpec(t *testing.T) {
	t.Run("Upsert should insert an OracleSpec when the id does not exist in the current block", testInsertIntoNewBlock)
	t.Run("Upsert should update an OracleSpec when the id already exists in the current block", testUpdateExistingInBlock)
	t.Run("GetSpecByID should retrieve the latest version of the specified OracleSpec", testGetSpecByID)
	t.Run("ListOracleSpecs should retrieve the latest versions of all OracleSpecs", testGetSpecs)
}

func setupOracleSpecTest(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.OracleSpec, *pgx.Conn) {
	t.Helper()
	DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	os := sqlstore.NewOracleSpec(connectionSource)

	config := NewTestConfig(testDBPort)
	conn, err := pgx.Connect(ctx, config.ConnectionConfig.GetConnectionString())
	require.NoError(t, err)

	return bs, os, conn
}

func testInsertIntoNewBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	proto := specProtos[0]
	data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
	require.NoError(t, err)
	assert.NoError(t, os.Upsert(context.Background(), data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)
}

func testUpdateExistingInBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	proto := specProtos[0]
	data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
	require.NoError(t, err)
	assert.NoError(t, os.Upsert(context.Background(), data))

	data.Status = entities.OracleSpecDeactivated
	assert.NoError(t, os.Upsert(context.Background(), data))

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 1, rowCount)
}

func testGetSpecByID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	for _, proto := range specProtos {
		data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, os.Upsert(context.Background(), data))
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)

	got, err := os.GetSpecByID(ctx, "DEADBEEF")
	require.NoError(t, err)

	want, err := entities.OracleSpecFromProto(specProtos[0], block.VegaTime)
	assert.NoError(t, err)
	// truncate the time to microseconds as postgres doesn't support nanosecond granularity.
	want.UpdatedAt = want.UpdatedAt.Truncate(time.Microsecond)
	want.CreatedAt = want.CreatedAt.Truncate(time.Microsecond)
	assert.Equal(t, *want, got)
}

func testGetSpecs(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, os, conn := setupOracleSpecTest(t, ctx)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	specProtos := getTestSpecs()

	want := make([]entities.OracleSpec, 0)

	for _, proto := range specProtos {
		data, err := entities.OracleSpecFromProto(proto, block.VegaTime)
		require.NoError(t, err)
		assert.NoError(t, os.Upsert(context.Background(), data))

		// truncate the time to microseconds as postgres doesn't support nanosecond granularity.
		data.CreatedAt = data.CreatedAt.Truncate(time.Microsecond)
		data.UpdatedAt = data.UpdatedAt.Truncate(time.Microsecond)
		want = append(want, *data)
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_specs").Scan(&rowCount))
	assert.Equal(t, 3, rowCount)

	got, err := os.GetSpecs(ctx, entities.OffsetPagination{})
	require.NoError(t, err)
	assert.ElementsMatch(t, want, got)
}

func getTestSpecs() []*oraclespb.OracleSpec {
	return []*oraclespb.OracleSpec{
		{
			Id:        "deadbeef",
			CreatedAt: time.Now().UnixNano(),
			UpdatedAt: time.Now().UnixNano(),
			PubKeys:   []string{"b105f00d", "baddcafe"},
			Filters: []*oraclespb.Filter{
				{
					Key: &oraclespb.PropertyKey{
						Name: "Ticker",
						Type: oraclespb.PropertyKey_TYPE_STRING,
					},
					Conditions: []*oraclespb.Condition{
						{
							Operator: oraclespb.Condition_OPERATOR_EQUALS,
							Value:    "USDETH",
						},
					},
				},
			},
			Status: oraclespb.OracleSpec_STATUS_ACTIVE,
		},
		{
			Id:        "cafed00d",
			CreatedAt: time.Now().UnixNano(),
			UpdatedAt: time.Now().UnixNano(),
			PubKeys:   []string{"b105f00d", "baddcafe"},
			Filters: []*oraclespb.Filter{
				{
					Key: &oraclespb.PropertyKey{
						Name: "Ticker",
						Type: oraclespb.PropertyKey_TYPE_STRING,
					},
					Conditions: []*oraclespb.Condition{
						{
							Operator: oraclespb.Condition_OPERATOR_EQUALS,
							Value:    "USDBTC",
						},
					},
				},
			},
			Status: oraclespb.OracleSpec_STATUS_ACTIVE,
		},
		{
			Id:        "deadbaad",
			CreatedAt: time.Now().UnixNano(),
			UpdatedAt: time.Now().UnixNano(),
			PubKeys:   []string{"b105f00d", "baddcafe"},
			Filters: []*oraclespb.Filter{
				{
					Key: &oraclespb.PropertyKey{
						Name: "Ticker",
						Type: oraclespb.PropertyKey_TYPE_STRING,
					},
					Conditions: []*oraclespb.Condition{
						{
							Operator: oraclespb.Condition_OPERATOR_EQUALS,
							Value:    "USDSOL",
						},
					},
				},
			},
			Status: oraclespb.OracleSpec_STATUS_ACTIVE,
		},
	}
}

func TestOracleSpec_GetSpecsWithCursorPagination(t *testing.T) {
	t.Run("should return the request spec of spec id is requested", testOracleSpecPaginationGetSpecID)
	t.Run("should return all specs if no spec id and no pagination is provided", testOracleSpecPaginationNoPagination)
	t.Run("should return the first page if no spec id and first is provided", testOracleSpecPaginationFirst)
	t.Run("should return the last page if no spec id and last is provided", testOracleSpecPaginationLast)
	t.Run("should return the requested page if no spec id and first and after is provided", testOracleSpecPaginationFirstAfter)
	t.Run("should return the requested page if no spec id and last and before is provided", testOracleSpecPaginationLastBefore)

	t.Run("should return all specs if no spec id and no pagination is provided - newest first", testOracleSpecPaginationNoPaginationNewestFirst)
	t.Run("should return the first page if no spec id and first is provided - newest first", testOracleSpecPaginationFirstNewestFirst)
	t.Run("should return the last page if no spec id and last is provided - newest first", testOracleSpecPaginationLastNewestFirst)
	t.Run("should return the requested page if no spec id and first and after is provided - newest first", testOracleSpecPaginationFirstAfterNewestFirst)
	t.Run("should return the requested page if no spec id and last and before is provided - newest first", testOracleSpecPaginationLastBeforeNewestFirst)
}

func createOracleSpecPaginationTestData(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, os *sqlstore.OracleSpec) []entities.OracleSpec {
	specs := make([]entities.OracleSpec, 0, 10)

	block := addTestBlockForTime(t, bs, time.Now().Truncate(time.Second))

	for i := 0; i < 10; i++ {
		pubKey, err := hex.DecodeString(generateID())
		require.NoError(t, err)
		spec := entities.OracleSpec{
			ID:         entities.NewSpecID(fmt.Sprintf("deadbeef%02d", i+1)),
			CreatedAt:  time.Now().Truncate(time.Microsecond),
			UpdatedAt:  time.Now().Truncate(time.Microsecond),
			PublicKeys: entities.PublicKeys{pubKey},
			Filters:    nil,
			Status:     entities.OracleSpecActive,
			VegaTime:   block.VegaTime,
		}

		err = os.Upsert(ctx, &spec)
		require.NoError(t, err)
		specs = append(specs, spec)
	}

	return specs
}

func testOracleSpecPaginationGetSpecID(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "deadbeef05", entities.CursorPagination{})
	require.NoError(t, err)

	assert.Equal(t, specs[4], got[0])
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     specs[4].Cursor().Encode(),
		EndCursor:       specs[4].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationNoPagination(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", entities.CursorPagination{})
	require.NoError(t, err)

	assert.Equal(t, specs, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[2].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLast(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     specs[7].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirstAfter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	first := int32(3)
	after := specs[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[3].Cursor().Encode(),
		EndCursor:       specs[5].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLastBefore(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := createOracleSpecPaginationTestData(t, ctx, bs, os)
	last := int32(3)
	before := specs[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[4].Cursor().Encode(),
		EndCursor:       specs[6].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", entities.CursorPagination{NewestFirst: true})
	require.NoError(t, err)

	assert.Equal(t, specs, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirstNewestFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     specs[0].Cursor().Encode(),
		EndCursor:       specs[2].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLastNewestFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     specs[7].Cursor().Encode(),
		EndCursor:       specs[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationFirstAfterNewestFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	first := int32(3)
	after := specs[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[3].Cursor().Encode(),
		EndCursor:       specs[5].Cursor().Encode(),
	}, pageInfo)
}

func testOracleSpecPaginationLastBeforeNewestFirst(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs, os, _ := setupOracleSpecTest(t, ctx)
	specs := entities.ReverseSlice(createOracleSpecPaginationTestData(t, ctx, bs, os))
	last := int32(3)
	before := specs[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := os.GetSpecsWithCursorPagination(ctx, "", pagination)
	require.NoError(t, err)

	assert.Equal(t, specs[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     specs[4].Cursor().Encode(),
		EndCursor:       specs[6].Cursor().Encode(),
	}, pageInfo)
}
