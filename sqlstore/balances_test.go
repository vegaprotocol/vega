package sqlstore_test

import (
	"context"
	"sort"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestBalance(t *testing.T, store *sqlstore.Balances, block entities.Block, acc entities.Account, balance int64) {
	bal := entities.Balance{
		AccountID: acc.ID,
		VegaTime:  block.VegaTime,
		Balance:   decimal.NewFromInt(balance),
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
	bals, err := balanceStore.Query(entities.AccountFilter{Asset: asset}, []entities.AccountField{})
	require.NoError(t, err)

	expected_blocks := []int{0, 1, 2, 3, 4}
	expected_bals := []int64{2, 5, 5 + 10, 5 + 10 + 100, 30 + 10 + 100}
	assertBalanceCorrect(t, expected_blocks, expected_bals, blocks[:], *bals)

	// Try just for our first account/party
	filter := entities.AccountFilter{
		Asset:   asset,
		Parties: []entities.Party{parties[0]},
	}
	bals, err = balanceStore.Query(filter, []entities.AccountField{})
	require.NoError(t, err)

	expected_blocks = []int{0, 1, 4}
	expected_bals = []int64{2, 5, 30}
	assertBalanceCorrect(t, expected_blocks, expected_bals, blocks[:], *bals)

	// Now try grouping - if we do it by account id it should split out balances for each account.
	bals, err = balanceStore.Query(entities.AccountFilter{Asset: asset}, []entities.AccountField{entities.AccountFieldID})
	require.NoError(t, err)

	expected_blocks = []int{0, 1, 2, 2, 3, 3, 3, 4, 4, 4}
	expected_bals = []int64{2, 5, 5, 10, 5, 10, 100, 30, 10, 100}
	assertBalanceCorrect(t, expected_blocks, expected_bals, blocks[:], *bals)
}
