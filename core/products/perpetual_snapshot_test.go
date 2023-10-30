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

package products_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/datasource"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/products/mocks"
	"code.vegaprotocol.io/vega/core/types"
	tmocks "code.vegaprotocol.io/vega/core/vegatime/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/logging"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
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

	restoreTime := time.Unix(1000000, 100)
	perps2, scheduleSrc := testPerpetualSnapshot(t, perps.ctrl, state2, restoreTime)

	// now we serialize again, and check the payload are same

	state3 := perps2.perpetual.Serialize()
	serialized2, err := proto.Marshal(state3)
	assert.NoError(t, err)

	assert.Equal(t, serialized1, serialized2)

	// check the the time-trigger has been set properly
	cfg := scheduleSrc.Data.GetInternalTimeTriggerSpecConfiguration()

	// trigger time in the past should fail, it should be set to restoreTime so should trigger
	// on a future time only. The trigger times are precision seconds so we pass it in truncated.
	assert.False(t, cfg.IsTriggered(restoreTime.Truncate(time.Second)))
	assert.True(t, cfg.IsTriggered(restoreTime.Add(time.Second)))
}

func TestPerpetualSnapshotNotStarted(t *testing.T) {
	perps := testPerpetual(t)

	// get fresh state before we've started the first period
	state1 := perps.perpetual.Serialize()

	serialized1, err := proto.Marshal(state1)
	assert.NoError(t, err)

	state2 := &snapshotpb.Product{}
	err = proto.Unmarshal(serialized1, state2)
	assert.NoError(t, err)

	restoreTime := time.Unix(1000000, 100)
	perps2, _ := testPerpetualSnapshot(t, perps.ctrl, state2, restoreTime)

	// now we serialize again, and check the payload are same

	state3 := perps2.perpetual.Serialize()
	serialized2, err := proto.Marshal(state3)
	assert.NoError(t, err)
	assert.Equal(t, serialized1, serialized2)
}

func testPerpetualSnapshot(t *testing.T, ctrl *gomock.Controller, state *snapshotpb.Product, tm time.Time) (*tstPerp, *datasource.Spec) {
	t.Helper()

	log := logging.NewTestLogger()
	oe := mocks.NewMockOracleEngine(ctrl)
	broker := mocks.NewMockBroker(ctrl)
	ts := tmocks.NewMockTimeService(ctrl)
	dp := uint32(1)

	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}
	factor, _ := num.DecimalFromString("0.5")
	settlementSrc := &datasource.Spec{
		Data: datasource.NewDefinition(
			datasource.ContentTypeOracle,
		).SetOracleConfig(
			&signedoracle.SpecConfiguration{
				Signers: pubKeys,
				Filters: []*dstypes.SpecFilter{
					{
						Key: &dstypes.SpecPropertyKey{
							Name:                "foo",
							Type:                datapb.PropertyKey_TYPE_INTEGER,
							NumberDecimalPlaces: ptr.From(uint64(dp)),
						},
						Conditions: nil,
					},
				},
			},
		),
	}

	definition := datasource.NewDefinition(
		datasource.ContentTypeInternalTimeTriggerTermination,
	).SetTimeTriggerTriggersConfig(
		dstypes.InternalTimeTriggers{
			&dstypes.InternalTimeTrigger{
				Initial: &tm,
				Every:   5,
			},
		},
	).SetTimeTriggerConditionConfig(
		[]*dstypes.SpecCondition{
			{
				Operator: datapb.Condition_OPERATOR_GREATER_THAN,
				Value:    "0",
			},
		},
	)
	scheduleSrc := datasource.SpecFromProto(vegapb.NewDataSourceSpec(definition.IntoProto()))

	perp := &types.Perps{
		MarginFundingFactor:                 factor,
		DataSourceSpecForSettlementData:     settlementSrc,
		DataSourceSpecForSettlementSchedule: scheduleSrc,
		DataSourceSpecBinding: &datasource.SpecBindingForPerps{
			SettlementDataProperty:     "foo",
			SettlementScheduleProperty: "vegaprotocol.builtin.timetrigger",
		},
	}
	oe.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(spec.SubscriptionID(1), func(_ context.Context, _ spec.SubscriptionID) {}, nil)
	ts.EXPECT().GetTimeNow().Times(1).Return(tm)
	perpetual, err := products.NewPerpetualFromSnapshot(context.Background(), log, perp, "", ts, oe, broker, state.GetPerps(), dp)
	if err != nil {
		t.Fatalf("couldn't create a perp for testing: %v", err)
	}

	return &tstPerp{
		perpetual: perpetual,
		oe:        oe,
		broker:    broker,
		ctrl:      ctrl,
	}, scheduleSrc
}
