package sqlstore_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestTransfers_GetTransfersFromPartyAndGetToParty(t *testing.T) {

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
		Market:          "",
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

	sourceTransferProto2 := &eventspb.Transfer{
		Id:              "deadd0d0",
		From:            accountFrom.PartyID.String(),
		FromAccountType: accountFrom.Type,
		To:              accountTo.PartyID.String(),
		ToAccountType:   accountTo.Type,
		Asset:           accountFrom.AssetID.String(),
		Market:          "",
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

	retrieved, err := transfers.GetTransfersFromParty(ctx, accountFrom.PartyID)
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, err = transfers.GetTransfersToParty(ctx, accountTo.PartyID)
	if err != nil {
		t.Fatalf("f%s", err)
	}
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

}

func TestTransfers_GetFromAccountAndGetToAccount(t *testing.T) {
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
			EndEpoch:   &vega.Uint64Value{Value: 56},
			Factor:     "3.12",
		}},
	}

	transfer, _ = entities.TransferFromProto(context.Background(), sourceTransferProto2, block.VegaTime, accounts)
	transfers.Upsert(context.Background(), transfer)

	retrieved, _ := transfers.GetAll(ctx)
	assert.Equal(t, 2, len(retrieved))

	retrieved, _ = transfers.GetTransfersFromAccount(ctx, accountFrom.ID)
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _ = transfers.GetTransfersToAccount(ctx, accountTo.ID)
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto1, retrievedTransferProto)

	retrieved, _ = transfers.GetTransfersFromAccount(ctx, accountTo.ID)
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

	retrieved, _ = transfers.GetTransfersToAccount(ctx, accountFrom.ID)
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ = retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto2, retrievedTransferProto)

}

func TestTransfers_UpdatesInDifferentBlocks(t *testing.T) {
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

	retrieved, _ := transfers.GetAll(ctx)
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

}

func TestTransfers_UpdateInSameBlock(t *testing.T) {
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

	retrieved, _ := transfers.GetAll(ctx)
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

}

func TestTransfers_AddAndRetrieveOneOffTransfer(t *testing.T) {
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
	retrieved, _ := transfers.GetAll(ctx)
	assert.Equal(t, 1, len(retrieved))
	retrievedTransferProto, _ := retrieved[0].ToProto(accounts)
	assert.Equal(t, sourceTransferProto, retrievedTransferProto)

}

func TestTransfers_AddAndRetrieveRecurringTransfer(t *testing.T) {
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

	retrieved, _ := transfers.GetAll(ctx)
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
		Quantum:       1,
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
