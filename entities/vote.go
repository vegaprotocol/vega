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

func (p Vote) ToProtoEdge(_ ...any) (*v2.VoteEdge, error) {
	return &v2.VoteEdge{
		Node:   p.ToProto(),
		Cursor: p.Cursor().Encode(),
	}, nil
}

func (p Vote) Cursor() *Cursor {
	pc := VoteCursor{
		PartyID:  p.PartyID,
		VegaTime: p.VegaTime,
	}

	return NewCursor(pc.String())
}

type VoteCursor struct {
	PartyID  PartyID   `json:"party_id"`
	VegaTime time.Time `json:"vega_time"`
}

func (rc VoteCursor) String() string {
	bs, err := json.Marshal(rc)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("could not marshal order cursor: %w", err))
	}
	return string(bs)
}

func (rc *VoteCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), rc)
}
