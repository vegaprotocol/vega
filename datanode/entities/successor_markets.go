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
)

type SuccessorMarket struct {
	Market    Market
	Proposals []*Proposal
}

func (s SuccessorMarket) ToProtoEdge(...any) (*v2.SuccessorMarketEdge, error) {
	props := make([]*vega.GovernanceData, len(s.Proposals))

	for i, p := range s.Proposals {
		props[i] = &vega.GovernanceData{
			Proposal: p.ToProto(),
		}
	}

	e := &v2.SuccessorMarketEdge{
		Node: &v2.SuccessorMarket{
			Market:    s.Market.ToProto(),
			Proposals: props,
		},
		Cursor: s.Market.Cursor().Encode(),
	}

	return e, nil
}

func (s SuccessorMarket) Cursor() *Cursor {
	c := SuccessorMarketCursor{
		VegaTime: s.Market.VegaTime,
	}
	return NewCursor(c.String())
}

type SuccessorMarketCursor struct {
	VegaTime time.Time `json:"vegaTime"`
}

func (mc SuccessorMarketCursor) String() string {
	bs, err := json.Marshal(mc)
	if err != nil {
		panic(fmt.Errorf("could not marshal market cursor: %w", err))
	}
	return string(bs)
}

func (mc *SuccessorMarketCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}

	return json.Unmarshal([]byte(cursorString), mc)
}
