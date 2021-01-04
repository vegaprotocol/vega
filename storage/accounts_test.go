package storage_test

import (
	"testing"

	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/storage"

	"github.com/stretchr/testify/assert"
)

const (
	testAccountParty1  = "g0ldman"
	testAccountParty2  = "m3tr0"
	testAccountStore   = "accountstore-test"
	testAccountMarket1 = "m@rk3t1"
	testAccountMarket2 = "tr@d1nG"
	testAssetGBP       = "GBP"
	testAssetUSD       = "USD"
	testAssetEUR       = "EUR"
)

func TestAccount_GetByPartyAndAsset(t *testing.T) {
	dir, tidy, err := storage.TempDir(testAccountStore)
	if err != nil {
		t.Fatalf("Error creating tmp dir for account store")
	}
	defer tidy()

	accountStore := createAccountStore(t, dir)

	err = accountStore.SaveBatch(getTestAccounts())
	assert.Nil(t, err)

	accs, err := accountStore.GetPartyAccounts(testAccountParty2, "", testAssetEUR, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, accs[0].Asset, testAssetEUR)
	assert.Equal(t, accs[1].Asset, testAssetEUR)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", testAssetEUR, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, err)
	assert.Len(t, accs, 0)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", testAssetUSD, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, accs[0].Asset, testAssetUSD)
	assert.Equal(t, accs[1].Asset, testAssetUSD)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", testAssetGBP, types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, accs[0].Asset, testAssetGBP)
	assert.Equal(t, accs[1].Asset, testAssetGBP)

	err = accountStore.Close()
	assert.NoError(t, err)
}

func TestAccount_GetByPartyAndType(t *testing.T) {
	invalid := "invalid type for query"

	dir, tidy, err := storage.TempDir(testAccountStore)
	if err != nil {
		t.Fatalf("Error creating tmp dir for account store")
	}
	defer tidy()

	accountStore := createAccountStore(t, dir)

	err = accountStore.SaveBatch(getTestAccounts())
	assert.Nil(t, err)

	// General accounts
	accs, err := accountStore.GetPartyAccounts(testAccountParty1, "", "", types.AccountType_ACCOUNT_TYPE_GENERAL)
	assert.Nil(t, err)
	assert.Len(t, accs, 2)

	// Margin accounts
	accs, err = accountStore.GetPartyAccounts(testAccountParty1, "", "", types.AccountType_ACCOUNT_TYPE_MARGIN)
	assert.Nil(t, err)
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
	assert.NoError(t, err)
}

func TestAccount_GetByPartyAndMarket(t *testing.T) {
	dir, tidy, err := storage.TempDir(testAccountStore)
	if err != nil {
		t.Fatalf("Error creating tmp dir for account store")
	}
	defer tidy()

	accountStore := createAccountStore(t, dir)

	err = accountStore.SaveBatch(getTestAccounts())
	assert.Nil(t, err)

	accs, err := accountStore.GetPartyAccounts(testAccountParty1, testAccountMarket1, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, testAccountMarket1, accs[0].MarketID)
	assert.Equal(t, testAccountMarket1, accs[1].MarketID)

	accs, err = accountStore.GetPartyAccounts(testAccountParty1, testAccountMarket2, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, err)
	assert.Len(t, accs, 2)
	assert.Equal(t, testAccountMarket2, accs[0].MarketID)
	assert.Equal(t, testAccountMarket2, accs[1].MarketID)

	err = accountStore.Close()
	assert.NoError(t, err)
}

func TestAccount_GetByParty(t *testing.T) {
	dir, tidy, err := storage.TempDir(testAccountStore)
	if err != nil {
		t.Fatalf("Error creating tmp dir for account store")
	}
	defer tidy()

	accountStore := createAccountStore(t, dir)

	err = accountStore.SaveBatch(getTestAccounts())
	assert.Nil(t, err)

	accs, err := accountStore.GetPartyAccounts(testAccountParty1, "", "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, err)
	assert.Len(t, accs, 4)
	assert.Equal(t, testAccountMarket1, accs[0].MarketID)

	err = accountStore.Close()
	assert.NoError(t, err)
}

func getTestAccounts() []*types.Account {
	accs := []*types.Account{
		{
			Owner:    testAccountParty1,
			MarketID: testAccountMarket1,
			Type:     types.AccountType_ACCOUNT_TYPE_GENERAL,
			Asset:    testAssetGBP,
			Balance:  1024,
		},
		{
			Owner:    testAccountParty1,
			MarketID: testAccountMarket1,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
			Asset:    testAssetGBP,
			Balance:  1024,
		},
		{
			Owner:    testAccountParty1,
			MarketID: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_GENERAL,
			Asset:    testAssetUSD,
			Balance:  1,
		},
		{
			Owner:    testAccountParty1,
			MarketID: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
			Asset:    testAssetUSD,
			Balance:  9,
		},
		{
			Owner:    testAccountParty2,
			MarketID: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_GENERAL,
			Asset:    testAssetEUR,
			Balance:  2048,
		},
		{
			Owner:    testAccountParty2,
			MarketID: testAccountMarket2,
			Type:     types.AccountType_ACCOUNT_TYPE_MARGIN,
			Asset:    testAssetEUR,
			Balance:  1024,
		},
	}
	return accs
}

func createAccountStore(t *testing.T, dir string) *storage.Account {
	config := storage.Config{
		Level:           encoding.LogLevel{Level: logging.DebugLevel},
		Accounts:        storage.DefaultStoreOptions(),
		AccountsDirPath: dir,
	}
	accountStore, err := storage.NewAccounts(logging.NewTestLogger(), config, func() {})
	assert.NoError(t, err)
	assert.NotNil(t, accountStore)

	if accountStore == nil {
		t.Fatalf("Error creating account store in unit test(s)")
	}

	return accountStore
}
