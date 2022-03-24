package sqlstore_test

import (
	"context"
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

func setupWithdrawalStoreTests(t *testing.T, ctx context.Context) (*sqlstore.Blocks, *sqlstore.Withdrawals, *pgx.Conn) {
	t.Helper()
	err := testStore.DeleteEverything()
	require.NoError(t, err)

	bs := sqlstore.NewBlocks(testStore)
	ws := sqlstore.NewWithdrawals(testStore)

	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, connectionString(config))
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
	withdrawalProto := getTestWithdrawal()

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
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
	withdrawalProto := getTestWithdrawal()

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime.Add(time.Second))
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
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
	withdrawalProto := getTestWithdrawal()

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	withdrawal.Status = entities.WithdrawalStatus(vega.Withdrawal_STATUS_FINALIZED)

	err = ws.Upsert(withdrawal)
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
	withdrawalProto := getTestWithdrawal()

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	withdrawalProto.Status = vega.Withdrawal_STATUS_FINALIZED
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
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
	withdrawalProto := getTestWithdrawal()

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
	require.NoError(t, err)
	err = conn.QueryRow(ctx, `select count(*) from withdrawals`).Scan(&rowCount)
	assert.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	time.Sleep(time.Second)

	block = addTestBlock(t, bs)
	withdrawalProto.Status = vega.Withdrawal_STATUS_FINALIZED
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
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
	withdrawalProto1 := getTestWithdrawal()
	withdrawalProto1.Id = "deadbeef01"

	withdrawalProto2 := getTestWithdrawal()
	withdrawalProto2.Id = "deadbeef02"

	want := make([]entities.Withdrawal, 0)

	withdrawal, err := entities.WithdrawalFromProto(withdrawalProto1, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	withdrawalProto1.Status = vega.Withdrawal_STATUS_FINALIZED
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto1, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
	require.NoError(t, err)

	withdrawal.Expiry = withdrawal.Expiry.Truncate(time.Microsecond)
	withdrawal.CreatedTimestamp = withdrawal.CreatedTimestamp.Truncate(time.Microsecond)
	withdrawal.WithdrawnTimestamp = withdrawal.WithdrawnTimestamp.Truncate(time.Microsecond)

	want = append(want, *withdrawal)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto2, block.VegaTime)
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
	require.NoError(t, err)

	time.Sleep(time.Millisecond * 500)

	block = addTestBlock(t, bs)
	withdrawal, err = entities.WithdrawalFromProto(withdrawalProto2, block.VegaTime)
	withdrawalProto2.Status = vega.Withdrawal_STATUS_FINALIZED
	require.NoError(t, err, "Converting withdrawal proto to database entity")

	err = ws.Upsert(withdrawal)
	require.NoError(t, err)

	withdrawal.Expiry = withdrawal.Expiry.Truncate(time.Microsecond)
	withdrawal.CreatedTimestamp = withdrawal.CreatedTimestamp.Truncate(time.Microsecond)
	withdrawal.WithdrawnTimestamp = withdrawal.WithdrawnTimestamp.Truncate(time.Microsecond)

	want = append(want, *withdrawal)

	got := ws.GetByParty(ctx, withdrawalProto1.PartyId, false, entities.Pagination{})

	assert.Equal(t, want, got)
}

func getTestWithdrawal() *vega.Withdrawal {
	now := time.Now().UnixNano()
	return &vega.Withdrawal{
		Id:                 "deadbeef",
		PartyId:            "deadbeef",
		Amount:             "1000",
		Asset:              "deadbeef",
		Status:             vega.Withdrawal_STATUS_OPEN,
		Ref:                "deadbeef",
		Expiry:             now + 1e9,
		TxHash:             "deadbeef",
		CreatedTimestamp:   now,
		WithdrawnTimestamp: now,
		Ext: &vega.WithdrawExt{
			Ext: &vega.WithdrawExt_Erc20{
				Erc20: &vega.Erc20WithdrawExt{
					ReceiverAddress: "0x1234",
				},
			},
		},
	}
}
