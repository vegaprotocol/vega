// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/ptr"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetTransferByID(t *testing.T) {
	ctx := tempTransaction(t)

	blocksStore := sqlstore.NewBlocks(connectionSource)
	assetsStore := sqlstore.NewAssets(connectionSource)
	accountsStore := sqlstore.NewAccounts(connectionSource)
	transfersStore := sqlstore.NewTransfers(connectionSource)

	block := addTestBlockForTime(t, ctx, blocksStore, time.Now())

	asset := CreateAsset(t, ctx, assetsStore, block)

	account1 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)
	account2 := CreateAccount(t, ctx, accountsStore, block,
		AccountWithType(vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD),
		AccountForAsset(asset),
	)

	transfer := NewTransfer(t, ctx, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}),
	)

	transferUpdateFromSameTx := NewTransfer(t, ctx, accountsStore, block,
		TransferWithID(transfer.ID),
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 15,
			EndEpoch:   nil,
			Factor:     "0.15",
		}),
	)
	transferUpdateFromSameTx.TxHash = transfer.TxHash

	transferUpdateFromDifferentTx := NewTransfer(t, ctx, accountsStore, block,
		TransferWithID(transfer.ID),
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 20,
			EndEpoch:   ptr.From(uint64(25)),
			Factor:     "0.2",
		}),
	)

	// Ensure we have different transfers so the test is meaningful.
	RequireAllDifferent(t, transfer, transferUpdateFromSameTx, transferUpdateFromDifferentTx)

	t.Run("Save transfers", func(t *testing.T) {
		require.NoError(t, transfersStore.Upsert(ctx, transfer))
		require.NoError(t, transfersStore.Upsert(ctx, transferUpdateFromSameTx))
		require.NoError(t, transfersStore.Upsert(ctx, transferUpdateFromDifferentTx))
	})

	t.Run("Retrieve the transfer by ID returns the latest version", func(t *testing.T) {
		retrieved, err := transfersStore.GetByID(ctx, transfer.ID.String())
		require.NoError(t, err)
		assert.Equal(t, *transferUpdateFromDifferentTx, retrieved.Transfer)
	})
}

func TestGetTransfersByHash(t *testing.T) {
	ctx := tempTransaction(t)

	blocksStore := sqlstore.NewBlocks(connectionSource)
	assetsStore := sqlstore.NewAssets(connectionSource)
	accountsStore := sqlstore.NewAccounts(connectionSource)
	transfersStore := sqlstore.NewTransfers(connectionSource)

	block1 := addTestBlockForTime(t, ctx, blocksStore, time.Now().Add(-2*time.Minute))
	block2 := addTestBlockForTime(t, ctx, blocksStore, time.Now().Add(-1*time.Minute))

	asset := CreateAsset(t, ctx, assetsStore, block1)

	account1 := CreateAccount(t, ctx, accountsStore, block1,
		AccountForAsset(asset),
	)
	account2 := CreateAccount(t, ctx, accountsStore, block1,
		AccountWithType(vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD),
		AccountForAsset(asset),
	)

	transfer1 := NewTransfer(t, ctx, accountsStore, block1,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}),
	)

	transfer2 := NewTransfer(t, ctx, accountsStore, block1,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   nil,
			Factor:     "0.1",
		}),
	)
	transfer2.TxHash = transfer1.TxHash

	transfer1UpdateFromSameTx := NewTransfer(t, ctx, accountsStore, block1,
		TransferWithID(transfer1.ID),
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 15,
			EndEpoch:   nil,
			Factor:     "0.15",
		}),
	)
	transfer1UpdateFromSameTx.TxHash = transfer1.TxHash

	transfer1UpdateFromDifferentTx := NewTransfer(t, ctx, accountsStore, block2,
		TransferWithID(transfer1.ID),
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 20,
			EndEpoch:   ptr.From(uint64(25)),
			Factor:     "0.2",
		}),
	)

	// Ensure we have different transfers so the test is meaningful.
	RequireAllDifferent(t, transfer1, transfer2, transfer1UpdateFromSameTx, transfer1UpdateFromDifferentTx)

	t.Run("Save transfers", func(t *testing.T) {
		require.NoError(t, transfersStore.Upsert(ctx, transfer1))
		require.NoError(t, transfersStore.Upsert(ctx, transfer2))
		require.NoError(t, transfersStore.Upsert(ctx, transfer1UpdateFromSameTx))
		require.NoError(t, transfersStore.Upsert(ctx, transfer1UpdateFromDifferentTx))
	})

	t.Run("Retrieve the transfer by hash returns all matching the hash", func(t *testing.T) {
		retrieved, err := transfersStore.GetByTxHash(ctx, transfer1.TxHash)
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2, *transfer1UpdateFromSameTx},
			retrieved,
		)
	})
}

func TestGetTransfersToOrFromParty(t *testing.T) {
	ctx := tempTransaction(t)

	blocksStore := sqlstore.NewBlocks(connectionSource)
	assetsStore := sqlstore.NewAssets(connectionSource)
	accountsStore := sqlstore.NewAccounts(connectionSource)
	transfersStore := sqlstore.NewTransfers(connectionSource)

	block := addTestBlockForTime(t, ctx, blocksStore, time.Now())

	asset := CreateAsset(t, ctx, assetsStore, block)

	account1 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)
	account2 := CreateAccount(t, ctx, accountsStore, block,
		AccountWithType(vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD),
		AccountForAsset(asset),
	)
	account3 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)

	transfer1 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account2, account1),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 5,
			EndEpoch:   ptr.From(uint64(15)),
			Factor:     "0.1",
			DispatchStrategy: &vegapb.DispatchStrategy{
				AssetForMetric: "deadd0d0",
				Markets:        []string{"beefdead", "feebaad"},
				Metric:         vegapb.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
			},
		}),
	)
	transfer2 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   ptr.From(uint64(20)),
			Factor:     "0.1",
		}),
	)
	transfer3 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account3),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 25,
			EndEpoch:   nil,
			Factor:     "0.1",
		}),
	)
	transfer4 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account3, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 15,
			EndEpoch:   ptr.From(uint64(20)),
			Factor:     "0.1",
		}),
	)

	t.Run("Retrieve all transfers from/to party", func(t *testing.T) {
		retrieved, _, err := transfersStore.GetTransfersToOrFromParty(ctx, entities.DefaultCursorPagination(true), sqlstore.ListTransfersFilters{}, account2.PartyID)
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer1, *transfer2, *transfer4},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers from/to party with epoch range", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(16)),
			ToEpoch:   ptr.From(uint64(20)),
		}

		retrievedFromAccount1, _, err := transfersStore.GetTransfersToOrFromParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2},
			TransferDetailsAsTransfers(t, retrievedFromAccount1),
		)

		retrievedFromAccount3, _, err := transfersStore.GetTransfersToOrFromParty(ctx, entities.DefaultCursorPagination(true), filters, account3.PartyID)
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer4},
			TransferDetailsAsTransfers(t, retrievedFromAccount3),
		)
	})

	t.Run("Retrieve transfers from/to party from epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(20)),
		}

		retrievedFromAccount1, _, err := transfersStore.GetTransfersToOrFromParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2, *transfer3},
			TransferDetailsAsTransfers(t, retrievedFromAccount1),
		)

		retrievedFromAccount3, _, err := transfersStore.GetTransfersToOrFromParty(ctx, entities.DefaultCursorPagination(true), filters, account3.PartyID)
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer3, *transfer4},
			TransferDetailsAsTransfers(t, retrievedFromAccount3),
		)
	})

	t.Run("Retrieve transfers from/to party to epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			ToEpoch: ptr.From(uint64(10)),
		}

		retrievedFromAccount1, _, err := transfersStore.GetTransfersToOrFromParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer1, *transfer2},
			TransferDetailsAsTransfers(t, retrievedFromAccount1),
		)

		retrievedFromAccount3, _, err := transfersStore.GetTransfersToOrFromParty(ctx, entities.DefaultCursorPagination(true), filters, account3.PartyID)
		require.NoError(t, err)
		assert.Empty(t, retrievedFromAccount3)
	})
}

func TestGetTransfersByParty(t *testing.T) {
	ctx := tempTransaction(t)

	blocksStore := sqlstore.NewBlocks(connectionSource)
	assetsStore := sqlstore.NewAssets(connectionSource)
	accountsStore := sqlstore.NewAccounts(connectionSource)
	transfersStore := sqlstore.NewTransfers(connectionSource)

	block := addTestBlockForTime(t, ctx, blocksStore, time.Now())

	asset := CreateAsset(t, ctx, assetsStore, block)

	account1 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)
	account2 := CreateAccount(t, ctx, accountsStore, block,
		AccountWithType(vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD),
		AccountForAsset(asset),
	)

	transfer1 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 5,
			EndEpoch:   ptr.From(uint64(15)),
			Factor:     "0.1",
		}),
	)
	transfer2 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account2, account1),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   ptr.From(uint64(17)),
			Factor:     "0.1",
		}),
	)
	transfer3 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account2, account1),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 15,
			EndEpoch:   ptr.From(uint64(20)),
			Factor:     "0.1",
		}),
	)
	transfer4 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 15,
			EndEpoch:   ptr.From(uint64(20)),
			Factor:     "0.1",
		}),
	)

	t.Run("Retrieve transfers from party", func(t *testing.T) {
		retrieved, _, err := transfersStore.GetTransfersFromParty(ctx, entities.DefaultCursorPagination(true), sqlstore.ListTransfersFilters{}, account1.PartyID)

		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer1, *transfer4},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers from party with epoch range", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(5)),
			ToEpoch:   ptr.From(uint64(10)),
		}

		retrieved, _, err := transfersStore.GetTransfersFromParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)

		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer1},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers from party from epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(17)),
		}

		retrieved, _, err := transfersStore.GetTransfersFromParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)
		require.NoError(t, err)

		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer4},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers from party to epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			ToEpoch: ptr.From(uint64(13)),
		}

		retrieved, _, err := transfersStore.GetTransfersFromParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)
		require.NoError(t, err)

		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer1},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers to party", func(t *testing.T) {
		retrieved, _, err := transfersStore.GetTransfersToParty(ctx, entities.DefaultCursorPagination(true), sqlstore.ListTransfersFilters{}, account1.PartyID)

		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2, *transfer3},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers to party with epoch range", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(5)),
			ToEpoch:   ptr.From(uint64(10)),
		}

		retrieved, _, err := transfersStore.GetTransfersToParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)

		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers to party from epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(18)),
		}

		retrieved, _, err := transfersStore.GetTransfersToParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)
		require.NoError(t, err)

		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer3},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers to party to epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			ToEpoch: ptr.From(uint64(13)),
		}

		retrieved, _, err := transfersStore.GetTransfersToParty(ctx, entities.DefaultCursorPagination(true), filters, account1.PartyID)
		require.NoError(t, err)

		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})
}

func TestGetAllTransfers(t *testing.T) {
	ctx := tempTransaction(t)

	blocksStore := sqlstore.NewBlocks(connectionSource)
	assetsStore := sqlstore.NewAssets(connectionSource)
	accountsStore := sqlstore.NewAccounts(connectionSource)
	transfersStore := sqlstore.NewTransfers(connectionSource)

	block := addTestBlockForTime(t, ctx, blocksStore, time.Now())

	asset := CreateAsset(t, ctx, assetsStore, block)

	account1 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)
	account2 := CreateAccount(t, ctx, accountsStore, block,
		AccountWithType(vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD),
		AccountForAsset(asset),
	)
	account3 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)

	transfer1 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account2, account1),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 5,
			EndEpoch:   ptr.From(uint64(15)),
			Factor:     "0.1",
			DispatchStrategy: &vegapb.DispatchStrategy{
				AssetForMetric:  "deadd0d0",
				Metric:          vegapb.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE,
				Markets:         []string{"beefdead", "feebaad"},
				IndividualScope: vegapb.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
			},
		}),
	)
	transfer2 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account2),
		TransferWithStatus(entities.TransferStatusDone),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 10,
			EndEpoch:   ptr.From(uint64(20)),
			Factor:     "0.1",
		}),
	)
	transfer3 := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
		TransferWithAsset(asset),
		TransferFromToAccounts(account1, account3),
		TransferWithStatus(entities.TransferStatusCancelled),
		TransferAsRecurring(&eventspb.RecurringTransfer{
			StartEpoch: 25,
			EndEpoch:   nil,
			Factor:     "0.1",
			DispatchStrategy: &vegapb.DispatchStrategy{
				TeamScope: []string{
					"beefdeadfeebaad",
				},
			},
		}),
		TransferWithGameID(ptr.From("c001d00d")),
	)

	t.Run("Retrieve all transfers", func(t *testing.T) {
		retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(true), sqlstore.ListTransfersFilters{})
		require.NoError(t, err)
		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer1, *transfer2, *transfer3},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers with epoch range", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(16)),
			ToEpoch:   ptr.From(uint64(28)),
		}

		retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(true), filters)
		require.NoError(t, err)

		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2, *transfer3},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve all transfers from epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			FromEpoch: ptr.From(uint64(20)),
		}

		retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(true), filters)
		require.NoError(t, err)

		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer2, *transfer3},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers to epoch", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			ToEpoch: ptr.From(uint64(10)),
		}

		retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(true), filters)
		require.NoError(t, err)

		assert.ElementsMatch(t,
			[]entities.Transfer{*transfer1, *transfer2},
			TransferDetailsAsTransfers(t, retrieved),
		)
	})

	t.Run("Retrieve transfers by status", func(t *testing.T) {
		matrix := map[entities.TransferStatus][]entities.Transfer{
			entities.TransferStatusPending:   {*transfer1},
			entities.TransferStatusCancelled: {*transfer3},
			entities.TransferStatusRejected:  {},
		}

		for status, expected := range matrix {
			filters := sqlstore.ListTransfersFilters{
				Status: ptr.From(status),
			}

			retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(true), filters)
			require.NoError(t, err)
			assert.Equal(t, expected, TransferDetailsAsTransfers(t, retrieved))
		}
	})

	t.Run("Retrieve transfers by scope", func(t *testing.T) {
		matrix := map[entities.TransferScope][]entities.Transfer{
			entities.TransferScopeIndividual: {*transfer1},
			entities.TransferScopeTeam:       {*transfer3},
		}

		for scope, expected := range matrix {
			filters := sqlstore.ListTransfersFilters{
				Scope: ptr.From(scope),
			}

			retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(true), filters)
			require.NoError(t, err)
			assert.Equal(t, expected, TransferDetailsAsTransfers(t, retrieved))
		}
	})

	t.Run("Retrieve transfers by game ID", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			GameID: ptr.From(entities.GameID("c001d00d")),
		}
		retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(true), filters)
		require.NoError(t, err)
		assert.Equal(t, []entities.Transfer{*transfer3}, TransferDetailsAsTransfers(t, retrieved))
	})
}

func TestGetAllTransfersWithPagination(t *testing.T) {
	ctx := tempTransaction(t)

	blocksStore := sqlstore.NewBlocks(connectionSource)
	assetsStore := sqlstore.NewAssets(connectionSource)
	accountsStore := sqlstore.NewAccounts(connectionSource)
	transfersStore := sqlstore.NewTransfers(connectionSource)

	block := addTestBlockForTime(t, ctx, blocksStore, time.Now())

	asset := CreateAsset(t, ctx, assetsStore, block)

	account1 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)
	account2 := CreateAccount(t, ctx, accountsStore, block,
		AccountWithType(vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD),
		AccountForAsset(asset),
	)

	transfers := make([]entities.Transfer, 0, 10)
	for i := 0; i < 10; i++ {
		block := addTestBlockForTime(t, ctx, blocksStore, time.Now().Add(time.Duration(i)*time.Second))
		transfer := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
			TransferWithAsset(asset),
			TransferFromToAccounts(account1, account2),
			TransferAsRecurring(&eventspb.RecurringTransfer{
				StartEpoch: 5,
				EndEpoch:   ptr.From(uint64(15)),
				Factor:     "0.1",
			}),
		)
		transfers = append(transfers, *transfer)
	}

	noFilters := sqlstore.ListTransfersFilters{}

	t.Run("Paginate with oldest first", func(t *testing.T) {
		pagination, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
		require.NoError(t, err)

		retrieved, pageInfo, err := transfersStore.GetAll(ctx, pagination, noFilters)
		require.NoError(t, err)
		assert.Equal(t, transfers, TransferDetailsAsTransfers(t, retrieved))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: false,
			StartCursor:     transfers[0].Cursor().Encode(),
			EndCursor:       transfers[9].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Paginate first 3 transfers", func(t *testing.T) {
		first := int32(3)
		pagination, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)

		retrieved, pageInfo, err := transfersStore.GetAll(ctx, pagination, noFilters)
		require.NoError(t, err)
		assert.Equal(t, transfers[:3], TransferDetailsAsTransfers(t, retrieved))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: false,
			StartCursor:     transfers[0].Cursor().Encode(),
			EndCursor:       transfers[2].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Paginate last 3 transfers", func(t *testing.T) {
		last := int32(3)
		pagination, err := entities.NewCursorPagination(nil, nil, &last, nil, false)
		require.NoError(t, err)

		retrieved, pageInfo, err := transfersStore.GetAll(ctx, pagination, noFilters)
		require.NoError(t, err)
		assert.Equal(t, transfers[7:], TransferDetailsAsTransfers(t, retrieved))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     false,
			HasPreviousPage: true,
			StartCursor:     transfers[7].Cursor().Encode(),
			EndCursor:       transfers[9].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Paginate first 3 transfers after third one", func(t *testing.T) {
		first := int32(3)
		after := transfers[2].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)

		retrieved, pageInfo, err := transfersStore.GetAll(ctx, pagination, noFilters)
		require.NoError(t, err)
		assert.Equal(t, transfers[3:6], TransferDetailsAsTransfers(t, retrieved))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     transfers[3].Cursor().Encode(),
			EndCursor:       transfers[5].Cursor().Encode(),
		}, pageInfo)
	})

	t.Run("Paginate last 3 transfers before seventh one", func(t *testing.T) {
		last := int32(3)
		before := transfers[7].Cursor().Encode()
		pagination, err := entities.NewCursorPagination(nil, nil, &last, &before, false)
		require.NoError(t, err)

		retrieved, pageInfo, err := transfersStore.GetAll(ctx, pagination, noFilters)
		require.NoError(t, err)
		assert.Equal(t, transfers[4:7], TransferDetailsAsTransfers(t, retrieved))
		assert.Equal(t, entities.PageInfo{
			HasNextPage:     true,
			HasPreviousPage: true,
			StartCursor:     transfers[4].Cursor().Encode(),
			EndCursor:       transfers[6].Cursor().Encode(),
		}, pageInfo)
	})
}

func TestGetAllRewardTransfers(t *testing.T) {
	ctx := tempTransaction(t)

	blocksStore := sqlstore.NewBlocks(connectionSource)
	assetsStore := sqlstore.NewAssets(connectionSource)
	accountsStore := sqlstore.NewAccounts(connectionSource)
	transfersStore := sqlstore.NewTransfers(connectionSource)

	vegaTime := time.Now().Truncate(time.Microsecond)

	block := addTestBlockForTime(t, ctx, blocksStore, vegaTime)

	asset := CreateAsset(t, ctx, assetsStore, block)

	account1 := CreateAccount(t, ctx, accountsStore, block,
		AccountForAsset(asset),
	)
	account2 := CreateAccount(t, ctx, accountsStore, block,
		AccountWithType(vegapb.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD),
		AccountForAsset(asset),
	)

	allTransfers := make([]entities.Transfer, 0, 20)
	for i := 0; i < 10; i++ {
		vegaTime = vegaTime.Add(time.Second)
		block := addTestBlockForTime(t, ctx, blocksStore, vegaTime)

		transfer := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
			TransferWithAsset(asset),
			TransferFromToAccounts(account1, account2),
			TransferAsOneOff(eventspb.OneOffTransfer{
				DeliverOn: vegaTime.UnixNano(),
			}),
		)
		allTransfers = append(allTransfers, *transfer)
	}

	rewardTransfers := make([]entities.Transfer, 0, 10)
	for i := 0; i < 10; i++ {
		vegaTime = vegaTime.Add(time.Second)
		block := addTestBlockForTime(t, ctx, blocksStore, vegaTime)

		var kindOption TransferOption
		var statusOption TransferOption
		if i%2 == 0 {
			kindOption = TransferAsRecurringGovernance(eventspb.RecurringGovernanceTransfer{
				StartEpoch: 15,
				EndEpoch:   nil,
				DispatchStrategy: &vegapb.DispatchStrategy{
					Metric:          vegapb.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN,
					LockPeriod:      uint64((i % 7) + 1),
					IndividualScope: vegapb.IndividualScope_INDIVIDUAL_SCOPE_ALL,
				},
			})
			statusOption = TransferWithStatus(entities.TransferStatusDone)
		} else {
			kindOption = TransferAsRecurring(&eventspb.RecurringTransfer{
				StartEpoch: 15,
				EndEpoch:   nil,
				Factor:     "0.15",
				DispatchStrategy: &vegapb.DispatchStrategy{
					Metric:     vegapb.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING,
					LockPeriod: uint64((i % 7) + 1),
					TeamScope:  []string{"deadfbeefc0ffeed00d"},
				},
			})

			statusOption = TransferWithStatus(entities.TransferStatusPending)
		}

		transfer := CreateTransfer(t, ctx, transfersStore, accountsStore, block,
			TransferWithAsset(asset),
			TransferFromToAccounts(account1, account2),
			kindOption,
			statusOption,
		)
		rewardTransfers = append(rewardTransfers, *transfer)
		allTransfers = append(allTransfers, *transfer)
	}

	noFilters := sqlstore.ListTransfersFilters{}

	t.Run("Get all transfers", func(t *testing.T) {
		retrieved, _, err := transfersStore.GetAll(ctx, entities.DefaultCursorPagination(false), noFilters)
		require.NoError(t, err)
		assert.Equal(t, allTransfers, TransferDetailsAsTransfers(t, retrieved))
	})

	t.Run("Get only reward transfers", func(t *testing.T) {
		retrieved, _, err := transfersStore.GetAllRewards(ctx, entities.DefaultCursorPagination(false), noFilters)
		require.NoError(t, err)
		assert.Equal(t, rewardTransfers, TransferDetailsAsTransfers(t, retrieved))
	})

	t.Run("Retrieve transfers by status pending", func(t *testing.T) {
		filters := sqlstore.ListTransfersFilters{
			Status: ptr.From(entities.TransferStatusDone),
		}

		retrieved, _, err := transfersStore.GetAllRewards(ctx, entities.DefaultCursorPagination(false), filters)
		require.NoError(t, err)
		assert.Equal(t,
			[]entities.Transfer{rewardTransfers[0], rewardTransfers[2], rewardTransfers[4], rewardTransfers[6], rewardTransfers[8]},
			TransferDetailsAsTransfers(t, retrieved))
	})

	t.Run("Retrieve transfers by scope", func(t *testing.T) {
		matrix := map[entities.TransferScope][]entities.Transfer{
			entities.TransferScopeIndividual: {rewardTransfers[0], rewardTransfers[2], rewardTransfers[4], rewardTransfers[6], rewardTransfers[8]},
			entities.TransferScopeTeam:       {rewardTransfers[1], rewardTransfers[3], rewardTransfers[5], rewardTransfers[7], rewardTransfers[9]},
		}

		for scope, expected := range matrix {
			filters := sqlstore.ListTransfersFilters{
				Scope: ptr.From(scope),
			}

			retrieved, _, err := transfersStore.GetAllRewards(ctx, entities.DefaultCursorPagination(false), filters)
			require.NoError(t, err)
			assert.Equal(t, expected, TransferDetailsAsTransfers(t, retrieved))
		}
	})
}
