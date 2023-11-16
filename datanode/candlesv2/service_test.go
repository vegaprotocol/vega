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

package candlesv2_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/candlesv2"
	"code.vegaprotocol.io/vega/datanode/candlesv2/mocks"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestCandleSubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockCandleStore(ctrl)
	store.EXPECT().CandleExists(
		gomock.Any(),
		gomock.Any()).Return(true, nil)

	expectedCandle := createCandle(time.Now(), time.Now(), 1, 2, 2, 1, 10, 100)

	store.EXPECT().GetCandleDataForTimeSpan(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		Return([]entities.Candle{expectedCandle}, entities.PageInfo{}, nil).AnyTimes()

	svc := candlesv2.NewService(context.Background(), logging.NewTestLogger(), candlesv2.NewDefaultConfig(), store)

	candleID := "candle1"
	_, out1, err := svc.Subscribe(context.Background(), candleID)
	if err != nil {
		t.Fatalf("failed to Subscribe: %s", err)
	}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)
}

func TestCandleUnsubscribe(t *testing.T) {
	ctrl := gomock.NewController(t)

	mockStore := mocks.NewMockCandleStore(ctrl)
	mockStore.EXPECT().CandleExists(
		gomock.Any(),
		gomock.Any()).Return(true, nil)

	testStore := &testStore{
		CandleStore: mockStore,
		candles:     make(chan []entities.Candle),
	}

	svc := candlesv2.NewService(context.Background(), logging.NewTestLogger(), candlesv2.NewDefaultConfig(), testStore)

	candleID := "candle1"
	subscriptionID, out1, err := svc.Subscribe(context.Background(), candleID)
	if err != nil {
		t.Fatalf("failed to Subscribe: %s", err)
	}

	expectedCandle := createCandle(time.Now(), time.Now(), 1, 2, 2, 1, 10, 100)
	testStore.candles <- []entities.Candle{expectedCandle}

	candle1 := <-out1
	assert.Equal(t, expectedCandle, candle1)

	svc.Unsubscribe(subscriptionID)

	_, ok := <-out1
	assert.False(t, ok, "channel should be closed")
}

type testStore struct {
	candlesv2.CandleStore
	candles chan []entities.Candle
}

func (t *testStore) GetCandleDataForTimeSpan(ctx context.Context, candleID string, from *time.Time, to *time.Time, p entities.CursorPagination) ([]entities.Candle, entities.PageInfo, error) {
	return <-t.candles, entities.PageInfo{}, nil
}
