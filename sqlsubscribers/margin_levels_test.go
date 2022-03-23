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
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
)

func TestMarginLevels_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockMarginLevelsStore(ctrl)

	store.EXPECT().Upsert(gomock.Any()).Times(1)
	subscriber := sqlsubscribers.NewMarginLevels(store, logging.NewTestLogger())
	subscriber.Push(events.NewTime(context.Background(), time.Now()))
	subscriber.Push(events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{
		MaintenanceMargin:      num.NewUint(1000),
		SearchLevel:            num.NewUint(1000),
		InitialMargin:          num.NewUint(1000),
		CollateralReleaseLevel: num.NewUint(1000),
		Party:                  "DEADBEEF",
		MarketID:               "DEADBEEF",
		Asset:                  "DEADBEEF",
		Timestamp:              time.Now().UnixNano(),
	}))
}
