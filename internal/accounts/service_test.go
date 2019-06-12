package accounts_test

import (
	"testing"

	"code.vegaprotocol.io/vega/internal/accounts"
	"code.vegaprotocol.io/vega/internal/accounts/mocks"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	storcfg "code.vegaprotocol.io/vega/internal/storage/config"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

// just all account types as vars, so we don't have to clutter the tests with too many arguments when getting account return values
var (
	allTypes = []types.AccountType{
		types.AccountType_MARGIN,
		types.AccountType_MARKET,
		types.AccountType_GENERAL,
		types.AccountType_INSURANCE,
		types.AccountType_SETTLEMENT,
	}

	// trader has first 3 account types
	traderTypes = allTypes[:3]
	// system has general, insurance, settlement
	systemTypes = allTypes[2:]

	// just general type for non-market specific accounts
	nomarketTypes = allTypes[2:3]
)

type tstService struct {
	*accounts.Svc
	ctrl    *gomock.Controller
	storage *mocks.MockAccountStore
}

func TestAccountsService(t *testing.T) {
	t.Run("Get trader accounts success", testGetTraderAccountsSuccess)
	t.Run("Get trader accounts fails", testGetTraderAccountsErr)
}

func testGetTraderAccountsSuccess(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	owner, market1, market2 := "test", "BTC/DEC19", "ETH/DEC19"
	firstMarket := getTestAccounts(owner, market1, traderTypes...)
	secondMarket := getTestAccounts(owner, market2, traderTypes...)
	general := append(firstMarket[2:3], secondMarket[2:3]...)
	accounts := append(firstMarket, secondMarket...)
	svc.storage.EXPECT().GetAccountsForOwner(owner).Times(1).Return(accounts, nil)
	accs, err := svc.GetTraderAccounts(owner)
	assert.NoError(t, err)
	assert.Equal(t, accounts, accs)
	// now see if we get the expected accounts (only BTC accounts) if we get trader balance for a market
	svc.storage.EXPECT().GetMarketAccountsForOwner(owner, market1).Times(1).Return(firstMarket[:2], nil)           // get the first 2
	svc.storage.EXPECT().GetAccountsForOwnerByType(owner, types.AccountType_GENERAL).Times(1).Return(general, nil) // return all general accounts
	accs, err = svc.GetTraderMarketBalance(owner, market1)
	assert.NoError(t, err)
	assert.Equal(t, len(firstMarket), len(accs))
	for i := range accs {
		// this slice should basically match first market, even though we returned general account for second market
		// but that used a different asset
		assert.Equal(t, firstMarket[i], accs[i])
	}
}

func testGetTraderAccountsErr(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	owner := "test"
	svc.storage.EXPECT().GetAccountsForOwner(owner).Times(1).Return(nil, storage.ErrOwnerNotFound)
	accs, err := svc.GetTraderAccounts(owner)
	assert.Error(t, err)
	assert.Nil(t, accs)
	assert.Equal(t, storage.ErrOwnerNotFound, err)
	// accounts not set up, so we can test the errors for trader market balance here, too
	market := "BTC/DEC19"
	svc.storage.EXPECT().GetMarketAccountsForOwner(owner, market).Times(1).Return(nil, storage.ErrOwnerNotFound)
	accs, err = svc.GetTraderMarketBalance(owner, market)
	assert.Nil(t, accs)
	assert.Error(t, err)
	// check we're returning the correct error
	assert.Equal(t, accounts.ErrOwnerNotInMarket, err)
	svc.storage.EXPECT().GetMarketAccountsForOwner(owner, market).Times(1).Return(nil, storage.ErrMarketNotFound)
	accs, err = svc.GetTraderMarketBalance(owner, market)
	assert.Nil(t, accs)
	assert.Equal(t, storage.ErrMarketNotFound, err)
	// now error cases on general account
	// no general account
	traderAccs := getTestAccounts(owner, market, traderTypes[:2]...) // do not create general account
	svc.storage.EXPECT().GetMarketAccountsForOwner(owner, market).Times(1).Return(traderAccs, nil)
	svc.storage.EXPECT().GetAccountsForOwnerByType(owner, types.AccountType_GENERAL).Times(1).Return(nil, storage.ErrAccountNotFound)
	accs, err = svc.GetTraderMarketBalance(owner, market)
	assert.Nil(t, accs)
	assert.Equal(t, accounts.ErrNoGeneralAccount, err)
	// owner not found when getting general type account (should be impossible)
	svc.storage.EXPECT().GetMarketAccountsForOwner(owner, market).Times(1).Return(traderAccs, nil)
	svc.storage.EXPECT().GetAccountsForOwnerByType(owner, types.AccountType_GENERAL).Times(1).Return(nil, storage.ErrOwnerNotFound)
	accs, err = svc.GetTraderMarketBalance(owner, market)
	assert.Nil(t, accs)
	assert.Equal(t, storage.ErrOwnerNotFound, err)
}

func getTestService(t *testing.T) *tstService {
	ctrl := gomock.NewController(t)
	acc := mocks.NewMockAccountStore(ctrl)
	conf := storcfg.NewDefaultAccountsConfig("somedir")
	svc := accounts.NewService(logging.NewTestLogger(), conf, acc)
	return &tstService{
		Svc:     svc,
		ctrl:    ctrl,
		storage: acc,
	}
}

func getTestAccounts(owner, market string, accTypes ...types.AccountType) []*types.Account {
	asset := "BTC"
	if len(market) >= 3 {
		asset = market[:3] // first 3 chars are asset
	}
	ret := make([]*types.Account, 0, len(accTypes))
	for _, t := range accTypes {
		acc := &types.Account{
			Id:       uuid.NewV4().String(),
			Owner:    owner,
			Balance:  0,
			Asset:    asset,
			MarketID: market,
			Type:     t,
		}
		// general accounts don't have a market ID
		if t == types.AccountType_GENERAL {
			acc.MarketID = ""
		}
		ret = append(ret, acc)
	}
	return ret
}
