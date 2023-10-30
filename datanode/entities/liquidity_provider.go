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

	"code.vegaprotocol.io/vega/protos/vega"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
)

type LiquidityProvider struct {
	PartyID    PartyID
	MarketID   MarketID
	Ordinality int64
	FeeShare   *vega.LiquidityProviderFeeShare
	SLA        *vega.LiquidityProviderSLA
}

type LiquidityProviderCursor struct {
	MarketID   MarketID `json:"marketId"`
	PartyID    PartyID  `json:"partyId"`
	Ordinality int64    `json:"ordinality"`
}

func (c LiquidityProviderCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal liquidity provision cursor: %w", err))
	}
	return string(bs)
}

func (c *LiquidityProviderCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), c)
}

func (lp LiquidityProvider) ToProto() *v2.LiquidityProvider {
	return &v2.LiquidityProvider{
		PartyId:  lp.PartyID.String(),
		MarketId: lp.MarketID.String(),
		FeeShare: lp.FeeShare,
		Sla:      lp.SLA,
	}
}

func (lp LiquidityProvider) Cursor() *Cursor {
	c := LiquidityProviderCursor{
		PartyID: lp.PartyID,
	}

	return NewCursor(c.String())
}

func (lp LiquidityProvider) ToProtoEdge(...any) (*v2.LiquidityProviderEdge, error) {
	return &v2.LiquidityProviderEdge{
		Node:   lp.ToProto(),
		Cursor: lp.Cursor().Encode(),
	}, nil
}
