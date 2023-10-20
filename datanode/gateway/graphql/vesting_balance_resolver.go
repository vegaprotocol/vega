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

	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type partyVestingBalanceResolver VegaResolverRoot

func (r *partyVestingBalanceResolver) Asset(
	ctx context.Context, obj *eventspb.PartyVestingBalance,
) (*protoTypes.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

type partyLockedBalanceResolver VegaResolverRoot

func (r *partyLockedBalanceResolver) Asset(
	ctx context.Context, obj *eventspb.PartyLockedBalance,
) (*protoTypes.Asset, error) {
	return r.r.getAssetByID(ctx, obj.Asset)
}

func (t *partyLockedBalanceResolver) UntilEpoch(_ context.Context, obj *eventspb.PartyLockedBalance) (int, error) {
	return int(obj.UntilEpoch), nil
}

type partyVestingBalancesSummary VegaResolverRoot

func (t *partyVestingBalancesSummary) Epoch(
	_ context.Context, obj *v2.GetVestingBalancesSummaryResponse,
) (*int, error) {
	if obj.EpochSeq == nil {
		return nil, nil
	}
	return ptr.From(int(*obj.EpochSeq)), nil
}
