// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"
	"testing"

	"github.com/golang/mock/gomock"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"
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
