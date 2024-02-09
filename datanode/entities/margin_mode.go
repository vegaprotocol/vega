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

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type PartyMarginMode struct {
	MarketID                   MarketID
	PartyID                    PartyID
	MarginMode                 vega.MarginMode
	MarginFactor               *num.Decimal
	MinTheoreticalMarginFactor *num.Decimal
	MaxTheoreticalLeverage     *num.Decimal
	AtEpoch                    uint64
}

func (t PartyMarginMode) Cursor() *Cursor {
	tc := PartyMarginModeCursor{
		MarketID: t.MarketID,
		PartyID:  t.PartyID,
	}
	return NewCursor(tc.String())
}

func (t PartyMarginMode) ToProto() *v2.PartyMarginMode {
	var marginFactor, minTheoreticalMarginFactor, maxTheoreticalLeverage *string

	if t.MarginFactor != nil {
		factor := t.MarginFactor.String()
		marginFactor = &factor
	}

	if t.MinTheoreticalMarginFactor != nil {
		factor := t.MinTheoreticalMarginFactor.String()
		minTheoreticalMarginFactor = &factor
	}

	if t.MaxTheoreticalLeverage != nil {
		leverage := t.MaxTheoreticalLeverage.String()
		maxTheoreticalLeverage = &leverage
	}

	return &v2.PartyMarginMode{
		MarketId:                   string(t.MarketID),
		PartyId:                    string(t.PartyID),
		MarginMode:                 t.MarginMode,
		MarginFactor:               marginFactor,
		MinTheoreticalMarginFactor: minTheoreticalMarginFactor,
		MaxTheoreticalLeverage:     maxTheoreticalLeverage,
		AtEpoch:                    t.AtEpoch,
	}
}

func (t PartyMarginMode) ToProtoEdge(_ ...any) (*v2.PartyMarginModeEdge, error) {
	return &v2.PartyMarginModeEdge{
		Node:   t.ToProto(),
		Cursor: t.Cursor().Encode(),
	}, nil
}

func PartyMarginModeFromProto(update *eventspb.PartyMarginModeUpdated) PartyMarginMode {
	var marginFactor, minTheoreticalMarginFactor, maxTheoreticalLeverage *num.Decimal

	if update.MarginFactor != nil {
		factor, _ := num.DecimalFromString(*update.MarginFactor)
		marginFactor = &factor
	}

	if update.MinTheoreticalMarginFactor != nil {
		factor, _ := num.DecimalFromString(*update.MinTheoreticalMarginFactor)
		minTheoreticalMarginFactor = &factor
	}

	if update.MaxTheoreticalLeverage != nil {
		factor, _ := num.DecimalFromString(*update.MaxTheoreticalLeverage)
		maxTheoreticalLeverage = &factor
	}

	return PartyMarginMode{
		MarketID:                   MarketID(update.MarketId),
		PartyID:                    PartyID(update.PartyId),
		MarginMode:                 update.MarginMode,
		MarginFactor:               marginFactor,
		MinTheoreticalMarginFactor: minTheoreticalMarginFactor,
		MaxTheoreticalLeverage:     maxTheoreticalLeverage,
		AtEpoch:                    update.AtEpoch,
	}
}

type PartyMarginModeCursor struct {
	MarketID MarketID
	PartyID  PartyID
}

func (tc PartyMarginModeCursor) String() string {
	bs, err := json.Marshal(tc)
	if err != nil {
		panic(fmt.Errorf("could not marshal party margin mode cursor: %v", err))
	}
	return string(bs)
}

func (tc *PartyMarginModeCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), tc)
}
