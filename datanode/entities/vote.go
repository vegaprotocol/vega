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

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/shopspring/decimal"
)

type PerMarketELSWeight struct {
	Market string `json:"market"`
	ELS    string `json:"els"`
}

type Vote struct {
	PartyID                        PartyID
	ProposalID                     ProposalID
	Value                          VoteValue
	TotalGovernanceTokenBalance    decimal.Decimal
	TotalGovernanceTokenWeight     decimal.Decimal
	TotalEquityLikeShareWeight     decimal.Decimal
	PerMarketEquityLikeShareWeight []PerMarketELSWeight
	InitialTime                    time.Time // First vote for this party/proposal
	TxHash                         TxHash
	VegaTime                       time.Time // Time of last vote update
}

func (v Vote) ToProto() *vega.Vote {
	var perMarketELSWeight []*vega.VoteELSPair

	if ln := len(v.PerMarketEquityLikeShareWeight); ln > 0 {
		perMarketELSWeight = make([]*vega.VoteELSPair, 0, ln)
		for _, w := range v.PerMarketEquityLikeShareWeight {
			perMarketELSWeight = append(perMarketELSWeight, &vega.VoteELSPair{
				MarketId: w.Market,
				Els:      w.ELS,
			})
		}
	}

	return &vega.Vote{
		PartyId:                     v.PartyID.String(),
		ProposalId:                  v.ProposalID.String(),
		Value:                       vega.Vote_Value(v.Value),
		TotalGovernanceTokenBalance: v.TotalGovernanceTokenBalance.String(),
		TotalGovernanceTokenWeight:  v.TotalGovernanceTokenWeight.String(),
		TotalEquityLikeShareWeight:  v.TotalEquityLikeShareWeight.String(),
		Timestamp:                   v.InitialTime.UnixNano(),
		ELSPerMarket:                perMarketELSWeight,
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

	// We need deterministic ordering of the share weights to prevent network history segment hashes from diverting
	perMarketELSWeight := make([]PerMarketELSWeight, 0)

	for _, pair := range pv.ELSPerMarket {
		perMarketELSWeight = append(perMarketELSWeight, PerMarketELSWeight{
			Market: pair.MarketId,
			ELS:    pair.Els,
		})
	}

	v := Vote{
		PartyID:                        PartyID(pv.PartyId),
		ProposalID:                     ProposalID(pv.ProposalId),
		Value:                          VoteValue(pv.Value),
		TotalGovernanceTokenBalance:    totalGovernanceTokenBalance,
		TotalGovernanceTokenWeight:     totalGovernanceTokenWeight,
		TotalEquityLikeShareWeight:     totalEquityLikeShareWeight,
		InitialTime:                    NanosToPostgresTimestamp(pv.Timestamp),
		PerMarketEquityLikeShareWeight: perMarketELSWeight,
		TxHash:                         txHash,
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
