package sqlstore_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/vega/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func addTestAccount(t *testing.T,
	accountStore *sqlstore.Accounts,
	party entities.Party,
	asset entities.Asset,
	block entities.Block,
) entities.Account {
	account := entities.Account{
		PartyID:  party.ID,
		AssetID:  asset.ID,
		MarketID: entities.NewMarketID(generateID()),
		Type:     1,
		VegaTime: block.VegaTime,
	}

	err := accountStore.Add(context.Background(), &account)
	require.NoError(t, err)
	return account
}

func TestAccount(t *testing.T) {
	defer DeleteEverything()

	blockStore := sqlstore.NewBlocks(connectionSource)
	assetStore := sqlstore.NewAssets(connectionSource)
	accountStore := sqlstore.NewAccounts(connectionSource)
	partyStore := sqlstore.NewParties(connectionSource)

	// Account store should be empty to begin with
	accounts, err := accountStore.GetAll()
	assert.NoError(t, err)
	assert.Empty(t, accounts)

	// Add an account
	block := addTestBlock(t, blockStore)
	asset := addTestAsset(t, assetStore, block)
	party := addTestParty(t, partyStore, block)
	account := addTestAccount(t, accountStore, party, asset, block)

	// Add it again, we should get a primary key violation
	err = accountStore.Add(context.Background(), &account)
	assert.Error(t, err)

	// Query and check we've got back an asset the same as the one we put in
	fetchedAccount, err := accountStore.GetByID(account.ID)
	assert.NoError(t, err)
	assert.Equal(t, account, fetchedAccount)

	// Add a second account, same asset - different party
	party2 := addTestParty(t, partyStore, block)
	account2 := addTestAccount(t, accountStore, party2, asset, block)

	// Query by asset, should have 2 accounts
	filter := entities.AccountFilter{Asset: asset}
	accs, err := accountStore.Query(filter)
	assert.NoError(t, err)
	assert.Len(t, accs, 2)

	// Query by asset + party should have only 1 account
	filter = entities.AccountFilter{Asset: asset, Parties: []entities.Party{party2}}
	accs, err = accountStore.Query(filter)
	require.NoError(t, err)
	assert.Len(t, accs, 1)
	assert.Equal(t, accs[0], account2)

	// Query by asset + invalid type, should have 0 accounts
	filter = entities.AccountFilter{Asset: asset, AccountTypes: []types.AccountType{100}}
	accs, err = accountStore.Query(filter)
	assert.NoError(t, err)
	assert.Len(t, accs, 0)

	// Query by asset + invalid market, should have 0 accounts
	filter = entities.AccountFilter{Asset: asset, Markets: []entities.Market{{ID: entities.NewMarketID("ffff")}}}
	accs, err = accountStore.Query(filter)
	require.NoError(t, err)
	assert.Len(t, accs, 0)

	// QueryBalance correctly filters on marketID
	filter = entities.AccountFilter{Asset: asset, Markets: []entities.Market{{ID: entities.NewMarketID(account.MarketID.String())}}}
	_, err = accountStore.QueryBalances(context.Background(), filter, entities.OffsetPagination{})
	require.NoError(t, err)
}
