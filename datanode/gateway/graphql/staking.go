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
