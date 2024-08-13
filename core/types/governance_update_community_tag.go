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

package types

import (
	"fmt"
	"slices"

	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsUpdateMarketCommunityTags struct {
	UpdateMarketCommunityTags *UpdateMarketCommunityTags
}

func (f ProposalTermsUpdateMarketCommunityTags) String() string {
	return fmt.Sprintf(
		"updateCommunityTags(%s)",
		stringer.PtrToString(f.UpdateMarketCommunityTags),
	)
}

func (f ProposalTermsUpdateMarketCommunityTags) IntoProto() *vegapb.UpdateMarketCommunityTags {
	var updateCommunityTags *vegapb.UpdateMarketCommunityTags
	if f.UpdateMarketCommunityTags != nil {
		updateCommunityTags = f.UpdateMarketCommunityTags.IntoProto()
	}
	return updateCommunityTags
}

func (f ProposalTermsUpdateMarketCommunityTags) isPTerm() {}

func (a ProposalTermsUpdateMarketCommunityTags) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_UpdateMarketCommunityTags{
		UpdateMarketCommunityTags: a.IntoProto(),
	}
}

func (a ProposalTermsUpdateMarketCommunityTags) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_UpdateMarketCommunityTags{
		UpdateMarketCommunityTags: a.IntoProto(),
	}
}

func (f ProposalTermsUpdateMarketCommunityTags) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateMarketCommunityTags
}

func (f ProposalTermsUpdateMarketCommunityTags) DeepClone() ProposalTerm {
	if f.UpdateMarketCommunityTags == nil {
		return &ProposalTermsUpdateMarketCommunityTags{}
	}
	return &ProposalTermsUpdateMarketCommunityTags{
		UpdateMarketCommunityTags: f.UpdateMarketCommunityTags.DeepClone(),
	}
}

func NewUpdateMarketCommunityTagsFromProto(p *vegapb.UpdateMarketCommunityTags) *ProposalTermsUpdateMarketCommunityTags {
	return &ProposalTermsUpdateMarketCommunityTags{
		UpdateMarketCommunityTags: &UpdateMarketCommunityTags{
			MarketID:   p.Changes.MarketId,
			AddTags:    slices.Clone(p.Changes.AddTags),
			RemoveTags: slices.Clone(p.Changes.RemoveTags),
		},
	}
}

type UpdateMarketCommunityTags struct {
	MarketID   string
	AddTags    []string
	RemoveTags []string
}

func (u UpdateMarketCommunityTags) IntoProto() *vegapb.UpdateMarketCommunityTags {
	return &vegapb.UpdateMarketCommunityTags{
		Changes: &vegapb.MarketCommunityTags{
			MarketId:   u.MarketID,
			AddTags:    slices.Clone(u.AddTags),
			RemoveTags: slices.Clone(u.RemoveTags),
		},
	}
}

func (u UpdateMarketCommunityTags) String() string {
	return fmt.Sprintf(
		"marketId(%v) addTags(%v) removeTags(%v)",
		u.MarketID,
		u.AddTags,
		u.RemoveTags,
	)
}

func (u UpdateMarketCommunityTags) DeepClone() *UpdateMarketCommunityTags {
	return &UpdateMarketCommunityTags{
		MarketID:   u.MarketID,
		AddTags:    slices.Clone(u.AddTags),
		RemoveTags: slices.Clone(u.RemoveTags),
	}
}
