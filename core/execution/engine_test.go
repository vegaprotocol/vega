package execution_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/oracles"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

func TestMarketSuccession(t *testing.T) {
	exec := getMockedEngine(t)
	defer exec.ctrl.Finish()
	knownAssets := map[string]*assets.Asset{}
	mkt := getMarketConfig()
	mkt.ID = "parentID"
	// sendCount, batchCount := 0, 0
	ctx := context.Background()
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
	exec.broker.EXPECT().Send(gomock.Any()).Times(10)
	// exec.broker.EXPECT().Send(gomock.Any()).AnyTimes().Do(func(_ events.Event) {
	// sendCount++
	// })
	// exec.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes().Do(func(_ []events.Event) {
	// batchCount++
	// })
	exec.collateral.EXPECT().AssetExists(gomock.Any()).AnyTimes().Return(true)
	// exec.collateral.EXPECT().AssetExists(gomock.Any()).AnyTimes().DoAndReturn(func(a string) bool {
	// if s, ok := knownAssets[a]; ok && s != nil {
	// return true
	// }
	// return false
	// })
	exec.collateral.EXPECT().CreateMarketAccounts(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	exec.oracle.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(oracles.SubscriptionID(0), func(_ context.Context, _ oracles.SubscriptionID) {})
	exec.statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	exec.statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	exec.timeSvc.EXPECT().GetTimeNow().AnyTimes()

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
	child2.State = types.MarketStateProposed
	// submit successor markets
	err = exec.SubmitMarket(ctx, child1, "", time.Now())
	require.NoError(t, err)
	err = exec.SubmitMarket(ctx, child2, "", time.Now())
	require.NoError(t, err)

	// when enacting a successor market, a lot of stuff happens:

	// Transfer insurance pool fraction
	exec.collateral.EXPECT().SuccessorInsuranceFraction(ctx, child1.ID, child1.ParentMarketID, gomock.Any(), child1.InsurancePoolFraction).Times(1).Return(&types.LedgerMovement{})
	// which in turn emits an event with ledger movements
	exec.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		// insurance pool fraction updated
		require.Equal(t, events.LedgerMovementsEvent, evt.Type())
	})
	// we get the parent market state to pass in the ELS and stuff:
	exec.collateral.EXPECT().GetInsurancePoolBalance(child1.ParentMarketID, gomock.Any()).Times(1).Return(num.NewUint(100), true) // the balance doesn't matter for this test
	// Any accounts associated with the now rejected successor market will be removed
	exec.collateral.EXPECT().ClearMarket(ctx, child2.ID, gomock.Any(), gomock.Any()).Times(1).Return(nil, nil)
	// statevars associated with the rejected successor market are unregistered
	exec.statevar.EXPECT().UnregisterStateVariable(gomock.Any(), child2.ID).AnyTimes()
	// the other succesor markets are rejected and removed, which emits market update events
	exec.broker.EXPECT().Send(gomock.Any()).Times(1).Do(func(evt events.Event) {
		require.Equal(t, events.MarketUpdatedEvent, evt.Type())
	})
	// set parent market to be settled
	err = exec.StartOpeningAuction(ctx, child1.ID)
	require.NoError(t, err)
	mkt.State = types.MarketStateSettled
	// start opening auction for the successor market
	err = exec.SucceedMarket(ctx, child1.ID, child1.ParentMarketID, child1.InsurancePoolFraction)
	require.NoError(t, err)
}
