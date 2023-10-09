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

type ProposalTermsUpdateNetworkParameter struct {
	UpdateNetworkParameter *UpdateNetworkParameter
}

func (a ProposalTermsUpdateNetworkParameter) String() string {
	return fmt.Sprintf(
		"updateNetworkParameter(%s)",
		stringer.ReflectPointerToString(a.UpdateNetworkParameter),
	)
}

func (a ProposalTermsUpdateNetworkParameter) IntoProto() *vegapb.ProposalTerms_UpdateNetworkParameter {
	return &vegapb.ProposalTerms_UpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.IntoProto(),
	}
}

func (a ProposalTermsUpdateNetworkParameter) isPTerm() {}

func (a ProposalTermsUpdateNetworkParameter) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsUpdateNetworkParameter) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateNetworkParameter
}

func (a ProposalTermsUpdateNetworkParameter) DeepClone() proposalTerm {
	if a.UpdateNetworkParameter == nil {
		return &ProposalTermsUpdateNetworkParameter{}
	}
	return &ProposalTermsUpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.DeepClone(),
	}
}

func NewUpdateNetworkParameterFromProto(
	p *vegapb.ProposalTerms_UpdateNetworkParameter,
) *ProposalTermsUpdateNetworkParameter {
	var updateNP *UpdateNetworkParameter
	if p.UpdateNetworkParameter != nil {
		updateNP = &UpdateNetworkParameter{}

		if p.UpdateNetworkParameter.Changes != nil {
			updateNP.Changes = NetworkParameterFromProto(p.UpdateNetworkParameter.Changes)
		}
	}

	return &ProposalTermsUpdateNetworkParameter{
		UpdateNetworkParameter: updateNP,
	}
}

type UpdateNetworkParameter struct {
	Changes *NetworkParameter
}

func (n UpdateNetworkParameter) IntoProto() *vegapb.UpdateNetworkParameter {
	return &vegapb.UpdateNetworkParameter{
		Changes: n.Changes.IntoProto(),
	}
}

func (n UpdateNetworkParameter) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.ReflectPointerToString(n.Changes),
	)
}

func (n UpdateNetworkParameter) DeepClone() *UpdateNetworkParameter {
	if n.Changes == nil {
		return &UpdateNetworkParameter{}
	}
	return &UpdateNetworkParameter{
		Changes: n.Changes.DeepClone(),
	}
}
