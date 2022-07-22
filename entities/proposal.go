// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"
	"code.vegaprotocol.io/protos/vega"
	"google.golang.org/protobuf/encoding/protojson"
)

type ProposalType string

var (
	ProposalTypeNewMarket              = ProposalType("newMarket")
	ProposalTypeNewAsset               = ProposalType("newAsset")
	ProposalTypeUpdateMarket           = ProposalType("updateMarket")
	ProposalTypeUpdateNetworkParameter = ProposalType("updateNetworkParameter")
	ProposalTypeNewFreeform            = ProposalType("newFreeform")
)

type ProposalID struct{ ID }

func NewProposalID(id string) ProposalID {
	return ProposalID{ID: ID(id)}
}

type Proposal struct {
	ID           ProposalID
	Reference    string
	PartyID      PartyID
	State        ProposalState
	Rationale    ProposalRationale
	Terms        ProposalTerms
	Reason       ProposalError
	ErrorDetails string
	ProposalTime time.Time
	VegaTime     time.Time
}

func (p *Proposal) ToProto() *vega.Proposal {
	pp := vega.Proposal{
		Id:           p.ID.String(),
		Reference:    p.Reference,
		PartyId:      p.PartyID.String(),
		State:        vega.Proposal_State(p.State),
		Rationale:    p.Rationale.ProposalRationale,
		Timestamp:    p.ProposalTime.UnixNano(),
		Terms:        p.Terms.ProposalTerms,
		Reason:       vega.ProposalError(p.Reason),
		ErrorDetails: p.ErrorDetails,
	}
	return &pp
}

func (p Proposal) Cursor() *Cursor {
	pc := ProposalCursor{
		State:    p.State,
		VegaTime: p.VegaTime,
		ID:       p.ID.String(),
	}
	return NewCursor(pc.String())
}

func (p Proposal) ToProtoEdge(_ ...any) (*v2.GovernanceDataEdge, error) {
	return &v2.GovernanceDataEdge{
		Node: &vega.GovernanceData{
			Proposal: p.ToProto(),
		},
		Cursor: p.Cursor().Encode(),
	}, nil
}

func ProposalFromProto(pp *vega.Proposal) (Proposal, error) {
	p := Proposal{
		ID:           NewProposalID(pp.Id),
		Reference:    pp.Reference,
		PartyID:      NewPartyID(pp.PartyId),
		State:        ProposalState(pp.State),
		Rationale:    ProposalRationale{pp.Rationale},
		Terms:        ProposalTerms{pp.Terms},
		Reason:       ProposalError(pp.Reason),
		ErrorDetails: pp.ErrorDetails,
		ProposalTime: time.Unix(0, pp.Timestamp),
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
	return protojson.Unmarshal(b, pt)
}

type ProposalCursor struct {
	State    ProposalState `json:"state"`
	VegaTime time.Time     `json:"vega_time"`
	ID       string        `json:"id"`
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
