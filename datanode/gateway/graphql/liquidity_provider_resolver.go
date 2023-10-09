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
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type liquidityProviderResolver VegaResolverRoot

func (r *liquidityProviderResolver) FeeShare(_ context.Context, obj *v2.LiquidityProvider) (*LiquidityProviderFeeShare, error) {
	return &LiquidityProviderFeeShare{
		Party:                 &vegapb.Party{Id: obj.PartyId},
		EquityLikeShare:       obj.FeeShare.EquityLikeShare,
		AverageEntryValuation: obj.FeeShare.AverageEntryValuation,
		AverageScore:          obj.FeeShare.AverageScore,
		VirtualStake:          obj.FeeShare.VirtualStake,
	}, nil
}

func (r *liquidityProviderResolver) SLA(_ context.Context, obj *v2.LiquidityProvider) (*LiquidityProviderSLA, error) {
	return &LiquidityProviderSLA{
		Party:                            &vegapb.Party{Id: obj.PartyId},
		CurrentEpochFractionOfTimeOnBook: obj.Sla.CurrentEpochFractionOfTimeOnBook,
		LastEpochFractionOfTimeOnBook:    obj.Sla.LastEpochFractionOfTimeOnBook,
		LastEpochFeePenalty:              obj.Sla.LastEpochFeePenalty,
		LastEpochBondPenalty:             obj.Sla.LastEpochBondPenalty,
		HysteresisPeriodFeePenalties:     obj.Sla.HysteresisPeriodFeePenalties,
	}, nil
}
