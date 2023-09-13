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
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestBalance(t *testing.T,
	store *sqlstore.Balances, block entities.Block,
	acc entities.Account, balance int64,
	txHash entities.TxHash,
) {
	t.Helper()
	bal := entities.AccountBalance{
		Account:  &acc,
		VegaTime: block.VegaTime,
		Balance:  decimal.NewFromInt(balance),
		TxHash:   txHash,
	}

	err := store.Add(bal)
	require.NoError(t, err)
}

func aggBalLessThan(x, y entities.AggregatedBalance) bool {
	if !x.VegaTime.Equal(y.VegaTime) {
		return x.VegaTime.Before(y.VegaTime)
	}
	if x.AccountID != y.AccountID {
		return x.AccountID.String() < y.AccountID.String()
	}
	return false
}

func assertBalanceCorrect(t *testing.T, expected, actual *[]entities.AggregatedBalance) {
	t.Helper()
	diff := cmp.Diff(expected, actual, cmpopts.SortSlices(aggBalLessThan))
	assert.Empty(t, diff)
}

func TestBalances(t *testing.T) {
	ctx := tempTransaction(t)

	blockStore := sqlstore.NewBlocks(connectionSource)
	assetStore := sqlstore.NewAssets(connectionSource)
	accountStore := sqlstore.NewAccounts(connectionSource)
	balanceStore := sqlstore.NewBalances(connectionSource)
	partyStore := sqlstore.NewParties(connectionSource)

	// Set up a test environment with a bunch of blocks/parties/accounts
	asset := addTestAsset(t, ctx, assetStore, addTestBlock(t, ctx, blockStore))

	var blocks []entities.Block
	var parties []entities.Party
	var accounts []entities.Account
	for i := 0; i < 5; i++ {
		blocks = append(blocks, addTestBlock(t, ctx, blockStore))
		parties = append(parties, addTestParty(t, ctx, partyStore, blocks[0]))
		accounts = append(accounts, helpers.AddTestAccount(t, ctx, accountStore, parties[i], asset, types.AccountTypeGeneral, blocks[0]))
	}

	// And add some dummy balances
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 1, defaultTxHash)
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 2, defaultTxHash) // Second balance on same acc/block should override first
	addTestBalance(t, balanceStore, blocks[1], accounts[0], 5, defaultTxHash)
	addTestBalance(t, balanceStore, blocks[2], accounts[1], 10, defaultTxHash)
	addTestBalance(t, balanceStore, blocks[3], accounts[2], 100, defaultTxHash)
	addTestBalance(t, balanceStore, blocks[4], accounts[0], 30, defaultTxHash)

	balanceStore.Flush(ctx)

	dateRange := entities.DateRange{}
	pagination := entities.CursorPagination{}

	mkAggBal := func(blockI, bal int64, acc entities.Account) entities.AggregatedBalance {
		return entities.AggregatedBalance{
			VegaTime: blocks[blockI].VegaTime,
			Balance:  decimal.NewFromInt(bal),
			AssetID:  &acc.AssetID,
			PartyID:  &acc.PartyID,
			MarketID: &acc.MarketID,
			Type:     &acc.Type,
		}
	}

	allExpected := []entities.AggregatedBalance{
		mkAggBal(0, 2, accounts[0]),   // accounts[0] -> 2
		mkAggBal(1, 5, accounts[0]),   // accounts[0] -> 5
		mkAggBal(2, 10, accounts[1]),  // accounts[1] -> 10;
		mkAggBal(3, 100, accounts[2]), // accounts[1] -> 10;
		mkAggBal(4, 30, accounts[0]),  // accounts[1] -> 10;
	}

	t.Run("Query should return all balances", func(t *testing.T) {
		// Query all the balances (they're all for the same asset)
		actual, _, err := balanceStore.Query(ctx, entities.AccountFilter{AssetID: asset.ID}, dateRange, pagination)
		require.NoError(t, err)
		assertBalanceCorrect(t, &allExpected, actual)
	})

	t.Run("Query should return transactions for party", func(t *testing.T) {
		// Try just for our first account/party
		filter := entities.AccountFilter{
			AssetID:  asset.ID,
			PartyIDs: []entities.PartyID{parties[0].ID},
		}
		actual, _, err := balanceStore.Query(ctx, filter, dateRange, pagination)
		require.NoError(t, err)

		// only accounts[0] is for  party[0]
		expected := &[]entities.AggregatedBalance{
			mkAggBal(0, 2, accounts[0]),  // accounts[0] -> 2
			mkAggBal(1, 5, accounts[0]),  // accounts[0] -> 5
			mkAggBal(4, 30, accounts[0]), // accounts[0] -> 30
		}
		assertBalanceCorrect(t, expected, actual)
	})

	t.Run("Query should return results paged", func(t *testing.T) {
		first := int32(3)
		after := allExpected[2].Cursor().Encode()
		p, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		actual, _, err := balanceStore.Query(ctx, entities.AccountFilter{AssetID: asset.ID}, dateRange, p)
		require.NoError(t, err)
		expected := allExpected[3:5]
		assertBalanceCorrect(t, &expected, actual)
	})

	t.Run("Query should return results between dates", func(t *testing.T) {
		p, err := entities.NewCursorPagination(nil, nil, nil, nil, false)
		require.NoError(t, err)
		startTime := blocks[1].VegaTime
		endTime := blocks[4].VegaTime
		dateRange := entities.DateRange{
			Start: &startTime,
			End:   &endTime,
		}
		actual, _, err := balanceStore.Query(ctx, entities.AccountFilter{AssetID: asset.ID}, dateRange, p)
		require.NoError(t, err)

		expected := allExpected[1:4]
		assertBalanceCorrect(t, &expected, actual)
	})

	t.Run("Query should return results paged between dates", func(t *testing.T) {
		first := int32(3)
		p, err := entities.NewCursorPagination(&first, nil, nil, nil, false)
		require.NoError(t, err)
		startTime := blocks[1].VegaTime
		endTime := blocks[4].VegaTime
		dateRange := entities.DateRange{
			Start: &startTime,
			End:   &endTime,
		}
		actual, _, err := balanceStore.Query(ctx, entities.AccountFilter{AssetID: asset.ID}, dateRange, p)
		require.NoError(t, err)

		expected := allExpected[1:4]
		assertBalanceCorrect(t, &expected, actual)
	})
}
