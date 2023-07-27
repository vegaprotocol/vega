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

package products_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPerpetualSnapshot(t *testing.T) {
	perps := testPerpetual(t)

	// set of the data points such that difference in averages is 0
	points := getTestDataPoints(t)
	ctx := context.Background()

	// tell the perpetual that we are ready to accept settlement stuff
	perps.broker.EXPECT().Send(gomock.Any()).Times(1)
	perps.perpetual.OnLeaveOpeningAuction(ctx, 1000)

	// send in some data points
	perps.broker.EXPECT().Send(gomock.Any()).Times(len(points) * 2)
	for _, p := range points {
		// send in an external and a matching internal
		require.NoError(t, perps.perpetual.SubmitDataPoint(ctx, p.price, p.t))
		perps.perpetual.AddTestExternalPoint(ctx, p.price, p.t)
	}

	// now get the serialised state, and try to load it
	state1 := perps.perpetual.Serialize()

	serialized1, err := proto.Marshal(state1)
	assert.NoError(t, err)

	state2 := &snapshotpb.Product{}
	err = proto.Unmarshal(serialized1, state2)
	assert.NoError(t, err)

	perps2 := testPerpetualSnapshot(t, perps.ctrl, state2)

	// now we serialize again, and check the payload are same

	state3 := perps2.perpetual.Serialize()
	serialized2, err := proto.Marshal(state3)
	assert.NoError(t, err)

	assert.Equal(t, serialized1, serialized2)
}

func testPerpetualSnapshot(t *testing.T, ctrl *gomock.Controller, state *snapshotpb.Product) *tstPerp {
	t.Helper()

	log := logging.NewTestLogger()
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)

	factor, _ := num.DecimalFromString("0.5")
	perp := &types.Perpetual{
		MarginFundingFactor: &factor,
	}

	perpetual, err := products.NewPerpetualFromSnapshot(context.Background(), log, perp, oe, broker, state.GetPerps())
	if err != nil {
		t.Fatalf("couldn't create a Future for testing: %v", err)
	}
	return &tstPerp{
		perpetual: perpetual,
		oe:        oe,
		broker:    broker,
		ctrl:      ctrl,
	}
}
