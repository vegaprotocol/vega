// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
	"github.com/stretchr/testify/require"
)

var testAsset = assets.NewAsset(builtin.New("VGT", &types.AssetDetails{
	Name:   "VEGA TOKEN",
	Symbol: "VGT",
}))

type testEngine struct {
	*banking.Engine
	ctrl                  *gomock.Controller
	witness               *fakeWitness
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
	witness := &fakeWitness{}
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
	epoch.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).AnyTimes()
	eng := banking.New(logging.NewTestLogger(), banking.NewDefaultConfig(), col, witness, tsvc, assets, notary, broker, top, marketActivityTracker, bridgeView, ethSource)

	eng.OnMaxQuantumAmountUpdate(context.Background(), num.DecimalOne())

	return &testEngine{
		Engine:                eng,
		ctrl:                  ctrl,
		witness:               witness,
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

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	// call the deposit function
	eng.tsvc.EXPECT().GetTimeNow().Times(2).Return(time.Now())
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake witness
	eng.witness.r.Check(context.Background())
	eng.witness.f(eng.witness.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.LedgerMovement{}, nil)

	eng.OnTick(context.Background(), time.Now())
}

func testDepositSuccessNoTxDuplicate(t *testing.T) {
	eng := getTestEngine(t)

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(2).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())

	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	// call the deposit function
	eng.tsvc.EXPECT().GetTimeNow().Times(2).Return(time.Now())
	require.NoError(t, eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42))

	// then we call the callback from the fake witness
	eng.witness.r.Check(context.Background())
	eng.witness.f(eng.witness.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.LedgerMovement{}, nil)

	eng.OnTick(context.Background(), time.Now())

	// call the deposit function
	eng.tsvc.EXPECT().GetTimeNow().Times(2).Return(time.Now())
	require.NoError(t, eng.DepositBuiltinAsset(context.Background(), bad, "depositid2", 43))

	// then we call the callback from the fake witness
	eng.witness.r.Check(context.Background())
	eng.witness.f(eng.witness.r, true)

	// then we call time update, which should call the collateral to
	// to do the deposit
	eng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.LedgerMovement{}, nil)

	eng.OnTick(context.Background(), time.Now())
}

func testDepositFailure(t *testing.T) {
	eng := getTestEngine(t)

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	// call the deposit function
	eng.tsvc.EXPECT().GetTimeNow().Times(2).Return(time.Now())
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake witness
	eng.witness.r.Check(context.Background())
	eng.witness.f(eng.witness.r, false)

	// then we call time update, expect collateral to never be called
	eng.OnTick(context.Background(), time.Now())
}

func testDepositError(t *testing.T) {
	eng := getTestEngine(t)

	eng.broker.EXPECT().Send(gomock.Any()).Times(1)
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	// set an error to be return by the fake witness
	expectError := errors.New("bad bad bad")
	eng.witness.err = expectError

	// call the deposit function
	eng.tsvc.EXPECT().GetTimeNow().Times(2).Return(time.Now())
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.EqualError(t, err, expectError.Error())
}

func testDepositFailureNotBuiltin(t *testing.T) {
	eng := getTestEngine(t)

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
	eng.tsvc.EXPECT().GetTimeNow().Times(1).Return(time.Now())
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.EqualError(t, err, expectError.Error())
}

type fakeWitness struct {
	r validators.Resource
	f func(interface{}, bool)
	t time.Time

	err error
}

func (f *fakeWitness) StartCheck(r validators.Resource, fn func(interface{}, bool), t time.Time) error {
	f.r = r
	f.f = fn
	f.t = t
	return f.err
}

func (f *fakeWitness) RestoreResource(r validators.Resource, fn func(interface{}, bool)) error {
	f.r = r
	f.f = fn
	return nil
}
