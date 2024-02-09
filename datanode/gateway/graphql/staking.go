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
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	vgproto "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type stakeLinkingResolver VegaResolverRoot

func (s *stakeLinkingResolver) Timestamp(_ context.Context, obj *eventspb.StakeLinking) (int64, error) {
	// returning the time in nano as the timestamp marshallar expects it that way
	return time.Unix(obj.Ts, 0).UnixNano(), nil
}

func (s *stakeLinkingResolver) Party(_ context.Context, obj *eventspb.StakeLinking) (*vgproto.Party, error) {
	return &vgproto.Party{Id: obj.Party}, nil
}

func (s *stakeLinkingResolver) FinalizedAt(_ context.Context, obj *eventspb.StakeLinking) (*int64, error) {
	if obj.FinalizedAt == 0 {
		return nil, nil
	}
	return ptr.From(obj.FinalizedAt), nil
}

func (s *stakeLinkingResolver) BlockHeight(_ context.Context, obj *eventspb.StakeLinking) (string, error) {
	return fmt.Sprintf("%d", obj.BlockHeight), nil
}

type partyStakeResolver VegaResolverRoot

func (p *partyStakeResolver) Linkings(_ context.Context, obj *v2.GetStakeResponse) ([]*eventspb.StakeLinking, error) {
	linkingEdges := obj.GetStakeLinkings().GetEdges()
	linkings := make([]*eventspb.StakeLinking, 0, len(linkingEdges))
	for i := range linkingEdges {
		linkings[i] = linkingEdges[i].GetNode()
	}
	return linkings, nil
}
