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
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransfers(t *testing.T) {
	t.Run("Retrieve transfers to or from a party", testTransfersGetTransferToOrFromParty)
	t.Run("Retrieve transfer to and from a party ", testTransfersGetTransfersByParty)
	t.Run("Retrieve transfer to and from an account", testTransfersGetFromAccountAndGetToAccount)
	t.Run("Retrieves latest transfer version after updates in different block", testTransfersUpdatesInDifferentBlocks)
	t.Run("Retrieves latest transfer version after updates in different block", testTransfersUpdateInSameBlock)
	t.Run("Test add and retrieve of one off transfer", testTransfersAddAndRetrieveOneOffTransfer)
	t.Run("Test add and retrieve of recurring transfer", testTransfersAddAndRetrieveRecurringTransfer)
}

func TestTransfersPagination(t *testing.T) {
	t.Run("should return all transfers if no pagination is specified", testTransferPaginationNoPagination)
	t.Run("should return the first page of results if first is provided", testTransferPaginationFirst)
	t.Run("should return the last page of results if last is provided", testTransferPaginationLast)
	t.Run("should return the specified page of results if first and after are provided", testTransferPaginationFirstAfter)
	t.Run("should return the specified page of results if last and before are provided", testTransferPaginationLastBefore)
}

func testTransfersGetTransferToOrFromParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	now := time.Now()
	block := getTestBlock(t, ctx, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	sourceTransferProto := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountTo.PartyID.String(),
		FromAccountType: accountTo.Type,
		To:              accountFrom.PartyID.String(),
		ToAccountType:   accountFrom.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.Transfer_Recurring{Recurring: &eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
			DispatchStrategy: &vega.DispatchStrategy{
				AssetForMetric: "deadd0d0",
				Markets:        []string{"beefdead", "feebaad"},
				Metric:         vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			},
		}},
	}

	transfer, err := entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	assert.NoError(t, err)
	err = transfers.Upsert(ctx, transfer)
	assert.NoError(t, err)

	sourceTransferProto2 := &eventspb.Transfer{
		Id:              "deadd0d1",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref2",
		Status:          eventspb.Transfer_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.Transfer_Recurring{Recurring: &eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transfer, err = entities.TransferFromProto(ctx, sourceTransferProto2, generateTxHash(), block.VegaTime, accounts)
	assert.NoError(t, err)
	err = transfers.Upsert(ctx, transfer)
	assert.NoError(t, err)

	retrieved, _, err := transfers.GetTransfersToOrFromParty(ctx, accountTo.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 2, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

	retrievedTransferProto, _ = retrieved[1].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)
}

func testTransfersGetTransfersByParty(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	now := time.Now()
	block := getTestBlock(t, ctx, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	reason := "some terrible reason"
	sourceTransferProto := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.Transfer_Recurring{Recurring: &eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
			DispatchStrategy: &vega.DispatchStrategy{
				AssetForMetric: "deadd0d0",
				Markets:        []string{"beefdead", "feebaad"},
				Metric:         vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			},
		}},
		Reason: &reason,
	}

	transfer, _ := entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	sourceTransferProto2 := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.Transfer_Recurring{Recurring: &eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transfer, _ = entities.TransferFromProto(ctx, sourceTransferProto2, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	retrieved, _, err := transfers.GetTransfersFromParty(ctx, accountFrom.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, _, err = transfers.GetTransfersToParty(ctx, accountTo.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)
}

func testTransfersGetFromAccountAndGetToAccount(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	now := time.Now()
	block := getTestBlock(t, ctx, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	sourceTransferProto1 := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.Transfer_Recurring{Recurring: &eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transfer, _ := entities.TransferFromProto(ctx, sourceTransferProto1, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	sourceTransferProto2 := &eventspb.Transfer{
		Id:              "deadd0d1",
		From:            accountTo.PartyID.String(),
		FromAccountType: accountTo.Type,
		To:              accountFrom.PartyID.String(),
		ToAccountType:   accountFrom.Type,
		Asset:           accountTo.AssetID.String(),
		Amount:          "50",
		Reference:       "Ref2",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.Transfer_Recurring{Recurring: &eventspb.RecurringTransfer{
			StartEpoch: 45,
			EndEpoch:   toPtr(uint64(56)),
			Factor:     "3.12",
		}},
	}

	transfer, _ = entities.TransferFromProto(ctx, sourceTransferProto2, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 2, len(retrieved))

	retrieved, _, _ = transfers.GetTransfersFromAccount(ctx, accountFrom.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _, _ = transfers.GetTransfersToAccount(ctx, accountTo.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _, _ = transfers.GetTransfersFromAccount(ctx, accountTo.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, _, _ = transfers.GetTransfersToAccount(ctx, accountFrom.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)
}

func testTransfersUpdatesInDifferentBlocks(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	now := time.Now()
	block := getTestBlock(t, ctx, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	deliverOn := block.VegaTime.Add(1 * time.Hour)

	sourceTransferProto := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.UnixNano()}},
	}

	transfer, _ := entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	block = getTestBlock(t, ctx, block.VegaTime.Add(1*time.Microsecond))
	deliverOn = deliverOn.Add(1 * time.Minute)
	sourceTransferProto = &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "40",
		Reference:       "Ref2",
		Status:          eventspb.Transfer_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.UnixNano()}},
	}
	transfer, _ = entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func testTransfersUpdateInSameBlock(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	now := time.Now()
	block := getTestBlock(t, ctx, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	deliverOn := block.VegaTime.Add(1 * time.Hour)

	sourceTransferProto := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.UnixNano()}},
	}

	transfer, _ := entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	deliverOn = deliverOn.Add(1 * time.Minute)
	sourceTransferProto = &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "40",
		Reference:       "Ref2",
		Status:          eventspb.Transfer_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.UnixNano()}},
	}
	transfer, _ = entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func testTransfersAddAndRetrieveOneOffTransfer(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	now := time.Now()
	block := getTestBlock(t, ctx, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	deliverOn := block.VegaTime.Add(1 * time.Hour)

	sourceTransferProto := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.UnixNano()}},
	}

	transfer, _ := entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)
	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func testTransfersAddAndRetrieveRecurringTransfer(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	now := time.Now()
	block := getTestBlock(t, ctx, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	sourceTransferProto := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.Transfer_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.Transfer_Recurring{Recurring: &eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transfer, _ := entities.TransferFromProto(ctx, sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transfers.Upsert(ctx, transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(ctx, accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func getTestBlock(t *testing.T, ctx context.Context, testTime time.Time) entities.Block {
	t.Helper()
	blocks := sqlstore.NewBlocks(connectionSource)
	vegaTime := time.UnixMicro(testTime.UnixMicro())
	block := addTestBlockForTime(t, ctx, blocks, vegaTime)
	return block
}

func getTestAccounts(t *testing.T, ctx context.Context, accounts *sqlstore.Accounts, block entities.Block) (accountFrom entities.Account,
	accountTo entities.Account,
) {
	t.Helper()
	assets := sqlstore.NewAssets(connectionSource)

	testAssetID := entities.AssetID(helpers.GenerateID())
	testAsset := entities.Asset{
		ID:            testAssetID,
		Name:          "testAssetName",
		Symbol:        "tan",
		Decimals:      1,
		Quantum:       decimal.NewFromInt(1),
		Source:        "TS",
		ERC20Contract: "ET",
		VegaTime:      block.VegaTime,
	}

	err := assets.Add(ctx, testAsset)
	if err != nil {
		t.Fatalf("failed to add test asset:%s", err)
	}

	accountFrom = entities.Account{
		PartyID:  entities.PartyID(helpers.GenerateID()),
		AssetID:  testAssetID,
		Type:     vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
		VegaTime: block.VegaTime,
	}
	err = accounts.Obtain(ctx, &accountFrom)
	if err != nil {
		t.Fatalf("failed to obtain from account:%s", err)
	}

	accountTo = entities.Account{
		PartyID: entities.PartyID(helpers.GenerateID()),
		AssetID: testAssetID,

		Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
		VegaTime: block.VegaTime,
	}
	err = accounts.Obtain(ctx, &accountTo)
	if err != nil {
		t.Fatalf("failed to obtain to account:%s", err)
	}

	return accountFrom, accountTo
}

func testTransferPaginationNoPagination(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)

	testTransfers := addTransfers(ctx, t, bs, transfers)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(ctx, pagination)

	require.NoError(t, err)
	assert.Equal(t, testTransfers, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[0].VegaTime,
		ID:       testTransfers[0].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[9].VegaTime,
		ID:       testTransfers[9].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationFirst(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()
	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)

	testTransfers := addTransfers(ctx, t, bs, transfers)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(ctx, pagination)

	require.NoError(t, err)
	want := testTransfers[:3]
	assert.Equal(t, want, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[0].VegaTime,
		ID:       testTransfers[0].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[2].VegaTime,
		ID:       testTransfers[2].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationLast(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)
	testTransfers := addTransfers(ctx, t, bs, transfers)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(ctx, pagination)

	require.NoError(t, err)
	want := testTransfers[7:]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[7].VegaTime,
		ID:       testTransfers[7].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[9].VegaTime,
		ID:       testTransfers[9].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationFirstAfter(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)
	testTransfers := addTransfers(ctx, t, bs, transfers)

	first := int32(3)
	after := testTransfers[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(ctx, pagination)

	require.NoError(t, err)
	want := testTransfers[3:6]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[3].VegaTime,
		ID:       testTransfers[3].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[5].VegaTime,
		ID:       testTransfers[5].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationLastBefore(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)
	testTransfers := addTransfers(ctx, t, bs, transfers)

	last := int32(3)
	before := entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[7].VegaTime,
		ID:       testTransfers[7].ID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(ctx, pagination)

	require.NoError(t, err)
	want := testTransfers[4:7]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[4].VegaTime,
		ID:       testTransfers[4].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferCursor{
		VegaTime: testTransfers[6].VegaTime,
		ID:       testTransfers[6].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func addTransfers(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, transferStore *sqlstore.Transfers) []entities.Transfer {
	t.Helper()
	vegaTime := time.Now().Truncate(time.Microsecond)
	block := addTestBlockForTime(t, ctx, bs, vegaTime)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, ctx, accounts, block)

	transfers := make([]entities.Transfer, 0, 10)
	for i := 0; i < 10; i++ {
		vegaTime = vegaTime.Add(time.Second)
		addTestBlockForTime(t, ctx, bs, vegaTime)

		amount, _ := decimal.NewFromString("10")
		transfer := entities.Transfer{
			ID:                  entities.TransferID(fmt.Sprintf("deadbeef%02d", i+1)),
			VegaTime:            vegaTime,
			FromAccountID:       accountFrom.ID,
			ToAccountID:         accountTo.ID,
			AssetID:             entities.AssetID(""),
			Amount:              amount,
			Reference:           "",
			Status:              0,
			TransferType:        0,
			DeliverOn:           nil,
			StartEpoch:          nil,
			EndEpoch:            nil,
			Factor:              nil,
			DispatchMetric:      nil,
			DispatchMetricAsset: nil,
			DispatchMarkets:     nil,
		}

		err := transferStore.Upsert(ctx, &transfer)
		require.NoError(t, err)
		transfers = append(transfers, transfer)
	}

	return transfers
}

func toPtr[T any](t T) *T {
	return &t
}
