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
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferInstructions(t *testing.T) {
	t.Run("Retrieve transfer instructions to or from a party", testTransferInstructionsGetTransferToOrFromParty)
	t.Run("Retrieve transfer instruction to and from a party ", testTransferInstructionsGetTransfersByParty)
	t.Run("Retrieve transfer instruction to and from an account", testTransferInstructionsGetFromAccountAndGetToAccount)
	t.Run("Retrieves latest transfer instructionversion after updates in different block", testTransferInstructionsUpdatesInDifferentBlocks)
	t.Run("Retrieves latest transfer instruction version after updates in different block", testTransferInstructionsUpdateInSameBlock)
	t.Run("Test add and retrieve of one off transfer instruction", testTransferInstructionsAddAndRetrieveOneOffTransferInstruction)
	t.Run("Test add and retrieve of recurring transfer instruction", testTransferInstructionsAddAndRetrieveRecurringTransferInstruction)
}

func TestTransferInstructionsPagination(t *testing.T) {
	t.Run("should return all transfer instructions if no pagination is specified", testTransferInstructionPaginationNoPagination)
	t.Run("should return the first page of results if first is provided", testTransferInstructionPaginationFirst)
	t.Run("should return the last page of results if last is provided", testTransferInstructionPaginationLast)
	t.Run("should return the specified page of results if first and after are provided", testTransferInstructionPaginationFirstAfter)
	t.Run("should return the specified page of results if last and before are provided", testTransferInstructionPaginationLastBefore)
}

func testTransferInstructionsGetTransferToOrFromParty(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	sourceTransferProto := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountTo.PartyID.String(),
		FromAccountType: accountTo.Type,
		To:              accountFrom.PartyID.String(),
		ToAccountType:   accountFrom.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.TransferInstruction_Recurring{Recurring: &eventspb.RecurringTransferInstruction{
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

	transferInstruction, err := entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	assert.NoError(t, err)
	err = transferInstructions.Upsert(context.Background(), transferInstruction)
	assert.NoError(t, err)

	sourceTransferProto2 := &eventspb.TransferInstruction{
		Id:              "deadd0d1",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref2",
		Status:          eventspb.TransferInstruction_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.TransferInstruction_Recurring{Recurring: &eventspb.RecurringTransferInstruction{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transferInstruction, err = entities.TransferInstructionFromProto(context.Background(), sourceTransferProto2, generateTxHash(), block.VegaTime, accounts)
	assert.NoError(t, err)

	err = transferInstructions.Upsert(context.Background(), transferInstruction)
	assert.NoError(t, err)

	retrieved, _, err := transferInstructions.GetTransferInstructionsToOrFromParty(ctx, accountTo.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 2, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

	retrievedTransferProto, _ = retrieved[1].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)
}

func testTransferInstructionsGetTransfersByParty(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	sourceTransferProto := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.TransferInstruction_Recurring{Recurring: &eventspb.RecurringTransferInstruction{
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

	transferInstruction, _ := entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	sourceTransferProto2 := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.TransferInstruction_Recurring{Recurring: &eventspb.RecurringTransferInstruction{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transferInstruction, _ = entities.TransferInstructionFromProto(context.Background(), sourceTransferProto2, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	retrieved, _, err := transferInstructions.GetTransferInstructionsFromParty(ctx, accountFrom.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, _, err = transferInstructions.GetTransferInstructionsToParty(ctx, accountTo.PartyID, entities.CursorPagination{})
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)
}

func testTransferInstructionsGetFromAccountAndGetToAccount(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	sourceTransferProto1 := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.TransferInstruction_Recurring{Recurring: &eventspb.RecurringTransferInstruction{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transferInstruction, _ := entities.TransferInstructionFromProto(context.Background(), sourceTransferProto1, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	sourceTransferProto2 := &eventspb.TransferInstruction{
		Id:              "deadd0d1",
		From:            accountTo.PartyID.String(),
		FromAccountType: accountTo.Type,
		To:              accountFrom.PartyID.String(),
		ToAccountType:   accountFrom.Type,
		Asset:           accountTo.AssetID.String(),
		Amount:          "50",
		Reference:       "Ref2",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.TransferInstruction_Recurring{Recurring: &eventspb.RecurringTransferInstruction{
			StartEpoch: 45,
			EndEpoch:   toPtr(uint64(56)),
			Factor:     "3.12",
		}},
	}

	transferInstruction, _ = entities.TransferInstructionFromProto(context.Background(), sourceTransferProto2, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	retrieved, _, _ := transferInstructions.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 2, len(retrieved))

	retrieved, _, _ = transferInstructions.GetTransferInstructionsFromAccount(ctx, accountFrom.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _, _ = transferInstructions.GetTransferInstructionsToAccount(ctx, accountTo.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _, _ = transferInstructions.GetTransferInstructionsFromAccount(ctx, accountTo.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, _, _ = transferInstructions.GetTransferInstructionsToAccount(ctx, accountFrom.ID, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)
}

func testTransferInstructionsUpdatesInDifferentBlocks(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	deliverOn := block.VegaTime.Add(1 * time.Hour)

	sourceTransferProto := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.TransferInstruction_OneOff{OneOff: &eventspb.OneOffTransferInstruction{DeliverOn: deliverOn.Unix()}},
	}

	transferInstruction, _ := entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	block = getTestBlock(t, block.VegaTime.Add(1*time.Microsecond))
	deliverOn = deliverOn.Add(1 * time.Minute)
	sourceTransferProto = &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "40",
		Reference:       "Ref2",
		Status:          eventspb.TransferInstruction_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.TransferInstruction_OneOff{OneOff: &eventspb.OneOffTransferInstruction{DeliverOn: deliverOn.Unix()}},
	}
	transferInstruction, _ = entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	retrieved, _, _ := transferInstructions.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func testTransferInstructionsUpdateInSameBlock(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	deliverOn := block.VegaTime.Add(1 * time.Hour)

	sourceTransferProto := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.TransferInstruction_OneOff{OneOff: &eventspb.OneOffTransferInstruction{DeliverOn: deliverOn.Unix()}},
	}

	transferInstruction, _ := entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	deliverOn = deliverOn.Add(1 * time.Minute)
	sourceTransferProto = &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "40",
		Reference:       "Ref2",
		Status:          eventspb.TransferInstruction_STATUS_DONE,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.TransferInstruction_OneOff{OneOff: &eventspb.OneOffTransferInstruction{DeliverOn: deliverOn.Unix()}},
	}
	transferInstruction, _ = entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	retrieved, _, _ := transferInstructions.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func testTransferInstructionsAddAndRetrieveOneOffTransferInstruction(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	deliverOn := block.VegaTime.Add(1 * time.Hour)

	sourceTransferProto := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind:            &eventspb.TransferInstruction_OneOff{OneOff: &eventspb.OneOffTransferInstruction{DeliverOn: deliverOn.Unix()}},
	}

	transferInstruction, _ := entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)
	retrieved, _, _ := transferInstructions.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func testTransferInstructionsAddAndRetrieveRecurringTransferInstruction(t *testing.T) {
	defer DeleteEverything()

	now := time.Now()
	block := getTestBlock(t, now)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTimeout := time.Second * 10
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	sourceTransferProto := &eventspb.TransferInstruction{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Amount:          "30",
		Reference:       "Ref1",
		Status:          eventspb.TransferInstruction_STATUS_PENDING,
		Timestamp:       block.VegaTime.UnixNano(),
		Kind: &eventspb.TransferInstruction_Recurring{Recurring: &eventspb.RecurringTransferInstruction{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}},
	}

	transferInstruction, _ := entities.TransferInstructionFromProto(context.Background(), sourceTransferProto, generateTxHash(), block.VegaTime, accounts)
	transferInstructions.Upsert(context.Background(), transferInstruction)

	retrieved, _, _ := transferInstructions.GetAll(ctx, entities.CursorPagination{})
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)
}

func getTestBlock(t *testing.T, testTime time.Time) entities.Block {
	t.Helper()
	blocks := sqlstore.NewBlocks(connectionSource)
	vegaTime := time.UnixMicro(testTime.UnixMicro())
	block := addTestBlockForTime(t, blocks, vegaTime)
	return block
}

func getTestAccounts(t *testing.T, accounts *sqlstore.Accounts, block entities.Block) (accountFrom entities.Account,
	accountTo entities.Account,
) {
	t.Helper()
	assets := sqlstore.NewAssets(connectionSource)

	testAssetID := entities.AssetID(generateID())
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

	err := assets.Add(context.Background(), testAsset)
	if err != nil {
		t.Fatalf("failed to add test asset:%s", err)
	}

	accountFrom = entities.Account{
		PartyID:  entities.PartyID(generateID()),
		AssetID:  testAssetID,
		Type:     vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD,
		VegaTime: block.VegaTime,
	}
	err = accounts.Obtain(context.Background(), &accountFrom)
	if err != nil {
		t.Fatalf("failed to obtain from account:%s", err)
	}

	accountTo = entities.Account{
		PartyID: entities.PartyID(generateID()),
		AssetID: testAssetID,

		Type:     vega.AccountType_ACCOUNT_TYPE_GENERAL,
		VegaTime: block.VegaTime,
	}
	err = accounts.Obtain(context.Background(), &accountTo)
	if err != nil {
		t.Fatalf("failed to obtain to account:%s", err)
	}

	return accountFrom, accountTo
}

func testTransferInstructionPaginationNoPagination(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTransferInstructions := addTransferInstructions(timeoutCtx, t, bs, transferInstructions)

	pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transferInstructions.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	assert.Equal(t, testTransferInstructions, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[0].VegaTime,
		ID:       testTransferInstructions[0].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[9].VegaTime,
		ID:       testTransferInstructions[9].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferInstructionPaginationFirst(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	bs := sqlstore.NewBlocks(connectionSource)
	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)

	testTransferInstructions := addTransferInstructions(timeoutCtx, t, bs, transferInstructions)

	first := int32(3)
	pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transferInstructions.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransferInstructions[:3]
	assert.Equal(t, want, got)
	assert.False(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[0].VegaTime,
		ID:       testTransferInstructions[0].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[2].VegaTime,
		ID:       testTransferInstructions[2].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferInstructionPaginationLast(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)
	testTransferInstructions := addTransferInstructions(timeoutCtx, t, bs, transferInstructions)

	last := int32(3)
	pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transferInstructions.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransferInstructions[7:]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.False(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[7].VegaTime,
		ID:       testTransferInstructions[7].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[9].VegaTime,
		ID:       testTransferInstructions[9].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferInstructionPaginationFirstAfter(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)
	testTransferInstructions := addTransferInstructions(timeoutCtx, t, bs, transferInstructions)

	first := int32(3)
	after := testTransferInstructions[2].Cursor().Encode()
	pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
	require.NoError(t, err)
	got, pageInfo, err := transferInstructions.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransferInstructions[3:6]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[3].VegaTime,
		ID:       testTransferInstructions[3].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[5].VegaTime,
		ID:       testTransferInstructions[5].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func testTransferInstructionPaginationLastBefore(t *testing.T) {
	timeoutCtx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()

	bs := sqlstore.NewBlocks(connectionSource)
	transferInstructions := sqlstore.NewTransferInstructions(connectionSource)
	testTransferInstructions := addTransferInstructions(timeoutCtx, t, bs, transferInstructions)

	last := int32(3)
	before := entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[7].VegaTime,
		ID:       testTransferInstructions[7].ID,
	}.String()).Encode()
	pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
	require.NoError(t, err)
	got, pageInfo, err := transferInstructions.GetAll(timeoutCtx, pagination)

	require.NoError(t, err)
	want := testTransferInstructions[4:7]
	assert.Equal(t, want, got)
	assert.True(t, pageInfo.HasPreviousPage)
	assert.True(t, pageInfo.HasNextPage)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[4].VegaTime,
		ID:       testTransferInstructions[4].ID,
	}.String()).Encode(), pageInfo.StartCursor)
	assert.Equal(t, entities.NewCursor(entities.TransferInstructionCursor{
		VegaTime: testTransferInstructions[6].VegaTime,
		ID:       testTransferInstructions[6].ID,
	}.String()).Encode(), pageInfo.EndCursor)
}

func addTransferInstructions(ctx context.Context, t *testing.T, bs *sqlstore.Blocks, transferStore *sqlstore.TransferInstructions) []entities.TransferInstruction {
	t.Helper()
	vegaTime := time.Now().Truncate(time.Microsecond)
	block := addTestBlockForTime(t, bs, vegaTime)
	accounts := sqlstore.NewAccounts(connectionSource)
	accountFrom, accountTo := getTestAccounts(t, accounts, block)

	transferInstructions := make([]entities.TransferInstruction, 0, 10)
	for i := 0; i < 10; i++ {
		vegaTime = vegaTime.Add(time.Second)
		addTestBlockForTime(t, bs, vegaTime)

		amount, _ := decimal.NewFromString("10")
		transferInstruction := entities.TransferInstruction{
			ID:                  entities.TransferInstructionID(fmt.Sprintf("deadbeef%02d", i+1)),
			VegaTime:            vegaTime,
			FromAccountID:       accountFrom.ID,
			ToAccountID:         accountTo.ID,
			AssetID:             entities.AssetID(""),
			Amount:              amount,
			Reference:           "",
			Status:              0,
			TransferInstructionType:        0,
			DeliverOn:           nil,
			StartEpoch:          nil,
			EndEpoch:            nil,
			Factor:              nil,
			DispatchMetric:      nil,
			DispatchMetricAsset: nil,
			DispatchMarkets:     nil,
		}

		err := transferStore.Upsert(ctx, &transferInstruction)
		require.NoError(t, err)
		transferInstructions = append(transferInstructions, transferInstruction)
	}

	return transferInstructions
}

func toPtr[T any](t T) *T {
	return &t
}
