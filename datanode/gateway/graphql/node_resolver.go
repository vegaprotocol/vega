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
	proto "code.vegaprotocol.io/vega/protos/vega"
)

type nodeResolver VegaResolverRoot

func (r *nodeResolver) RankingScore(ctx context.Context, obj *proto.Node) (proto.RankingScore, error) {
	return *obj.RankingScore, nil
}

func (r *nodeResolver) RewardScore(ctx context.Context, obj *proto.Node) (proto.RewardScore, error) {
	return *obj.RewardScore, nil
}

func (r *nodeResolver) DelegationsConnection(ctx context.Context, node *proto.Node, partyID *string, pagination *v2.Pagination) (*v2.DelegationsConnection, error) {
	var nodeID *string
	if node != nil {
		nodeID = &node.Id
	}
	return handleDelegationConnectionRequest(ctx, r.tradingDataClientV2, partyID, nodeID, nil, pagination)
}
