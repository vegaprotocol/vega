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

	"code.vegaprotocol.io/data-node/vegatime"
	dnapiproto "code.vegaprotocol.io/protos/data-node/api/v1"
	vgproto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
)

type stakeLinkingResolver VegaResolverRoot

func (s *stakeLinkingResolver) Type(ctx context.Context, obj *eventspb.StakeLinking) (StakeLinkingType, error) {
	return convertStakeLinkingTypeFromProto(obj.Type)
}
func (s *stakeLinkingResolver) Timestamp(ctx context.Context, obj *eventspb.StakeLinking) (string, error) {
	return vegatime.Format(vegatime.Unix(obj.Ts, 0)), nil
}
func (s *stakeLinkingResolver) Party(ctx context.Context, obj *eventspb.StakeLinking) (*vgproto.Party, error) {
	return &vgproto.Party{Id: obj.Party}, nil
}

func (s *stakeLinkingResolver) Status(ctx context.Context, obj *eventspb.StakeLinking) (StakeLinkingStatus, error) {
	return convertStakeLinkingStatusFromProto(obj.Status)
}
func (s *stakeLinkingResolver) FinalizedAt(ctx context.Context, obj *eventspb.StakeLinking) (*string, error) {
	if obj.FinalizedAt == 0 {
		return nil, nil
	}
	fa := vegatime.Format(vegatime.UnixNano(obj.FinalizedAt))
	return &fa, nil
}

type partyStakeResolver VegaResolverRoot

func (p *partyStakeResolver) Linkings(ctx context.Context, obj *dnapiproto.PartyStakeResponse) ([]*eventspb.StakeLinking, error) {
	return obj.StakeLinkings, nil
}
