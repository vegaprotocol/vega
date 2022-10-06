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
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestBalance(t *testing.T, store *sqlstore.Balances, block entities.Block, acc entities.Account, balance int64) {
	t.Helper()
	bal := entities.AccountBalance{
		Account:  &acc,
		VegaTime: block.VegaTime,
		Balance:  decimal.NewFromInt(balance),
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
	defer DeleteEverything()
	ctx := context.Background()

	blockStore := sqlstore.NewBlocks(connectionSource)
	assetStore := sqlstore.NewAssets(connectionSource)
	accountStore := sqlstore.NewAccounts(connectionSource)
	balanceStore := sqlstore.NewBalances(connectionSource)
	partyStore := sqlstore.NewParties(connectionSource)

	// Set up a test environment with a bunch of blocks/parties/accounts
	asset := addTestAsset(t, assetStore, addTestBlock(t, blockStore))

	var blocks []entities.Block
	var parties []entities.Party
	var accounts []entities.Account
	for i := 0; i < 5; i++ {
		blocks = append(blocks, addTestBlock(t, blockStore))
		parties = append(parties, addTestParty(t, partyStore, blocks[0]))
		accounts = append(accounts, addTestAccount(t, accountStore, parties[i], asset, blocks[0]))
	}

	// And add some dummy balances
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 1)
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 2) // Second balance on same acc/block should override first
	addTestBalance(t, balanceStore, blocks[1], accounts[0], 5)
	addTestBalance(t, balanceStore, blocks[2], accounts[1], 10)
	addTestBalance(t, balanceStore, blocks[3], accounts[2], 100)
	addTestBalance(t, balanceStore, blocks[4], accounts[0], 30)

	balanceStore.Flush(ctx)

	dateRange := entities.DateRange{}
	pagination := entities.CursorPagination{}

	// Helper function to make an aggregated balance that isn't grouped by anything
	mkAggBal := func(blockI, bal int64) entities.AggregatedBalance {
		return entities.AggregatedBalance{
			VegaTime: blocks[blockI].VegaTime,
			Balance:  decimal.NewFromInt(bal),
		}
	}

	// Helper function to make an aggregated balance that is grouped (i.e. broken out by) account
	mkAggBalAcc := func(blockI, accI, bal int64) entities.AggregatedBalance {
		return entities.AggregatedBalance{
			VegaTime:  blocks[blockI].VegaTime,
			AccountID: &accounts[accI].ID,
			Balance:   decimal.NewFromInt(bal),
		}
	}

	// Helper function to make an aggregated balance that is grouped (i.e. broken out by) account
	mkAggBalGeneral := func(blockI, bal int64) entities.AggregatedBalance {
		accType := types.AccountTypeGeneral
		return entities.AggregatedBalance{
			VegaTime: blocks[blockI].VegaTime,
			Type:     &accType,
			Balance:  decimal.NewFromInt(bal),
		}
	}

	t.Run("Query should return all balances", func(t *testing.T) {
		// Query all the balances (they're all for the same asset)
		actual, _, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{}, dateRange, pagination)
		require.NoError(t, err)

		expected := &[]entities.AggregatedBalance{
			mkAggBal(0, 2),         // accounts[0] -> 2
			mkAggBal(1, 5),         // accounts[0] -> 5
			mkAggBal(2, 5+10),      // accounts[1] -> 10;  want accounts[0] + accounts[1]
			mkAggBal(3, 5+10+100),  // accounts[2] -> 100; want accounts[0] + accounts[1] + accounts[2]
			mkAggBal(4, 30+10+100), // accounts[0] -> 30;  want accounts[0] + accounts[1] + accounts[2]
		}

		assertBalanceCorrect(t, expected, actual)
	})

	t.Run("Query should return transactions for party", func(t *testing.T) {
		// Try just for our first account/party
		filter := entities.AccountFilter{
			AssetID:  asset.ID,
			PartyIDs: []entities.PartyID{parties[0].ID},
		}
		actual, _, err := balanceStore.Query(filter, []entities.AccountField{}, dateRange, pagination)
		require.NoError(t, err)

		// only accounts[0] is for  party[0]
		expected := &[]entities.AggregatedBalance{
			mkAggBal(0, 2),  // accounts[0] -> 2
			mkAggBal(1, 5),  // accounts[0] -> 5
			mkAggBal(4, 30), // accounts[0] -> 30
		}
		assertBalanceCorrect(t, expected, actual)
	})

	expectedGroupedByAccount := []entities.AggregatedBalance{
		mkAggBalAcc(0, 0, 2),   // accounts[0] -> 2
		mkAggBalAcc(1, 0, 5),   // accounts[0] -> 5
		mkAggBalAcc(2, 0, 5),   // accounts[0] -> <no change>
		mkAggBalAcc(2, 1, 10),  // accounts[1] -> 10;
		mkAggBalAcc(3, 0, 5),   // accounts[0] -> <no change>
		mkAggBalAcc(3, 1, 10),  // accounts[1] -> <no change>
		mkAggBalAcc(3, 2, 100), // accounts[2] -> 100;
		mkAggBalAcc(4, 0, 30),  // accounts[0] -> 30;
		mkAggBalAcc(4, 1, 10),  // accounts[1] -> <no change>
		mkAggBalAcc(4, 2, 100), // accounts[2] -> <no change>
	}

	// So this is a bit complicated; balanceStore.Query will sort first by vegaTime and
	// then by whatever we have grouped by (in this case account id).
	sortFn := func(x, y int) bool {
		if !expectedGroupedByAccount[x].VegaTime.Equal(expectedGroupedByAccount[y].VegaTime) {
			return expectedGroupedByAccount[x].VegaTime.Before(expectedGroupedByAccount[y].VegaTime)
		}
		return string(*expectedGroupedByAccount[x].AccountID) < string(*expectedGroupedByAccount[y].AccountID)
	}

	sort.Slice(expectedGroupedByAccount, sortFn)

	t.Run("Query should group results by account", func(t *testing.T) {
		// Now try grouping - if we do it by account id it should split out balances for each account.
		actual, _, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{entities.AccountFieldID}, dateRange, pagination)
		require.NoError(t, err)

		expected := &expectedGroupedByAccount
		assertBalanceCorrect(t, expected, actual)
	})

	t.Run("Query should group by account type", func(t *testing.T) {
		// Now try grouping by account type (they are all 'General' so all accounts should be summed)
		actual, _, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{entities.AccountFieldType}, dateRange, pagination)

		require.NoError(t, err)

		expected := &[]entities.AggregatedBalance{
			mkAggBalGeneral(0, 2),         // accounts[0] -> 2
			mkAggBalGeneral(1, 5),         // accounts[0] -> 5
			mkAggBalGeneral(2, 5+10),      // accounts[1] -> 10;  want accounts[0] + accounts[1]
			mkAggBalGeneral(3, 5+10+100),  // accounts[2] -> 100; want accounts[0] + accounts[1] + accounts[2]
			mkAggBalGeneral(4, 30+10+100), // accounts[0] -> 30;  want accounts[0] + accounts[1] + accounts[2]
		}

		assertBalanceCorrect(t, expected, actual)
	})

	t.Run("Query should return results paged", func(t *testing.T) {
		first := int32(3)
		after := expectedGroupedByAccount[2].Cursor().Encode()
		p, err := entities.NewCursorPagination(&first, &after, nil, nil, false)
		require.NoError(t, err)
		actual, _, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{entities.AccountFieldID}, dateRange, p)
		require.NoError(t, err)
		expected := expectedGroupedByAccount[3:6]
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
		actual, _, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{entities.AccountFieldID}, dateRange, p)
		require.NoError(t, err)

		expected := expectedGroupedByAccount[1:7]
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
		actual, _, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{entities.AccountFieldID}, dateRange, p)
		require.NoError(t, err)

		expected := expectedGroupedByAccount[1:4]
		assertBalanceCorrect(t, &expected, actual)
	})
}

func TestBalancesDataRetention(t *testing.T) {
	defer DeleteEverything()
	ctx := context.Background()

	blockStore := sqlstore.NewBlocks(connectionSource)
	assetStore := sqlstore.NewAssets(connectionSource)
	accountStore := sqlstore.NewAccounts(connectionSource)
	balanceStore := sqlstore.NewBalances(connectionSource)
	partyStore := sqlstore.NewParties(connectionSource)

	// Set up a test environment with a bunch of blocks/parties/accounts
	asset := addTestAsset(t, assetStore, addTestBlock(t, blockStore))

	var blocks []entities.Block
	var parties []entities.Party
	var accounts []entities.Account
	for i := 0; i < 5; i++ {
		blocks = append(blocks, addTestBlockForTime(t, blockStore, time.Now().Add((-26*time.Hour)-(time.Duration(5-i)*time.Second))))
		parties = append(parties, addTestParty(t, partyStore, blocks[0]))
		accounts = append(accounts, addTestAccount(t, accountStore, parties[i], asset, blocks[0]))
	}

	// And add some dummy balances
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 1)
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 2) // Second balance on same acc/block should override first
	addTestBalance(t, balanceStore, blocks[1], accounts[0], 5)
	addTestBalance(t, balanceStore, blocks[2], accounts[1], 10)
	addTestBalance(t, balanceStore, blocks[3], accounts[2], 100)
	balanceStore.Flush(ctx)

	// Conflate the data and add some new positions so all tests run against a mix of conflated and non-conflated data
	now := time.Now()
	refreshQuery := fmt.Sprintf("CALL refresh_continuous_aggregate('conflated_balances', '%s', '%s');",
		now.Add(-48*time.Hour).Format("2006-01-02"),
		time.Now().Format("2006-01-02"))
	_, err := connectionSource.Connection.Exec(context.Background(), refreshQuery)

	assert.NoError(t, err)

	// The refresh of the continuous aggregate completes asynchronously so the following loop is necessary to ensure the data has been materialized
	// before the test continues
	for {
		var counter int
		connectionSource.Connection.QueryRow(context.Background(), "SELECT count(*) FROM conflated_balances").Scan(&counter)
		if counter == 3 {
			break
		}
	}

	_, err = connectionSource.Connection.Exec(context.Background(), "delete from balances")
	assert.NoError(t, err)

	addTestBalance(t, balanceStore, blocks[4], accounts[0], 30)
	balanceStore.Flush(ctx)

	dateRange := entities.DateRange{}
	pagination := entities.CursorPagination{}

	// Query all the balances (they're all for the same asset)
	actual, _, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{}, dateRange, pagination)
	require.NoError(t, err)

	// Helper function to make an aggregated balance that isn't grouped by anything
	mkAggBal := func(blockI, bal int64) entities.AggregatedBalance {
		return entities.AggregatedBalance{
			VegaTime: blocks[blockI].VegaTime,
			Balance:  decimal.NewFromInt(bal),
		}
	}
	expected := []entities.AggregatedBalance{
		// mkAggBal(0, 2),         // accounts[0] -> 2
		mkAggBal(1, 5),         // accounts[0] -> 5
		mkAggBal(2, 5+10),      // accounts[1] -> 10;  want accounts[0] + accounts[1]
		mkAggBal(3, 5+10+100),  // accounts[2] -> 100; want accounts[0] + accounts[1] + accounts[2]
		mkAggBal(4, 30+10+100), // accounts[0] -> 30;  want accounts[0] + accounts[1] + accounts[2]
	}

	assertBalanceCorrect(t, &expected, actual)
}
