package accounts_test

import (
	"testing"

	"code.vegaprotocol.io/vega/accounts"
	"code.vegaprotocol.io/vega/accounts/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/storage"

	"github.com/golang/mock/gomock"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/assert"
)

type tstService struct {
	*accounts.Svc
	ctrl    *gomock.Controller
	storage *mocks.MockAccountStore
}

func TestAccountsService(t *testing.T) {
	t.Run("Get party accounts success", testGetPartyAccountsSuccess)
	t.Run("Get party accounts fails", testGetPartyAccountsErr)
}

func testGetPartyAccountsSuccess(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	owner, market1, market2 := "test", "BTC/DEC19", "ETH/DEC19"

	firstMarket := getTestAccounts(owner, market1,
		types.AccountType_ACCOUNT_TYPE_MARGIN,
		types.AccountType_ACCOUNT_TYPE_GENERAL,
	)
	secondMarket := getTestAccounts(owner, market2,
		types.AccountType_ACCOUNT_TYPE_MARGIN,
		types.AccountType_ACCOUNT_TYPE_GENERAL,
	)

	all := append(firstMarket, secondMarket...)

	svc.storage.EXPECT().GetPartyAccounts(owner, "", "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED).Times(1).Return(all, nil)
	accs, err := svc.GetPartyAccounts(owner, "", "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.NoError(t, err)
	assert.Equal(t, all, accs)
	// now see if we get the expected accounts (only BTC accounts) if we get party balance for a market
	svc.storage.EXPECT().GetPartyAccounts(owner, market1, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED).Times(1).Return(firstMarket[:2], nil)
	accs, err = svc.GetPartyAccounts(owner, market1, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.NoError(t, err)
	assert.Equal(t, len(firstMarket), len(accs))
	for i := range accs {
		// this slice should basically match first market, even though we returned general account for second market
		// but that used a different asset
		assert.Equal(t, firstMarket[i], accs[i])
	}
}

func testGetPartyAccountsErr(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	owner := "test"
	svc.storage.EXPECT().GetPartyAccounts(owner, "", "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED).Times(1).Return(nil, storage.ErrOwnerNotFound)
	accs, err := svc.GetPartyAccounts(owner, "", "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Error(t, err)
	assert.Nil(t, accs)
	assert.Equal(t, storage.ErrOwnerNotFound, err)

	// accounts not set up, so we can test the errors for party market balance here, too
	market := "BTC/DEC19"
	svc.storage.EXPECT().GetPartyAccounts(owner, market, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED).Times(1).Return(nil, storage.ErrOwnerNotFound)
	accs, err = svc.GetPartyAccounts(owner, market, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, accs)
	assert.Error(t, err)

	// check we're returning the correct error
	assert.Equal(t, storage.ErrOwnerNotFound, err)
	svc.storage.EXPECT().GetPartyAccounts(owner, market, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED).Times(1).Return(nil, storage.ErrMarketNotFound)
	accs, err = svc.GetPartyAccounts(owner, market, "", types.AccountType_ACCOUNT_TYPE_UNSPECIFIED)
	assert.Nil(t, accs)
	assert.Equal(t, storage.ErrMarketNotFound, err)
}

func getTestService(t *testing.T) *tstService {
	ctrl := gomock.NewController(t)
	acc := mocks.NewMockAccountStore(ctrl)
	conf := accounts.NewDefaultConfig()
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
			MarketId: market,
			Type:     t,
		}
		// general accounts don't have a market ID
		if t == types.AccountType_ACCOUNT_TYPE_GENERAL {
			acc.MarketId = ""
		}
		ret = append(ret, acc)
	}
	return ret
}
