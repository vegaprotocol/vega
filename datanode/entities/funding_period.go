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
	VegaTime         time.Time
	TxHash           TxHash
}

func NewFundingPeriodFromProto(fp *events.FundingPeriod, txHash TxHash, vegaTime time.Time) (*FundingPeriod, error) {
	var (
		endTime        *time.Time
		fundingPayment *num.Decimal
		fundingRate    *num.Decimal
		err            error
	)

	if fp.End != nil {
		endTime = ptr.From(NanosToPostgresTimestamp(*fp.End))
	}

	if fp.FundingPayment != nil {
		*fundingPayment, err = num.DecimalFromString(*fp.FundingPayment)
	}

	if fp.FundingRate != nil {
		*fundingRate, err = num.DecimalFromString(*fp.FundingRate)
	}

	return &FundingPeriod{
		MarketID:         MarketID(fp.MarketId),
		FundingPeriodSeq: fp.Seq,
		StartTime:        NanosToPostgresTimestamp(fp.Start),
		EndTime:          endTime,
		FundingPayment:   fundingPayment,
		FundingRate:      fundingRate,
		VegaTime:         vegaTime,
		TxHash:           txHash,
	}, err
}

func (fp FundingPeriod) ToProto() *events.FundingPeriod {
	var (
		endTime        *int64
		fundingPayment *string
		fundingRate    *string
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

	return &events.FundingPeriod{
		MarketId:       fp.MarketID.String(),
		Seq:            fp.FundingPeriodSeq,
		Start:          fp.StartTime.UnixNano(),
		End:            endTime,
		FundingPayment: fundingPayment,
		FundingRate:    fundingRate,
	}
}

func (fp FundingPeriod) Cursor() *Cursor {
	pc := FundingPeriodCursor{
		VegaTime:         fp.VegaTime,
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
	Timestamp        time.Time
	VegaTime         time.Time
	TxHash           TxHash
}

func NewFundingPeriodDataPointFromProto(fpdp *events.FundingPeriodDataPoint, txHash TxHash, vegaTime time.Time) (*FundingPeriodDataPoint, error) {
	price, err := num.DecimalFromString(fpdp.Price)
	if err != nil {
		return nil, err
	}

	return &FundingPeriodDataPoint{
		MarketID:         MarketID(fpdp.MarketId),
		FundingPeriodSeq: fpdp.Seq,
		DataPointType:    FundingPeriodDataPointSource(fpdp.DataPointType),
		Price:            price,
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
		Timestamp:     dp.Timestamp.UnixNano(),
	}
}

func (dp FundingPeriodDataPoint) Cursor() *Cursor {
	pc := FundingPeriodDataPointCursor{
		VegaTime:         dp.VegaTime,
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
	VegaTime         time.Time `json:"vegaTime"`
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
	VegaTime         time.Time                    `json:"vegaTime"`
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
