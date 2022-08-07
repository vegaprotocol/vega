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
	"code.vegaprotocol.io/vega/logging"
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
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{}))
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{}))
	subscriber.Flush(context.Background())

	// Now push a non duplicate

	subscriber.Push(context.Background(), events.NewMarginLevelsEvent(context.Background(), types.MarginLevels{InitialMargin: num.NewUint(6)}))
	subscriber.Flush(context.Background())
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
	subscriber.Flush(context.Background())
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

	subscriber.Flush(context.Background())
}

type TestAccountSource struct{}

func (TestAccountSource) Obtain(ctx context.Context, a *entities.Account) error {
	a.ID = 1
	return nil
}

func (TestAccountSource) GetByID(id int64) (entities.Account, error) {
	panic("implement me")
}
