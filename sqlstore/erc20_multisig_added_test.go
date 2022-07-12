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

	sa := getTestSignerEvent(t, "fc677151d0c93726", generateEthereumAddress(), "12", true)
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

	err = ms.Add(ctx, getTestSignerEvent(t, vID1, generateEthereumAddress(), "12", true))
	require.NoError(t, err)

	// same validator different epoch
	err = ms.Add(ctx, getTestSignerEvent(t, vID1, generateEthereumAddress(), "24", true))
	require.NoError(t, err)

	// same epoch different validator
	err = ms.Add(ctx, getTestSignerEvent(t, vID2, generateEthereumAddress(), "12", true))
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
	submitter := generateEthereumAddress()
	wrongSubmitter := generateEthereumAddress()

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	err = ms.Add(ctx, getTestSignerEvent(t, vID1, generateEthereumAddress(), "12", true))
	require.NoError(t, err)

	// same validator different epoch
	err = ms.Add(ctx, getTestSignerEvent(t, vID1, submitter, "24", false))
	require.NoError(t, err)

	// same epoch different validator
	err = ms.Add(ctx, getTestSignerEvent(t, vID2, generateEthereumAddress(), "12", true))
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
				NewSigner:   generateEthereumAddress(),
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
				OldSigner:   generateEthereumAddress(),
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
