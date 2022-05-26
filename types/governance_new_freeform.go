package types

import (
	"fmt"

	vegapb "code.vegaprotocol.io/protos/vega"
)

type ProposalTermsNewFreeform struct {
	NewFreeform *NewFreeform
}

func (f ProposalTermsNewFreeform) String() string {
	return fmt.Sprintf(
		"newFreeForm(%s)",
		reflectPointerToString(f.NewFreeform),
	)
}

func (f ProposalTermsNewFreeform) IntoProto() *vegapb.ProposalTerms_NewFreeform {
	var newFreeform *vegapb.NewFreeform
	if f.NewFreeform != nil {
		newFreeform = f.NewFreeform.IntoProto()
	}
	return &vegapb.ProposalTerms_NewFreeform{
		NewFreeform: newFreeform,
	}
}

func (f ProposalTermsNewFreeform) isPTerm() {}

func (f ProposalTermsNewFreeform) oneOfProto() interface{} {
	return f.IntoProto()
}

func (f ProposalTermsNewFreeform) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewFreeform
}

func (f ProposalTermsNewFreeform) DeepClone() proposalTerm {
	if f.NewFreeform == nil {
		return &ProposalTermsNewFreeform{}
	}
	return &ProposalTermsNewFreeform{
		NewFreeform: f.NewFreeform.DeepClone(),
	}
}

func NewNewFreeformFromProto(_ *vegapb.ProposalTerms_NewFreeform) *ProposalTermsNewFreeform {
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
