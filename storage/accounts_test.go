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

package storage_test

import (
	"testing"

	"code.vegaprotocol.io/data-node/config/encoding"
	vgtesting "code.vegaprotocol.io/data-node/libs/testing"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/storage"
	types "code.vegaprotocol.io/protos/vega"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/assert"
)

const (
	testAccountParty1  = "g0ldman"
	testAccountParty2  = "m3tr0"
	testAccountMarket1 = "m@rk3t1"
	testAccountMarket2 = "tr@d1nG"
	testAssetGBP       = "GBP"
	testAssetUSD       = "USD"
	testAssetEUR       = "EUR"
)

func TestAccount_GetByPartyAndAsset(t *testing.T) {
	accountStore, cleanupFn := createAccountStore(t)
	defer cleanupFn()

	err := accountStore.SaveBatch(getTestAccounts())
	require.NoError(t, err)

	accs, err := accountStore.GetPartyAccounts(testAccountParty2, "", testAssetEUR, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, accs[0].Asset, testAssetEUR)
	assert.Equal(t, accs[1].Asset, testAssetEUR)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", testAssetEUR, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Len(t, accs, 0)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", testAssetUSD, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, accs[0].Asset, testAssetUSD)
	assert.Equal(t, accs[1].Asset, testAssetUSD)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", testAssetGBP, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, accs[0].Asset, testAssetGBP)
	assert.Equal(t, accs[1].Asset, testAssetGBP)

	err = accountStore.Close()
	require.NoError(t, err)
}

func TestAccount_GetByPartyAndType(t *testing.T) {
	invalid := "invalid type for query"

	accountStore, cleanupFn := createAccountStore(t)
	defer cleanupFn()

	err := accountStore.SaveBatch(getTestAccounts())
	require.NoError(t, err)

	// General accounts
	accs, err := accountStore.GetPartyAccounts(testAccountParty1, "", "", types.AccountType_ACCOUNT_TYPE_GENERAL)
	require.NoError(t, err)
	assert.Len(t, accs, 2)

	// Margin accounts
	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", "", types.AccountType_ACCOUNT_TYPE_MARGIN)
	require.NoError(t, err)
	assert.Len(t, accs, 2)

	// Invalid account type
	accs, err = accountStore.GetPartyAccounts(testAccountParty2, "", "", types.AccountType_ACCOUNT_TYPE_INSURANCE)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), invalid)
	assert.Nil(t, accs)

	// Invalid account type
	accs, err = accountStore.GetPartyAccounts(testAccountParty2, "", "", types.AccountType_ACCOUNT_TYPE_SETTLEMENT)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), invalid)
	assert.Nil(t, accs)

	err = accountStore.Close()
	require.NoError(t, err)
}

func TestAccount_GetByPartyAndMarket(t *testing.T) {
	accountStore, _ := createAccountStore(t)

	err := accountStore.SaveBatch(getTestAccounts())
	require.NoError(t, err)

	accs, err := accountStore.GetPartyAccounts(testAccountParty1, testAccountMarket1, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, testAccountMarket1, accs[0].MarketId)
	assert.Equal(t, testAccountMarket1, accs[1].MarketId)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, testAccountMarket2, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, testAccountMarket2, accs[0].MarketId)
	assert.Equal(t, testAccountMarket2, accs[1].MarketId)

	err = accountStore.Close()
	require.NoError(t, err)
}

func TestAccount_GetByParty(t *testing.T) {
	accountStore, _ := createAccountStore(t)

	err := accountStore.SaveBatch(getTestAccounts())
	require.NoError(t, err)

	accs, err := accountStore.GetPartyAccounts(testAccountParty1, "", "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	require.NoError(t, err)
	assert.Len(t, accs, 4)
	assert.Equal(t, testAccountMarket1, accs[0].MarketId)

	err = accountStore.Close()
	require.NoError(t, err)
}

func getTestAccounts() []*types.Account {
	accs := []*types.Account{
		{
			Owner:    testAccountParty1,
			MarketId: testAccountMarket1,
			Type:     types.AccountType_ACCOUNT_TYPE_GENERAL,
			Asset:    testAssetGBP,
			Balance:  "1024",
		},
		{
			Owner:    testAccountParty1,
			MarketId: testAccountMarket1,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
			Asset:    testAssetGBP,
			Balance:  "1024",
		},
		{
			Owner:    testAccountParty1,
			MarketId: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_GENERAL,
			Asset:    testAssetUSD,
			Balance:  "1",
		},
		{
			Owner:    testAccountParty1,
			MarketId: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
			Asset:    testAssetUSD,
			Balance:  "9",
		},
		{
			Owner:    testAccountParty2,
			MarketId: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_GENERAL,
			Asset:    testAssetEUR,
			Balance:  "2048",
		},
		{
			Owner:    testAccountParty2,
			MarketId: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
			Asset:    testAssetEUR,
			Balance:  "1024",
		},
	}
	return accs
}

func createAccountStore(t *testing.T) (*storage.Account, func()) {
	config := storage.Config{
		Level:    encoding.LogLevel{Level: logging.DebugLevel},
		Accounts: storage.DefaultStoreOptions(),
	}

	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	st, err := storage.InitialiseStorage(vegaPaths)
	require.NoError(t, err)

	accountStore, err := storage.NewAccounts(logging.NewTestLogger(), st.AccountsHome, config, func() {})
	require.NoError(t, err)
	require.NotNil(t, accountStore)

	return accountStore, cleanupFn
}
