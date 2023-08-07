package entities

import (
	"encoding/json"
	"fmt"
	"time"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"

	"code.vegaprotocol.io/vega/libs/ptr"

	events "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"code.vegaprotocol.io/vega/libs/num"
)

type FundingPeriod struct {
	MarketID         MarketID
	FundingPeriodSeq uint64
	StartTime        time.Time
	EndTime          *time.Time
	FundingPayment   *num.Decimal
	FundingRate      *num.Decimal
	ExternalTwap     *num.Decimal
	InternalTwap     *num.Decimal
	VegaTime         time.Time
	TxHash           TxHash
}

func NewFundingPeriodFromProto(fp *events.FundingPeriod, txHash TxHash, vegaTime time.Time) (*FundingPeriod, error) {
	fundingPeriod := &FundingPeriod{
		MarketID:         MarketID(fp.MarketId),
		FundingPeriodSeq: fp.Seq,
		StartTime:        NanosToPostgresTimestamp(fp.Start),
		VegaTime:         vegaTime,
		TxHash:           txHash,
	}

	if fp.End != nil {
		fundingPeriod.EndTime = ptr.From(NanosToPostgresTimestamp(*fp.End))
	}

	if fp.FundingPayment != nil {
		fundingPayment, err := num.DecimalFromString(*fp.FundingPayment)
		if err != nil {
			return nil, err
		}
		fundingPeriod.FundingPayment = &fundingPayment
	}

	if fp.FundingRate != nil {
		fundingRate, err := num.DecimalFromString(*fp.FundingRate)
		if err != nil {
			return nil, err
		}
		fundingPeriod.FundingRate = &fundingRate
	}

	if fp.ExternalTwap != nil {
		externalTwap, err := num.DecimalFromString(*fp.ExternalTwap)
		if err != nil {
			return nil, err
		}
		fundingPeriod.ExternalTwap = &externalTwap
	}

	if fp.InternalTwap != nil {
		internalTwap, err := num.DecimalFromString(*fp.InternalTwap)
		if err != nil {
			return nil, err
		}
		fundingPeriod.InternalTwap = &internalTwap
	}

	return fundingPeriod, nil
}

func (fp FundingPeriod) ToProto() *events.FundingPeriod {
	var (
		endTime        *int64
		fundingPayment *string
		fundingRate    *string
		externalTwap   *string
		internalTwap   *string
	)

	if fp.EndTime != nil {
		endTime = ptr.From(fp.EndTime.UnixNano())
	}

	if fp.FundingPayment != nil {
		fundingPayment = ptr.From(fp.FundingPayment.String())
	}

	if fp.FundingRate != nil {
		fundingRate = ptr.From(fp.FundingRate.String())
	}

	if fp.ExternalTwap != nil {
		externalTwap = ptr.From(fp.ExternalTwap.String())
	}

	if fp.InternalTwap != nil {
		internalTwap = ptr.From(fp.InternalTwap.String())
	}

	return &events.FundingPeriod{
		MarketId:       fp.MarketID.String(),
		Seq:            fp.FundingPeriodSeq,
		Start:          fp.StartTime.UnixNano(),
		End:            endTime,
		FundingPayment: fundingPayment,
		FundingRate:    fundingRate,
		ExternalTwap:   externalTwap,
		InternalTwap:   internalTwap,
	}
}

func (fp FundingPeriod) Cursor() *Cursor {
	pc := FundingPeriodCursor{
		StartTime:        fp.StartTime,
		MarketID:         fp.MarketID,
		FundingPeriodSeq: fp.FundingPeriodSeq,
	}

	return NewCursor(pc.String())
}

func (fp FundingPeriod) ToProtoEdge(_ ...any) (*v2.FundingPeriodEdge, error) {
	return &v2.FundingPeriodEdge{
		Node:   fp.ToProto(),
		Cursor: fp.Cursor().Encode(),
	}, nil
}

type FundingPeriodDataPoint struct {
	MarketID         MarketID
	FundingPeriodSeq uint64
	DataPointType    FundingPeriodDataPointSource
	Price            num.Decimal
	Twap             num.Decimal
	Timestamp        time.Time
	VegaTime         time.Time
	TxHash           TxHash
}

func NewFundingPeriodDataPointFromProto(fpdp *events.FundingPeriodDataPoint, txHash TxHash, vegaTime time.Time) (*FundingPeriodDataPoint, error) {
	price, err := num.DecimalFromString(fpdp.Price)
	if err != nil {
		return nil, err
	}

	twap, err := num.DecimalFromString(fpdp.Twap)
	if err != nil {
		return nil, err
	}
	return &FundingPeriodDataPoint{
		MarketID:         MarketID(fpdp.MarketId),
		FundingPeriodSeq: fpdp.Seq,
		DataPointType:    FundingPeriodDataPointSource(fpdp.DataPointType),
		Price:            price,
		Twap:             twap,
		Timestamp:        NanosToPostgresTimestamp(fpdp.Timestamp),
		VegaTime:         vegaTime,
		TxHash:           txHash,
	}, nil
}

func (dp FundingPeriodDataPoint) ToProto() *events.FundingPeriodDataPoint {
	return &events.FundingPeriodDataPoint{
		MarketId:      dp.MarketID.String(),
		Seq:           dp.FundingPeriodSeq,
		DataPointType: events.FundingPeriodDataPoint_Source(dp.DataPointType),
		Price:         dp.Price.String(),
		Twap:          dp.Twap.String(),
		Timestamp:     dp.Timestamp.UnixNano(),
	}
}

func (dp FundingPeriodDataPoint) Cursor() *Cursor {
	pc := FundingPeriodDataPointCursor{
		Timestamp:        dp.Timestamp,
		MarketID:         dp.MarketID,
		FundingPeriodSeq: dp.FundingPeriodSeq,
		DataPointType:    dp.DataPointType,
	}

	return NewCursor(pc.String())
}

func (dp FundingPeriodDataPoint) ToProtoEdge(_ ...any) (*v2.FundingPeriodDataPointEdge, error) {
	return &v2.FundingPeriodDataPointEdge{
		Node:   dp.ToProto(),
		Cursor: dp.Cursor().Encode(),
	}, nil
}

type FundingPeriodCursor struct {
	// We're using start-time over vega-time for the cursor because the funding period record can be updated
	// on settlement and the vega time and tx hash will change, but the start time will not.
	StartTime        time.Time `json:"startTime"`
	MarketID         MarketID  `json:"marketID"`
	FundingPeriodSeq uint64    `json:"fundingPeriodSeq"`
}

func (c FundingPeriodCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal funding period cursor: %w", err))
	}
	return string(bs)
}

func (c *FundingPeriodCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}

type FundingPeriodDataPointCursor struct {
	// We want to use the timestamp of the data point over the vega time for the cursor because the timestamp represents a
	// point in time between the start time and end time of the funding period and it is also used for filtering
	// results when querying via the API.
	Timestamp        time.Time                    `json:"timestamp"`
	MarketID         MarketID                     `json:"marketID"`
	FundingPeriodSeq uint64                       `json:"fundingPeriodSeq"`
	DataPointType    FundingPeriodDataPointSource `json:"dataPointType"`
}

func (c FundingPeriodDataPointCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal funding period data point cursor: %w", err))
	}
	return string(bs)
}

func (c *FundingPeriodDataPointCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
