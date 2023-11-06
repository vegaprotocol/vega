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

type ProposalTermsCancelTransfer struct {
	CancelTransfer *CancelTransfer
}

func (a ProposalTermsCancelTransfer) String() string {
	return fmt.Sprintf(
		"cancelTransfer(%s)",
		stringer.PtrToString(a.CancelTransfer),
	)
}

func (a ProposalTermsCancelTransfer) IntoProto() *vegapb.ProposalTerms_CancelTransfer {
	return &vegapb.ProposalTerms_CancelTransfer{
		CancelTransfer: a.CancelTransfer.IntoProto(),
	}
}

func (a ProposalTermsCancelTransfer) isPTerm() {}

func (a ProposalTermsCancelTransfer) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsCancelTransfer) GetTermType() ProposalTermsType {
	return ProposalTermsTypeCancelTransfer
}

func (a ProposalTermsCancelTransfer) DeepClone() proposalTerm {
	if a.CancelTransfer == nil {
		return &ProposalTermsCancelTransfer{}
	}
	return &ProposalTermsCancelTransfer{
		CancelTransfer: a.CancelTransfer.DeepClone(),
	}
}

func NewCancelGovernanceTransferFromProto(p *vegapb.ProposalTerms_CancelTransfer) (*ProposalTermsCancelTransfer, error) {
	var cancelTransfer *CancelTransfer
	if p.CancelTransfer != nil {
		cancelTransfer = &CancelTransfer{}

		if p.CancelTransfer.Changes != nil {
			cancelTransfer.Changes = &CancelTransferConfiguration{
				TransferID: p.CancelTransfer.Changes.TransferId,
			}
		}
	}

	return &ProposalTermsCancelTransfer{
		CancelTransfer: cancelTransfer,
	}, nil
}

type CancelTransfer struct {
	Changes *CancelTransferConfiguration
}

func (c CancelTransfer) IntoProto() *vegapb.CancelTransfer {
	return &vegapb.CancelTransfer{
		Changes: &vegapb.CancelTransferConfiguration{
			TransferId: c.Changes.TransferID,
		},
	}
}

func (c CancelTransfer) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.PtrToString(c.Changes),
	)
}

func (c CancelTransfer) DeepClone() *CancelTransfer {
	return &CancelTransfer{
		Changes: &CancelTransferConfiguration{
			TransferID: c.Changes.TransferID,
		},
	}
}

type CancelTransferConfiguration struct {
	TransferID string
}

func (c CancelTransferConfiguration) String() string {
	return fmt.Sprintf("transferID(%s)", c.TransferID)
}
