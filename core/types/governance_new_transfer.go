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
	"errors"
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/stringer"
	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
)

type ProposalTermsNewTransfer struct {
	NewTransfer *NewTransfer
}

func (a ProposalTermsNewTransfer) String() string {
	return fmt.Sprintf(
		"newTransfer(%s)",
		stringer.PtrToString(a.NewTransfer),
	)
}

func (a ProposalTermsNewTransfer) IntoProto() *vegapb.ProposalTerms_NewTransfer {
	return &vegapb.ProposalTerms_NewTransfer{
		NewTransfer: a.NewTransfer.IntoProto(),
	}
}

func (a ProposalTermsNewTransfer) isPTerm() {}

func (a ProposalTermsNewTransfer) oneOfProto() interface{} {
	return a.IntoProto()
}

func (a ProposalTermsNewTransfer) GetTermType() ProposalTermsType {
	return ProposalTermsTypeNewTransfer
}

func (a ProposalTermsNewTransfer) DeepClone() proposalTerm {
	if a.NewTransfer == nil {
		return &ProposalTermsNewTransfer{}
	}
	return &ProposalTermsNewTransfer{
		NewTransfer: a.NewTransfer.DeepClone(),
	}
}

func NewNewTransferFromProto(p *vegapb.ProposalTerms_NewTransfer) (*ProposalTermsNewTransfer, error) {
	var newTransfer *NewTransfer
	if p.NewTransfer != nil {
		newTransfer = &NewTransfer{}

		if p.NewTransfer.Changes != nil {
			var err error
			newTransfer.Changes, err = NewTransferConfigurationFromProto(p.NewTransfer.Changes)
			if err != nil {
				return nil, err
			}
		}
	}

	return &ProposalTermsNewTransfer{
		NewTransfer: newTransfer,
	}, nil
}

type NewTransfer struct {
	Changes *NewTransferConfiguration
}

func (n NewTransfer) IntoProto() *vegapb.NewTransfer {
	var changes *vegapb.NewTransferConfiguration
	if n.Changes != nil {
		changes = n.Changes.IntoProto()
	}
	return &vegapb.NewTransfer{
		Changes: changes,
	}
}

func (n NewTransfer) DeepClone() *NewTransfer {
	cpy := NewTransfer{}
	if n.Changes != nil {
		cpy.Changes = n.Changes.DeepClone()
	}
	return &cpy
}

func (n NewTransfer) String() string {
	return fmt.Sprintf(
		"changes(%s)",
		stringer.PtrToString(n.Changes),
	)
}

type TransferKind int

const (
	TransferKindOneOff TransferKind = iota
	TransferKindRecurring
)

type NewTransferConfiguration struct {
	SourceType              AccountType
	DestinationType         AccountType
	Asset                   string
	Source                  string
	Destination             string
	TransferType            vega.GovernanceTransferType
	MaxAmount               *num.Uint
	FractionOfBalance       num.Decimal
	Kind                    TransferKind
	OneOffTransferConfig    *vega.OneOffTransfer
	RecurringTransferConfig *vega.RecurringTransfer
}

func (n NewTransferConfiguration) IntoProto() *vegapb.NewTransferConfiguration {
	c := &vegapb.NewTransferConfiguration{
		SourceType:        n.SourceType,
		Source:            n.Source,
		TransferType:      n.TransferType,
		Asset:             n.Asset,
		Amount:            n.MaxAmount.String(),
		FractionOfBalance: n.FractionOfBalance.String(),
		DestinationType:   n.DestinationType,
		Destination:       n.Destination,
	}
	if n.Kind == TransferKind(TransferCommandKindOneOff) {
		c.Kind = &vegapb.NewTransferConfiguration_OneOff{
			OneOff: n.OneOffTransferConfig,
		}
	} else {
		c.Kind = &vegapb.NewTransferConfiguration_Recurring{
			Recurring: n.RecurringTransferConfig,
		}
	}
	return c
}

func (n NewTransferConfiguration) DeepClone() *NewTransferConfiguration {
	return &NewTransferConfiguration{
		SourceType:              n.SourceType,
		Source:                  n.Source,
		TransferType:            n.TransferType,
		Asset:                   n.Asset,
		MaxAmount:               n.MaxAmount.Clone(),
		FractionOfBalance:       n.FractionOfBalance,
		DestinationType:         n.DestinationType,
		Destination:             n.Destination,
		Kind:                    n.Kind,
		OneOffTransferConfig:    n.OneOffTransferConfig,
		RecurringTransferConfig: n.RecurringTransferConfig,
	}
}

func (n NewTransferConfiguration) String() string {
	return fmt.Sprintf(
		"sourceType(%v) source(%s) asset(%s) maxAmount(%s) fractionalBalance(%s) destinationType(%v) destination(%s), kind(%d)",
		n.SourceType,
		n.Source,
		n.Asset,
		n.MaxAmount.String(),
		n.FractionOfBalance.String(),
		n.DestinationType,
		n.Destination,
		n.Kind,
	)
}

func NewTransferConfigurationFromProto(p *vegapb.NewTransferConfiguration) (*NewTransferConfiguration, error) {
	maxAmount, overflow := num.UintFromString(p.Amount, 10)
	if overflow {
		return nil, errors.New("invalid max amount for transfer")
	}
	fraction, err := num.DecimalFromString(p.FractionOfBalance)
	if err != nil {
		return nil, err
	}

	oneOff := p.GetOneOff()
	recurring := p.GetRecurring()
	kind := TransferKindOneOff
	if recurring != nil {
		kind = TransferKindRecurring
	}

	return &NewTransferConfiguration{
		SourceType:              p.SourceType,
		Source:                  p.Source,
		Asset:                   p.Asset,
		MaxAmount:               maxAmount,
		FractionOfBalance:       fraction,
		DestinationType:         p.DestinationType,
		TransferType:            p.TransferType,
		Destination:             p.Destination,
		OneOffTransferConfig:    oneOff,
		RecurringTransferConfig: recurring,
		Kind:                    kind,
	}, nil
}
