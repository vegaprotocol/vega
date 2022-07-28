// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"context"

	"code.vegaprotocol.io/data-node/datanode/vegatime"
	proto "code.vegaprotocol.io/protos/vega"
)

type voteResolver VegaResolverRoot

func (r *voteResolver) Value(_ context.Context, obj *proto.Vote) (VoteValue, error) {
	return convertVoteValueFromProto(obj.Value)
}
func (r *voteResolver) Party(_ context.Context, obj *proto.Vote) (*proto.Party, error) {
	return &proto.Party{Id: obj.PartyId}, nil
}

func (r *voteResolver) Datetime(_ context.Context, obj *proto.Vote) (string, error) {
	return vegatime.Format(vegatime.UnixNano(obj.Timestamp)), nil
}

func (r *voteResolver) GovernanceTokenBalance(_ context.Context, obj *proto.Vote) (string, error) {
	return obj.TotalGovernanceTokenBalance, nil
}

func (r *voteResolver) GovernanceTokenWeight(_ context.Context, obj *proto.Vote) (string, error) {
	return obj.TotalGovernanceTokenWeight, nil
}
