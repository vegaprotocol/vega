package sqlsubscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	oraclespb "code.vegaprotocol.io/protos/vega/oracles/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/golang/mock/gomock"
)

func TestOracleData_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockOracleDataStore(ctrl)

	store.EXPECT().Add(context.Background(), gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewOracleData(store, logging.NewTestLogger())
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))
	subscriber.Push(context.Background(), events.NewOracleDataEvent(context.Background(), oraclespb.OracleData{
		PubKeys:        nil,
		Data:           nil,
		MatchedSpecIds: nil,
		BroadcastAt:    0,
	}))
}
