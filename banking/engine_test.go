package banking_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/banking/mocks"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/validators"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var (
	testAsset = assets.NewAsset(builtin.New("VGT", &types.BuiltinAsset{
		Name:   "VEGA TOKEN",
		Symbol: "VGT",
	}))
)

type testEngine struct {
	*banking.Engine
	ctrl   *gomock.Controller
	erc    *fakeERC
	col    *mocks.MockCollateral
	assets *mocks.MockAssets
	tsvc   *mocks.MockTimeService
	broker *mocks.MockBroker
}

func getTestEngine(t *testing.T) *testEngine {
	ctrl := gomock.NewController(t)
	erc := &fakeERC{}
	col := mocks.NewMockCollateral(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	tsvc := mocks.NewMockTimeService(ctrl)
	notary := mocks.NewMockNotary(ctrl)
	broker := mocks.NewMockBroker(ctrl)

	tsvc.EXPECT().NotifyOnTick(gomock.Any()).Times(1)
	eng := banking.New(logging.NewTestLogger(), banking.NewDefaultConfig(), col, erc, tsvc, assets, notary, broker)

	return &testEngine{
		Engine: eng,
		ctrl:   ctrl,
		erc:    erc,
		col:    col,
		assets: assets,
		tsvc:   tsvc,
		broker: broker,
	}
}

func TestBanking(t *testing.T) {
	t.Run("test deposit success", testDepositSuccess)
	t.Run("test deposit success - no tx duplicate", testDepositSuccessNoTxDuplicate)
	t.Run("test deposit failure", testDepositFailure)
	t.Run("test deposit failure - not builtin", testDepositFailureNotBuiltin)
	t.Run("test deposit error - start check fail", testDepositError)
}

func testDepositSuccess(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	now := time.Now()
	eng.OnTick(context.Background(), now)
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      42,
	}

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check()
	eng.erc.f(eng.erc.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.TransferResponse{}, nil)

	eng.OnTick(context.Background(), now.Add(1*time.Second))
}

func testDepositSuccessNoTxDuplicate(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(2).Return(testAsset, nil)
	now := time.Now()
	eng.OnTick(context.Background(), now.Add(1*time.Second))
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      42,
	}

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check()
	eng.erc.f(eng.erc.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.TransferResponse{}, nil)

	eng.OnTick(context.Background(), now.Add(1*time.Second))

	// call the deposit function
	err = eng.DepositBuiltinAsset(context.Background(), bad, "depositid2", 43)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check()
	eng.erc.f(eng.erc.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.TransferResponse{}, nil)

	eng.OnTick(context.Background(), now.Add(1*time.Second))
}

func testDepositFailure(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	now := time.Now()
	eng.OnTick(context.Background(), now)
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      42,
	}

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check()
	eng.erc.f(eng.erc.r, false)

	// then we call time update, expect collateral to never be called
	eng.col.EXPECT().Deposit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	eng.OnTick(context.Background(), now.Add(1*time.Second))
}

func testDepositError(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	now := time.Now()
	eng.OnTick(context.Background(), now)
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      42,
	}

	// set an error to be return by the fake erc
	expectError := errors.New("bad bad bad")
	eng.erc.err = expectError

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.EqualError(t, err, expectError.Error())
}

func testDepositFailureNotBuiltin(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	expectError := errors.New("bad bad bad")
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, expectError)
	now := time.Now()
	eng.OnTick(context.Background(), now)
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      42,
	}

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.EqualError(t, err, expectError.Error())
}

type fakeERC struct {
	r validators.Resource
	f func(interface{}, bool)
	t time.Time

	err error
}

func (f *fakeERC) StartCheck(r validators.Resource, fn func(interface{}, bool), t time.Time) error {
	f.r = r
	f.f = fn
	f.t = t
	return f.err
}
