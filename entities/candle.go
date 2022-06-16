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
