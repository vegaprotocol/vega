package entities

import (
	"encoding/json"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type FundingPayment struct {
	PartyID          PartyID
	MarketID         MarketID
	FundingPeriodSeq uint64
	Amount           num.Decimal
	VegaTime         time.Time
	TxHash           TxHash
}

func NewFundingPaymentsFromProto(
	fps *eventspb.FundingPayments,
	txHash TxHash,
	vegaTime time.Time,
) ([]*FundingPayment, error) {
	payments := make([]*FundingPayment, 0, len(fps.Payments))

	marketID := MarketID(fps.MarketId)
	for _, v := range fps.Payments {
		amount, err := num.DecimalFromString(v.Amount)
		if err != nil {
			return nil, err
		}

		payments = append(payments,
			&FundingPayment{
				PartyID:          PartyID(v.PartyId),
				MarketID:         marketID,
				FundingPeriodSeq: fps.Seq,
				Amount:           amount,
				VegaTime:         vegaTime,
				TxHash:           txHash,
			},
		)
	}

	return payments, nil
}

func (fp FundingPayment) ToProto() *v2.FundingPayment {
	return &v2.FundingPayment{
		PartyId:          fp.PartyID.String(),
		MarketId:         fp.MarketID.String(),
		FundingPeriodSeq: fp.FundingPeriodSeq,
		Timestamp:        fp.VegaTime.UnixNano(),
		Amount:           fp.Amount.String(),
	}
}

func (fp FundingPayment) Cursor() *Cursor {
	pc := FundingPaymentCursor{
		VegaTime: fp.VegaTime,
		MarketID: fp.MarketID,
		PartyID:  fp.PartyID,
	}

	return NewCursor(pc.String())
}

func (fp FundingPayment) ToProtoEdge(_ ...any) (*v2.FundingPaymentEdge, error) {
	return &v2.FundingPaymentEdge{
		Node:   fp.ToProto(),
		Cursor: fp.Cursor().Encode(),
	}, nil
}

type FundingPaymentCursor struct {
	VegaTime time.Time `json:"vegaTime"`
	MarketID MarketID  `json:"marketID"`
	PartyID  PartyID   `json:"partyID"`
}

func (c FundingPaymentCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		panic(fmt.Errorf("could not marshal funding payment cursor: %w", err))
	}
	return string(bs)
}

func (c *FundingPaymentCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}
