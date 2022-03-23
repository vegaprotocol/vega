package entities

import (
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type Vote struct {
	PartyID                     []byte
	ProposalID                  []byte
	Value                       VoteValue
	TotalGovernanceTokenBalance decimal.Decimal
	TotalGovernanceTokenWeight  decimal.Decimal
	TotalEquityLikeShareWeight  decimal.Decimal
	VegaTime                    time.Time
}

func (v *Vote) PartyHexID() string {
	return Party{ID: v.PartyID}.HexID()
}

func (v *Vote) ProposalHexID() string {
	return Proposal{ID: v.ProposalID}.HexID()
}

func (v *Vote) ToProto() *vega.Vote {
	return &vega.Vote{
		PartyId:                     v.PartyHexID(),
		ProposalId:                  v.ProposalHexID(),
		Value:                       vega.Vote_Value(v.Value),
		TotalGovernanceTokenBalance: v.TotalGovernanceTokenBalance.String(),
		TotalGovernanceTokenWeight:  v.TotalGovernanceTokenWeight.String(),
		TotalEquityLikeShareWeight:  v.TotalEquityLikeShareWeight.String(),
		Timestamp:                   v.VegaTime.UnixNano(),
	}
}

func VoteFromProto(pv *vega.Vote) (Vote, error) {
	partyID, err := MakePartyID(pv.PartyId)
	if err != nil {
		return Vote{}, err
	}

	proposalID, err := MakeProposalID(pv.ProposalId)
	if err != nil {
		return Vote{}, err
	}

	totalGovernanceTokenBalance, err := decimal.NewFromString(pv.TotalGovernanceTokenBalance)
	if err != nil {
		return Vote{}, err
	}

	totalGovernanceTokenWeight, err := decimal.NewFromString(pv.TotalGovernanceTokenWeight)
	if err != nil {
		return Vote{}, err
	}

	totalEquityLikeShareWeight, err := decimal.NewFromString(pv.TotalEquityLikeShareWeight)
	if err != nil {
		return Vote{}, err
	}

	v := Vote{
		PartyID:                     partyID,
		ProposalID:                  proposalID,
		Value:                       VoteValue(pv.Value),
		TotalGovernanceTokenBalance: totalGovernanceTokenBalance,
		TotalGovernanceTokenWeight:  totalGovernanceTokenWeight,
		TotalEquityLikeShareWeight:  totalEquityLikeShareWeight,
		VegaTime:                    time.Unix(0, pv.Timestamp),
	}

	return v, nil
}
