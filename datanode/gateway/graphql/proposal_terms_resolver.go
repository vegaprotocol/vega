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
	"errors"

	types "code.vegaprotocol.io/protos/vega"
)

var ErrUnsupportedProposalTermsChanges = errors.New("unsupported proposal terms changes")

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
	case *types.ProposalTerms_NewFreeform:
		return obj.GetNewFreeform(), nil
	default:
		return nil, ErrUnsupportedProposalTermsChanges
	}
}
