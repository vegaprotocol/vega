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

package execution_test

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMarketSuccession(t *testing.T) {
	exec := getMockedEngine(t)
	exec.timeSvc.EXPECT().GetTimeNow().AnyTimes().Return(time.Now())
	exec.OnSuccessorMarketTimeWindowUpdate(context.Background(), 100*time.Second)
	defer exec.ctrl.Finish()
	knownAssets := map[string]*assets.Asset{}
	mkt := getMarketConfig()
	mkt.ID = "parentID"
	// sendCount, batchCount := 0, 0
	ctx := vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("0deadbeef")))
	// for now, we don't care about this much
	// exec.epoch.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).AnyTimes()
	exec.asset.EXPECT().Get(gomock.Any()).AnyTimes().DoAndReturn(func(asset string) (*assets.Asset, error) {
		a, ok := knownAssets[asset]
		if !ok {
			a = NewAssetStub(asset, 0)
			knownAssets[asset] = a
		}
		if a == nil {
			return nil, fmt.Errorf("unknown asset")
		}
		return a, nil
	})
	// this is to propose the parent market and the 2 successors, and starting the opening auction for the eventual successor
	seen := false
	exec.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(e events.Event) {
		if e.Type() == events.MarketUpdatedEvent {
			seen = true
		}
	})
	exec.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	exec.collateral.EXPECT().GetAssetQuantum("Ethereum/Ether").AnyTimes().Return(num.DecimalFromInt64(1), nil)
	exec.collateral.EXPECT().AssetExists(gomock.Any()).AnyTimes().Return(true)
	exec.collateral.EXPECT().CreateMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	exec.oracle.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(spec.SubscriptionID(0), func(_ context.Context, _ spec.SubscriptionID) {}, nil)
	exec.statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	exec.statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// create parent market
	err := exec.SubmitMarket(ctx, mkt, "", time.Now())
	require.NoError(t, err)

	// create successors
	child1 := getMarketConfig()
	child1.ParentMarketID = mkt.ID
	child1.ID = "child1"
	child1.InsurancePoolFraction = num.DecimalFromFloat(.5)
	child1.State = types.MarketStateProposed

	child2 := getMarketConfig()
	child2.ParentMarketID = mkt.ID
	child2.ID = "child2"
	child2.InsurancePoolFraction = num.DecimalFromFloat(.33)
	child2.State = types.MarketStateActive
	// submit successor markets
	err = exec.SubmitMarket(ctx, child1, "", time.Now())
	require.NoError(t, err)
	err = exec.SubmitMarket(ctx, child2, "", time.Now())
	require.NoError(t, err)

	// when enacting a successor market, a lot of stuff happens:

	// Transfer insurance pool fraction
	acc := &types.Account{
		Balance: num.UintZero(),
	}
	exec.collateral.EXPECT().GetMarketLiquidityFeeAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(acc, nil)
	// exec.collateral.EXPECT().SuccessorInsuranceFraction(ctx, child1.ID, child1.ParentMarketID, gomock.Any(), child1.InsurancePoolFraction).Times(1).Return(&types.LedgerMovement{})
	// which in turn emits an event with ledger movements
	// we get the parent market state to pass in the ELS and stuff:
	exec.collateral.EXPECT().GetInsurancePoolBalance(child1.ParentMarketID, gomock.Any()).AnyTimes().Return(num.NewUint(100), true) // the balance doesn't matter for this test
	exec.collateral.EXPECT().GetInsurancePoolBalance(child1.ID, gomock.Any()).AnyTimes().Return(num.NewUint(50), true)              // the balance doesn't matter for this test
	exec.collateral.EXPECT().GetInsurancePoolBalance(child2.ID, gomock.Any()).AnyTimes().Return(num.NewUint(50), true)              // the balance doesn't matter for this test
	// Any accounts associated with the now rejected successor market will be removed
	// exec.collateral.EXPECT().ClearMarket(ctx, child2.ID, gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
	// statevars associated with the rejected successor market are unregistered
	exec.statevar.EXPECT().UnregisterStateVariable(gomock.Any(), child2.ID).AnyTimes()
	// the other succesor markets are rejected and removed, which emits market update events
	// set parent market to be settled
	err = exec.StartOpeningAuction(ctx, child1.ID)
	require.NoError(t, err)
	mkt.State = types.MarketStateSettled
	child1.State = types.MarketStateProposed
	// start opening auction for the successor market
	err = exec.SucceedMarket(ctx, child1.ID, child1.ParentMarketID)
	require.NoError(t, err)
	exec.OnTick(ctx, time.Now())
	require.True(t, seen)
}

func TestUpdateMarginUpdate(t *testing.T) {
	engine, ctrl := createEngine(t)
	defer ctrl.Finish()

	require.Equal(t, types.ErrInvalidMarketID, engine.UpdateMarginMode(context.Background(), "zohar", "unknown", types.MarginModeIsolatedMargin, num.DecimalOne()))

	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}
	mkt := newMarket("market-id", pubKey)
	require.NoError(t, engine.SubmitMarket(context.Background(), mkt, "zohar", time.Now()))

	// rfShort, rfLong = 1
	require.Equal(t, "margin factor (0.5) must be greater than max(riskFactorLong (1), riskFactorShort (1))", engine.UpdateMarginMode(context.Background(), "zohar", "market-id", types.MarginModeIsolatedMargin, num.DecimalFromFloat(0.5)).Error())

	// all good, just not supported yet
	require.Error(t, errors.New("Unsupported"), engine.UpdateMarginMode(context.Background(), "zohar", "market-id", types.MarginModeIsolatedMargin, num.DecimalFromFloat(1)))
}
