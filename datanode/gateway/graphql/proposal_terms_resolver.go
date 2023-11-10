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
	case *types.ProposalTerms_CancelTransfer:
		return obj.GetCancelTransfer(), nil
	case *types.ProposalTerms_NewTransfer:
		return obj.GetNewTransfer(), nil
	case *types.ProposalTerms_NewSpotMarket:
		return obj.GetNewSpotMarket(), nil
	case *types.ProposalTerms_UpdateSpotMarket:
		return obj.GetUpdateSpotMarket(), nil
	case *types.ProposalTerms_UpdateMarketState:
		return obj.GetUpdateMarketState(), nil
	case *types.ProposalTerms_UpdateReferralProgram:
		return obj.GetUpdateReferralProgram(), nil
	case *types.ProposalTerms_UpdateVolumeDiscountProgram:
		return obj.GetUpdateVolumeDiscountProgram(), nil
	default:
		return nil, ErrUnsupportedProposalTermsChanges
	}
}
