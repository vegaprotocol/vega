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

type Candle struct {
	PeriodStart        time.Time
	LastUpdateInPeriod time.Time
	Open               decimal.Decimal
	Close              decimal.Decimal
	High               decimal.Decimal
	Low                decimal.Decimal
	Volume             uint64
	Notional           uint64
}

func (c *Candle) ToV1CandleProto(interval vega.Interval) (*vega.Candle, error) {
	return &vega.Candle{
		Timestamp: c.PeriodStart.UnixNano(),
		Datetime:  c.LastUpdateInPeriod.Format(time.RFC3339Nano),
		High:      c.High.String(),
		Low:       c.Low.String(),
		Open:      c.Open.String(),
		Close:     c.Close.String(),
		Volume:    c.Volume,
		Notional:  c.Notional,
		Interval:  interval,
	}, nil
}

func (c *Candle) ToV2CandleProto() *v2.Candle {
	var openPx, highPx, lowPx, closePx string

	if c.Open.GreaterThan(decimal.Zero) {
		openPx = c.Open.String()
	}

	if c.High.GreaterThan(decimal.Zero) {
		highPx = c.High.String()
	}

	if c.Low.GreaterThan(decimal.Zero) {
		lowPx = c.Low.String()
	}

	if c.Close.GreaterThan(decimal.Zero) {
		closePx = c.Close.String()
	}

	return &v2.Candle{
		Start:      c.PeriodStart.UnixNano(),
		LastUpdate: c.LastUpdateInPeriod.UnixNano(),
		High:       highPx,
		Low:        lowPx,
		Open:       openPx,
		Close:      closePx,
		Volume:     c.Volume,
		Notional:   c.Notional,
	}
}

func (c Candle) Cursor() *Cursor {
	cc := CandleCursor{
		PeriodStart: c.PeriodStart,
	}
	return NewCursor(cc.String())
}

func (c Candle) ToProtoEdge(_ ...any) (*v2.CandleEdge, error) {
	return &v2.CandleEdge{
		Node:   c.ToV2CandleProto(),
		Cursor: c.Cursor().Encode(),
	}, nil
}

type CandleCursor struct {
	PeriodStart time.Time `json:"periodStart"`
}

func (c CandleCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal candle cursor: %w", err))
	}
	return string(bs)
}

func (c *CandleCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
