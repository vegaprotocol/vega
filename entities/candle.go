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
	"time"

	v2 "code.vegaprotocol.io/protos/data-node/api/v2"

	"code.vegaprotocol.io/protos/vega"

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
		Interval:  interval,
	}, nil
}

func (c *Candle) ToV2CandleProto() *v2.Candle {
	return &v2.Candle{
		Start:      c.PeriodStart.UnixNano(),
		LastUpdate: c.LastUpdateInPeriod.UnixNano(),
		High:       c.High.String(),
		Low:        c.Low.String(),
		Open:       c.Open.String(),
		Close:      c.Close.String(),
		Volume:     c.Volume,
	}
}

func (c Candle) Cursor() *Cursor {
	return NewCursor(c.PeriodStart.Format(time.RFC3339Nano))
}

func (c Candle) ToProtoEdge(_ ...any) (*v2.CandleEdge, error) {
	return &v2.CandleEdge{
		Node:   c.ToV2CandleProto(),
		Cursor: c.Cursor().Encode(),
	}, nil
}
