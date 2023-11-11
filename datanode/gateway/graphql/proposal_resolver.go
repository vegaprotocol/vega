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

	"code.vegaprotocol.io/vega/libs/num"
	types "code.vegaprotocol.io/vega/protos/vega"
	vega "code.vegaprotocol.io/vega/protos/vega"
)

type proposalResolver VegaResolverRoot

func (r *proposalResolver) RejectionReason(_ context.Context, data *types.GovernanceData) (*vega.ProposalError, error) {
	return data.Proposal.Reason, nil
}

func (r *proposalResolver) ID(_ context.Context, data *types.GovernanceData) (*string, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	return &data.Proposal.Id, nil
}

func (r *proposalResolver) Reference(_ context.Context, data *types.GovernanceData) (string, error) {
	if data == nil || data.Proposal == nil {
		return "", ErrInvalidProposal
	}
	return data.Proposal.Reference, nil
}

func (r *proposalResolver) Party(ctx context.Context, data *types.GovernanceData) (*types.Party, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	p, err := getParty(ctx, r.log, r.tradingDataClientV2, data.Proposal.PartyId)
	if p == nil && err == nil {
		// the api could return an nil party in some cases
		// e.g: when a party does not exists in the stores
		// this is not an error, but here we are not checking
		// if a party exists or not, but what party did propose
		p = &types.Party{Id: data.Proposal.PartyId}
	}
	return p, err
}

func (r *proposalResolver) State(_ context.Context, data *types.GovernanceData) (vega.Proposal_State, error) {
	if data == nil || data.Proposal == nil {
		return vega.Proposal_STATE_UNSPECIFIED, ErrInvalidProposal
	}
	return data.Proposal.State, nil
}

func (r *proposalResolver) Datetime(_ context.Context, data *types.GovernanceData) (int64, error) {
	if data == nil || data.Proposal == nil {
		return 0, ErrInvalidProposal
	}
	return data.Proposal.Timestamp, nil
}

func (r *proposalResolver) Rationale(_ context.Context, data *types.GovernanceData) (*types.ProposalRationale, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	return data.Proposal.Rationale, nil
}

func (r *proposalResolver) Terms(_ context.Context, data *types.GovernanceData) (*types.ProposalTerms, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	return data.Proposal.Terms, nil
}

func (r *proposalResolver) Votes(_ context.Context, obj *types.GovernanceData) (*ProposalVotes, error) {
	if obj == nil {
		return nil, ErrInvalidProposal
	}

	var yesWeight float64
	yesToken := num.UintZero()
	var yesLPWeight num.Decimal
	for _, yes := range obj.Yes {
		weight, err := strconv.ParseFloat(yes.TotalGovernanceTokenWeight, 64)
		if err != nil {
			return nil, err
		}
		yesWeight += weight
		yesUint, notOk := num.UintFromString(yes.TotalGovernanceTokenBalance, 10)
		if notOk {
			continue
		}
		yesToken.Add(yesToken, yesUint)
		weightLP, err := num.DecimalFromString(yes.TotalEquityLikeShareWeight)
		if err != nil {
			return nil, err
		}
		yesLPWeight = yesLPWeight.Add(weightLP)
	}
	var noWeight float64
	noToken := num.UintZero()
	var noLPWeight num.Decimal
	for _, no := range obj.No {
		weight, err := strconv.ParseFloat(no.TotalGovernanceTokenWeight, 64)
		if err != nil {
			return nil, err
		}
		noWeight += weight
		noUint, notOk := num.UintFromString(no.TotalGovernanceTokenBalance, 10)
		if notOk {
			continue
		}
		noToken.Add(noToken, noUint)
		weightLP, err := num.DecimalFromString(no.TotalEquityLikeShareWeight)
		if err != nil {
			return nil, err
		}
		noLPWeight = noLPWeight.Add(weightLP)
	}

	votes := &ProposalVotes{
		Yes: &ProposalVoteSide{
			Votes:                      obj.Yes,
			TotalNumber:                strconv.Itoa(len(obj.Yes)),
			TotalWeight:                strconv.FormatFloat(yesWeight, 'f', -1, 64),
			TotalTokens:                yesToken.String(),
			TotalEquityLikeShareWeight: yesLPWeight.String(),
		},
		No: &ProposalVoteSide{
			Votes:                      obj.No,
			TotalNumber:                strconv.Itoa(len(obj.No)),
			TotalWeight:                strconv.FormatFloat(noWeight, 'f', -1, 64),
			TotalTokens:                noToken.String(),
			TotalEquityLikeShareWeight: noLPWeight.String(),
		},
	}

	return votes, nil
}

func (r *proposalResolver) ErrorDetails(_ context.Context, data *types.GovernanceData) (*string, error) {
	return data.Proposal.ErrorDetails, nil
}

func (r *proposalResolver) RequiredMajority(_ context.Context, data *types.GovernanceData) (string, error) {
	return data.Proposal.RequiredMajority, nil
}

func (r *proposalResolver) RequiredParticipation(_ context.Context, data *types.GovernanceData) (string, error) {
	return data.Proposal.RequiredParticipation, nil
}

func (r *proposalResolver) RequiredLpMajority(_ context.Context, data *types.GovernanceData) (*string, error) {
	return data.Proposal.RequiredLiquidityProviderMajority, nil
}

func (r *proposalResolver) RequiredLpParticipation(_ context.Context, data *types.GovernanceData) (*string, error) {
	return data.Proposal.RequiredLiquidityProviderParticipation, nil
}
