package referral_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/referral"
	"code.vegaprotocol.io/vega/core/referral/mocks"
	"code.vegaprotocol.io/vega/core/types"
	typespb "code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
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
		Action: typespb.EpochAction_EPOCH_ACTION_START,
	})
}
