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
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
)

func TestGameScore_Push(t *testing.T) {
	ctrl := gomock.NewController(t)

	store := mocks.NewMockGameScoreStore(ctrl)

	store.EXPECT().AddPartyScore(gomock.Any(), gomock.Any()).Times(2)
	store.EXPECT().AddTeamScore(gomock.Any(), gomock.Any()).Times(2)
	subscriber := sqlsubscribers.NewGameScore(store)
	subscriber.Flush(context.Background())
	subscriber.Push(context.Background(), events.NewTeamGameScoresEvent(
		context.Background(),
		1,
		"game1",
		time.Now(),
		[]*types.PartyContributionScore{{Party: "team1", Score: num.DecimalOne()}, {Party: "team2", Score: num.DecimalOne()}},
		map[string][]*types.PartyContributionScore{
			"team1": {{Party: "party1", Score: num.DecimalOne(), StakingBalance: num.UintZero(), OpenVolume: num.UintZero(), TotalFeesPaid: num.UintZero()}},
			"team2": {{Party: "party2", Score: num.DecimalOne(), StakingBalance: num.UintZero(), OpenVolume: num.UintZero(), TotalFeesPaid: num.UintZero()}},
		}))
}
