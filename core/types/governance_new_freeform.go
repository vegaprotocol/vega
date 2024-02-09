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

	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsNewFreeform struct {
	NewFreeform *NewFreeform
}

func (f ProposalTermsNewFreeform) String() string {
	return fmt.Sprintf(
		"newFreeForm(%s)",
		stringer.PtrToString(f.NewFreeform),
	)
}

func (f ProposalTermsNewFreeform) IntoProto() *vegapb.NewFreeform {
	var newFreeform *vegapb.NewFreeform
	if f.NewFreeform != nil {
		newFreeform = f.NewFreeform.IntoProto()
	}
	return newFreeform
}

func (f ProposalTermsNewFreeform) isPTerm() {}

func (a ProposalTermsNewFreeform) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_NewFreeform{
		NewFreeform: a.IntoProto(),
	}
}

func (a ProposalTermsNewFreeform) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_NewFreeform{
		NewFreeform: a.IntoProto(),
	}
}

func (f ProposalTermsNewFreeform) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewFreeform
}

func (f ProposalTermsNewFreeform) DeepClone() ProposalTerm {
	if f.NewFreeform == nil {
		return &ProposalTermsNewFreeform{}
	}
	return &ProposalTermsNewFreeform{
		NewFreeform: f.NewFreeform.DeepClone(),
	}
}

func NewNewFreeformFromProto(_ *vegapb.NewFreeform) *ProposalTermsNewFreeform {
	return &ProposalTermsNewFreeform{
		NewFreeform: &NewFreeform{},
	}
}

type NewFreeform struct{}

func (n NewFreeform) IntoProto() *vegapb.NewFreeform {
	return &vegapb.NewFreeform{}
}

func (n NewFreeform) String() string {
	return ""
}

func (n NewFreeform) DeepClone() *NewFreeform {
	return &NewFreeform{}
}
