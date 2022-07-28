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

package price_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/libs/crypto"
	"code.vegaprotocol.io/vega/core/libs/proto"
	"code.vegaprotocol.io/vega/core/monitor/price"
	"code.vegaprotocol.io/vega/core/monitor/price/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createPriceMonitor(t *testing.T, ctrl *gomock.Controller) *price.Engine {
	t.Helper()

	riskModel, auctionState, settings := createPriceMonitorDeps(t, ctrl)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any())

	pm, err := price.NewMonitor("asset", "market", riskModel, auctionState, settings, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm)

	return pm
}

func createPriceMonitorDeps(t *testing.T, ctrl *gomock.Controller) (*mocks.MockRangeProvider, *mocks.MockAuctionState, *types.PriceMonitoringSettings) {
	t.Helper()
	riskModel := mocks.NewMockRangeProvider(ctrl)
	auctionStateMock := mocks.NewMockAuctionState(ctrl)

	settings := &types.PriceMonitoringSettings{
		Parameters: &types.PriceMonitoringParameters{
			Triggers: []*types.PriceMonitoringTrigger{},
		},
	}

	auctionStateMock.EXPECT().IsFBA().Return(false).AnyTimes()
	auctionStateMock.EXPECT().InAuction().Return(false).AnyTimes()

	return riskModel, auctionStateMock, settings
}

func getHash(pe *price.Engine) []byte {
	state := pe.GetState()
	pmproto := state.IntoProto()
	bytes, _ := proto.Marshal(pmproto)
	return crypto.Hash(bytes)
}

func TestEmpty(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pm1 := createPriceMonitor(t, ctrl)
	assert.NotNil(t, pm1)

	// Get the initial state
	hash1 := getHash(pm1)
	state1 := pm1.GetState()

	// Create a new market and restore into it
	riskModel, auctionState, settings := createPriceMonitorDeps(t, ctrl)

	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	pm2, err := price.NewMonitorFromSnapshot("marketID", "assetID", state1, settings, riskModel, auctionState, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm2)

	// Now get the state again and check it against the original
	hash2 := getHash(pm2)

	assert.Equal(t, hash1, hash2)
}

func TestChangedState(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	pm1 := createPriceMonitor(t, ctrl)
	assert.NotNil(t, pm1)

	// Get the initial state
	hash1 := getHash(pm1)

	// Perform some actions on the object
	as := mocks.NewMockAuctionState(ctrl)
	as.EXPECT().IsFBA().Return(false).Times(10)
	as.EXPECT().InAuction().Return(false).Times(10)

	now := time.Now()

	for i := 0; i < 10; i++ {
		pm1.OnTimeUpdate(now)
		p := []*types.Trade{{Price: num.NewUint(uint64(100 + i)), Size: uint64(100 + i)}}
		b := pm1.CheckPrice(context.Background(), as, p, true)
		now = now.Add(time.Minute * 1)
		require.False(t, b)
	}

	// Check something has changed
	assert.True(t, pm1.Changed())

	// Get the new hash after the change
	hash2 := getHash(pm1)
	assert.NotEqual(t, hash1, hash2)

	// Now try reloading the state
	state := pm1.GetState()
	assert.Len(t, state.PricesNow, 1)
	assert.Len(t, state.PricesPast, 9)

	riskModel, auctionState, settings := createPriceMonitorDeps(t, ctrl)
	statevar := mocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	pm2, err := price.NewMonitorFromSnapshot("marketID", "assetID", state, settings, riskModel, auctionState, statevar, logging.NewTestLogger())
	require.NoError(t, err)
	require.NotNil(t, pm2)

	hash3 := getHash(pm2)
	assert.Equal(t, hash2, hash3)

	state2 := pm1.GetState()
	assert.Len(t, state2.PricesNow, 1)
	assert.Len(t, state2.PricesPast, 9)

	asProto := state2.IntoProto()
	state3 := types.PriceMonitorFromProto(asProto)
	assert.Len(t, state3.PricesNow, 1)
	assert.Len(t, state3.PricesPast, 9)
	assert.Equal(t, state2.Now.UnixNano(), state3.Now.UnixNano())
	assert.Equal(t, state2.Update.UnixNano(), state3.Update.UnixNano())
	assert.Equal(t, state2.PricesPast[0].Time.UnixNano(), state3.PricesPast[0].Time.UnixNano())
}
