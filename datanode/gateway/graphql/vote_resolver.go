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

	proto "code.vegaprotocol.io/vega/protos/vega"
)

type voteResolver VegaResolverRoot

func (r *voteResolver) Party(_ context.Context, obj *proto.Vote) (*proto.Party, error) {
	return &proto.Party{Id: obj.PartyId}, nil
}

func (r *voteResolver) Datetime(_ context.Context, obj *proto.Vote) (int64, error) {
	return obj.Timestamp, nil
}

func (r *voteResolver) GovernanceTokenBalance(_ context.Context, obj *proto.Vote) (string, error) {
	return obj.TotalGovernanceTokenBalance, nil
}

func (r *voteResolver) GovernanceTokenWeight(_ context.Context, obj *proto.Vote) (string, error) {
	return obj.TotalGovernanceTokenWeight, nil
}

func (r *voteResolver) EquityLikeShareWeight(_ context.Context, obj *proto.Vote) (string, error) {
	return obj.TotalEquityLikeShareWeight, nil
}

func (r *voteResolver) EquityLikeSharePerMarket(_ context.Context, obj *proto.Vote) ([]*proto.VoteELSPair, error) {
	return obj.ElsPerMarket, nil
}
