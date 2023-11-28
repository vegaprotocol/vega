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
		stringer.PtrToString(a.UpdateNetworkParameter),
	)
}

func (a ProposalTermsUpdateNetworkParameter) IntoProto() *vegapb.ProposalTerms_UpdateNetworkParameter {
	return &vegapb.ProposalTerms_UpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.IntoProto(),
	}
}

func (a ProposalTermsUpdateNetworkParameter) isPTerm() {}

func (a ProposalTermsUpdateNetworkParameter) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_UpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.IntoProto(),
	}
}

func (a ProposalTermsUpdateNetworkParameter) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return &vegapb.BatchProposalTermsChange_UpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.IntoProto(),
	}
}

func (a ProposalTermsUpdateNetworkParameter) GetTermType() ProposalTermsType {
	return ProposalTermsTypeUpdateNetworkParameter
}

func (a ProposalTermsUpdateNetworkParameter) DeepClone() ProposalTerm {
	if a.UpdateNetworkParameter == nil {
		return &ProposalTermsUpdateNetworkParameter{}
	}
	return &ProposalTermsUpdateNetworkParameter{
		UpdateNetworkParameter: a.UpdateNetworkParameter.DeepClone(),
	}
}

func NewUpdateNetworkParameterFromProto(
	updateNetworkParamProto *vegapb.UpdateNetworkParameter,
) *ProposalTermsUpdateNetworkParameter {
	var updateNP *UpdateNetworkParameter
	if updateNetworkParamProto != nil {
		updateNP = &UpdateNetworkParameter{}

		if updateNetworkParamProto.Changes != nil {
			updateNP.Changes = NetworkParameterFromProto(updateNetworkParamProto.Changes)
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
		stringer.PtrToString(n.Changes),
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
