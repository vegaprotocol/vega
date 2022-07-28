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
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransfers(t *testing.T) {
	t.Run("Retrieve transfers to or from a party", testTransfers_GetTransferToOrFromParty)
	t.Run("Retrieve transfer to and from a party ", testTransfers_GetTransfersByParty)
	t.Run("Retrieve transfer to and from an account", testTransfers_GetFromAccountAndGetToAccount)
	t.Run("Retrieves latest transfer version after updates in different block", testTransfers_UpdatesInDifferentBlocks)
	t.Run("Retrieves latest transfer version after updates in different block", testTransfers_UpdateInSameBlock)
	t.Run("Test add and retrieve of one off transfer", testTransfers_AddAndRetrieveOneOffTransfer)
	t.Run("Test add and retrieve of recurring transfer", testTransfers_AddAndRetrieveRecurringTransfer)
}

func TestTransfersPagination(t *testing.T) {
	t.Run("should return all transfers if no pagination is specified", testTransferPaginationNoPagination)
	t.Run("should return the first page of results if first is provided", testTransferPaginationFirst)
	t.Run("should return the last page of results if last is provided", testTransferPaginationLast)
	t.Run("should return the specified page of results if first and after are provided", testTransferPaginationFirstAfter)
	t.Run("should return the specified page of results if last and before are provided", testTransferPaginationLastBefore)
}

func testTransfers_GetTransferToOrFromParty(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

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

	transfer, err := entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	assert.NoError(t, err)
	err = transfers.Upsert(context.Background(), transfer)
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

	transfer, err = entities.TransferFromProto(context.Background(), sourceTransferProto2, block.VegaTime, accounts)
	assert.NoError(t, err)
	err = transfers.Upsert(context.Background(), transfer)
	assert.NoError(t, err)

	retrieved, _, err := transfers.GetTransfersToOrFromParty(ctx, accountTo.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 2, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

	retrievedTransferProto, _ = retrieved[1].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

}

func testTransfers_GetTransfersByParty(t *testing.T) {

	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

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
	}

	transfer, _ := entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

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

	transfer, _ = entities.TransferFromProto(context.Background(), sourceTransferProto2, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

	retrieved, _, err := transfers.GetTransfersFromParty(ctx, accountFrom.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, _, err = transfers.GetTransfersToParty(ctx, accountTo.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)
}

func testTransfers_GetFromAccountAndGetToAccount(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

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

	transfer, _ := entities.TransferFromProto(context.Background(), sourceTransferProto1, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

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

	transfer, _ = entities.TransferFromProto(context.Background(), sourceTransferProto2, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 2, len(retrieved))

	retrieved, _, _ = transfers.GetTransfersFromAccount(ctx, accountFrom.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _, _ = transfers.GetTransfersToAccount(ctx, accountTo.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _, _ = transfers.GetTransfersFromAccount(ctx, accountTo.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, _, _ = transfers.GetTransfersToAccount(ctx, accountFrom.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

}

func testTransfers_UpdatesInDifferentBlocks(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

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
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.Unix()}},
	}

	transfer, _ := entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

	block = getTestBlock(t, block.VegaTime.Add(1*time.Microsecond))
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
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.Unix()}},
	}
	transfer, _ = entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

}

func testTransfers_UpdateInSameBlock(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

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
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.Unix()}},
	}

	transfer, _ := entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

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
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.Unix()}},
	}
	transfer, _ = entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

}

func testTransfers_AddAndRetrieveOneOffTransfer(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

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
		Kind:            &eventspb.Transfer_OneOff{OneOff: &eventspb.OneOffTransfer{DeliverOn: deliverOn.Unix()}},
	}

	transfer, _ := entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)
	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

}

func testTransfers_AddAndRetrieveRecurringTransfer(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := sqlstore.NewTransfers(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

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

	transfer, _ := entities.TransferFromProto(context.Background(), sourceTransferProto, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

	retrieved, _, _ := transfers.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

}

func getTestBlock(t *testing.T, testTime time.Time) entities.Block {
	blocks := sqlstore.NewBlocks(connectionSource)
	vegaTime := time.UnixMicro(testTime.UnixMicro())
	block := addTestBlockForTime(t, blocks, vegaTime)
	return block
}

func getTestAccounts(t *testing.T, accounts *sqlstore.Accounts, block entities.Block) (accountFrom entities.Account,
	accountTo entities.Account) {

	assets := sqlstore.NewAssets(connectionSource)

	testAssetId := entities.AssetID{ID: entities.ID(generateID())}
	testAsset := entities.Asset{
		ID:            testAssetId,
		Name:          "testAssetName",
		Symbol:        "tan",
		TotalSupply:   decimal.NewFromInt(20),
		Decimals:      1,
		Quantum:       decimal.NewFromInt(1),
		Source:        "TS",
		ERC20Contract: "ET",
		VegaTime:      block.VegaTime,
	}

	err := assets.Add(context.Background(), testAsset)
	if err != nil {
		t.Fatalf("failed to add test asset:%s", err)
	}

	accountFrom = entities.Account{
		PartyID:  entities.PartyID{ID: entities.ID(generateID())},
		AssetID:  testAssetId,
		Type:     vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
		VegaTime: block.VegaTime,
	}
	err = accounts.Obtain(context.Background(), &accountFrom)
	if err != nil {
		t.Fatalf("failed to obtain from account:%s", err)
	}

	accountTo = entities.Account{
		PartyID: entities.PartyID{ID: entities.ID(generateID())},
		AssetID: testAssetId,

		Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
		VegaTime: block.VegaTime,
	}
	err = accounts.Obtain(context.Background(), &accountTo)
	if err != nil {
		t.Fatalf("failed to obtain to account:%s", err)
	}

	return
}

func testTransferPaginationNoPagination(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)

	testTransfers := addTransfers(timeoutCtx, t, bs, transfers)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	assert.Equal(t, testTransfers, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[0].VegaTime,
		ID:       testTransfers[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[9].VegaTime,
		ID:       testTransfers[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationFirst(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)

	testTransfers := addTransfers(timeoutCtx, t, bs, transfers)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransfers[:3]
	assert.Equal(t, want, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[0].VegaTime,
		ID:       testTransfers[0].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[2].VegaTime,
		ID:       testTransfers[2].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationLast(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)
	testTransfers := addTransfers(timeoutCtx, t, bs, transfers)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransfers[7:]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[7].VegaTime,
		ID:       testTransfers[7].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[9].VegaTime,
		ID:       testTransfers[9].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationFirstAfter(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)
	testTransfers := addTransfers(timeoutCtx, t, bs, transfers)

	first := int32(3)
	after := entities.NewCursor(entities.DepositCursor{
		VegaTime: testTransfers[2].VegaTime,
		ID:       testTransfers[2].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransfers[3:6]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[3].VegaTime,
		ID:       testTransfers[3].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[5].VegaTime,
		ID:       testTransfers[5].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferPaginationLastBefore(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transfers := sqlstore.NewTransfers(connectionSource)
	testTransfers := addTransfers(timeoutCtx, t, bs, transfers)

	last := int32(3)
	before := entities.NewCursor(entities.LiquidityProvisionCursor{
		VegaTime: testTransfers[7].VegaTime,
		ID:       testTransfers[7].ID.String(),
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := transfers.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransfers[4:7]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[4].VegaTime,
		ID:       testTransfers[4].ID.String(),
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.WithdrawalCursor{
		VegaTime: testTransfers[6].VegaTime,
		ID:       testTransfers[6].ID.String(),
	}.String()).Encode(), pageInfo.EndCursor)
}

func addTransfers(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, transferStore *sqlstore.Transfers) []entities.Transfer {

	vegaTime := time.Now().Truncate(time.Microsecond)
	block := addTestBlockForTime(t, bs, vegaTime)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transfers := make([]entities.Transfer, 0, 10)
	for i := 0; i < 10; i++ {
		vegaTime = vegaTime.Add(time.Second)
		addTestBlockForTime(t, bs, vegaTime)

		amount, _ := decimal.NewFromString("10")
		transfer := entities.Transfer{
			ID:                  entities.NewTransferID(fmt.Sprintf("deadbeef%02d", i+1)),
			VegaTime:            vegaTime,
			FromAccountId:       accountFrom.ID,
			ToAccountId:         accountTo.ID,
			AssetId:             entities.AssetID{},
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
