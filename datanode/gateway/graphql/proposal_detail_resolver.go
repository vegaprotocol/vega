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

	types "code.vegaprotocol.io/vega/protos/vega"
	vega "code.vegaprotocol.io/vega/protos/vega"
)

type proposalDetailResolver VegaResolverRoot

func (r *proposalDetailResolver) Party(ctx context.Context, obj *vega.Proposal) (*vega.Party, error) {
	p, err := getParty(ctx, r.log, r.tradingDataClientV2, obj.PartyId)
	if p == nil && err == nil {
		// the api could return an nil party in some cases
		// e.g: when a party does not exists in the stores
		// this is not an error, but here we are not checking
		// if a party exists or not, but what party did propose
		p = &types.Party{Id: obj.PartyId}
	}
	return p, err
}

func (r *proposalDetailResolver) Datetime(ctx context.Context, obj *vega.Proposal) (int64, error) {
	return obj.Timestamp, nil
}

func (r *proposalDetailResolver) RejectionReason(ctx context.Context, obj *vega.Proposal) (*vega.ProposalError, error) {
	return obj.Reason, nil
}

func (r *proposalDetailResolver) RequiredLpMajority(ctx context.Context, obj *vega.Proposal) (*string, error) {
	return obj.RequiredLiquidityProviderMajority, nil
}

func (r *proposalDetailResolver) RequiredLpParticipation(ctx context.Context, obj *vega.Proposal) (*string, error) {
	return obj.RequiredLiquidityProviderParticipation, nil
}
