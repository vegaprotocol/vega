package sqlstore_test

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/protos/vega"
	"github.com/georgysavva/scany/pgxscan"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testID     = "deadbeef"
	testAmount = "1000"
)

func TestDeposits(t *testing.T) {
	t.Run("Upsert should insert deposits if one doesn't exist for the block", testAddDepositForNewBlock)
	t.Run("Upsert should error if the vega block does not exist", testErrorIfBlockDoesNotExist)
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
}

func setupDepositStoreTests(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.Deposits, *pgx.Conn) {
	t.Helper()
	DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	ds := sqlstore.NewDeposits(connectionSource)

	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, config.ConnectionConfig.GetConnectionString())
	require.NoError(t, err)

	return bs, ds, conn
}

func testAddDepositForNewBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ds, conn := setupDepositStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)

	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testErrorIfBlockDoesNotExist(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ds, conn := setupDepositStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, block.VegaTime.Add(time.Second))
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.Error(t, err, "Should error if the block does not exist")
}

func testUpdateDepositForBlockIfExists(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ds, conn := setupDepositStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	deposit.Status = entities.DepositStatus(vega.Deposit_STATUS_FINALIZED)

	err = ds.Upsert(context.Background(), deposit)
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
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ds, conn := setupDepositStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	depositProto.Status = vega.Deposit_STATUS_FINALIZED
	deposit, err = entities.DepositFromProto(depositProto, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
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
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ds, conn := setupDepositStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	depositProto := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())

	deposit, err := entities.DepositFromProto(depositProto, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	depositProto.Status = vega.Deposit_STATUS_FINALIZED
	deposit, err = entities.DepositFromProto(depositProto, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)

	got, err := ds.GetByID(ctx, depositProto.Id)
	assert.NoError(t, err)

	// We need to truncate the timestamp because the postgres database will truncate to microseconds
	deposit.CreatedTimestamp = deposit.CreatedTimestamp.Truncate(time.Microsecond)
	deposit.CreditedTimestamp = deposit.CreditedTimestamp.Truncate(time.Microsecond)

	assert.Equal(t, *deposit, got)
}

func testDepositsGetByParty(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ds, conn := setupDepositStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from deposits`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	depositProto1 := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())
	depositProto1.Id = "deadbeef01"

	depositProto2 := getTestDeposit(testID, testID, testID, testAmount, testID, time.Now().UnixNano())
	depositProto2.Id = "deadbeef02"

	want := make([]entities.Deposit, 0)

	deposit, err := entities.DepositFromProto(depositProto1, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	depositProto1.Status = vega.Deposit_STATUS_FINALIZED
	deposit, err = entities.DepositFromProto(depositProto1, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)

	deposit.CreatedTimestamp = deposit.CreatedTimestamp.Truncate(time.Microsecond)
	deposit.CreditedTimestamp = deposit.CreditedTimestamp.Truncate(time.Microsecond)

	want = append(want, *deposit)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	deposit, err = entities.DepositFromProto(depositProto2, block.VegaTime)
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	deposit, err = entities.DepositFromProto(depositProto2, block.VegaTime)
	depositProto2.Status = vega.Deposit_STATUS_FINALIZED
	require.NoError(t, err, "Converting market proto to database entity")

	err = ds.Upsert(context.Background(), deposit)
	require.NoError(t, err)

	deposit.CreatedTimestamp = deposit.CreatedTimestamp.Truncate(time.Microsecond)
	deposit.CreditedTimestamp = deposit.CreditedTimestamp.Truncate(time.Microsecond)

	want = append(want, *deposit)

	got, _, err := ds.GetByParty(ctx, depositProto1.PartyId, false, entities.OffsetPagination{})
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
	vegaTime := time.Now().Truncate(time.Microsecond)
	amount := int64(1000)
	deposits := make([]entities.Deposit, 0, 10)
	for i := 0; i < 10; i++ {
		addTestBlockForTime(t, bs, vegaTime)

		depositProto := getTestDeposit(fmt.Sprintf("deadbeef%02d", i+1), testID, testID,
			strconv.FormatInt(amount, 10), generateID(), vegaTime.UnixNano())
		deposit, err := entities.DepositFromProto(depositProto, vegaTime)
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
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ds, _ := setupDepositStoreTests(t, timeoutCtx)

	testDeposits := addDeposits(timeoutCtx, t, bs, ds)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	assert.Equal(t, testDeposits, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[0].VegaTime,
		ID:       testDeposits[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[9].VegaTime,
		ID:       testDeposits[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testDepositsPaginationFirst(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ds, _ := setupDepositStoreTests(t, timeoutCtx)

	testDeposits := addDeposits(timeoutCtx, t, bs, ds)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testDeposits[:3]
	assert.Equal(t, want, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[0].VegaTime,
		ID:       testDeposits[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[2].VegaTime,
		ID:       testDeposits[2].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testDepositsPaginationLast(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ds, _ := setupDepositStoreTests(t, timeoutCtx)

	testDeposits := addDeposits(timeoutCtx, t, bs, ds)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testDeposits[7:]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[7].VegaTime,
		ID:       testDeposits[7].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[9].VegaTime,
		ID:       testDeposits[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testDepositsPaginationFirstAfter(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ds, _ := setupDepositStoreTests(t, timeoutCtx)

	testDeposits := addDeposits(timeoutCtx, t, bs, ds)

	first := int32(3)
	after := entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[2].VegaTime,
		ID:       testDeposits[2].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testDeposits[3:6]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[3].VegaTime,
		ID:       testDeposits[3].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[5].VegaTime,
		ID:       testDeposits[5].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testDepositsPaginationLastBefore(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ds, _ := setupDepositStoreTests(t, timeoutCtx)

	testDeposits := addDeposits(timeoutCtx, t, bs, ds)

	last := int32(3)
	before := entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[7].VegaTime,
		ID:       testDeposits[7].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before)
	require.NoError(t, err)
	got, pageInfo, err := ds.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testDeposits[4:7]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[4].VegaTime,
		ID:       testDeposits[4].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.DepositCursor{
		VegaTime: testDeposits[6].VegaTime,
		ID:       testDeposits[6].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}
