package sqlsubscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
)

func TestMarginLevelsDuplicate_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockMarginLevelsStore(ctrl)

	store.EXPECT().Add(gomock.Any()).Times(2)
	store.EXPECT().Flush(gomock.Any()).Times(4)

	accountSource := TestAccountSource{}

	subscriber := sqlsubscribers.NewMarginLevels(store, accountSource, logging.NewTestLogger())
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))
	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{}))
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))
	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{}))
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))

	// Now push a non duplicate

	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{InitialMargin: num.NewUint(6)}))
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))

}

func TestMarginLevels_Push(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	accountSource := mocks.NewMockAccountSource(ctrl)
	accountSource.EXPECT().Obtain(gomock.Any(), gomock.Any()).AnyTimes().Return(nil)

	store := mocks.NewMockMarginLevelsStore(ctrl)

	store.EXPECT().Add(gomock.Any()).Times(1)
	store.EXPECT().Flush(gomock.Any()).Times(2)
	subscriber := sqlsubscribers.NewMarginLevels(store, accountSource, logging.NewTestLogger())
	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))
	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{
		MaintenanceMargin:      num.NewUint(1000),
		SearchLevel:            num.NewUint(1000),
		InitialMargin:          num.NewUint(1000),
		CollateralReleaseLevel: num.NewUint(1000),
		Party:                  "DEADBEEF",
		MarketID:               "DEADBEEF",
		Asset:                  "DEADBEEF",
		Timestamp:              time.Now().UnixNano(),
	}))

	subscriber.Push(context.Background(), events.NewTime(context.Background(), time.Now()))
}

type TestAccountSource struct{}

func (TestAccountSource) Obtain(ctx context.Context, a *entities.Account) error {
	a.ID = 1
	return nil
}

func (TestAccountSource) GetByID(id int64) (entities.Account, error) {
	panic("implement me")
}
