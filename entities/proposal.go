package entities

import (
	"encoding/hex"
	"fmt"
	"time"

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

type Proposal struct {
	ID           []byte
	Reference    string
	PartyID      []byte
	State        ProposalState
	Terms        ProposalTerms
	Reason       ProposalError
	ErrorDetails string
	ProposalTime time.Time
	VegaTime     time.Time
}

func (p *Proposal) PartyHexID() string {
	return Party{ID: p.PartyID}.HexID()
}

func (p Proposal) HexID() string {
	return hex.EncodeToString(p.ID)
}

func MakeProposalID(stringID string) ([]byte, error) {
	id, err := hex.DecodeString(stringID)
	if err != nil {
		return nil, fmt.Errorf("proposal id is not valid hex string: %v", stringID)
	}
	return id, nil
}

func (p *Proposal) ToProto() *vega.Proposal {
	pp := vega.Proposal{
		Id:           p.HexID(),
		Reference:    p.Reference,
		PartyId:      p.PartyHexID(),
		State:        vega.Proposal_State(p.State),
		Timestamp:    p.ProposalTime.UnixNano(),
		Terms:        p.Terms.ProposalTerms,
		Reason:       vega.ProposalError(p.Reason),
		ErrorDetails: p.ErrorDetails,
	}
	return &pp
}

func ProposalFromProto(pp *vega.Proposal) (Proposal, error) {
	id, err := MakeProposalID(pp.Id)
	if err != nil {
		return Proposal{}, err
	}

	partyID, err := MakePartyID(pp.PartyId)
	if err != nil {
		return Proposal{}, err
	}

	p := Proposal{
		ID:           id,
		Reference:    pp.Reference,
		PartyID:      partyID,
		State:        ProposalState(pp.State),
		Terms:        ProposalTerms{pp.Terms},
		Reason:       ProposalError(pp.Reason),
		ErrorDetails: pp.ErrorDetails,
		ProposalTime: time.Unix(0, pp.Timestamp),
	}
	return p, nil
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
