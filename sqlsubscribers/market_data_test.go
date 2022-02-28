package sqlsubscribers

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"github.com/golang/mock/gomock"
)

func Test_MarketData_Push(t *testing.T) {
	t.Run("Should call market data store Add", testShouldCallStoreAdd)
}

func testShouldCallStoreAdd(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	store := mocks.NewMockMarketDataStore(ctrl)

	timeout := time.Second * 5

	store.EXPECT().Add(gomock.Any()).Times(1)

	subscriber := NewMarketData(store, logging.NewTestLogger(), timeout)
	subscriber.Push(events.NewMarketDataEvent(context.Background(), types.MarketData{}))
}
