package candlesv2_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/candlesv2"
	"code.vegaprotocol.io/data-node/candlesv2/mocks"
	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

func TestCandleSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	store := mocks.NewMockCandleStore(ctrl)
	store.EXPECT().CandleExists(
		gomock.Any(),
		gomock.Any()).Return(true, nil)

	expectedCandle := createCandle(time.Now(), time.Now(), 1, 2, 2, 1, 10)

	store.EXPECT().GetCandleDataForTimeSpan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]entities.Candle{expectedCandle}, entities.PageInfo{}, nil).AnyTimes()

	svc := candlesv2.NewService(context.Background(), logging.NewTestLogger(), candlesv2.NewDefaultConfig(), store)

	candleId := "candle1"
	_, out1, err := svc.Subscribe(context.Background(), candleId)
	if err != nil {
		t.Fatalf("failed to Subscribe: %s", err)
	}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)
}

func TestCandleUnsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()
	mockStore := mocks.NewMockCandleStore(ctrl)
	mockStore.EXPECT().CandleExists(
		gomock.Any(),
		gomock.Any()).Return(true, nil)

	testStore := &testStore{
		CandleStore: mockStore,
		candles:     make(chan []entities.Candle),
	}

	svc := candlesv2.NewService(context.Background(), logging.NewTestLogger(), candlesv2.NewDefaultConfig(), testStore)

	candleId := "candle1"
	subscriptionId, out1, err := svc.Subscribe(context.Background(), candleId)
	if err != nil {
		t.Fatalf("failed to Subscribe: %s", err)
	}

	expectedCandle := createCandle(time.Now(), time.Now(), 1, 2, 2, 1, 10)
	testStore.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	svc.Unsubscribe(subscriptionId)

	_, ok := <-out1
	assert.False(t, ok, "channel should be closed")
}

type testStore struct {
	candlesv2.CandleStore
	candles chan []entities.Candle
}

func (t *testStore) GetCandleDataForTimeSpan(ctx context.Context, candleId string, from *time.Time, to *time.Time, p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error) {
	return <-t.candles, entities.PageInfo{}, nil
}
