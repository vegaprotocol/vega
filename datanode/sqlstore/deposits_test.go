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
	"encoding/hex"
	"fmt"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testAmount = "1000"

var testID = hex.EncodeToString([]byte(vgrand.RandomStr(5)))

func TestDeposits(t *testing.T) {
	t.Run("Upsert should insert deposits if one doesn't exist for the block", testAddDepositForNewBlock)
	t.Run("Upsert should update deposits if one already exists for the block", testUpdateDepositForBlockIfExists)
	t.Run("Upsert should insert deposit updates if the same deposit id is inserted in a different block", testInsertDepositUpdatesIfNewBlock)
	t.Run("GetByID should retrieve the latest state of the deposit with the given ID", testDepositsGetByID)
	t.Run("GetByParty should retrieve the latest state of all deposits for a given party", testDepositsGetByParty)
}

func TestDepositsPagination(t *testing.T) {
	t.Run("should return all deposits if no pagination is specified", testDepositsPaginationNoPagination)
	t.Run("should return the first page of results if first is provided", testDepositsPaginationFirst)
	t.Run("should return the last page of results if last is provided", testDepositsPaginationLast)
	t.Run("should return the specified page of results if first and after are provided", testDepositsPaginationFirstAfter)
	t.Run("should return the specified page of results if last and before are provided", testDepositsPaginationLastBefore)

	t.Run("should return all deposits if no pagination is specified - newest first", testDepositsPaginationNoPaginationNewestFirst)
	t.Run("should return the first page of results if first is provided - newest first", testDepositsPaginationFirstNewestFirst)
	t.Run("should return the last page of results if last is provided - newest first", testDepositsPaginationLastNewestFirst)
	t.Run("should return the specified page of results if first and after are provided - newest first", testDepositsPaginationFirstAfterNewestFirst)
	t.Run("should return the specified page of results if last and before are provided - newest first", testDepositsPaginationLastBeforeNewestFirst)

	t.Run("should return all deposits between dates if no pagination is specified", testDepositsPaginationBetweenDatesNoPagination)
	t.Run("should return the first page of results between dates if first is provided", testDepositsPaginationBetweenDatesFirst)
	t.Run("should return the last page of results between dates if last is provided", testDepositsPaginationBetweenDatesLast)
	t.Run("should return the specified page of results between dates if first and after are provided", testDepositsPaginationBetweenDatesFirstAfter)
	t.Run("should return the specified page of results between dates if last and before are provided", testDepositsPaginationBetweenDatesLastBefore)

	t.Run("should return all deposits between dates if no pagination is specified - newest first", testDepositsPaginationBetweenDatesNoPaginationNewestFirst)
	t.Run("should return the first page of results between dates if first is provided - newest first", testDepositsPaginationBetweenDatesFirstNewestFirst)
	t.Run("should return the last page of results between dates if last is provided - newest first", testDepositsPaginationBetweenDatesLastNewestFirst)
	t.Run("should return the specified page of results between dates if first and after are provided - newest first", testDepositsPaginationBetweenDatesFirstAfterNewestFirst)
	t.Run("should return the specified page of results between dates if last and before are provided - newest first", testDepositsPaginationBetweenDatesLastBeforeNewestFirst)
}

func setupDepositStoreTests(t *testing.T) (*sqlstore.Blocks, *sqlstore.Deposits, sqlstore.Connection) {
	t.Helper()
	bs := sqlstore.NewBlocks(connectionSource)
	ds := sqlstore.NewDeposits(connectionSource)
	return bs, ds, connectionSource.Connection
}

func testAddDepositForNewBlock(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, conn := setupDepositStoreTests(t)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)

	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testUpdateDepositForBlockIfExists(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, conn := setupDepositStoreTests(t)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	deposit.Status = entities.DepositStatus(vega.Deposit_STATUS_FINALIZED)

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	var status entities.DepositStatus
	err = pgxscan.Get(ctx, conn, &status, `select status from deposits where id = $1 and vega_time = $2`, deposit.ID, deposit.VegaTime)
	assert.NoError(t, err)
	assert.Equal(t, entities.DepositStatusFinalized, status)
}

func testInsertDepositUpdatesIfNewBlock(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs, ds, conn := setupDepositStoreTests(t)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, ctx, bs)
	depositProto.Status = vega.Deposit_STATUS_FINALIZED
	deposit, err = entities.DepositFromProto(depositProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	var status entities.DepositStatus
	err = pgxscan.Get(ctx, conn, &status, `select status from deposits where id = $1 and vega_time = $2`, deposit.ID, deposit.VegaTime)
	assert.NoError(t, err)
	assert.Equal(t, entities.DepositStatusFinalized, status)
}

func testDepositsGetByID(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, conn := setupDepositStoreTests(t)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, ctx, bs)
	depositProto.Status = vega.Deposit_STATUS_FINALIZED
	deposit, err = entities.DepositFromProto(depositProto, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)

	got, err := ds.GetByID(ctx, depositProto.Id)
	assert.NoError(t, err)

	// We need to truncate the timestamp because the postgres database will truncate to microseconds
	deposit.CreatedTimestamp = deposit.CreatedTimestamp.Truncate(time.Microsecond)
	deposit.CreditedTimestamp = deposit.CreditedTimestamp.Truncate(time.Microsecond)

	assert.Equal(t, *deposit, got)
}

func testDepositsGetByParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, conn := setupDepositStoreTests(t)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, ctx, bs)
	depositProto1 := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())
	depositProto1.Id = "deadbeef01"

	depositProto2 := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())
	depositProto2.Id = "deadbeef02"

	want := make([]entities.Deposit, 0)

	deposit, err := entities.DepositFromProto(depositProto1, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, ctx, bs)
	depositProto1.Status = vega.Deposit_STATUS_FINALIZED
	deposit, err = entities.DepositFromProto(depositProto1, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)

	deposit.CreatedTimestamp = deposit.CreatedTimestamp.Truncate(time.Microsecond)
	deposit.CreditedTimestamp = deposit.CreditedTimestamp.Truncate(time.Microsecond)

	want = append(want, *deposit)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, ctx, bs)
	deposit, err = entities.DepositFromProto(depositProto2, generateTxHash(), block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, ctx, bs)
	deposit, err = entities.DepositFromProto(depositProto2, generateTxHash(), block.VegaTime)
	depositProto2.Status = vega.Deposit_STATUS_FINALIZED
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(ctx, deposit)
	require.NoError(t, err)

	deposit.CreatedTimestamp = deposit.CreatedTimestamp.Truncate(time.Microsecond)
	deposit.CreditedTimestamp = deposit.CreditedTimestamp.Truncate(time.Microsecond)

	want = append(want, *deposit)

	got, _, err := ds.GetByParty(ctx, depositProto1.PartyId, false, entities.OffsetPagination{}, entities.DateRange{})
	assert.NoError(t, err)
	assert.Equal(t, want, got)
}

func getTestDeposit(id, party, asset, amount, txHash string, ts int64) *vega.Deposit {
	return &vega.Deposit{
		Id:                id,
		Status:            vega.Deposit_STATUS_OPEN,
		PartyId:           party,
		Asset:             asset,
		Amount:            amount,
		TxHash:            txHash,
		CreditedTimestamp: ts,
		CreatedTimestamp:  ts,
	}
}

func addDeposits(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, ds *sqlstore.Deposits) []entities.Deposit {
	t.Helper()
	vegaTime := time.Now().Truncate(time.Microsecond)
	amount := int64(1000)
	deposits := make([]entities.Deposit, 0, 10)
	for i := 0; i < 10; i++ {
		addTestBlockForTime(t, ctx, bs, vegaTime)

		depositProto := getTestDeposit(fmt.Sprintf("deadbeef%02d", i+1), testID, testID,
			strconv.FormatInt(amount, 10), helpers.GenerateID(), vegaTime.UnixNano())
		deposit, err := entities.DepositFromProto(depositProto, generateTxHash(), vegaTime)
		require.NoError(t, err, "Converting deposit proto to database entity")
		err = ds.Upsert(ctx, deposit)
		deposits = append(deposits, *deposit)
		require.NoError(t, err)

		vegaTime = vegaTime.Add(time.Second)
		amount += 100
	}

	return deposits
}

func testDepositsPaginationNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	assert.Equal(t, testDeposits, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     testDeposits[0].Cursor().Encode(),
		EndCursor:       testDeposits[9].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     testDeposits[0].Cursor().Encode(),
		EndCursor:       testDeposits[2].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     testDeposits[7].Cursor().Encode(),
		EndCursor:       testDeposits[9].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationFirstAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	first := int32(3)
	after := testDeposits[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     testDeposits[3].Cursor().Encode(),
		EndCursor:       testDeposits[5].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationLastBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	last := int32(3)
	before := testDeposits[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     testDeposits[4].Cursor().Encode(),
		EndCursor:       testDeposits[6].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := entities.ReverseSlice(addDeposits(ctx, t, bs, ds))

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	assert.Equal(t, testDeposits, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     testDeposits[0].Cursor().Encode(),
		EndCursor:       testDeposits[9].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationFirstNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := entities.ReverseSlice(addDeposits(ctx, t, bs, ds))

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     testDeposits[0].Cursor().Encode(),
		EndCursor:       testDeposits[2].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationLastNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := entities.ReverseSlice(addDeposits(ctx, t, bs, ds))

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     testDeposits[7].Cursor().Encode(),
		EndCursor:       testDeposits[9].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationFirstAfterNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := entities.ReverseSlice(addDeposits(ctx, t, bs, ds))

	first := int32(3)
	after := testDeposits[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     testDeposits[3].Cursor().Encode(),
		EndCursor:       testDeposits[5].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationLastBeforeNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := entities.ReverseSlice(addDeposits(ctx, t, bs, ds))

	last := int32(3)
	before := testDeposits[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{})

	require.NoError(t, err)
	want := testDeposits[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     testDeposits[4].Cursor().Encode(),
		EndCursor:       testDeposits[6].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	startDate := testDeposits[3].VegaTime
	endDate := testDeposits[8].VegaTime

	t.Run("Between start and end dates", func(t *testing.T) {
		got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
			Start: &startDate,
			End:   &endDate,
		})

		want := testDeposits[3:8]

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     testDeposits[3].Cursor().Encode(),
			EndCursor:       testDeposits[7].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("From start date only", func(t *testing.T) {
		got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
			Start: &startDate,
		})

		want := testDeposits[3:]

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     testDeposits[3].Cursor().Encode(),
			EndCursor:       testDeposits[9].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("To end date only", func(t *testing.T) {
		got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
			End: &endDate,
		})

		want := testDeposits[:8]

		require.NoError(t, err)
		assert.Equal(t, want, got)
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     testDeposits[0].Cursor().Encode(),
			EndCursor:       testDeposits[7].Cursor().Encode(),
		}, pageInfo)
	})
}

func testDepositsPaginationBetweenDatesFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime

	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	want := testDeposits[2:5]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     testDeposits[2].Cursor().Encode(),
		EndCursor:       testDeposits[4].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime

	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	want := testDeposits[5:8]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     testDeposits[5].Cursor().Encode(),
		EndCursor:       testDeposits[7].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesFirstAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	first := int32(3)
	after := testDeposits[4].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime

	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	want := testDeposits[5:8]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     testDeposits[5].Cursor().Encode(),
		EndCursor:       testDeposits[7].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesLastBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	last := int32(3)
	before := testDeposits[6].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime

	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	want := testDeposits[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     testDeposits[3].Cursor().Encode(),
		EndCursor:       testDeposits[5].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesNoPaginationNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)
	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime
	want := entities.ReverseSlice(testDeposits[2:8])

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[5].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesFirstNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)
	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime
	want := entities.ReverseSlice(testDeposits[2:8])

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	want = want[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesLastNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime

	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})
	want := entities.ReverseSlice(testDeposits[2:8])

	require.NoError(t, err)
	want = want[3:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesFirstAfterNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)
	want := entities.ReverseSlice(testDeposits[2:8])

	first := int32(3)
	after := want[1].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime

	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	want = want[2:5]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testDepositsPaginationBetweenDatesLastBeforeNewestFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs, ds, _ := setupDepositStoreTests(t)

	testDeposits := addDeposits(ctx, t, bs, ds)
	want := entities.ReverseSlice(testDeposits[2:8])

	last := int32(3)
	before := want[4].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	startDate := testDeposits[2].VegaTime
	endDate := testDeposits[8].VegaTime

	got, pageInfo, err := ds.GetByParty(ctx, testID, false, pagination, entities.DateRange{
		Start: &startDate,
		End:   &endDate,
	})

	require.NoError(t, err)
	want = want[1:4]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}
