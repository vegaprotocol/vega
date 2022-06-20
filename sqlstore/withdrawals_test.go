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

func TestWithdrawals(t *testing.T) {
	t.Run("Upsert should insert withdrawals if one doesn't exist for the block", testAddWithdrawalForNewBlock)
	t.Run("Upsert should error if the vega block does not exist", testWithdrawalErrorIfBlockDoesNotExist)
	t.Run("Upsert should update withdrawals if one already exists for the block", testUpdateWithdrawalForBlockIfExists)
	t.Run("Upsert should insert withdrawal updates if the same withdrawal id is inserted in a different block", testInsertWithdrawalUpdatesIfNewBlock)
	t.Run("GetByID should retrieve the latest state of the withdrawal with the given ID", testWithdrawalsGetByID)
	t.Run("GetByParty should retrieve the latest state of all withdrawals for a given party", testWithdrawalsGetByParty)
}

func TestWithdrawalsPagination(t *testing.T) {
	t.Run("should return all withdrawals if no pagination is specified", testWithdrawalsPaginationNoPagination)
	t.Run("should return the first page of results if first is provided", testWithdrawalsPaginationFirst)
	t.Run("should return the last page of results if last is provided", testWithdrawalsPaginationLast)
	t.Run("should return the specified page of results if first and after are provided", testWithdrawalsPaginationFirstAfter)
	t.Run("should return the specified page of results if last and before are provided", testWithdrawalsPaginationLastBefore)
}

func setupWithdrawalStoreTests(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.Withdrawals, *pgx.Conn) {
	t.Helper()
	DeleteEverything()

	bs := sqlstore.NewBlocks(connectionSource)
	ws := sqlstore.NewWithdrawals(connectionSource)

	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, config.ConnectionConfig.GetConnectionString())
	require.NoError(t, err)

	return bs, ws, conn
}

func testAddWithdrawalForNewBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ws, conn := setupWithdrawalStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	withdrawalProto := getTestWithdrawal(testID, testID, testID, testAmount, testID, block.VegaTime)

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testWithdrawalErrorIfBlockDoesNotExist(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ws, conn := setupWithdrawalStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	withdrawalProto := getTestWithdrawal(testID, testID, testID, testAmount, testID, block.VegaTime)

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime.Add(time.Second))
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.Error(t, err, "Should error if the block does not exist")
}

func testUpdateWithdrawalForBlockIfExists(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ws, conn := setupWithdrawalStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	withdrawalProto := getTestWithdrawal(testID, testID, testID, testAmount, testID, block.VegaTime)

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	withdrawal.Status = entities.WithdrawalStatus(vega.Withdrawal_STATUS_FINALIZED)

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	var status entities.WithdrawalStatus
	err = pgxscan.Get(ctx, conn, &status, `select status from withdrawals where id = $1 and vega_time = $2`, withdrawal.ID, withdrawal.VegaTime)
	assert.NoError(t, err)
	assert.Equal(t, entities.WithdrawalStatusFinalized, status)
}

func testInsertWithdrawalUpdatesIfNewBlock(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ws, conn := setupWithdrawalStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	withdrawalProto := getTestWithdrawal(testID, testID, testID, testAmount, testID, block.VegaTime)

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	withdrawalProto.Status = vega.Withdrawal_STATUS_FINALIZED
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 2, rowCount)

	var status entities.WithdrawalStatus
	err = pgxscan.Get(ctx, conn, &status, `select status from withdrawals where id = $1 and vega_time = $2`, withdrawal.ID, withdrawal.VegaTime)
	assert.NoError(t, err)
	assert.Equal(t, entities.WithdrawalStatusFinalized, status)
}

func testWithdrawalsGetByID(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ws, conn := setupWithdrawalStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	withdrawalProto := getTestWithdrawal(testID, testID, testID, testAmount, testID, block.VegaTime)

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	withdrawalProto.Status = vega.Withdrawal_STATUS_FINALIZED
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)

	got, err := ws.GetByID(ctx, withdrawalProto.Id)
	assert.NoError(t, err)

	// We need to truncate the timestamp because the postgres database will truncate to microseconds
	withdrawal.Expiry = withdrawal.Expiry.Truncate(time.Microsecond)
	withdrawal.CreatedTimestamp = withdrawal.CreatedTimestamp.Truncate(time.Microsecond)
	withdrawal.WithdrawnTimestamp = withdrawal.WithdrawnTimestamp.Truncate(time.Microsecond)

	assert.Equal(t, *withdrawal, got)
}

func testWithdrawalsGetByParty(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	bs, ws, conn := setupWithdrawalStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	block := addTestBlock(t, bs)
	withdrawalProto1 := getTestWithdrawal(testID, testID, testID, testAmount, testID, block.VegaTime)
	withdrawalProto1.Id = "deadbeef01"

	withdrawalProto2 := getTestWithdrawal(testID, testID, testID, testAmount, testID, block.VegaTime)
	withdrawalProto2.Id = "deadbeef02"

	want := make([]entities.Withdrawal, 0)

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto1, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	withdrawalProto1.Status = vega.Withdrawal_STATUS_FINALIZED
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto1, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)

	withdrawal.Expiry = withdrawal.Expiry.Truncate(time.Microsecond)
	withdrawal.CreatedTimestamp = withdrawal.CreatedTimestamp.Truncate(time.Microsecond)
	withdrawal.WithdrawnTimestamp = withdrawal.WithdrawnTimestamp.Truncate(time.Microsecond)

	want = append(want, *withdrawal)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto2, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto2, block.VegaTime)
	withdrawalProto2.Status = vega.Withdrawal_STATUS_FINALIZED
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(context.Background(), withdrawal)
	require.NoError(t, err)

	withdrawal.Expiry = withdrawal.Expiry.Truncate(time.Microsecond)
	withdrawal.CreatedTimestamp = withdrawal.CreatedTimestamp.Truncate(time.Microsecond)
	withdrawal.WithdrawnTimestamp = withdrawal.WithdrawnTimestamp.Truncate(time.Microsecond)

	want = append(want, *withdrawal)

	got, _, _ := ws.GetByParty(ctx, withdrawalProto1.PartyId, false, entities.OffsetPagination{})

	assert.Equal(t, want, got)
}

func getTestWithdrawal(id, party, asset, amount, txHash string, ts time.Time) *vega.Withdrawal {
	return &vega.Withdrawal{
		Id:                 id,
		PartyId:            party,
		Amount:             amount,
		Asset:              asset,
		Status:             vega.Withdrawal_STATUS_OPEN,
		Ref:                "deadbeef",
		Expiry:             ts.Unix() + 1,
		TxHash:             txHash,
		CreatedTimestamp:   ts.UnixNano(),
		WithdrawnTimestamp: ts.UnixNano(),
		Ext: &vega.WithdrawExt{
			Ext: &vega.WithdrawExt_Erc20{
				Erc20: &vega.Erc20WithdrawExt{
					ReceiverAddress: "0x1234",
				},
			},
		},
	}
}

func addWithdrawals(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, ws *sqlstore.Withdrawals) []entities.Withdrawal {
	vegaTime := time.Now().Truncate(time.Microsecond)
	amount := int64(1000)
	withdrawals := make([]entities.Withdrawal, 0, 10)
	for i := 0; i < 10; i++ {
		addTestBlockForTime(t, bs, vegaTime)

		withdrawalProto := getTestWithdrawal(fmt.Sprintf("deadbeef%02d", i+1), testID, testID,
			strconv.FormatInt(amount, 10), generateID(), vegaTime)
		withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, vegaTime)
		require.NoError(t, err, "Converting withdrawal proto to database entity")
		err = ws.Upsert(ctx, withdrawal)
		withdrawals = append(withdrawals, *withdrawal)
		require.NoError(t, err)

		vegaTime = vegaTime.Add(time.Second)
		amount += 100
	}

	return withdrawals
}

func testWithdrawalsPaginationNoPagination(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ws, _ := setupWithdrawalStoreTests(t, timeoutCtx)

	testWithdrawals := addWithdrawals(timeoutCtx, t, bs, ws)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil)
	require.NoError(t, err)
	got, pageInfo, err := ws.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	assert.Equal(t, testWithdrawals, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[0].VegaTime,
		ID:       testWithdrawals[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[9].VegaTime,
		ID:       testWithdrawals[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testWithdrawalsPaginationFirst(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ws, _ := setupWithdrawalStoreTests(t, timeoutCtx)

	testWithdrawals := addWithdrawals(timeoutCtx, t, bs, ws)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil)
	require.NoError(t, err)
	got, pageInfo, err := ws.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testWithdrawals[:3]
	assert.Equal(t, want, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[0].VegaTime,
		ID:       testWithdrawals[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[2].VegaTime,
		ID:       testWithdrawals[2].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testWithdrawalsPaginationLast(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ws, _ := setupWithdrawalStoreTests(t, timeoutCtx)

	testWithdrawals := addWithdrawals(timeoutCtx, t, bs, ws)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil)
	require.NoError(t, err)
	got, pageInfo, err := ws.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testWithdrawals[7:]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[7].VegaTime,
		ID:       testWithdrawals[7].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[9].VegaTime,
		ID:       testWithdrawals[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testWithdrawalsPaginationFirstAfter(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ws, _ := setupWithdrawalStoreTests(t, timeoutCtx)

	testWithdrawals := addWithdrawals(timeoutCtx, t, bs, ws)

	first := int32(3)
	after := entities.NewCursor(entities.DepositCursor{
		VegaTime: testWithdrawals[2].VegaTime,
		ID:       testWithdrawals[2].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil)
	require.NoError(t, err)
	got, pageInfo, err := ws.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testWithdrawals[3:6]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[3].VegaTime,
		ID:       testWithdrawals[3].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[5].VegaTime,
		ID:       testWithdrawals[5].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testWithdrawalsPaginationLastBefore(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs, ws, _ := setupWithdrawalStoreTests(t, timeoutCtx)

	testWithdrawals := addWithdrawals(timeoutCtx, t, bs, ws)

	last := int32(3)
	before := entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[7].VegaTime,
		ID:       testWithdrawals[7].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before)
	require.NoError(t, err)
	got, pageInfo, err := ws.GetByParty(timeoutCtx, testID, false, pagination)

	require.NoError(t, err)
	want := testWithdrawals[4:7]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[4].VegaTime,
		ID:       testWithdrawals[4].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testWithdrawals[6].VegaTime,
		ID:       testWithdrawals[6].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}
