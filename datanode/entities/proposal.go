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

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"google.golang.org/protobuf/encoding/protojson"
)

type ProposalType v2.ListGovernanceDataRequest_Type

var (
	ProposalTypeAll                         = ProposalType(v2.ListGovernanceDataRequest_TYPE_ALL)
	ProposalTypeNewMarket                   = ProposalType(v2.ListGovernanceDataRequest_TYPE_NEW_MARKET)
	ProposalTypeNewAsset                    = ProposalType(v2.ListGovernanceDataRequest_TYPE_NEW_ASSET)
	ProposalTypeUpdateAsset                 = ProposalType(v2.ListGovernanceDataRequest_TYPE_UPDATE_ASSET)
	ProposalTypeUpdateMarket                = ProposalType(v2.ListGovernanceDataRequest_TYPE_UPDATE_MARKET)
	ProposalTypeUpdateNetworkParameter      = ProposalType(v2.ListGovernanceDataRequest_TYPE_NETWORK_PARAMETERS)
	ProposalTypeNewFreeform                 = ProposalType(v2.ListGovernanceDataRequest_TYPE_NEW_FREE_FORM)
	ProposalTypeNewSpotMarket               = ProposalType(v2.ListGovernanceDataRequest_TYPE_NEW_SPOT_MARKET)
	ProposalTypeUpdateSpotMarket            = ProposalType(v2.ListGovernanceDataRequest_TYPE_UPDATE_SPOT_MARKET)
	ProposalTypeNewTransfer                 = ProposalType(v2.ListGovernanceDataRequest_TYPE_NEW_TRANSFER)
	ProposalTypeCancelTransfer              = ProposalType(v2.ListGovernanceDataRequest_TYPE_CANCEL_TRANSFER)
	ProposalTypeUpdateMarketState           = ProposalType(v2.ListGovernanceDataRequest_TYPE_UPDATE_MARKET_STATE)
	ProposalTypeUpdateReferralProgram       = ProposalType(v2.ListGovernanceDataRequest_TYPE_UPDATE_REFERRAL_PROGRAM)
	ProposalTypeUpdateVolumeDiscountProgram = ProposalType(v2.ListGovernanceDataRequest_TYPE_UPDATE_VOLUME_DISCOUNT_PROGRAM)
	ProposalTypeAutomatedPurchase           = ProposalType(v2.ListGovernanceDataRequest_TYPE_NEW_AUTOMATED_PURCHASE)
)

func (p *ProposalType) String() string {
	if p == nil {
		return ""
	}
	switch *p {
	case ProposalTypeAll:
		return "all"
	case ProposalTypeNewMarket:
		return "newMarket"
	case ProposalTypeNewAsset:
		return "newAsset"
	case ProposalTypeUpdateAsset:
		return "updateAsset"
	case ProposalTypeUpdateMarket:
		return "updateMarket"
	case ProposalTypeUpdateNetworkParameter:
		return "updateNetworkParameter"
	case ProposalTypeNewFreeform:
		return "newFreeform"
	case ProposalTypeNewSpotMarket:
		return "newSpotMarket"
	case ProposalTypeUpdateSpotMarket:
		return "updateSpotMarket"
	case ProposalTypeNewTransfer:
		return "newTransfer"
	case ProposalTypeCancelTransfer:
		return "cancelTransfer"
	case ProposalTypeUpdateMarketState:
		return "updateMarketState"
	case ProposalTypeUpdateReferralProgram:
		return "updateReferralProgram"
	case ProposalTypeUpdateVolumeDiscountProgram:
		return "updateVolumeDiscountProgram"
	case ProposalTypeAutomatedPurchase:
		return "NewProtocolAutomatedPurchase"
	default:
		return "unknown"
	}
}

type _Proposal struct{}

type ProposalID = ID[_Proposal]

type Proposal struct {
	ID                      ProposalID
	BatchID                 ProposalID
	Reference               string
	PartyID                 PartyID
	State                   ProposalState
	Rationale               ProposalRationale
	Terms                   ProposalTerms
	BatchTerms              BatchProposalTerms
	Reason                  ProposalError
	ErrorDetails            string
	ProposalTime            time.Time
	VegaTime                time.Time
	RequiredMajority        num.Decimal
	RequiredParticipation   num.Decimal
	RequiredLPMajority      *num.Decimal
	RequiredLPParticipation *num.Decimal
	TxHash                  TxHash
	Proposals               []Proposal
}

func (p Proposal) IsBatch() bool {
	return p.BatchTerms.BatchProposalTerms != nil
}

func (p Proposal) BelongsToBatch() bool {
	return len(p.BatchID) > 0
}

func (p Proposal) ToProto() *vega.Proposal {
	var lpMajority *string
	if !p.RequiredLPMajority.IsZero() {
		lpMajority = toPtr(p.RequiredLPMajority.String())
	}
	var lpParticipation *string
	if !p.RequiredLPParticipation.IsZero() {
		lpParticipation = toPtr(p.RequiredLPParticipation.String())
	}

	var reason *vega.ProposalError
	if p.Reason != ProposalErrorUnspecified {
		reason = ptr.From(vega.ProposalError(p.Reason))
	}

	var errDetails *string
	if len(p.ErrorDetails) > 0 {
		errDetails = ptr.From(p.ErrorDetails)
	}

	var batchID *string
	if len(p.BatchID) > 0 {
		batchID = ptr.From(p.BatchID.String())
	}

	pp := vega.Proposal{
		Id:                                     p.ID.String(),
		BatchId:                                batchID,
		Reference:                              p.Reference,
		PartyId:                                p.PartyID.String(),
		State:                                  vega.Proposal_State(p.State),
		Rationale:                              p.Rationale.ProposalRationale,
		Timestamp:                              p.ProposalTime.UnixNano(),
		Terms:                                  p.Terms.ProposalTerms,
		BatchTerms:                             p.BatchTerms.BatchProposalTerms,
		Reason:                                 reason,
		ErrorDetails:                           errDetails,
		RequiredMajority:                       p.RequiredMajority.String(),
		RequiredParticipation:                  p.RequiredParticipation.String(),
		RequiredLiquidityProviderMajority:      lpMajority,
		RequiredLiquidityProviderParticipation: lpParticipation,
	}
	return &pp
}

func (p Proposal) Cursor() *Cursor {
	pc := ProposalCursor{
		State:    p.State,
		VegaTime: p.VegaTime,
		ID:       p.ID,
	}
	return NewCursor(pc.String())
}

func (p Proposal) ToProtoEdge(_ ...any) (*v2.GovernanceDataEdge, error) {
	proposalsProto := make([]*vega.Proposal, len(p.Proposals))

	for i, proposal := range p.Proposals {
		proposalsProto[i] = proposal.ToProto()
	}

	return &v2.GovernanceDataEdge{
		Node: &vega.GovernanceData{
			Proposal:  p.ToProto(),
			Proposals: proposalsProto,
		},
		Cursor: p.Cursor().Encode(),
	}, nil
}

func ProposalFromProto(pp *vega.Proposal, txHash TxHash) (Proposal, error) {
	var err error
	var majority num.Decimal
	if len(pp.RequiredMajority) <= 0 {
		majority = num.DecimalZero()
	} else if majority, err = num.DecimalFromString(pp.RequiredMajority); err != nil {
		return Proposal{}, err
	}

	var participation num.Decimal
	if len(pp.RequiredParticipation) <= 0 {
		participation = num.DecimalZero()
	} else if participation, err = num.DecimalFromString(pp.RequiredParticipation); err != nil {
		return Proposal{}, err
	}

	lpMajority := num.DecimalZero()
	if pp.RequiredLiquidityProviderMajority != nil && len(*pp.RequiredLiquidityProviderMajority) > 0 {
		if lpMajority, err = num.DecimalFromString(*pp.RequiredLiquidityProviderMajority); err != nil {
			return Proposal{}, err
		}
	}
	lpParticipation := num.DecimalZero()
	if pp.RequiredLiquidityProviderParticipation != nil && len(*pp.RequiredLiquidityProviderParticipation) > 0 {
		if lpParticipation, err = num.DecimalFromString(*pp.RequiredLiquidityProviderParticipation); err != nil {
			return Proposal{}, err
		}
	}

	reason := ProposalErrorUnspecified
	if pp.Reason != nil {
		reason = ProposalError(*pp.Reason)
	}

	var errDetails string
	if pp.ErrorDetails != nil {
		errDetails = *pp.ErrorDetails
	}

	var batchID ProposalID
	if pp.BatchId != nil {
		batchID = ProposalID(*pp.BatchId)
	}

	p := Proposal{
		ID:                      ProposalID(pp.Id),
		BatchID:                 batchID,
		Reference:               pp.Reference,
		PartyID:                 PartyID(pp.PartyId),
		State:                   ProposalState(pp.State),
		Rationale:               ProposalRationale{pp.Rationale},
		Terms:                   ProposalTerms{pp.GetTerms()},
		BatchTerms:              BatchProposalTerms{pp.GetBatchTerms()},
		Reason:                  reason,
		ErrorDetails:            errDetails,
		ProposalTime:            time.Unix(0, pp.Timestamp),
		RequiredMajority:        majority,
		RequiredParticipation:   participation,
		RequiredLPMajority:      &lpMajority,
		RequiredLPParticipation: &lpParticipation,
		TxHash:                  txHash,
	}
	return p, nil
}

type ProposalRationale struct {
	*vega.ProposalRationale
}

func (pt ProposalRationale) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(pt)
}

func (pt *ProposalRationale) UnmarshalJSON(b []byte) error {
	pt.ProposalRationale = &vega.ProposalRationale{}
	return protojson.Unmarshal(b, pt)
}

type ProposalTerms struct {
	*vega.ProposalTerms
}

func (pt ProposalTerms) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(pt)
}

func (pt *ProposalTerms) UnmarshalJSON(b []byte) error {
	pt.ProposalTerms = &vega.ProposalTerms{}
	if err := protojson.Unmarshal(b, pt); err != nil {
		return err
	}

	if pt.ProposalTerms.Change == nil {
		pt.ProposalTerms = nil
	}

	return nil
}

type BatchProposalTerms struct {
	*vega.BatchProposalTerms
}

func (pt BatchProposalTerms) MarshalJSON() ([]byte, error) {
	return protojson.Marshal(pt)
}

func (pt *BatchProposalTerms) UnmarshalJSON(b []byte) error {
	pt.BatchProposalTerms = &vega.BatchProposalTerms{}
	if err := protojson.Unmarshal(b, pt); err != nil {
		return err
	}

	if pt.BatchProposalTerms.Changes == nil || len(pt.BatchProposalTerms.Changes) == 0 {
		pt.BatchProposalTerms = nil
	}

	return nil
}

type ProposalCursor struct {
	State    ProposalState `json:"state"`
	VegaTime time.Time     `json:"vega_time"`
	ID       ProposalID    `json:"id"`
}

func (pc ProposalCursor) String() string {
	bs, err := json.Marshal(pc)
	if err != nil {
		panic(fmt.Errorf("failed to marshal proposal cursor: %w", err))
	}
	return string(bs)
}

func (pc *ProposalCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), pc)
}

func toPtr[T any](t T) *T {
	return &t
}
