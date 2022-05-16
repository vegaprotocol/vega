package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	vgcrypto "code.vegaprotocol.io/shared/libs/crypto"
	"github.com/jackc/pgx/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestERC20MultiSigEvent(t *testing.T) {
	t.Run("Adding a single bundle", testAddSigner)
	t.Run("Get with filters", testGetWithFilters)
	t.Run("Get with add and removed events", testGetWithAddAndRemoveEvents)
}

func setupERC20MultiSigEventStoreTests(t *testing.T, ctx context.Context) (*sqlstore.ERC20MultiSigSignerEvent, *pgx.Conn) {
	t.Helper()
	DeleteEverything()
	ms := sqlstore.NewERC20MultiSigSignerEvent(connectionSource)

	config := NewTestConfig(testDBPort)

	conn, err := pgx.Connect(ctx, connectionString(config.ConnectionConfig))
	require.NoError(t, err)

	return ms, conn
}

func testAddSigner(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ms, conn := setupERC20MultiSigEventStoreTests(t, ctx)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	sa := getTestSignerEvent(t, "fc677151d0c93726", vgcrypto.RandomHash(), "12", true)
	err = ms.Add(ctx, sa)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 1, rowCount)

	// now add a duplicate and check we still remain with one
	err = ms.Add(ctx, sa)
	require.NoError(t, err)

	err = conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 1, rowCount)
}

func testGetWithFilters(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ms, conn := setupERC20MultiSigEventStoreTests(t, ctx)

	var rowCount int
	vID1 := "fc677151d0c93726"
	vID2 := "15d1d5fefa8988eb"

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	err = ms.Add(ctx, getTestSignerEvent(t, vID1, vgcrypto.RandomHash(), "12", true))
	require.NoError(t, err)

	// same validator different epoch
	err = ms.Add(ctx, getTestSignerEvent(t, vID1, vgcrypto.RandomHash(), "24", true))
	require.NoError(t, err)

	// same epoch different validator
	err = ms.Add(ctx, getTestSignerEvent(t, vID2, vgcrypto.RandomHash(), "12", true))
	require.NoError(t, err)

	res, err := ms.GetAddedEvents(ctx, vID1, nil, entities.OffsetPagination{})
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, vID1, res[0].ValidatorID.String())
	assert.Equal(t, vID1, res[1].ValidatorID.String())

	epoch := int64(12)
	res, err = ms.GetAddedEvents(ctx, vID1, &epoch, entities.OffsetPagination{})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, vID1, res[0].ValidatorID.String())
	assert.Equal(t, int64(12), res[0].EpochID)
}

func testGetWithAddAndRemoveEvents(t *testing.T) {
	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	ms, conn := setupERC20MultiSigEventStoreTests(t, ctx)

	var rowCount int
	vID1 := "fc677151d0c93726"
	vID2 := "15d1d5fefa8988eb"
	submitter := "15d1d5fefa8988bb"
	wrongSubmitter := "15d155fefa8988bb"

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	err = ms.Add(ctx, getTestSignerEvent(t, vID1, vgcrypto.RandomHash(), "12", true))
	require.NoError(t, err)

	// same validator different epoch
	err = ms.Add(ctx, getTestSignerEvent(t, vID1, submitter, "24", false))
	require.NoError(t, err)

	// same epoch different validator
	err = ms.Add(ctx, getTestSignerEvent(t, vID2, vgcrypto.RandomHash(), "12", true))
	require.NoError(t, err)

	res, err := ms.GetAddedEvents(ctx, vID1, nil, entities.OffsetPagination{})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, vID1, res[0].ValidatorID.String())

	res, err = ms.GetRemovedEvents(ctx, vID1, submitter, nil, entities.OffsetPagination{})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, vID1, res[0].ValidatorID.String())

	res, err = ms.GetRemovedEvents(ctx, vID1, wrongSubmitter, nil, entities.OffsetPagination{})
	require.NoError(t, err)
	require.Len(t, res, 0)
}

func getTestSignerEvent(t *testing.T, validatorID string, submitter string, epochSeq string, addedEvent bool) *entities.ERC20MultiSigSignerEvent {
	t.Helper()
	vgcrypto.RandomHash()

	var err error
	var evt *entities.ERC20MultiSigSignerEvent
	switch addedEvent {
	case true:
		evt, err = entities.ERC20MultiSigSignerEventFromAddedProto(
			&eventspb.ERC20MultiSigSignerAdded{
				SignatureId: vgcrypto.RandomHash(),
				ValidatorId: validatorID,
				NewSigner:   vgcrypto.RandomHash(),
				Submitter:   submitter,
				Nonce:       "nonce",
				EpochSeq:    epochSeq,
				Timestamp:   time.Unix(10000, 13).UnixNano(),
			},
		)
		require.NoError(t, err)
	case false:
		evts, err := entities.ERC20MultiSigSignerEventFromRemovedProto(
			&eventspb.ERC20MultiSigSignerRemoved{
				SignatureSubmitters: []*eventspb.ERC20MulistSigSignerRemovedSubmitter{
					{
						SignatureId: vgcrypto.RandomHash(),
						Submitter:   submitter,
					},
				},
				ValidatorId: validatorID,
				OldSigner:   vgcrypto.RandomHash(),
				Nonce:       "nonce",
				EpochSeq:    epochSeq,
				Timestamp:   time.Unix(10000, 13).UnixNano(),
			},
		)
		require.NoError(t, err)
		evt = evts[0]
	}

	require.NoError(t, err)
	return evt
}
