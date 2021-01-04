package gql

import (
	"context"
	"errors"

	types "code.vegaprotocol.io/vega/proto/gen/golang"
)

var (
	ErrUnsupportedProposalTermsChanges = errors.New("unsupported proposal terms changes")
)

type proposalTermsResolver VegaResolverRoot

func (r *proposalTermsResolver) ClosingDatetime(ctx context.Context, obj *types.ProposalTerms) (string, error) {
	return secondsTSToDatetime(obj.ClosingTimestamp), nil
}

func (r *proposalTermsResolver) EnactmentDatetime(ctx context.Context, obj *types.ProposalTerms) (string, error) {
	return secondsTSToDatetime(obj.EnactmentTimestamp), nil
}

func (r *proposalTermsResolver) Change(ctx context.Context, obj *types.ProposalTerms) (ProposalChange, error) {
	switch obj.Change.(type) {
	case *types.ProposalTerms_UpdateMarket:
		return obj.GetUpdateMarket(), nil
	case *types.ProposalTerms_UpdateNetworkParameter:
		return obj.GetUpdateNetworkParameter(), nil
	case *types.ProposalTerms_NewMarket:
		return obj.GetNewMarket(), nil
	case *types.ProposalTerms_NewAsset:
		return obj.GetNewAsset(), nil
	default:
		return nil, ErrUnsupportedProposalTermsChanges
	}
}
