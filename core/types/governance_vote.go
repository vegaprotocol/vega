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
	"sort"

	"code.vegaprotocol.io/vega/libs/num"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type VoteValue = vegapb.Vote_Value

const (
	// VoteValueUnspecified Default value, always invalid.
	VoteValueUnspecified VoteValue = vegapb.Vote_VALUE_UNSPECIFIED
	// VoteValueNo represents a vote against the proposal.
	VoteValueNo VoteValue = vegapb.Vote_VALUE_NO
	// VoteValueYes represents a vote in favour of the proposal.
	VoteValueYes VoteValue = vegapb.Vote_VALUE_YES
)

type VoteSubmission struct {
	// The ID of the proposal to vote for.
	ProposalID string
	// The actual value of the vote
	Value VoteValue
}

func NewVoteSubmissionFromProto(p *commandspb.VoteSubmission) *VoteSubmission {
	return &VoteSubmission{
		ProposalID: p.ProposalId,
		Value:      p.Value,
	}
}

func (v VoteSubmission) IntoProto() *commandspb.VoteSubmission {
	return &commandspb.VoteSubmission{
		ProposalId: v.ProposalID,
		Value:      v.Value,
	}
}

func (v VoteSubmission) String() string {
	return fmt.Sprintf(
		"proposalID(%s) value(%s)",
		v.ProposalID,
		v.Value.String(),
	)
}

// Vote represents a governance vote casted by a party for a given proposal.
type Vote struct {
	// PartyID is the party that casted the vote.
	PartyID string
	// ProposalID is the proposal identifier concerned by the vote.
	ProposalID string
	// Value is the actual position of the vote: yes or no.
	Value VoteValue
	// Timestamp is the date and time (in nanoseconds) at which the vote has
	// been casted.
	Timestamp int64
	// TotalGovernanceTokenBalance is the total number of tokens hold by the
	// party that casted the vote.
	TotalGovernanceTokenBalance *num.Uint
	// TotalGovernanceTokenWeight is the weight of the vote compared to the
	// total number of governance token.
	TotalGovernanceTokenWeight num.Decimal
	// TotalEquityLikeShareWeight is the weight of the vote compared to the
	// total number of equity-like share on the market.
	TotalEquityLikeShareWeight     num.Decimal
	PerMarketEquityLikeShareWeight map[string]num.Decimal
}

func (v Vote) IntoProto() *vegapb.Vote {
	var ELSMap []*vegapb.VoteELSPair
	if ln := len(v.PerMarketEquityLikeShareWeight); ln > 0 {
		ELSMap = make([]*vegapb.VoteELSPair, 0, ln)
		for mkt, els := range v.PerMarketEquityLikeShareWeight {
			ELSMap = append(ELSMap, &vegapb.VoteELSPair{
				MarketId: mkt,
				Els:      els.String(),
			})
		}
		sort.SliceStable(ELSMap, func(i, j int) bool {
			return ELSMap[i].MarketId > ELSMap[j].MarketId
		})
	}
	return &vegapb.Vote{
		PartyId:                     v.PartyID,
		Value:                       v.Value,
		ProposalId:                  v.ProposalID,
		Timestamp:                   v.Timestamp,
		TotalGovernanceTokenBalance: num.UintToString(v.TotalGovernanceTokenBalance),
		TotalGovernanceTokenWeight:  v.TotalGovernanceTokenWeight.String(),
		TotalEquityLikeShareWeight:  v.TotalEquityLikeShareWeight.String(),
		ElsPerMarket:                ELSMap,
	}
}

func VoteFromProto(v *vegapb.Vote) (*Vote, error) {
	ret := Vote{
		PartyID:    v.PartyId,
		Value:      v.Value,
		ProposalID: v.ProposalId,
		Timestamp:  v.Timestamp,
	}
	if len(v.TotalGovernanceTokenBalance) > 0 {
		ret.TotalGovernanceTokenBalance, _ = num.UintFromString(v.TotalGovernanceTokenBalance, 10)
	}
	if len(v.TotalGovernanceTokenWeight) > 0 {
		w, err := num.DecimalFromString(v.TotalGovernanceTokenWeight)
		if err != nil {
			return nil, err
		}
		ret.TotalGovernanceTokenWeight = w
	}
	if len(v.TotalEquityLikeShareWeight) > 0 {
		ret.TotalEquityLikeShareWeight, _ = num.DecimalFromString(v.TotalEquityLikeShareWeight)
	}
	if len(v.ElsPerMarket) > 0 {
		els := make(map[string]num.Decimal, len(v.ElsPerMarket))
		for _, pair := range v.ElsPerMarket {
			share, err := num.DecimalFromString(pair.Els)
			if err != nil {
				return nil, err
			}
			els[pair.MarketId] = share
		}
		ret.PerMarketEquityLikeShareWeight = els
	}
	return &ret, nil
}
