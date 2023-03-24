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
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOracleData(t *testing.T) {
	t.Run("Add should insert oracle data", testAddOracleData)
	t.Run("GetOracleDataBySpecID should return all data where matched spec ids contains the provided id", testGetOracleDataBySpecID)
	t.Run("GetByTxHash", testGetOracleDataByTxHash)
}

func setupOracleDataTest(t *testing.T) (*sqlstore.Blocks, *sqlstore.OracleData, sqlstore.Connection) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	od := sqlstore.NewOracleData(connectionSource)
	return bs, od, connectionSource.Connection
}

func testAddOracleData(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, od, conn := setupOracleDataTest(t)

	var rowCount int
	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount))
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	dataProtos := getTestOracleData()

	for i, proto := range dataProtos {
		data, err := entities.OracleDataFromProto(proto, generateTxHash(), block.VegaTime, uint64(i))
		require.NoError(t, err)
		assert.NoError(t, od.Add(ctx, data))
	}

	assert.NoError(t, conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount))
	assert.Equal(t, len(dataProtos), rowCount)
}

func testGetOracleDataBySpecID(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, od, conn := setupOracleDataTest(t)

	var rowCount int
	err := conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	testTime := time.Now()
	dataProtos := getTestOracleData()

	for i, proto := range dataProtos {
		block := addTestBlockForTime(t, ctx, bs, testTime)
		data, err := entities.OracleDataFromProto(proto, generateTxHash(), block.VegaTime, uint64(i))
		require.NoError(t, err)
		err = od.Add(ctx, data)
		require.NoError(t, err)
		testTime = testTime.Add(time.Minute)
	}

	err = conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, len(dataProtos), rowCount)

	got, _, err := od.GetOracleDataBySpecID(ctx, "deadbeef02", entities.OffsetPagination{})
	require.NoError(t, err)
	assert.Equal(t, 2, len(got))
}

func testGetOracleDataByTxHash(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, od, conn := setupOracleDataTest(t)

	var rowCount int
	err := conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	testTime := time.Now()
	dataProtos := getTestOracleData()

	datas := make([]entities.OracleData, 0, len(dataProtos))
	for i, proto := range dataProtos {
		block := addTestBlockForTime(t, ctx, bs, testTime)
		data, err := entities.OracleDataFromProto(proto, generateTxHash(), block.VegaTime, uint64(i))
		require.NoError(t, err)
		err = od.Add(ctx, data)
		require.NoError(t, err)
		testTime = testTime.Add(time.Minute)

		datas = append(datas, *data)
	}

	err = conn.QueryRow(ctx, "select count(*) from oracle_data").Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, len(dataProtos), rowCount)

	foundData, err := od.GetByTxHash(ctx, datas[0].ExternalData.Data.TxHash)
	require.NoError(t, err)
	assert.Equal(t, 1, len(foundData))
	assert.Equal(t, datas[0].ExternalData.Data, foundData[0].ExternalData.Data)

	foundData2, err := od.GetByTxHash(ctx, datas[1].ExternalData.Data.TxHash)
	require.NoError(t, err)
	assert.Equal(t, 1, len(foundData2))
	assert.Equal(t, datas[1].ExternalData.Data, foundData2[0].ExternalData.Data)
}

func getTestOracleData() []*vegapb.OracleData {
	pk1 := types.CreateSignerFromString("b105f00d", types.DataSignerTypePubKey)
	pk2 := types.CreateSignerFromString("baddcafe", types.DataSignerTypePubKey)

	return []*vegapb.OracleData{
		{ // 0
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "Ticker",
							Value: "USDBTC",
						},
					},
					MatchedSpecIds: []string{"deadbeef01"},
					BroadcastAt:    0,
				},
			},
		},
		//},
		{ // 1
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "Ticker",
							Value: "USDETH",
						},
					},
					MatchedSpecIds: []string{"deadbeef02"},
					BroadcastAt:    0,
				},
			},
		},
		{ // 2
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "Ticker",
							Value: "USDETH",
						},
					},
					MatchedSpecIds: []string{"deadbeef02"},
					BroadcastAt:    0,
				},
			},
		},
		{ // 3
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "Ticker",
							Value: "USDSOL",
						},
					},
					MatchedSpecIds: []string{"deadbeef03"},
					BroadcastAt:    0,
				},
			},
		},
		{ // 4
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "AAAA",
							Value: "AAAA",
						},
					},
					MatchedSpecIds: []string{"deadbeef04"},
				},
			},
		},
		{ // 5
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "BBBB",
							Value: "BBBB",
						},
					},
					MatchedSpecIds: []string{"deadbeef04"},
				},
			},
		},
		{ // 6
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "CCCC",
							Value: "CCCC",
						},
					},
					MatchedSpecIds: []string{"deadbeef04"},
				},
			},
		},
		{ // 7
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "DDDD",
							Value: "DDDD",
						},
					},
					MatchedSpecIds: []string{"deadbeef04"},
				},
			},
		},
		{ // 8
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "EEEE",
							Value: "EEEE",
						},
					},
					MatchedSpecIds: []string{"deadbeef04"},
				},
			},
		},
		{ // 9
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "FFFF",
							Value: "FFFF",
						},
					},
					MatchedSpecIds: []string{"deadbeef04"},
				},
			},
		},
		{ // 10
			ExternalData: &datapb.ExternalData{
				Data: &datapb.Data{
					Signers: []*datapb.Signer{pk1.IntoProto(), pk2.IntoProto()},
					Data: []*datapb.Property{
						{
							Name:  "GGGG",
							Value: "GGGG",
						},
					},
					MatchedSpecIds: []string{"deadbeef04"},
				},
			},
		},
	}
}

func getTestPaginationOracleData(t *testing.T, ctx context.Context, bs *sqlstore.Blocks, ds *sqlstore.OracleData) []entities.OracleData {
	t.Helper()
	protoData := getTestOracleData()
	data := make([]entities.OracleData, 0, len(protoData))

	blockTime := time.Now()

	for i, item := range protoData {
		block := addTestBlockForTime(t, ctx, bs, blockTime)
		odEntity, err := entities.OracleDataFromProto(item, generateTxHash(), block.VegaTime, uint64(i))
		require.NoError(t, err)

		err = ds.Add(ctx, odEntity)
		require.NoError(t, err)

		data = append(data, *odEntity)
		blockTime = blockTime.Add(time.Minute)
	}

	return data
}

func TestOracleData_GetOracleDataBySpecIDCursorPagination(t *testing.T) {
	t.Run("should return all data when no pagination is provided", testOracleDataGetBySpecNoPagination)
	t.Run("should return first page when first is provided", testOracleDataGetBySpecFirst)
	t.Run("should return last page when last is provided", testOracleDataGetBySpecLast)
	t.Run("should return requested page when first and after is provided", testOracleDataGetBySpecFirstAfter)
	t.Run("should return requested page when last and before is provided", testOracleDataGetBySpecLastBefore)
}

func testOracleDataGetBySpecNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.GetOracleDataBySpecID(ctx, "deadbeef04", pagination)
	require.NoError(t, err)
	assert.Equal(t, data[4:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     data[4].Cursor().Encode(),
		EndCursor:       data[10].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataGetBySpecFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.GetOracleDataBySpecID(ctx, "deadbeef04", pagination)
	require.NoError(t, err)
	assert.Equal(t, data[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     data[4].Cursor().Encode(),
		EndCursor:       data[6].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataGetBySpecLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.GetOracleDataBySpecID(ctx, "deadbeef04", pagination)
	require.NoError(t, err)
	assert.Equal(t, data[8:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     data[8].Cursor().Encode(),
		EndCursor:       data[10].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataGetBySpecFirstAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)

	first := int32(3)
	after := data[6].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.GetOracleDataBySpecID(ctx, "deadbeef04", pagination)
	require.NoError(t, err)
	assert.Equal(t, data[7:10], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     data[7].Cursor().Encode(),
		EndCursor:       data[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataGetBySpecLastBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)

	last := int32(3)
	before := data[8].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.GetOracleDataBySpecID(ctx, "deadbeef04", pagination)
	require.NoError(t, err)
	assert.Equal(t, data[5:8], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     data[5].Cursor().Encode(),
		EndCursor:       data[7].Cursor().Encode(),
	}, pageInfo)
}

func TestOracleData_ListOracleDataCursorPagination(t *testing.T) {
	t.Run("should return all data when no pagination is provided", testOracleDataListNoPagination)
	t.Run("should return first page when first is provided", testOracleDataListFirst)
	t.Run("should return last page when last is provided", testOracleDataListLast)
	t.Run("should return requested page when first and after is provided", testOracleDataListFirstAfter)
	t.Run("should return requested page when last and before is provided", testOracleDataListLastBefore)

	t.Run("should return all data when no pagination is provided - newest first", testOracleDataListNoPaginationNewestFirst)
	t.Run("should return first page when first is provided - newest first", testOracleDataListFirstNewestFirst)
	t.Run("should return last page when last is provided - newest first", testOracleDataListLastNewestFirst)
	t.Run("should return requested page when first and after is provided - newest first", testOracleDataListFirstAfterNewestFirst)
	t.Run("should return requested page when last and before is provided - newest first", testOracleDataListLastBeforeNewestFirst)
}

func testOracleDataListNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = append(want, data[2:]...)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     data[0].Cursor().Encode(),
		EndCursor:       data[10].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = append(want, data[2:]...)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[0:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = append(want, data[2:]...)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[7].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListFirstAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = append(want, data[2:]...)

	first := int32(3)
	after := want[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[3].Cursor().Encode(),
		EndCursor:       want[5].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListLastBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = append(want, data[2:]...)

	last := int32(3)
	before := want[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[4].Cursor().Encode(),
		EndCursor:       want[6].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListNoPaginationNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = entities.ReverseSlice(append(want, data[2:]...))

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListFirstNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = entities.ReverseSlice(append(want, data[2:]...))

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[0:3], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListLastNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = entities.ReverseSlice(append(want, data[2:]...))

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[7:], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[7].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListFirstAfterNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = entities.ReverseSlice(append(want, data[2:]...))

	first := int32(3)
	after := want[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[3:6], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[3].Cursor().Encode(),
		EndCursor:       want[5].Cursor().Encode(),
	}, pageInfo)
}

func testOracleDataListLastBeforeNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, _ := setupOracleDataTest(t)
	data := getTestPaginationOracleData(t, ctx, bs, ds)
	want := []entities.OracleData{data[0]}
	want = entities.ReverseSlice(append(want, data[2:]...))

	last := int32(3)
	before := want[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	got, pageInfo, err := ds.ListOracleData(ctx, pagination)
	require.NoError(t, err)
	assert.Equal(t, want[4:7], got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[4].Cursor().Encode(),
		EndCursor:       want[6].Cursor().Encode(),
	}, pageInfo)
}
