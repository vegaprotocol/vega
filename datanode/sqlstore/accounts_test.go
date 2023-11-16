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
	"encoding/hex"
	"testing"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/datanode/sqlstore/helpers"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAccount(t *testing.T) {
	ctx := tempTransaction(t)

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
	accTxHash := entities.TxHash(hex.EncodeToString([]byte("account_hash_1")))
	account := helpers.AddTestAccountWithTxHash(t, ctx, accountStore, party, asset, types.AccountTypeInsurance, block, accTxHash)

	// Add a second account, same asset - different party
	party2 := addTestParty(t, ctx, partyStore, block)

	accTxHash2 := entities.TxHash(hex.EncodeToString([]byte("account_hash_2")))
	account2 := helpers.AddTestAccountWithTxHash(t, ctx, accountStore, party2, asset, types.AccountTypeInsurance, block, accTxHash2)

	// Add a couple of test balances
	balTxHash := txHashFromString("balance_hash_1")
	balTxHash2 := txHashFromString("balance_hash_2")
	addTestBalance(t, balanceStore, block, account, 10, balTxHash)
	addTestBalance(t, balanceStore, block, account2, 100, balTxHash2)
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

	t.Run("get by tx hash", func(t *testing.T) {
		accounts, err := accountStore.GetByTxHash(ctx, accTxHash)
		require.NoError(t, err)
		require.Len(t, accounts, 1)
		assert.Equal(t, accounts[0], account)

		accounts2, err := accountStore.GetByTxHash(ctx, accTxHash2)
		require.NoError(t, err)
		require.Len(t, accounts2, 1)
		assert.Equal(t, accounts2[0], account2)
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

	t.Run("get balances by tx hash", func(t *testing.T) {
		balances1, err := accountStore.GetBalancesByTxHash(ctx, balTxHash)
		require.NoError(t, err)
		require.Len(t, balances1, 1)
		require.True(t, accBal1.Equal(balances1[0]))

		balances2, err := accountStore.GetBalancesByTxHash(ctx, balTxHash2)
		require.NoError(t, err)
		require.Len(t, balances2, 1)
		require.True(t, accBal2.Equal(balances2[0]))
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
