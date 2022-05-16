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

func TestDeposits(t *testing.T) {
	t.Run("Upsert should insert deposits if one doesn't exist for the block", testAddDepositForNewBlock)
	t.Run("Upsert should error if the vega block does not exist", testErrorIfBlockDoesNotExist)
	t.Run("Upsert should update deposits if one already exists for the block", testUpdateDepositForBlockIfExists)
	t.Run("Upsert should insert deposit updates if the same deposit id is inserted in a different block", testInsertDepositUpdatesIfNewBlock)
	t.Run("GetByID should retrieve the latest state of the deposit with the given ID", testDepositsGetByID)
	t.Run("GetByParty should retrieve the latest state of all deposits for a given party", testDepositsGetByParty)
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
	depositProto := getTestDeposit()

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
	depositProto := getTestDeposit()

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
	depositProto := getTestDeposit()

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
	depositProto := getTestDeposit()

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
	depositProto := getTestDeposit()

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
	depositProto1 := getTestDeposit()
	depositProto1.Id = "deadbeef01"

	depositProto2 := getTestDeposit()
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

	got := ds.GetByParty(ctx, depositProto1.PartyId, false, entities.Pagination{})

	assert.Equal(t, want, got)
}

func getTestDeposit() *vega.Deposit {
	now := time.Now().UnixNano()
	return &vega.Deposit{
		Id:                "deadbeef",
		Status:            vega.Deposit_STATUS_OPEN,
		PartyId:           "deadbeef",
		Asset:             "deadbeef",
		Amount:            "1000",
		TxHash:            "deadbeef",
		CreditedTimestamp: now,
		CreatedTimestamp:  now,
	}
}
