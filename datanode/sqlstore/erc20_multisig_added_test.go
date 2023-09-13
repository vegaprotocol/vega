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
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestERC20MultiSigEvent(t *testing.T) {
	t.Run("Adding a single bundle", testAddSigner)
	t.Run("Get with filters", testGetWithFilters)
	t.Run("Get with add and removed events", testGetWithAddAndRemoveEvents)
	t.Run("Get by tx hash with add and removed events", testGetByTxHashWithAddAndRemoveEvents)
}

func setupERC20MultiSigEventStoreTests(t *testing.T) (*sqlstore.ERC20MultiSigSignerEvent, sqlstore.Connection) {
	t.Helper()
	ms := sqlstore.NewERC20MultiSigSignerEvent(connectionSource)
	return ms, connectionSource.Connection
}

func testAddSigner(t *testing.T) {
	ctx := tempTransaction(t)

	ms, conn := setupERC20MultiSigEventStoreTests(t)

	var rowCount int

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	sa := getTestSignerEvent(t, vgcrypto.RandomHash(), "fc677151d0c93726", generateEthereumAddress(), "12", true)
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
	ctx := tempTransaction(t)

	ms, conn := setupERC20MultiSigEventStoreTests(t)

	var rowCount int
	vID1 := "fc677151d0c93726"
	vID2 := "15d1d5fefa8988eb"

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	vgcrypto.RandomHash()
	err = ms.Add(ctx, getTestSignerEvent(t, vgcrypto.RandomHash(), vID1, generateEthereumAddress(), "12", true))
	require.NoError(t, err)

	// same validator different epoch
	err = ms.Add(ctx, getTestSignerEvent(t, vgcrypto.RandomHash(), vID1, generateEthereumAddress(), "24", true))
	require.NoError(t, err)

	// same epoch different validator
	err = ms.Add(ctx, getTestSignerEvent(t, vgcrypto.RandomHash(), vID2, generateEthereumAddress(), "12", true))
	require.NoError(t, err)

	res, _, err := ms.GetAddedEvents(ctx, vID1, "", nil, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 2)
	assert.Equal(t, vID1, res[0].ValidatorID.String())
	assert.Equal(t, vID1, res[1].ValidatorID.String())

	epoch := int64(12)
	res, _, err = ms.GetAddedEvents(ctx, vID1, "", &epoch, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, vID1, res[0].ValidatorID.String())
	assert.Equal(t, int64(12), res[0].EpochID)
}

func testGetWithAddAndRemoveEvents(t *testing.T) {
	ctx := tempTransaction(t)

	ms, conn := setupERC20MultiSigEventStoreTests(t)

	var rowCount int
	vID1 := "fc677151d0c93726"
	vID2 := "15d1d5fefa8988eb"
	submitter := generateEthereumAddress()
	wrongSubmitter := generateEthereumAddress()

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	err = ms.Add(ctx, getTestSignerEvent(t, vgcrypto.RandomHash(), vID1, generateEthereumAddress(), "12", true))
	require.NoError(t, err)

	// same validator different epoch
	err = ms.Add(ctx, getTestSignerEvent(t, vgcrypto.RandomHash(), vID1, submitter, "24", false))
	require.NoError(t, err)

	// same epoch different validator
	err = ms.Add(ctx, getTestSignerEvent(t, vgcrypto.RandomHash(), vID2, generateEthereumAddress(), "12", true))
	require.NoError(t, err)

	res, _, err := ms.GetAddedEvents(ctx, vID1, "", nil, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, vID1, res[0].ValidatorID.String())

	res, _, err = ms.GetRemovedEvents(ctx, vID1, submitter, nil, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, vID1, res[0].ValidatorID.String())

	res, _, err = ms.GetRemovedEvents(ctx, vID1, wrongSubmitter, nil, entities.CursorPagination{})
	require.NoError(t, err)
	require.Len(t, res, 0)
}

func testGetByTxHashWithAddAndRemoveEvents(t *testing.T) {
	ctx := tempTransaction(t)

	ms, conn := setupERC20MultiSigEventStoreTests(t)

	var rowCount int
	vID1 := "fc677151d0c93726"
	submitter := generateEthereumAddress()

	err := conn.QueryRow(ctx, `select count(*) from erc20_multisig_signer_events`).Scan(&rowCount)
	require.NoError(t, err)
	assert.Equal(t, 0, rowCount)

	event1 := getTestSignerEvent(t, vgcrypto.RandomHash(), vID1, submitter, "12", true)
	err = ms.Add(ctx, event1)
	require.NoError(t, err)

	event2 := getTestSignerEvent(t, vgcrypto.RandomHash(), vID1, submitter, "24", false)
	err = ms.Add(ctx, event2)
	require.NoError(t, err)

	res, err := ms.GetAddedByTxHash(ctx, event1.TxHash)
	require.NoError(t, err)
	require.Len(t, res, 1)
	assert.Equal(t, event1.TxHash, res[0].TxHash)
	assert.Equal(t, event1.Event, res[0].Event)

	res2, err := ms.GetRemovedByTxHash(ctx, event2.TxHash)
	require.NoError(t, err)
	require.Len(t, res2, 1)
	assert.Equal(t, event2.TxHash, res2[0].TxHash)
	assert.Equal(t, event2.Event, res2[0].Event)
}

func getTestSignerEvent(t *testing.T, signatureID string, validatorID string, submitter string, epochSeq string, addedEvent bool) *entities.ERC20MultiSigSignerEvent {
	t.Helper()

	var err error
	var evt *entities.ERC20MultiSigSignerEvent
	switch addedEvent {
	case true:
		evt, err = entities.ERC20MultiSigSignerEventFromAddedProto(
			&eventspb.ERC20MultiSigSignerAdded{
				SignatureId: signatureID,
				ValidatorId: validatorID,
				NewSigner:   generateEthereumAddress(),
				Submitter:   submitter,
				Nonce:       "nonce",
				EpochSeq:    epochSeq,
				Timestamp:   time.Unix(10000000, 13000).UnixNano(),
			},
			generateTxHash(),
		)
		require.NoError(t, err)
	case false:
		evts, err := entities.ERC20MultiSigSignerEventFromRemovedProto(
			&eventspb.ERC20MultiSigSignerRemoved{
				SignatureSubmitters: []*eventspb.ERC20MultiSigSignerRemovedSubmitter{
					{
						SignatureId: signatureID,
						Submitter:   submitter,
					},
				},
				ValidatorId: validatorID,
				OldSigner:   generateEthereumAddress(),
				Nonce:       "nonce",
				EpochSeq:    epochSeq,
				Timestamp:   time.Unix(10000000, 13000).UnixNano(),
			},
			generateTxHash(),
		)
		require.NoError(t, err)
		evt = evts[0]
	}

	require.NoError(t, err)
	return evt
}

func TestERC20MultiSigEventPagination(t *testing.T) {
	t.Run("should return all added events if no pagination is specified", testERC20MultiSigAddedEventPaginationNoPagination)
	t.Run("should return first page of added events if first pagination is specified", testERC20MultiSigAddedEventPaginationFirst)
	t.Run("should return last page of added events if last pagination is specified", testERC20MultiSigAddedEventPaginationLast)
	t.Run("should return specified page of added events if first and after pagination is specified", testERC20MultiSigAddedEventPaginationFirstAfter)
	t.Run("should return specified page of added events if last and before pagination is specified", testERC20MultiSigAddedEventPaginationLastBefore)

	t.Run("should return all added events if no pagination is specified - newest first", testERC20MultiSigAddedEventPaginationNoPaginationNewestFirst)
	t.Run("should return first page of added events if first pagination is specified - newest first", testERC20MultiSigAddedEventPaginationFirstNewestFirst)
	t.Run("should return last page of added events if last pagination is specified - newest first", testERC20MultiSigAddedEventPaginationLastNewestFirst)
	t.Run("should return specified page of added events if first and after pagination is specified - newest first", testERC20MultiSigAddedEventPaginationFirstAfterNewestFirst)
	t.Run("should return specified page of added events if last and before pagination is specified - newest first", testERC20MultiSigAddedEventPaginationLastBeforeNewestFirst)

	t.Run("should return all removed events if no pagination is specified", testERC20MultiSigRemovedEventPaginationNoPagination)
	t.Run("should return first page of removed events if first pagination is specified", testERC20MultiSigRemovedEventPaginationFirst)
	t.Run("should return last page of removed events if last pagination is specified", testERC20MultiSigRemovedEventPaginationLast)
	t.Run("should return specified page of removed events if first and after pagination is specified", testERC20MultiSigRemovedEventPaginationFirstAfter)
	t.Run("should return specified page of removed events if last and before pagination is specified", testERC20MultiSigRemovedEventPaginationLastBefore)

	t.Run("should return all removed events if no pagination is specified - newest first", testERC20MultiSigRemovedEventPaginationNoPaginationNewestFirst)
	t.Run("should return first page of removed events if first pagination is specified - newest first", testERC20MultiSigRemovedEventPaginationFirstNewestFirst)
	t.Run("should return last page of removed events if last pagination is specified - newest first", testERC20MultiSigRemovedEventPaginationLastNewestFirst)
	t.Run("should return specified page of removed events if first and after pagination is specified - newest first", testERC20MultiSigRemovedEventPaginationFirstAfterNewestFirst)
	t.Run("should return specified page of removed events if last and before pagination is specified - newest first", testERC20MultiSigRemovedEventPaginationLastBeforeNewestFirst)
}

func testERC20MultiSigAddedEventPaginationNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := events[:10]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := events[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationLast(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := events[7:10]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationFirstAfter(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	after := events[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := events[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationLastBefore(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	before := events[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := events[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[:10])
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationFirstNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[:10])[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationLastNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[:10])[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationFirstAfterNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	after := events[7].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[:10])[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigAddedEventPaginationLastBeforeNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, _ := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	before := events[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetAddedEvents(ctx, validator, "", nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[:10])[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationNoPagination(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := events[10:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := events[10:13]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationLast(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := events[17:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationFirstAfter(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	after := events[12].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := events[13:16]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationLastBefore(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	before := events[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := events[14:17]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationNoPaginationNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[10:])
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[9].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationFirstNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[10:])[:3]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: false,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationLastNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[10:])[7:]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     false,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationFirstAfterNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	first := int32(3)
	after := events[17].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[10:])[3:6]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func testERC20MultiSigRemovedEventPaginationLastBeforeNewestFirst(t *testing.T) {
	ctx := tempTransaction(t)

	es, events, submitter := setupERC20MultiSigEventStorePaginationTests(t, ctx)
	last := int32(3)
	before := events[12].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, true)
	require.NoError(t, err)

	validator := "fc677151d0c93726"
	got, pageInfo, err := es.GetRemovedEvents(ctx, validator, submitter, nil, pagination)
	require.NoError(t, err)
	want := entities.ReverseSlice(events[10:])[4:7]
	assert.Equal(t, want, got)
	assert.Equal(t, entities.PageInfo{
		HasNextPage:     true,
		HasPreviousPage: true,
		StartCursor:     want[0].Cursor().Encode(),
		EndCursor:       want[2].Cursor().Encode(),
	}, pageInfo)
}

func setupERC20MultiSigEventStorePaginationTests(t *testing.T, ctx context.Context) (*sqlstore.ERC20MultiSigSignerEvent, []entities.ERC20MultiSigSignerEvent, string) {
	t.Helper()

	es := sqlstore.NewERC20MultiSigSignerEvent(connectionSource)

	validator := "fc677151d0c93726"
	submitter := generateEthereumAddress()

	events := make([]entities.ERC20MultiSigSignerEvent, 20)
	for i := 0; i < 10; i++ {
		e := getTestSignerEvent(t, fmt.Sprintf("deadbeef%02d", i+1), validator, submitter, fmt.Sprintf("%d", i+1), true)
		err := es.Add(ctx, e)
		require.NoError(t, err)
		events[i] = *e
	}

	for i := 0; i < 10; i++ {
		e := getTestSignerEvent(t, fmt.Sprintf("deadbeef%02d", 10+i+1), validator, submitter, fmt.Sprintf("%d", 10+i+1), false)
		err := es.Add(ctx, e)
		require.NoError(t, err)
		events[10+i] = *e
	}

	return es, events, submitter
}
