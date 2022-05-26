package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/protos/vega"
)

type ProposalTermsUpdateNetworkParameter struct {
	UpdateNetworkParameter *UpdateNetworkParameter
}

func (a ProposalTermsUpdateNetworkParameter) String() string {
	return fmt.Sprintf(
		"updateNetworkParameter(%s)",
		reflectPointerToString(a.UpdateNetworkParameter),
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
		reflectPointerToString(n.Changes),
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
