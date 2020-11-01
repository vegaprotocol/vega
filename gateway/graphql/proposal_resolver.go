package gql

import (
	"context"

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

func (r *proposalResolver) ID(ctx context.Context, data *types.GovernanceData) (*string, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	return &data.Proposal.ID, nil
}

func (r *proposalResolver) Reference(ctx context.Context, data *types.GovernanceData) (string, error) {
	if data == nil || data.Proposal == nil {
		return "", ErrInvalidProposal
	}
	return data.Proposal.Reference, nil
}

func (r *proposalResolver) Party(ctx context.Context, data *types.GovernanceData) (*types.Party, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	p, err := getParty(ctx, r.log, r.tradingDataClient, data.Proposal.PartyID)
	if p == nil && err == nil {
		// the api could return an nil party in some cases
		// e.g: when a party does not exists in the stores
		// this is not an error, but here we are not checking
		// if a party exists or not, but what party did propose
		p = &types.Party{Id: data.Proposal.PartyID}
	}
	return p, err
}

func (r *proposalResolver) State(ctx context.Context, data *types.GovernanceData) (ProposalState, error) {
	if data == nil || data.Proposal == nil {
		return "", ErrInvalidProposal
	}
	return convertProposalStateFromProto(data.Proposal.State)
}

func (r *proposalResolver) Datetime(ctx context.Context, data *types.GovernanceData) (string, error) {
	if data == nil || data.Proposal == nil {
		return "", ErrInvalidProposal
	}
	if data.Proposal.Timestamp == 0 {
		// no timestamp for prepared proposals
		return "", nil
	}
	return nanoTSToDatetime(data.Proposal.Timestamp), nil
}

func (r *proposalResolver) Terms(ctx context.Context, data *types.GovernanceData) (*types.ProposalTerms, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	return data.Proposal.Terms, nil
}

func (r *proposalResolver) convertVotes(ctx context.Context, data []*types.Vote) ([]*Vote, error) {
	result := make([]*Vote, len(data))
	for i, v := range data {
		voter, err := getParty(ctx, r.log, r.tradingDataClient, v.PartyID)
		if err != nil {
			return nil, err
		}
		value, err := convertVoteValueFromProto(v.Value)
		if err != nil {
			return nil, err
		}
		result[i] = &Vote{
			Value:    value,
			Party:    voter,
			Datetime: nanoTSToDatetime(v.Timestamp),
		}
	}
	return result, nil
}

func (r *proposalResolver) YesVotes(ctx context.Context, data *types.GovernanceData) ([]*Vote, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	return r.convertVotes(ctx, data.Yes)
}

func (r *proposalResolver) NoVotes(ctx context.Context, data *types.GovernanceData) ([]*Vote, error) {
	if data == nil || data.Proposal == nil {
		return nil, ErrInvalidProposal
	}
	return r.convertVotes(ctx, data.No)
}
