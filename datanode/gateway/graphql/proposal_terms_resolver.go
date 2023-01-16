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
	"errors"
	"time"

	"code.vegaprotocol.io/vega/libs/ptr"
	types "code.vegaprotocol.io/vega/protos/vega"
)

var ErrUnsupportedProposalTermsChanges = errors.New("unsupported proposal terms changes")

type proposalTermsResolver VegaResolverRoot

func (r *proposalTermsResolver) ClosingDatetime(ctx context.Context, obj *types.ProposalTerms) (int64, error) {
	// this is a unix timestamp (specified by users)
	// needs to convert to time then UnixNano for the Timestamp resolver to work
	return time.Unix(obj.ClosingTimestamp, 0).UnixNano(), nil
}

func (r *proposalTermsResolver) EnactmentDatetime(ctx context.Context, obj *types.ProposalTerms) (*int64, error) {
	var dt *int64
	if obj.EnactmentTimestamp != 0 {
		// this is a unix timestamp (specified by users)
		// needs to convert to time then UnixNano for the Timestamp resolver to work
		dt = ptr.From(time.Unix(obj.EnactmentTimestamp, 0).UnixNano())
	}
	return dt, nil
}

func (r *proposalTermsResolver) ValidationDatetime(ctx context.Context, obj *types.ProposalTerms) (*int64, error) {
	var dt *int64
	if obj.ValidationTimestamp != 0 {
		// this is a unix timestamp (specified by users)
		// needs to convert to time then UnixNano for the Timestamp resolver to work
		dt = ptr.From(time.Unix(obj.ValidationTimestamp, 0).UnixNano())
	}
	return dt, nil
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
	case *types.ProposalTerms_UpdateAsset:
		return obj.GetUpdateAsset(), nil
	default:
		return nil, ErrUnsupportedProposalTermsChanges
	}
}
