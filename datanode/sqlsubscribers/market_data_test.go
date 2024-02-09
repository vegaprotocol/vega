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

package sqlsubscribers

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"

	"github.com/golang/mock/gomock"
)

func Test_MarketData_Push(t *testing.T) {
	t.Run("Should call market data store Add", testShouldCallStoreAdd)
}

func testShouldCallStoreAdd(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockMarketDataStore(ctrl)

	store.EXPECT().Add(gomock.Any()).Times(1)

	subscriber := NewMarketData(store)
	subscriber.Push(context.Background(), events.NewMarketDataEvent(context.Background(), types.MarketData{}))
}
