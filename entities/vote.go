package entities

import (
	"time"

	"code.vegaprotocol.io/protos/vega"
	"github.com/shopspring/decimal"
)

type Vote struct {
	PartyID                     PartyID
	ProposalID                  ProposalID
	Value                       VoteValue
	TotalGovernanceTokenBalance decimal.Decimal
	TotalGovernanceTokenWeight  decimal.Decimal
	TotalEquityLikeShareWeight  decimal.Decimal
	InitialTime                 time.Time // First vote for this party/proposal
	VegaTime                    time.Time // Time of last vote update
}

func (v *Vote) ToProto() *vega.Vote {
	return &vega.Vote{
		PartyId:                     v.PartyID.String(),
		ProposalId:                  v.ProposalID.String(),
		Value:                       vega.Vote_Value(v.Value),
		TotalGovernanceTokenBalance: v.TotalGovernanceTokenBalance.String(),
		TotalGovernanceTokenWeight:  v.TotalGovernanceTokenWeight.String(),
		TotalEquityLikeShareWeight:  v.TotalEquityLikeShareWeight.String(),
		Timestamp:                   v.InitialTime.UnixNano(),
	}
}

func VoteFromProto(pv *vega.Vote) (Vote, error) {
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
		PartyID:                     NewPartyID(pv.PartyId),
		ProposalID:                  NewProposalID(pv.ProposalId),
		Value:                       VoteValue(pv.Value),
		TotalGovernanceTokenBalance: totalGovernanceTokenBalance,
		TotalGovernanceTokenWeight:  totalGovernanceTokenWeight,
		TotalEquityLikeShareWeight:  totalEquityLikeShareWeight,
		InitialTime:                 NanosToPostgresTimestamp(pv.Timestamp),
	}

	return v, nil
}
