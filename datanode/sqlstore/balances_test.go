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
	"sort"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/datanode/sqlstore"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestBalance(t *testing.T, store *sqlstore.Balances, block entities.Block, acc entities.Account, balance int64) {
	bal := entities.AccountBalance{
		Account:  &acc,
		VegaTime: block.VegaTime,
		Balance:  decimal.NewFromInt(balance),
	}

	err := store.Add(bal)
	require.NoError(t, err)
}

func assertBalanceCorrect(t *testing.T,
	expected_blocks []int, expected_bals []int64,
	blocks []entities.Block, bals []entities.AggregatedBalance) {
	assert.Len(t, bals, len(expected_blocks))
	for i := 0; i < len(expected_blocks); i++ {
		assert.Equal(t, blocks[expected_blocks[i]].VegaTime, (bals)[i].VegaTime)
		assert.Equal(t, decimal.NewFromInt(expected_bals[i]), (bals)[i].Balance)
	}
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

	// Sort the accounts by ID because that's how the DB will return them in a bit
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].ID < accounts[j].ID
	})

	// And add some dummy balances
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 1)
	addTestBalance(t, balanceStore, blocks[0], accounts[0], 2) // Second balance on same acc/block should override first
	addTestBalance(t, balanceStore, blocks[1], accounts[0], 5)
	addTestBalance(t, balanceStore, blocks[2], accounts[1], 10)
	addTestBalance(t, balanceStore, blocks[3], accounts[2], 100)
	addTestBalance(t, balanceStore, blocks[4], accounts[0], 30)

	balanceStore.Flush(ctx)

	// Query all the balances (they're all for the same asset)
	bals, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{})
	require.NoError(t, err)

	expected_blocks := []int{0, 1, 2, 3, 4}
	expected_bals := []int64{2, 5, 5 + 10, 5 + 10 + 100, 30 + 10 + 100}
	assertBalanceCorrect(t, expected_blocks, expected_bals, blocks[:], *bals)

	// Try just for our first account/party
	filter := entities.AccountFilter{
		AssetID:  asset.ID,
		PartyIDs: []entities.PartyID{parties[0].ID},
	}
	bals, err = balanceStore.Query(filter, []entities.AccountField{})
	require.NoError(t, err)

	expected_blocks = []int{0, 1, 4}
	expected_bals = []int64{2, 5, 30}
	assertBalanceCorrect(t, expected_blocks, expected_bals, blocks[:], *bals)

	// Now try grouping - if we do it by account id it should split out balances for each account.
	bals, err = balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{entities.AccountFieldID})
	require.NoError(t, err)

	expected_blocks = []int{0, 1, 2, 2, 3, 3, 3, 4, 4, 4}
	expected_bals = []int64{2, 5, 5, 10, 5, 10, 100, 30, 10, 100}
	assertBalanceCorrect(t, expected_blocks, expected_bals, blocks[:], *bals)
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

	// Sort the accounts by ID because that's how the DB will return them in a bit
	sort.Slice(accounts, func(i, j int) bool {
		return accounts[i].ID < accounts[j].ID
	})

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

	// Query all the balances (they're all for the same asset)
	bals, err := balanceStore.Query(entities.AccountFilter{AssetID: asset.ID}, []entities.AccountField{})
	require.NoError(t, err)

	expected_blocks := []int{1, 2, 3, 4}
	expected_bals := []int64{5, 5 + 10, 5 + 10 + 100, 30 + 10 + 100}
	assertBalanceCorrect(t, expected_blocks, expected_bals, blocks[:], *bals)
}
