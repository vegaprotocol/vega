// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package types

import (
	"fmt"

	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsNewAsset struct {
	NewAsset *NewAsset
}

func (a ProposalTermsNewAsset) String() string {
	return fmt.Sprintf(
		"newAsset(%v)",
		stringer.ReflectPointerToString(a.NewAsset),
	)
}

func (a ProposalTermsNewAsset) IntoProto() *vegapb.ProposalTerms_NewAsset {
	var newAsset *vegapb.NewAsset
	if a.NewAsset != nil {
		newAsset = a.NewAsset.IntoProto()
	}
	return &vegapb.ProposalTerms_NewAsset{
		NewAsset: newAsset,
	}
}

func (a ProposalTermsNewAsset) isPTerm() {}

func (a ProposalTermsNewAsset) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsNewAsset) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewAsset
}

func (a ProposalTermsNewAsset) DeepClone() proposalTerm {
	if a.NewAsset == nil {
		return &ProposalTermsNewAsset{}
	}
	return &ProposalTermsNewAsset{
		NewAsset: a.NewAsset.DeepClone(),
	}
}

func NewNewAssetFromProto(p *vegapb.ProposalTerms_NewAsset) (*ProposalTermsNewAsset, error) {
	var newAsset *NewAsset
	if p.NewAsset != nil {
		newAsset = &NewAsset{}

		if p.NewAsset.Changes != nil {
			var err error
			newAsset.Changes, err = AssetDetailsFromProto(p.NewAsset.Changes)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ProposalTermsNewAsset{
		NewAsset: newAsset,
	}, nil
}

type NewAsset struct {
	Changes *AssetDetails
}

func (n *NewAsset) Validate() (ProposalError, error) {
	if n.Changes == nil {
		return ProposalErrorInvalidAssetDetails, ErrChangesAreRequired
	}
	if perr, err := n.Changes.Validate(); err != nil {
		return perr, err
	}
	if n.Changes.Source == nil {
		return ProposalErrorInvalidAsset, ErrSourceIsRequired
	}
	return n.Changes.Source.Validate()
}

func (n *NewAsset) GetChanges() *AssetDetails {
	if n != nil {
		return n.Changes
	}
	return nil
}

func (n NewAsset) IntoProto() *vegapb.NewAsset {
	var changes *vegapb.AssetDetails
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &vegapb.NewAsset{
		Changes: changes,
	}
}

func (n NewAsset) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.ReflectPointerToString(n.Changes),
	)
}

func (n NewAsset) DeepClone() *NewAsset {
	if n.Changes == nil {
		return &NewAsset{}
	}
	return &NewAsset{
		Changes: n.Changes.DeepClone(),
	}
}
