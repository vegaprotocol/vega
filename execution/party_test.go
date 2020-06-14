package execution_test

import (
	"context"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
)

func TestNewParty(t *testing.T) {
	now := time.Now()
	trader1 := "v@d3R"
	trader2 := "B0b@f3tt"
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	lossBuf := mocks.NewMockLossSocializationBuf(ctrl)
	lossBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	lossBuf.EXPECT().Flush().AnyTimes()
	broker := mocks.NewMockBroker(ctrl)

	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), broker, lossBuf, now)
	assert.NoError(t, err)

	testMarket := getMarkets(now.AddDate(0, 0, 7))
	testMarketID := testMarket[0].Id

	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	party := execution.NewParty(log, collateralEngine, nil, broker)
	assert.NotNil(t, party)
	party = execution.NewParty(log, collateralEngine, testMarket, broker)
	assert.NotNil(t, party)

	res := party.GetByMarket("invalid")
	assert.Equal(t, 0, len(res))
	res = party.GetByMarket(testMarketID)
	assert.Equal(t, 0, len(res))

	notify1 := proto.NotifyTraderAccount{
		TraderID: trader1,
		Amount:   9876,
	}
	err = party.NotifyTraderAccount(context.Background(), &notify1)
	assert.NoError(t, err)
	err = party.NotifyTraderAccount(context.Background(), &notify1)
	assert.NoError(t, err)

	notify2 := proto.NotifyTraderAccount{
		TraderID: trader2,
		Amount:   1234,
	}

	res = party.GetByMarket(testMarketID)
	assert.Equal(t, 1, len(res))

	err = party.NotifyTraderAccountWithTopUpAmount(context.Background(), &notify2, uint64(4567))
	assert.NoError(t, err)

	res = party.GetByMarket(testMarketID)
	assert.Equal(t, 2, len(res))

	foundParty, err := party.GetByMarketAndID(testMarketID, trader1)
	assert.NoError(t, err)
	assert.NotNil(t, foundParty)
	assert.Equal(t, trader1, foundParty.Id)

	noParty, err := party.GetByMarketAndID(testMarketID, "L@nd099")
	assert.Error(t, err)
	assert.Nil(t, noParty)
	assert.Equal(t, err, execution.ErrPartyDoesNotExist)

	err = party.NotifyTraderAccountWithTopUpAmount(context.Background(), &notify2, 0)
	assert.NoError(t, err)

	notify1.Amount = 0
	err = party.NotifyTraderAccount(context.Background(), &notify1)
	assert.NoError(t, err)

	err = party.NotifyTraderAccount(context.Background(), nil)
	assert.Error(t, err)
	assert.Equal(t, err, execution.ErrNotifyPartyIdMissing)
}

func TestNewAccount(t *testing.T) {
	now := time.Now()

	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	lossBuf := mocks.NewMockLossSocializationBuf(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	lossBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	lossBuf.EXPECT().Flush().AnyTimes()

	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), broker, lossBuf, now)
	assert.NoError(t, err)

	testMarket := getMarkets(now.AddDate(0, 0, 7))
	testMarketID := testMarket[0].Id

	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	party := execution.NewParty(log, collateralEngine, nil, broker)
	assert.NotNil(t, party)
	party = execution.NewParty(log, collateralEngine, testMarket, broker)
	assert.NotNil(t, party)

	res := party.GetByMarket("invalid")
	assert.Equal(t, 0, len(res))
	res = party.GetByMarket(testMarketID)
	assert.Equal(t, 0, len(res))

	trader := "Finn the human"
	accs, err := party.MakeGeneralAccounts(context.Background(), trader)
	assert.NoError(t, err)
	assert.Len(t, accs, 1)

	accs, err = party.MakeGeneralAccounts(context.Background(), trader)
	assert.NoError(t, err)
	assert.Len(t, accs, 1)

	foundParty, err := party.GetByMarketAndID(testMarketID, trader)
	assert.NoError(t, err)
	assert.NotNil(t, foundParty)
	assert.Equal(t, trader, foundParty.Id)

	for accName, _ := range accs {
		acc, err := collateralEngine.GetAccountByID(accName)
		assert.NoError(t, err)
		assert.NotNil(t, acc)
		assert.Zero(t, acc.GetBalance())
	}
}
