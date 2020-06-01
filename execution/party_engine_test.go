package execution_test

import (
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/collateral"
	collateralmocks "code.vegaprotocol.io/vega/collateral/mocks"
	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/execution/mocks"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/proto"
)

func TestNewParty(t *testing.T) {
	now := time.Now()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	partyBuf := mocks.NewMockPartyBuf(ctrl)
	accountBuf := collateralmocks.NewMockAccountBuffer(ctrl)
	lossBuf := mocks.NewMockLossSocializationBuf(ctrl)
	lossBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	lossBuf.EXPECT().Flush().AnyTimes()

	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), accountBuf, lossBuf, now)
	assert.NoError(t, err)

	testMarket := getMarkets(now.AddDate(0, 0, 7))

	partyBuf.EXPECT().Add(gomock.Any()).AnyTimes().Return()
	accountBuf.EXPECT().Add(gomock.Any()).AnyTimes().Return()

	party := execution.NewPartyEngine(log, collateralEngine, nil, partyBuf)
	assert.NotNil(t, party)
	party = execution.NewPartyEngine(log, collateralEngine, testMarket, partyBuf)
	assert.NotNil(t, party)

	assert.Len(t, party.Parties, 0)
	notify1 := proto.NotifyTraderAccount{TraderID: "v@d3R"}
	err = party.NotifyTraderAccount(&notify1)
	assert.Len(t, party.Parties, 1)
	assert.NoError(t, err)

	asset, err := testMarket[0].GetAsset()
	assert.NoError(t, err)
	assert.NotEmpty(t, asset)
	acc, err := collateralEngine.GetPartyGeneralAccount(notify1.TraderID, asset)
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, uint64(execution.DefaultCredit), acc.GetBalance())

	foundParty, err := party.GetByID(notify1.TraderID)
	assert.NoError(t, err)
	assert.NotNil(t, foundParty)
	assert.Equal(t, notify1.TraderID, foundParty.Id)

	notify1.Amount = 9876
	err = party.NotifyTraderAccount(&notify1)
	assert.NoError(t, err)

	acc, err = collateralEngine.GetPartyGeneralAccount(notify1.TraderID, asset)
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, uint64(execution.DefaultCredit+9876), acc.GetBalance())

	notify2 := proto.NotifyTraderAccount{
		TraderID: "B0b@f3tt",
		Amount:   1234,
	}
	err = party.NotifyTraderAccount(&notify2)
	assert.NoError(t, err)

	acc, err = collateralEngine.GetPartyGeneralAccount(notify2.TraderID, asset)
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Equal(t, uint64(1234), acc.GetBalance())

	foundParty, err = party.GetByID(notify2.TraderID)
	assert.NoError(t, err)
	assert.NotNil(t, foundParty)
	assert.Equal(t, notify2.TraderID, foundParty.Id)

	noParty, err := party.GetByID("L@nd099")
	assert.Error(t, err)
	assert.Nil(t, noParty)
	assert.Equal(t, err, execution.ErrPartyDoesNotExist)

	err = party.NotifyTraderAccount(&notify2)
	assert.NoError(t, err)

	notify1.Amount = 0
	err = party.NotifyTraderAccount(&notify1)
	assert.NoError(t, err)

	err = party.NotifyTraderAccount(nil)
	assert.Error(t, err)
	assert.Equal(t, err, execution.ErrNotifyPartyIdMissing)
}

func TestNewAccount(t *testing.T) {
	now := time.Now()

	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	partyBuf := mocks.NewMockPartyBuf(ctrl)
	accountBuf := collateralmocks.NewMockAccountBuffer(ctrl)
	lossBuf := mocks.NewMockLossSocializationBuf(ctrl)
	lossBuf.EXPECT().Add(gomock.Any()).AnyTimes()
	lossBuf.EXPECT().Flush().AnyTimes()

	collateralEngine, err := collateral.New(log, collateral.NewDefaultConfig(), accountBuf, lossBuf, now)
	assert.NoError(t, err)

	markets := getMarkets(now.AddDate(0, 0, 7))

	partyBuf.EXPECT().Add(gomock.Any()).AnyTimes().Return()
	accountBuf.EXPECT().Add(gomock.Any()).AnyTimes().Return()

	engine := execution.NewPartyEngine(log, collateralEngine, markets, partyBuf)
	assert.NotNil(t, engine)

	trader := "Finn the human"
	assert.Empty(t, engine.Parties)

	added, err := engine.Add(trader)
	assert.NoError(t, err)
	assert.True(t, added)
	assert.Len(t, engine.Parties, 1, "adding party registers it with engine")
	assert.Equal(t, trader, engine.Parties[0])

	foundParty, err := engine.GetByID(trader)
	assert.NoError(t, err)
	assert.NotNil(t, foundParty)
	assert.Equal(t, trader, foundParty.Id)

	asset, err := markets[0].GetAsset()
	assert.NoError(t, err)
	assert.NotEmpty(t, asset)

	acc, err := collateralEngine.GetPartyGeneralAccount(trader, asset)
	assert.NoError(t, err)
	assert.NotNil(t, acc)
	assert.Zero(t, acc.GetBalance())
}
