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

	"code.vegaprotocol.io/vega/libs/stringer"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type ProposalTerm interface {
	isPTerm()
	oneOfSingleProto() vegapb.ProposalOneOffTermChangeType
	oneOfBatchProto() vegapb.ProposalOneOffTermBatchChangeType

	DeepClone() ProposalTerm
	GetTermType() ProposalTermsType
	String() string
}

type ProposalTerms struct {
	ClosingTimestamp    int64
	EnactmentTimestamp  int64
	ValidationTimestamp int64
	Change              ProposalTerm
}

func (p *ProposalTerms) IsMarketStateUpdate() bool {
	switch p.Change.(type) {
	case *ProposalTermsUpdateMarketState:
		return true
	default:
		return false
	}
}

func (p *ProposalTerms) IsMarketUpdate() bool {
	switch p.Change.(type) {
	case *ProposalTermsUpdateMarket:
		return true
	default:
		return false
	}
}

func (p *ProposalTerms) IsSpotMarketUpdate() bool {
	switch p.Change.(type) {
	case *ProposalTermsUpdateSpotMarket:
		return true
	default:
		return false
	}
}

func (p *ProposalTerms) IsReferralProgramUpdate() bool {
	switch p.Change.(type) {
	case *ProposalTermsUpdateReferralProgram:
		return true
	default:
		return false
	}
}

func (p *ProposalTerms) IsVolumeDiscountProgramUpdate() bool {
	switch p.Change.(type) {
	case *ProposalTermsUpdateVolumeDiscountProgram:
		return true
	default:
		return false
	}
}

func (p *ProposalTerms) IsVolumeRebateProgramUpdate() bool {
	switch p.Change.(type) {
	case *ProposalTermsUpdateVolumeRebateProgram:
		return true
	default:
		return false
	}
}

func (p *ProposalTerms) IsNewProtocolAutomatedPurchase() bool {
	switch p.Change.(type) {
	case *ProposalTermsNewProtocolAutomatedPurchase:
		return true
	default:
		return false
	}
}

func (p *ProposalTerms) MarketUpdate() *UpdateMarket {
	switch terms := p.Change.(type) {
	case *ProposalTermsUpdateMarket:
		return terms.UpdateMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) UpdateMarketState() *UpdateMarketState {
	switch terms := p.Change.(type) {
	case *ProposalTermsUpdateMarketState:
		return terms.UpdateMarketState
	default:
		return nil
	}
}

func (p *ProposalTerms) SpotMarketUpdate() *UpdateSpotMarket {
	switch terms := p.Change.(type) {
	case *ProposalTermsUpdateSpotMarket:
		return terms.UpdateSpotMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) IsNewMarket() bool {
	return p.Change.GetTermType() == ProposalTermsTypeNewMarket
}

func (p *ProposalTerms) NewMarket() *NewMarket {
	switch terms := p.Change.(type) {
	case *ProposalTermsNewMarket:
		return terms.NewMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) IsSuccessorMarket() bool {
	if p.Change == nil {
		return false
	}
	if nm := p.NewMarket(); nm != nil {
		return nm.Changes.Successor != nil
	}
	return false
}

func (p ProposalTerms) IntoProto() *vegapb.ProposalTerms {
	change := p.Change.oneOfSingleProto()
	terms := &vegapb.ProposalTerms{
		ClosingTimestamp:    p.ClosingTimestamp,
		EnactmentTimestamp:  p.EnactmentTimestamp,
		ValidationTimestamp: p.ValidationTimestamp,
	}

	switch ch := change.(type) {
	case *vegapb.ProposalTerms_NewMarket:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateMarket:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateNetworkParameter:
		terms.Change = ch
	case *vegapb.ProposalTerms_NewAsset:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateAsset:
		terms.Change = ch
	case *vegapb.ProposalTerms_NewFreeform:
		terms.Change = ch
	case *vegapb.ProposalTerms_NewTransfer:
		terms.Change = ch
	case *vegapb.ProposalTerms_CancelTransfer:
		terms.Change = ch
	case *vegapb.ProposalTerms_NewSpotMarket:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateSpotMarket:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateMarketState:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateReferralProgram:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateVolumeDiscountProgram:
		terms.Change = ch
	case *vegapb.ProposalTerms_UpdateVolumeRebateProgram:
		terms.Change = ch
	case *vegapb.ProposalTerms_NewProtocolAutomatedPurchase:
		terms.Change = ch
	}

	return terms
}

func (p ProposalTerms) DeepClone() *ProposalTerms {
	cpy := p
	cpy.Change = p.Change.DeepClone()
	return &cpy
}

func (p ProposalTerms) String() string {
	return fmt.Sprintf(
		"single term: validationTs(%v) closingTs(%v) enactmentTs(%v) change(%s)",
		p.ValidationTimestamp,
		p.ClosingTimestamp,
		p.EnactmentTimestamp,
		stringer.ObjToString(p.Change),
	)
}

func (p *ProposalTerms) GetNewTransfer() *NewTransfer {
	switch c := p.Change.(type) {
	case *ProposalTermsNewTransfer:
		return c.NewTransfer
	default:
		return nil
	}
}

func (p *ProposalTerms) GetCancelTransfer() *CancelTransfer {
	switch c := p.Change.(type) {
	case *ProposalTermsCancelTransfer:
		return c.CancelTransfer
	default:
		return nil
	}
}

func (p *ProposalTerms) GetMarketStateUpdate() *UpdateMarketState {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateMarketState:
		return c.UpdateMarketState
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewAsset() *NewAsset {
	switch c := p.Change.(type) {
	case *ProposalTermsNewAsset:
		return c.NewAsset
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateAsset() *UpdateAsset {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateAsset:
		return c.UpdateAsset
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewMarket() *NewMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsNewMarket:
		return c.NewMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateMarket() *UpdateMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateMarket:
		return c.UpdateMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewSpotMarket() *NewSpotMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsNewSpotMarket:
		return c.NewSpotMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateSpotMarket() *UpdateSpotMarket {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateSpotMarket:
		return c.UpdateSpotMarket
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateVolumeDiscountProgram() *UpdateVolumeDiscountProgram {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateVolumeDiscountProgram:
		return c.UpdateVolumeDiscountProgram
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateVolumeRebateProgram() *UpdateVolumeRebateProgram {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateVolumeRebateProgram:
		return c.UpdateVolumeRebateProgram
	default:
		return nil
	}
}

func (p *ProposalTerms) GetAutomatedPurchase() *NewProtocolAutomatedPurchase {
	switch c := p.Change.(type) {
	case *ProposalTermsNewProtocolAutomatedPurchase:
		return c.NewProtocolAutomatedPurchase
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateReferralProgram() *UpdateReferralProgram {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateReferralProgram:
		return c.UpdateReferralProgram
	default:
		return nil
	}
}

func (p *ProposalTerms) GetUpdateNetworkParameter() *UpdateNetworkParameter {
	switch c := p.Change.(type) {
	case *ProposalTermsUpdateNetworkParameter:
		return c.UpdateNetworkParameter
	default:
		return nil
	}
}

func (p *ProposalTerms) GetNewFreeform() *NewFreeform {
	switch c := p.Change.(type) {
	case *ProposalTermsNewFreeform:
		return c.NewFreeform
	default:
		return nil
	}
}

func ProposalTermsFromProto(p *vegapb.ProposalTerms) (*ProposalTerms, error) {
	var (
		err    error
		change ProposalTerm
	)
	switch ch := p.Change.(type) {
	case *vegapb.ProposalTerms_NewMarket:
		change, err = NewNewMarketFromProto(ch.NewMarket)
	case *vegapb.ProposalTerms_UpdateMarket:
		change, err = UpdateMarketFromProto(ch.UpdateMarket)
	case *vegapb.ProposalTerms_UpdateNetworkParameter:
		change = NewUpdateNetworkParameterFromProto(ch.UpdateNetworkParameter)
	case *vegapb.ProposalTerms_NewAsset:
		change, err = NewNewAssetFromProto(ch.NewAsset)
	case *vegapb.ProposalTerms_UpdateAsset:
		change, err = NewUpdateAssetFromProto(ch.UpdateAsset)
	case *vegapb.ProposalTerms_NewFreeform:
		change = NewNewFreeformFromProto(ch.NewFreeform)
	case *vegapb.ProposalTerms_NewSpotMarket:
		change, err = NewNewSpotMarketFromProto(ch.NewSpotMarket)
	case *vegapb.ProposalTerms_UpdateSpotMarket:
		change, err = UpdateSpotMarketFromProto(ch.UpdateSpotMarket)
	case *vegapb.ProposalTerms_NewTransfer:
		change, err = NewNewTransferFromProto(ch.NewTransfer)
	case *vegapb.ProposalTerms_CancelTransfer:
		change, err = NewCancelGovernanceTransferFromProto(ch.CancelTransfer)
	case *vegapb.ProposalTerms_UpdateMarketState:
		change, err = NewTerminateMarketFromProto(ch.UpdateMarketState)
	case *vegapb.ProposalTerms_UpdateReferralProgram:
		change, err = NewUpdateReferralProgramProposalFromProto(ch.UpdateReferralProgram)
	case *vegapb.ProposalTerms_UpdateVolumeDiscountProgram:
		change, err = NewUpdateVolumeDiscountProgramProposalFromProto(ch.UpdateVolumeDiscountProgram)
	case *vegapb.ProposalTerms_UpdateVolumeRebateProgram:
		change, err = NewUpdateVolumeRebateProgramProposalFromProto(ch.UpdateVolumeRebateProgram)
	case *vegapb.ProposalTerms_NewProtocolAutomatedPurchase:
		change, err = NewProtocolAutomatedPurchaseConfigurationProposalFromProto(ch.NewProtocolAutomatedPurchase)
	}
	if err != nil {
		return nil, err
	}

	return &ProposalTerms{
		ClosingTimestamp:    p.ClosingTimestamp,
		EnactmentTimestamp:  p.EnactmentTimestamp,
		ValidationTimestamp: p.ValidationTimestamp,
		Change:              change,
	}, nil
}

type BatchProposalChange struct {
	ID             string
	Change         ProposalTerm
	EnactmentTime  int64
	ValidationTime int64
}

type BatchProposalTerms struct {
	ClosingTimestamp int64
	Changes          []BatchProposalChange
}

func (p BatchProposalTerms) String() string {
	return fmt.Sprintf(
		"batch term: closingTs(%v) changes(%v)",
		p.ClosingTimestamp,
		p.Changes,
	)
}

func (p BatchProposalTerms) IntoProto() *vegapb.BatchProposalTerms {
	terms := &vegapb.BatchProposalTerms{
		ClosingTimestamp: p.ClosingTimestamp,
		Changes:          p.changesToProto(),
	}

	return terms
}

func (p BatchProposalTerms) IntoSubmissionProto() *commandspb.BatchProposalSubmissionTerms {
	return &commandspb.BatchProposalSubmissionTerms{
		ClosingTimestamp: p.ClosingTimestamp,
		Changes:          p.changesToProto(),
	}
}

func (p BatchProposalTerms) changesToProto() []*vegapb.BatchProposalTermsChange {
	out := make([]*vegapb.BatchProposalTermsChange, 0, len(p.Changes))

	for _, c := range p.Changes {
		change := c.Change.oneOfBatchProto()

		termsChange := &vegapb.BatchProposalTermsChange{
			EnactmentTimestamp: c.EnactmentTime,
		}

		switch ch := change.(type) {
		case *vegapb.BatchProposalTermsChange_NewMarket:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateMarket:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateNetworkParameter:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateAsset:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_NewAsset:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_NewFreeform:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_NewTransfer:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_CancelTransfer:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_NewSpotMarket:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateSpotMarket:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateMarketState:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateReferralProgram:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateVolumeDiscountProgram:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_UpdateVolumeRebateProgram:
			termsChange.Change = ch
		case *vegapb.BatchProposalTermsChange_NewProtocolAutomatedPurchase:
			termsChange.Change = ch
		}

		out = append(out, termsChange)
	}

	return out
}

func (p BatchProposalTerms) DeepClone() *BatchProposalTerms {
	cpy := p

	changes := make([]BatchProposalChange, 0, len(p.Changes))
	for _, v := range cpy.Changes {
		changes = append(changes, BatchProposalChange{
			EnactmentTime: v.EnactmentTime,
			Change:        v.Change.DeepClone(),
		})
	}

	cpy.Changes = changes
	return &cpy
}

func BatchProposalTermsSubmissionFromProto(p *commandspb.BatchProposalSubmissionTerms, ids []string) (*BatchProposalTerms, error) {
	changesLen := len(p.Changes)

	if changesLen != len(ids) {
		return nil, errors.New("failed to convert BatchProposalTerms to proto due missing IDs")
	}

	changes := make([]BatchProposalChange, 0, changesLen)

	var (
		err    error
		change ProposalTerm
	)

	for i, term := range p.Changes {
		if term == nil {
			continue
		}

		switch ch := term.Change.(type) {
		case *vegapb.BatchProposalTermsChange_NewAsset:
			change, err = NewNewAssetFromProto(ch.NewAsset)
		case *vegapb.BatchProposalTermsChange_NewMarket:
			change, err = NewNewMarketFromProto(ch.NewMarket)
		case *vegapb.BatchProposalTermsChange_UpdateMarket:
			change, err = UpdateMarketFromProto(ch.UpdateMarket)
		case *vegapb.BatchProposalTermsChange_UpdateNetworkParameter:
			change = NewUpdateNetworkParameterFromProto(ch.UpdateNetworkParameter)
		case *vegapb.BatchProposalTermsChange_UpdateAsset:
			change, err = NewUpdateAssetFromProto(ch.UpdateAsset)
		case *vegapb.BatchProposalTermsChange_NewFreeform:
			change = NewNewFreeformFromProto(ch.NewFreeform)
		case *vegapb.BatchProposalTermsChange_NewSpotMarket:
			change, err = NewNewSpotMarketFromProto(ch.NewSpotMarket)
		case *vegapb.BatchProposalTermsChange_UpdateSpotMarket:
			change, err = UpdateSpotMarketFromProto(ch.UpdateSpotMarket)
		case *vegapb.BatchProposalTermsChange_NewTransfer:
			change, err = NewNewTransferFromProto(ch.NewTransfer)
		case *vegapb.BatchProposalTermsChange_CancelTransfer:
			change, err = NewCancelGovernanceTransferFromProto(ch.CancelTransfer)
		case *vegapb.BatchProposalTermsChange_UpdateMarketState:
			change, err = NewTerminateMarketFromProto(ch.UpdateMarketState)
		case *vegapb.BatchProposalTermsChange_UpdateReferralProgram:
			change, err = NewUpdateReferralProgramProposalFromProto(ch.UpdateReferralProgram)
		case *vegapb.BatchProposalTermsChange_UpdateVolumeDiscountProgram:
			change, err = NewUpdateVolumeDiscountProgramProposalFromProto(ch.UpdateVolumeDiscountProgram)
		case *vegapb.BatchProposalTermsChange_UpdateVolumeRebateProgram:
			change, err = NewUpdateVolumeRebateProgramProposalFromProto(ch.UpdateVolumeRebateProgram)
		case *vegapb.BatchProposalTermsChange_NewProtocolAutomatedPurchase:
			change, err = NewProtocolAutomatedPurchaseConfigurationProposalFromProto(ch.NewProtocolAutomatedPurchase)
		}
		if err != nil {
			return nil, err
		}

		changes = append(changes, BatchProposalChange{
			ID:             ids[i],
			EnactmentTime:  term.EnactmentTimestamp,
			ValidationTime: term.ValidationTimestamp,
			Change:         change,
		})
	}

	return &BatchProposalTerms{
		ClosingTimestamp: p.ClosingTimestamp,
		Changes:          changes,
	}, nil
}
