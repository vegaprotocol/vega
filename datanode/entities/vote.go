// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/protos/vega"
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
	TxHash                      TxHash
	VegaTime                    time.Time // Time of last vote update
}

func (v Vote) ToProto() *vega.Vote {
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

func VoteFromProto(pv *vega.Vote, txHash TxHash) (Vote, error) {
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
		PartyID:                     PartyID(pv.PartyId),
		ProposalID:                  ProposalID(pv.ProposalId),
		Value:                       VoteValue(pv.Value),
		TotalGovernanceTokenBalance: totalGovernanceTokenBalance,
		TotalGovernanceTokenWeight:  totalGovernanceTokenWeight,
		TotalEquityLikeShareWeight:  totalEquityLikeShareWeight,
		InitialTime:                 NanosToPostgresTimestamp(pv.Timestamp),
		TxHash:                      txHash,
	}

	return v, nil
}

func (v Vote) ToProtoEdge(_ ...any) (*v2.VoteEdge, error) {
	return &v2.VoteEdge{
		Node:   v.ToProto(),
		Cursor: v.Cursor().Encode(),
	}, nil
}

func (v Vote) Cursor() *Cursor {
	pc := VoteCursor{
		PartyID:  v.PartyID,
		VegaTime: v.VegaTime,
	}

	return NewCursor(pc.String())
}

type VoteCursor struct {
	PartyID  PartyID   `json:"party_id"`
	VegaTime time.Time `json:"vega_time"`
}

func (vc VoteCursor) String() string {
	bs, err := json.Marshal(vc)
	if err != nil {
		// This should never happen.
		panic(fmt.Errorf("could not marshal order cursor: %w", err))
	}
	return string(bs)
}

func (vc *VoteCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), vc)
}
