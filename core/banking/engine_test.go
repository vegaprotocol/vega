// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package banking_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/banking/mocks"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/validators"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

var testAsset = assets.NewAsset(builtin.New("VGT", &types.AssetDetails{
	Name:   "VEGA TOKEN",
	Symbol: "VGT",
}))

type testEngine struct {
	*banking.Engine
	ctrl                  *gomock.Controller
	erc                   *fakeERC
	col                   *mocks.MockCollateral
	assets                *mocks.MockAssets
	tsvc                  *mocks.MockTimeService
	top                   *mocks.MockTopology
	broker                *bmocks.MockBroker
	epoch                 *mocks.MockEpochService
	bridgeView            *mocks.MockERC20BridgeView
	marketActivityTracker *mocks.MockMarketActivityTracker
	ethSource             *mocks.MockEthereumEventSource
}

func getTestEngine(t *testing.T) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	erc := &fakeERC{}
	col := mocks.NewMockCollateral(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	tsvc := mocks.NewMockTimeService(ctrl)
	notary := mocks.NewMockNotary(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	top := mocks.NewMockTopology(ctrl)
	epoch := mocks.NewMockEpochService(ctrl)
	bridgeView := mocks.NewMockERC20BridgeView(ctrl)
	marketActivityTracker := mocks.NewMockMarketActivityTracker(ctrl)
	ethSource := mocks.NewMockEthereumEventSource(ctrl)

	notary.EXPECT().OfferSignatures(gomock.Any(), gomock.Any()).AnyTimes()
	epoch.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).Times(1)
	eng := banking.New(logging.NewTestLogger(), banking.NewDefaultConfig(), col, erc, tsvc, assets, notary, broker, top, epoch, marketActivityTracker, bridgeView, ethSource)

	return &testEngine{
		Engine:                eng,
		ctrl:                  ctrl,
		erc:                   erc,
		col:                   col,
		assets:                assets,
		tsvc:                  tsvc,
		broker:                broker,
		top:                   top,
		epoch:                 epoch,
		bridgeView:            bridgeView,
		marketActivityTracker: marketActivityTracker,
		ethSource:             ethSource,
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

	eng.tsvc.EXPECT().GetTimeNow().Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	eng.tsvc.EXPECT().GetTimeNow().Times(2)

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check(context.Background())
	eng.erc.f(eng.erc.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.LedgerMovement{}, nil)

	eng.tsvc.EXPECT().GetTimeNow().Times(2)
	eng.OnTick(context.Background(), time.Now())
}

func testDepositSuccessNoTxDuplicate(t *testing.T) {
	eng := getTestEngine(t)

	eng.tsvc.EXPECT().GetTimeNow().Times(6)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(2).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check(context.Background())
	eng.erc.f(eng.erc.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.LedgerMovement{}, nil)

	eng.tsvc.EXPECT().GetTimeNow().Times(4)
	eng.OnTick(context.Background(), time.Now())

	// call the deposit function
	err = eng.DepositBuiltinAsset(context.Background(), bad, "depositid2", 43)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check(context.Background())
	eng.erc.f(eng.erc.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.LedgerMovement{}, nil)

	eng.tsvc.EXPECT().GetTimeNow().Times(2)
	eng.OnTick(context.Background(), time.Now())
}

func testDepositFailure(t *testing.T) {
	eng := getTestEngine(t)

	eng.tsvc.EXPECT().GetTimeNow().Times(5)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake erc
	eng.erc.r.Check(context.Background())
	eng.erc.f(eng.erc.r, false)

	// then we call time update, expect collateral to never be called
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.OnTick(context.Background(), time.Now())
}

func testDepositError(t *testing.T) {
	eng := getTestEngine(t)

	eng.tsvc.EXPECT().GetTimeNow().Times(4)
	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
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

	eng.tsvc.EXPECT().GetTimeNow().Times(3)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	expectError := errors.New("bad bad bad")
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, expectError)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
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

func (f *fakeERC) RestoreResource(r validators.Resource, fn func(interface{}, bool)) error {
	f.r = r
	f.f = fn
	return nil
}
