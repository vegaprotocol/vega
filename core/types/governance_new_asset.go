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

type ProposalTermsNewAsset struct {
	NewAsset *NewAsset
}

func (a ProposalTermsNewAsset) String() string {
	return fmt.Sprintf(
		"newAsset(%v)",
		stringer.PtrToString(a.NewAsset),
	)
}

func (a ProposalTermsNewAsset) IntoProto() *vegapb.NewAsset {
	var newAsset *vegapb.NewAsset
	if a.NewAsset != nil {
		newAsset = a.NewAsset.IntoProto()
	}
	return newAsset
}

func (a ProposalTermsNewAsset) isPTerm() {}

func (a ProposalTermsNewAsset) oneOfSingleProto() vegapb.ProposalOneOffTermChangeType {
	return &vegapb.ProposalTerms_NewAsset{
		NewAsset: a.IntoProto(),
	}
}

func (a ProposalTermsNewAsset) oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType {
	return nil
}

func (a ProposalTermsNewAsset) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsNewAsset) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewAsset
}

func (a ProposalTermsNewAsset) DeepClone() ProposalTerm {
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
		stringer.PtrToString(n.Changes),
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
