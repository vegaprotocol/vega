package sqlsubscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
)

func TestStakeLinking_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockStakeLinkingStore(ctrl)

	store.EXPECT().Upsert(gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewStakeLinking(store, logging.NewTestLogger())
	subscriber.Push(events.NewTime(context.Background(), time.Now()))
	subscriber.Push(events.NewStakeLinking(context.Background(), types.StakeLinking{}))
}
