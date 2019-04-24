package storage_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	types "code.vegaprotocol.io/vega/proto"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

func TestAccounts(t *testing.T) {
	t.Run("Create duplicate account", testAddDuplicate)
	t.Run("Create market accounts", testCreateMarketAccounts)
}

func testAddDuplicate(t *testing.T) {
	acc := getAccountStorage()
	rec := types.Account{
		Id:     uuid.NewV4().String(),
		Asset:  "GBP",
		Market: "market",
		Owner:  uuid.NewV4().String(),
		Type:   types.AccountType_GENERAL,
	}
	assert.NoError(t, acc.Create(&rec))
	assert.Equal(t, storage.ErrDuplicateAccount, acc.Create(&rec))
}

func testCreateMarketAccounts(t *testing.T) {
	acc := getAccountStorage()
	market := "market"
	settlement := int64(123)
	assert.NoError(t, acc.CreateMarketAccounts(market, settlement))
	assert.Equal(t, storage.ErrMarketAccountsExist, acc.CreateMarketAccounts(market, settlement))
	accounts, err := acc.GetMarketAccounts(market)
	assert.NoError(t, err)
	assert.NotEmpty(t, accounts)
	for _, account := range accounts {
		assert.Equal(t, market, account.Market)
		assert.Equal(t, storage.SystemOwner, account.Owner)
		if account.Type == types.AccountType_INSURANCE {
			assert.Equal(t, settlement, account.Balance)
		}
	}
	sysLen := len(accounts)
	err = acc.CreateMarketAccounts(market, settlement)
	assert.Error(t, err)
	assert.Equal(t, storage.ErrMarketAccountsExist, err)
	assert.NoError(t, acc.CreateTraderMarketAccounts(uuid.NewV4().String(), market))
	accounts, err = acc.GetMarketAccounts(market)
	assert.NotEqual(t, sysLen, len(accounts))
	types := []types.AccountType{
		types.AccountType_MARGIN,
		types.AccountType_MARKET,
	}
	var owner string
	for _, account := range accounts {
		if account.Owner != storage.SystemOwner {
			owner = account.Owner
			assert.Contains(t, types, account.Type)
		}
	}
	accounts, err = acc.GetAccountsForOwner(owner)
	assert.NoError(t, err)
	assert.Equal(t, 3, len(accounts))
	_, err = acc.GetAccountsForOwner("foobar")
	assert.Equal(t, storage.ErrOwnerNotFound, err)
	accounts, err = acc.GetMarketAccountsForOwner(market, owner)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(accounts))
}

func getAccountStorage() *storage.Account {
	conf, _ := storage.NewTestConfig()
	acc, _ := storage.NewAccounts(logging.NewTestLogger(), conf)
	return acc
}
