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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
)

func TestMarginLevels_Push(t *testing.T) {
	ctrl := gomock.NewController(t)

	accountSource := TestNullAccountSource{}

	store := mocks.NewMockMarginLevelsStore(ctrl)

	store.EXPECT().Add(gomock.Any()).Times(1)
	store.EXPECT().Flush(gomock.Any()).Times(2)
	subscriber := sqlsubscribers.NewMarginLevels(store, accountSource)
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{
		MaintenanceMargin:      num.NewUint(1000),
		SearchLevel:            num.NewUint(1000),
		InitialMargin:          num.NewUint(1000),
		CollateralReleaseLevel: num.NewUint(1000),
		OrderMargin:            num.UintZero(),
		Party:                  "DEADBEEF",
		MarketID:               "DEADBEEF",
		Asset:                  "DEADBEEF",
		Timestamp:              time.Now().UnixNano(),
	}))

	subscriber.Flush(context.Background())
}

type TestAccountSource struct{}

func (TestAccountSource) Obtain(ctx context.Context, a *entities.Account) error {
	a.ID = "1"
	return nil
}

func (TestAccountSource) GetByID(id entities.AccountID) (entities.Account, error) {
	panic("implement me")
}

type TestNullAccountSource struct{}

func (TestNullAccountSource) Obtain(ctx context.Context, a *entities.Account) error {
	return nil
}

func (TestNullAccountSource) GetByID(ctx context.Context, id entities.AccountID) (entities.Account, error) {
	panic("implement me")
}
