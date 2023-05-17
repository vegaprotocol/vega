package types

import (
	"errors"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

type StopOrderExpiryStrategy = vega.StopOrder_ExpiryStrategy

const (
	// Never valid
	StopOrdeExpiryStrategyUnspecified StopOrderExpiryStrategy = vega.StopOrder_EXPIRY_STRATEGY_UNSPECIFIED
	// The stop order should be cancelled if the expiry time is reached.
	StopOrderExpiryStrategyCancels = vega.StopOrder_EXPIRY_STRATEGY_CANCELS
	// The order should be submitted if the expiry time is reached.
	StopOrderExpiryStrategySubmit = vega.StopOrder_EXPIRY_STRATEGY_SUBMIT
)

type StopOrdersSubmission struct {
	RisesAbove *StopOrderSetup
	FallsBelow *StopOrderSetup
}

type StopOrderExpiry struct {
	ExpiresAt      *time.Time
	ExpiryStrategy *StopOrderExpiryStrategy
}

func (s *StopOrderExpiry) Expires() bool {
	return s.ExpiresAt != nil
}

type StopOrderTrigger struct {
	price                 *num.Uint
	trailingPercentOffset num.Decimal
}

func (s *StopOrderTrigger) IsPrice() bool {
	return s.price != nil
}

func (s *StopOrderTrigger) IsTrailingPercenOffset() bool {
	return s.price == nil
}

func (s *StopOrderTrigger) Price() *num.Uint {
	if s.price == nil {
		panic("invalid use of price trigger")
	}
	return s.price.Clone()
}

func (s *StopOrderTrigger) TrailingPercentOffset() num.Decimal {
	if s.price == nil {
		panic("invalid use of trailing percent offset trigger")
	}
	return s.trailingPercentOffset
}

type StopOrderSetup struct {
	OrderSubmission *OrderSubmission
	Expiry          *StopOrderExpiry
	Trigger         *StopOrderTrigger
}

func StopOrderSetupFromProto(psetup *commandspb.StopOrderSetup) (*StopOrderSetup, error) {
	orderSubmission, err := NewOrderSubmissionFromProto(psetup.OrderSubmission)
	if err != nil {
		return nil, err
	}

	trigger := &StopOrderTrigger{}
	switch t := psetup.Trigger.(type) {
	case *commandspb.StopOrderSetup_Price:
		var overflow bool
		// checking here, but seeing that the payload have been validated down
		// the line there's little to no changes this is invalid
		if trigger.price, overflow = num.UintFromString(t.Price, 10); overflow {
			return nil, errors.New("invalid trigger price")
		}
	case *commandspb.StopOrderSetup_TrailingPercentOffset:
		var err error
		// same stuff here
		if trigger.trailingPercentOffset, err = num.DecimalFromString(t.TrailingPercentOffset); err != nil {
			return nil, err
		}
	}

	expiry := &StopOrderExpiry{}
	if psetup.ExpiresAt != nil {
		expiry.ExpiresAt = ptr.From(time.Unix(*psetup.ExpiresAt, 0))
		expiry.ExpiryStrategy = psetup.ExpiryStrategy
	}

	return &StopOrderSetup{
		OrderSubmission: orderSubmission,
		Expiry:          expiry,
		Trigger:         trigger,
	}, nil
}

func NewStopOrderSubmissionFromProto(psubmission *commandspb.StopOrdersSubmission) (*StopOrdersSubmission, error) {
	var (
		fallsBelow, risesAbove *StopOrderSetup
		err                    error
	)
	if psubmission.FallsBelow != nil {
		if fallsBelow, err = StopOrderSetupFromProto(psubmission.FallsBelow); err != nil {
			return nil, err
		}
	}
	if psubmission.RisesAbove != nil {
		if risesAbove, err = StopOrderSetupFromProto(psubmission.RisesAbove); err != nil {
			return nil, err
		}
	}

	return &StopOrdersSubmission{
		FallsBelow: fallsBelow,
		RisesAbove: risesAbove,
	}, nil
}
