// Copyright (c) 2023 Gobalsky Labs Limited
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

package referral_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/referral/mocks"
	"code.vegaprotocol.io/vega/core/types"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type testEngine struct {
	engine *referral.Engine
	broker *mocks.MockBroker
}

func newEngine(t *testing.T) *testEngine {
	t.Helper()

	ctrl := gomock.NewController(t)

	epochEngine := mocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any())

	broker := mocks.NewMockBroker(ctrl)

	engine := referral.NewEngine(epochEngine, broker)

	return &testEngine{
		engine: engine,
		broker: broker,
	}
}

func endEpoch(t *testing.T, ctx context.Context, te *testEngine, endTime time.Time) {
	t.Helper()

	te.engine.OnEpoch(ctx, types.Epoch{
		Action:  typespb.EpochAction_EPOCH_ACTION_END,
		EndTime: endTime,
	})
	te.engine.OnEpoch(ctx, types.Epoch{
		StartTime: endTime.Add(1 * time.Second),
		Action:    typespb.EpochAction_EPOCH_ACTION_START,
	})
}

func expectReferralProgramStartedEvent(t *testing.T, engine *testEngine) {
	t.Helper()

	engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.ReferralProgramStarted)
		assert.True(t, ok, "Event should be a ReferralProgramStarted, but is %T", evt)
	}).Times(1)
}

func expectReferralProgramEndedEvent(t *testing.T, engine *testEngine) *gomock.Call {
	t.Helper()

	return engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.ReferralProgramEnded)
		assert.True(t, ok, "Event should be a ReferralProgramEnded, but is %T", evt)
	}).Times(1)
}

func expectReferralProgramUpdatedEvent(t *testing.T, engine *testEngine) *gomock.Call {
	t.Helper()

	return engine.broker.EXPECT().Send(gomock.Any()).Do(func(evt events.Event) {
		_, ok := evt.(*events.ReferralProgramUpdated)
		assert.True(t, ok, "Event should be a ReferralProgramUpdated, but is %T", evt)
	}).Times(1)
}
