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

package sqlsubscribers_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"

	"github.com/golang/mock/gomock"
)

func TestLiquidityProvision_Push(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockLiquidityProvisionStore(ctrl)

	store.EXPECT().Upsert(context.Background(), gomock.Any()).Times(1)
	store.EXPECT().Flush(gomock.Any()).Times(2)

	subscriber := sqlsubscribers.NewLiquidityProvision(store)
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewLiquidityProvisionEvent(context.Background(), &types.LiquidityProvision{}))
	subscriber.Flush(context.Background())
}
