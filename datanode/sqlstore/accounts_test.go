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

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"
)

func TestAccount(t *testing.T) {
	ctx, rollback := tempTransaction(t)
	defer rollback()

	blockStore := sqlstore.NewBlocks(connectionSource)
	assetStore := sqlstore.NewAssets(connectionSource)
	accountStore := sqlstore.NewAccounts(connectionSource)
	partyStore := sqlstore.NewParties(connectionSource)
	balanceStore := sqlstore.NewBalances(connectionSource)

	// Account store should be empty to begin with
	accounts, err := accountStore.GetAll(ctx)
	assert.NoError(t, err)
	assert.Empty(t, accounts)

	// Add an account
	block := addTestBlock(t, ctx, blockStore)
	asset := addTestAsset(t, ctx, assetStore, block)
	party := addTestParty(t, ctx, partyStore, block)
	account := helpers.AddTestAccount(t, ctx, accountStore, party, asset, types.AccountTypeInsurance, block)

	// Add a second account, same asset - different party
	party2 := addTestParty(t, ctx, partyStore, block)
	account2 := helpers.AddTestAccount(t, ctx, accountStore, party2, asset, types.AccountTypeInsurance, block)

	// Add a couple of test balances
	addTestBalance(t, balanceStore, block, account, 10)
	addTestBalance(t, balanceStore, block, account2, 100)
	_, err = balanceStore.Flush(ctx)
	require.NoError(t, err)

	t.Run("check we get same info back as we put in", func(t *testing.T) {
		fetchedAccount, err := accountStore.GetByID(ctx, account.ID)
		require.NoError(t, err)
		assert.Equal(t, account, fetchedAccount)
	})

	t.Run("query by asset", func(t *testing.T) {
		// Query by asset, should have 2 accounts
		filter := entities.AccountFilter{AssetID: asset.ID}
		accs, err := accountStore.Query(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, accs, 2)
	})

	t.Run("query by asset + party", func(t *testing.T) {
		// Query by asset + party should have only 1 account
		filter := entities.AccountFilter{AssetID: asset.ID, PartyIDs: []entities.PartyID{party2.ID}}
		accs, err := accountStore.Query(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, accs, 1)
		assert.Equal(t, accs[0], account2)
	})

	t.Run("query by asset + invalid type", func(t *testing.T) {
		// Query by asset + invalid type, should have 0 accounts
		filter := entities.AccountFilter{AssetID: asset.ID, AccountTypes: []types.AccountType{100}}
		accs, err := accountStore.Query(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, accs, 0)
	})

	t.Run("query by asset + invalid market", func(t *testing.T) {
		// Query by asset + invalid market, should have 0 accounts
		filter := entities.AccountFilter{AssetID: asset.ID, MarketIDs: []entities.MarketID{entities.MarketID("ffff")}}
		accs, err := accountStore.Query(ctx, filter)
		require.NoError(t, err)
		assert.Len(t, accs, 0)
	})

	accBal1 := entities.AccountBalance{Account: &account, Balance: decimal.NewFromInt(10)}
	accBal2 := entities.AccountBalance{Account: &account2, Balance: decimal.NewFromInt(100)}

	t.Run("query account balance", func(t *testing.T) {
		filter := entities.AccountFilter{AssetID: asset.ID, MarketIDs: []entities.MarketID{account.MarketID}}
		balances, pageInfo, err := accountStore.QueryBalances(ctx, filter, entities.CursorPagination{})
		require.NoError(t, err)
		require.Len(t, balances, 1)
		require.True(t, accBal1.Equal(balances[0]))
		require.False(t, pageInfo.HasNextPage)
		require.False(t, pageInfo.HasPreviousPage)
	})

	one := int32(1)
	noFilter := entities.AccountFilter{}
	firstPage, err := entities.NewCursorPagination(&one, nil, nil, nil, false)
	require.NoError(t, err)

	var cursor string

	t.Run("query account balance first page and last page", func(t *testing.T) {
		balances, pageInfo, err := accountStore.QueryBalances(ctx, noFilter, firstPage)
		require.NoError(t, err)
		require.Len(t, balances, 1)

		var lastPageAccBal entities.AccountBalance
		if accBal1.Equal(balances[0]) {
			lastPageAccBal = accBal2
		} else {
			lastPageAccBal = accBal1
		}

		require.True(t, accBal1.Equal(balances[0]) || accBal2.Equal(balances[0]))
		require.True(t, pageInfo.HasNextPage)
		require.False(t, pageInfo.HasPreviousPage)
		cursor = pageInfo.EndCursor

		lastPage, err := entities.NewCursorPagination(&one, &cursor, nil, nil, false)
		require.NoError(t, err)

		balances, pageInfo, err = accountStore.QueryBalances(ctx, noFilter, lastPage)
		require.NoError(t, err)
		require.Len(t, balances, 1)
		require.True(t, lastPageAccBal.Equal(balances[0]))
		require.False(t, pageInfo.HasNextPage)
		require.True(t, pageInfo.HasPreviousPage)
	})

	// Do this last as it will abort the transaction
	t.Run("fails if accounts are not unique", func(t *testing.T) {
		err = accountStore.Add(ctx, &account)
		assert.Error(t, err)
	})
}
