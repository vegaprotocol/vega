package gql

import (
	"context"
	"strconv"

	types "code.vegaprotocol.io/vega/proto"
)

type proposalResolver VegaResolverRoot

func (r *proposalResolver) RejectionReason(_ context.Context, data *types.GovernanceData) (*ProposalRejectionReason, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	p := data.Proposal
	if p.Reason == types.ProposalError_PROPOSAL_ERROR_UNSPECIFIED {
		return nil, nil
	}

	reason, err := convertProposalRejectionReasonFromProto(p.Reason)
	if err != nil {
		return nil, err
	}
	return &reason, nil
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
	p, err := getParty(ctx, r.log, r.tradingDataClient, data.Proposal.PartyId)
	if p == nil && err == nil {
		// the api could return an nil party in some cases
		// e.g: when a party does not exists in the stores
		// this is not an error, but here we are not checking
		// if a party exists or not, but what party did propose
		p = &types.Party{Id: data.Proposal.PartyId}
	}
	return p, err
}

func (r *proposalResolver) State(_ context.Context, data *types.GovernanceData) (ProposalState, error) {
	if data == nil || data.Proposal == nil {
		return "", ErrInvalidProposal
	}
	return convertProposalStateFromProto(data.Proposal.State)
}

func (r *proposalResolver) Datetime(_ context.Context, data *types.GovernanceData) (string, error) {
	if data == nil || data.Proposal == nil {
		return "", ErrInvalidProposal
	}
	if data.Proposal.Timestamp == 0 {
		// no timestamp for prepared proposals
		return "", nil
	}
	return nanoTSToDatetime(data.Proposal.Timestamp), nil
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
	var yesToken uint64
	for _, yes := range obj.Yes {
		weight, err := strconv.ParseFloat(yes.TotalGovernanceTokenWeight, 64)
		if err != nil {
			return nil, err
		}
		yesWeight += weight
		yesToken += yes.TotalGovernanceTokenBalance
	}
	var noWeight float64
	var noToken uint64
	for _, no := range obj.Yes {
		weight, err := strconv.ParseFloat(no.TotalGovernanceTokenWeight, 64)
		if err != nil {
			return nil, err
		}
		noWeight += weight
		noToken += no.TotalGovernanceTokenBalance
	}

	votes := &ProposalVotes{
		Yes: &ProposalVoteSide{
			Votes:       obj.Yes,
			TotalNumber: strconv.Itoa(len(obj.Yes)),
			TotalWeight: strconv.FormatFloat(yesWeight, 'f', -1, 64),
			TotalTokens: strconv.FormatUint(yesToken, 10),
		},
		No: &ProposalVoteSide{
			Votes:       obj.No,
			TotalNumber: strconv.Itoa(len(obj.No)),
			TotalWeight: strconv.FormatFloat(noWeight, 'f', -1, 64),
			TotalTokens: strconv.FormatUint(noToken, 10),
		},
	}

	return votes, nil
}
