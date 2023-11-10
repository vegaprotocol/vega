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
	"strconv"

	proto "code.vegaprotocol.io/vega/protos/vega"
)

type delegationResolver VegaResolverRoot

func (r *delegationResolver) Party(ctx context.Context, obj *proto.Delegation) (*proto.Party, error) {
	return &proto.Party{Id: obj.Party}, nil
}

func (r *delegationResolver) Node(ctx context.Context, obj *proto.Delegation) (*proto.Node, error) {
	return r.r.getNodeByID(ctx, obj.NodeId)
}

func (r *delegationResolver) Epoch(ctx context.Context, obj *proto.Delegation) (int, error) {
	seq, err := strconv.Atoi(obj.EpochSeq)
	if err != nil {
		return -1, err
	}

	return seq, nil
}
