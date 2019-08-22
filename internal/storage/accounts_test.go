package storage_test

import (
	types "code.vegaprotocol.io/vega/proto"
	"testing"

	"code.vegaprotocol.io/vega/internal/config/encoding"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"

	"github.com/stretchr/testify/assert"
)


const (
	testAccountStore   = "accountstore-test"
	testAccountMarket1 = "m@rk3t1"
	testAccountMarket2 = "tr@d1nG"
	testAssetGBP       = "GBP"
	testAssetUSD       = "USD"
	testAssetEUR       = "EUR"
)

//
//func TestMarkets(t *testing.T) {
//	dir, tidy, err := storage.TempDir("marketstore-test")
//	if err != nil {
//		t.Fatalf("Failed to create tmp dir: %s", err.Error())
//	}
//	defer tidy()
//
//	config := storage.Config{
//		Level:          encoding.LogLevel{Level: logging.DebugLevel},
//		Markets:        storage.DefaultMarketStoreOptions(),
//		MarketsDirPath: dir,
//	}
//	marketStore, err := storage.NewMarkets(logging.NewTestLogger(), config)
//	assert.NoError(t, err)
//	assert.NotNil(t, marketStore)
//	if marketStore == nil {
//		t.Fatalf("Could not create market store. Giving up.")
//	}
//	defer marketStore.Close()
//
//	err = marketStore.Commit() // no-op for in-memory store
//	assert.NoError(t, err)
//
//	config.Level.Level = logging.InfoLevel
//	marketStore.ReloadConf(config)
//
//	mkt := types.Market{
//		Id:   testMarketId,
//		Name: testMarketName,
//	}
//	err = marketStore.Post(&mkt)
//	assert.NoError(t, err)
//
//	mkt2, err := marketStore.GetByID("nonexistant_market")
//	assert.Equal(t, badger.ErrKeyNotFound, err)
//	assert.Nil(t, mkt2)
//
//	mkt3, err := marketStore.GetByID(testMarketId)
//	assert.NoError(t, err)
//	assert.NotNil(t, mkt3)
//	assert.Equal(t, mkt.Id, mkt3.Id)
//
//	mkts, err := marketStore.GetAll()
//	assert.NoError(t, err)
//	assert.NotNil(t, mkts)
//	assert.Equal(t, 1, len(mkts))
//	assert.Equal(t, mkt.Id, mkts[0].Id)
//
//	err = marketStore.Post(nil)
//	assert.Error(t, err)
//}

func TestAccount_New(t *testing.T) {

}

func TestAccount_GetByParty(t *testing.T) {
	dir, tidy := createTmpDir(t, testAccountStore)
	defer tidy()

	accountStore := createAccountStore(t, dir)

	err := accountStore.SaveBatch(getTestAccounts())
	assert.Nil(t, err)

	accs, err := accountStore.GetByParty(testParty)
	assert.Nil(t, err)
	assert.Len(t, accs, 4)
	assert.Equal(t, testAccountMarket1, accs[0].MarketID)

	err = accountStore.Close()
	assert.NoError(t, err)


}

func getTestAccounts() []*types.Account {
	accs := []*types.Account {
		{
			Owner: testParty,
			MarketID: testAccountMarket1,
			Type: types.AccountType_GENERAL,
			Asset: testAssetGBP,
		},
		{
			Owner: testParty,
			MarketID: testAccountMarket1,
			Type: types.AccountType_MARGIN,
			Asset: testAssetGBP,
		},
		{
			Owner: testParty,
			MarketID: testAccountMarket2,
			Type: types.AccountType_GENERAL,
			Asset: testAssetUSD,
		},
		{
			Owner: testParty,
			MarketID: testAccountMarket2,
			Type: types.AccountType_MARGIN,
			Asset: testAssetUSD,
		},
	}
	return accs
}

func createTmpDir(t *testing.T, storePath string) (string, func()) {
	dir, tidy, err := storage.TempDir(storePath)
	if err != nil {
		t.Fatalf("Failed to create tmp dir: %s", err.Error())
	}
	return dir, tidy
}

func createAccountStore(t *testing.T, dir string) *storage.Account {
	config := storage.Config{
		Level:           encoding.LogLevel{Level: logging.DebugLevel},
		Accounts:        storage.DefaultAccountStoreOptions(),
		AccountsDirPath: dir,
	}
	accountStore, err := storage.NewAccounts(logging.NewTestLogger(), config)
	assert.NoError(t, err)
	assert.NotNil(t, accountStore)

	if accountStore == nil {
		t.Fatalf("Error creating account store in unit test(s)")
	}

	return accountStore

}
