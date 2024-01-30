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

package gql

import (
	"context"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type teamResolver VegaResolverRoot

func (t teamResolver) TotalMembers(_ context.Context, obj *v2.Team) (int, error) {
	return int(obj.TotalMembers), nil
}

func (t teamResolver) CreatedAtEpoch(_ context.Context, obj *v2.Team) (int, error) {
	return int(obj.CreatedAtEpoch), nil
}

type teamRefereeResolver VegaResolverRoot

func (t teamRefereeResolver) JoinedAtEpoch(_ context.Context, obj *v2.TeamReferee) (int, error) {
	return int(obj.JoinedAtEpoch), nil
}

type teamRefereeHistoryResolver VegaResolverRoot

func (t teamRefereeHistoryResolver) JoinedAtEpoch(_ context.Context, obj *v2.TeamRefereeHistory) (int, error) {
	return int(obj.JoinedAtEpoch), nil
}

type teamStatsResolver VegaResolverRoot

func (t teamStatsResolver) TotalGamesPlayed(_ context.Context, obj *v2.TeamStatistics) (int, error) {
	return int(obj.TotalGamesPlayed), nil
}

type quantumRewardsPerEpochResolver VegaResolverRoot

func (q quantumRewardsPerEpochResolver) Epoch(_ context.Context, obj *v2.QuantumRewardsPerEpoch) (int, error) {
	return int(obj.Epoch), nil
}

type teamMemberStatsResolver VegaResolverRoot

func (t teamMemberStatsResolver) TotalGamesPlayed(_ context.Context, obj *v2.TeamMemberStatistics) (int, error) {
	return int(obj.TotalGamesPlayed), nil
}
