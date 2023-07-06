package types

import (
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type StopOrderExpiryStrategy = vega.StopOrder_ExpiryStrategy

const (
	// Never valid.
	StopOrderExpiryStrategyUnspecified StopOrderExpiryStrategy = vega.StopOrder_EXPIRY_STRATEGY_UNSPECIFIED
	// The stop order should be cancelled if the expiry time is reached.
	StopOrderExpiryStrategyCancels = vega.StopOrder_EXPIRY_STRATEGY_CANCELS
	// The order should be submitted if the expiry time is reached.
	StopOrderExpiryStrategySubmit = vega.StopOrder_EXPIRY_STRATEGY_SUBMIT
)

type StopOrderTriggerDirection = vega.StopOrder_TriggerDirection

const (
	// Never valid.
	StopOrderTriggerDirectionUnspecified StopOrderTriggerDirection = vega.StopOrder_TRIGGER_DIRECTION_UNSPECIFIED
	// The stop order is triggered once the price falls below a certain level.
	StopOrderTriggerDirectionFallsBelow = vega.StopOrder_TRIGGER_DIRECTION_FALLS_BELOW
	// The stop order is triggered once the price rises above a certain level.
	StopOrderTriggerDirectionRisesAbove = vega.StopOrder_TRIGGER_DIRECTION_RISES_ABOVE
)

type StopOrderStatus = vega.StopOrder_Status

const (
	// Never valid.
	StopOrderStatusUnspecified StopOrderStatus = vega.StopOrder_STATUS_UNSPECIFIED
	// Pending to be executed once the trigger is breached.
	StopOrderStatusPending = vega.StopOrder_STATUS_PENDING
	// Cancelled by the user.
	StopOrderStatusCancelled = vega.StopOrder_STATUS_CANCELLED
	// Stopped by the network, e.g: OCO other side has been triggered.
	StopOrderStatusStopped = vega.StopOrder_STATUS_STOPPED
	// Stop order has been triggered and generated an order.
	StopOrderStatusTriggered = vega.StopOrder_STATUS_TRIGGERED
	// Stop order has expired.
	StopOrderStatusExpired = vega.StopOrder_STATUS_EXPIRED
	// Stop order was rejected at submission.
	StopOrderStatusRejected = vega.StopOrder_STATUS_REJECTED
)

type StopOrderExpiry struct {
	ExpiresAt      *time.Time
	ExpiryStrategy *StopOrderExpiryStrategy
}

func (s StopOrderExpiry) String() string {
	return fmt.Sprintf(
		"expiresAt(%v) expiryStrategy(%v)",
		s.ExpiresAt,
		s.ExpiryStrategy,
	)
}

func (s *StopOrderExpiry) Expires() bool {
	return s.ExpiresAt != nil
}

type StopOrderTrigger struct {
	Direction             StopOrderTriggerDirection
	price                 *num.Uint
	trailingPercentOffset num.Decimal
}

func NewPriceStopOrderTrigger(
	direction StopOrderTriggerDirection,
	price *num.Uint,
) *StopOrderTrigger {
	return &StopOrderTrigger{
		Direction: direction,
		price:     price,
	}
}

func NewTrailingStopOrderTrigger(
	direction StopOrderTriggerDirection,
	trailingPercentOffset num.Decimal,
) *StopOrderTrigger {
	return &StopOrderTrigger{
		Direction:             direction,
		trailingPercentOffset: trailingPercentOffset,
	}
}

func (s StopOrderTrigger) String() string {
	return fmt.Sprintf(
		"price(%v) trailingPercentOffset(%v)",
		s.price,
		s.trailingPercentOffset,
	)
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
	if s.price != nil {
		panic("invalid use of trailing percent offset trigger")
	}
	return s.trailingPercentOffset
}

type StopOrderSetup struct {
	OrderSubmission *OrderSubmission
	Expiry          *StopOrderExpiry
	Trigger         *StopOrderTrigger
}

func (s StopOrderSetup) String() string {
	return fmt.Sprintf(
		"orderSubmission(%v) expiry(%v) trigger(%v)",
		s.OrderSubmission.String(),
		s.Expiry.String(),
		s.Trigger.String(),
	)
}

func StopOrderSetupFromProto(
	psetup *commandspb.StopOrderSetup,
	direction StopOrderTriggerDirection,
) (*StopOrderSetup, error) {
	orderSubmission, err := NewOrderSubmissionFromProto(psetup.OrderSubmission)
	if err != nil {
		return nil, err
	}

	trigger := &StopOrderTrigger{
		Direction: direction,
	}
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
		expiry.ExpiresAt = ptr.From(time.Unix(0, *psetup.ExpiresAt))
		expiry.ExpiryStrategy = psetup.ExpiryStrategy
	}

	return &StopOrderSetup{
		OrderSubmission: orderSubmission,
		Expiry:          expiry,
		Trigger:         trigger,
	}, nil
}

type StopOrdersSubmission struct {
	RisesAbove *StopOrderSetup
	FallsBelow *StopOrderSetup
}

func NewStopOrderSubmissionFromProto(psubmission *commandspb.StopOrdersSubmission) (*StopOrdersSubmission, error) {
	var (
		fallsBelow, risesAbove *StopOrderSetup
		err                    error
	)
	if psubmission.FallsBelow != nil {
		if fallsBelow, err = StopOrderSetupFromProto(psubmission.FallsBelow, StopOrderTriggerDirectionFallsBelow); err != nil {
			return nil, err
		}
	}
	if psubmission.RisesAbove != nil {
		if risesAbove, err = StopOrderSetupFromProto(psubmission.RisesAbove, StopOrderTriggerDirectionRisesAbove); err != nil {
			return nil, err
		}
	}

	return &StopOrdersSubmission{
		FallsBelow: fallsBelow,
		RisesAbove: risesAbove,
	}, nil
}

func (s *StopOrdersSubmission) IntoStopOrders(
	party, fallsBelowID, risesAboveID string,
	now time.Time,
) (fallsBelow, risesAbove *StopOrder) {
	if s.RisesAbove != nil {
		risesAbove = &StopOrder{
			ID:              risesAboveID,
			Party:           party,
			Market:          s.RisesAbove.OrderSubmission.MarketID,
			OrderSubmission: s.RisesAbove.OrderSubmission,
			OCOLinkID:       fallsBelowID,
			Expiry:          s.RisesAbove.Expiry,
			Trigger:         s.RisesAbove.Trigger,
			Status:          StopOrderStatusPending,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	if s.FallsBelow != nil {
		fallsBelow = &StopOrder{
			ID:              fallsBelowID,
			Party:           party,
			Market:          s.FallsBelow.OrderSubmission.MarketID,
			OrderSubmission: s.FallsBelow.OrderSubmission,
			OCOLinkID:       risesAboveID,
			Expiry:          s.FallsBelow.Expiry,
			Trigger:         s.FallsBelow.Trigger,
			Status:          StopOrderStatusPending,
			CreatedAt:       now,
			UpdatedAt:       now,
		}
	}

	return fallsBelow, risesAbove
}

func (s StopOrdersSubmission) String() string {
	rises, falls := "nil", "nil"
	if s.RisesAbove != nil {
		rises = s.RisesAbove.String()
	}
	if s.FallsBelow != nil {
		falls = s.FallsBelow.String()
	}
	return fmt.Sprintf(
		"risesAbove(%v) fallsBelow(%v)",
		rises,
		falls,
	)
}

type StopOrder struct {
	ID              string
	Party           string
	Market          string
	OrderSubmission *OrderSubmission
	OrderID         string
	OCOLinkID       string
	Expiry          *StopOrderExpiry
	Trigger         *StopOrderTrigger
	Status          StopOrderStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

func (s *StopOrder) String() string {
	return fmt.Sprintf(
		"id(%v) party(%v) market(%v) orderSubmission(%v) orderId(%v) ocoLink(%v) expiry(%v) trigger(%v) status(%v) createdAt(%v) updatedAt(%v)",
		s.ID,
		s.Party,
		s.Market,
		s.OrderSubmission.String(),
		s.OrderID,
		s.OCOLinkID,
		s.Expiry.String(),
		s.Trigger.String(),
		s.Status,
		s.CreatedAt,
		s.UpdatedAt,
	)
}

func NewStopOrderFromProto(p *eventspb.StopOrderEvent) *StopOrder {
	sub, err := NewOrderSubmissionFromProto(p.Submission)
	if err != nil {
		panic("submission should always be valid here")
	}

	trigger := &StopOrderTrigger{
		Direction: p.StopOrder.TriggerDirection,
	}
	switch t := p.StopOrder.Trigger.(type) {
	case *vega.StopOrder_Price:
		var overflow bool
		// checking here, but seeing that the payload have been validated down
		// the line there's little to no changes this is invalid
		if trigger.price, overflow = num.UintFromString(t.Price, 10); overflow {
			panic("invalid trigger price")
		}
	case *vega.StopOrder_TrailingPercentOffset:
		var err error
		// same stuff here
		if trigger.trailingPercentOffset, err = num.DecimalFromString(t.TrailingPercentOffset); err != nil {
			panic(err)
		}
	}

	expiry := &StopOrderExpiry{}
	if p.StopOrder.ExpiresAt != nil {
		expiry.ExpiresAt = ptr.From(time.Unix(0, *p.StopOrder.ExpiresAt))
		expiry.ExpiryStrategy = p.StopOrder.ExpiryStrategy
	}

	return &StopOrder{
		ID:              p.StopOrder.Id,
		Party:           p.StopOrder.PartyId,
		Market:          p.StopOrder.MarketId,
		OrderID:         p.StopOrder.OrderId,
		OCOLinkID:       ptr.UnBox(p.StopOrder.OcoLinkId),
		Status:          p.StopOrder.Status,
		CreatedAt:       time.Unix(0, p.StopOrder.CreatedAt),
		UpdatedAt:       time.Unix(0, ptr.UnBox(p.StopOrder.UpdatedAt)),
		OrderSubmission: sub,
		Trigger:         trigger,
		Expiry:          expiry,
	}
}

func (s *StopOrder) ToProtoEvent() *eventspb.StopOrderEvent {
	var updatedAt *int64
	if s.UpdatedAt != (time.Time{}) {
		updatedAt = ptr.From(s.UpdatedAt.UnixNano())
	}

	var ocoLinkID *string
	if len(s.OCOLinkID) > 0 {
		ocoLinkID = ptr.From(s.OCOLinkID)
	}

	ev := &eventspb.StopOrderEvent{
		Submission: s.OrderSubmission.IntoProto(),
		StopOrder: &vega.StopOrder{
			Id:               s.ID,
			PartyId:          s.Party,
			MarketId:         s.Market,
			OrderId:          s.OrderID,
			OcoLinkId:        ocoLinkID,
			Status:           s.Status,
			CreatedAt:        s.CreatedAt.UnixNano(),
			UpdatedAt:        updatedAt,
			TriggerDirection: s.Trigger.Direction,
		},
	}

	if s.Expiry.Expires() {
		ev.StopOrder.ExpiresAt = ptr.From(s.Expiry.ExpiresAt.UnixNano())
		ev.StopOrder.ExpiryStrategy = s.Expiry.ExpiryStrategy
	}

	switch {
	case s.Trigger.IsPrice():
		ev.StopOrder.Trigger = &vega.StopOrder_Price{
			Price: s.Trigger.Price().String(),
		}
	case s.Trigger.IsTrailingPercenOffset():
		ev.StopOrder.Trigger = &vega.StopOrder_TrailingPercentOffset{
			TrailingPercentOffset: s.Trigger.TrailingPercentOffset().String(),
		}
	}

	return ev
}

type StopOrdersCancellation struct {
	MarketID string
	OrderID  string
}

func NewStopOrderCancellationFromProto(
	soc *commandspb.StopOrdersCancellation,
) *StopOrdersCancellation {
	return &StopOrdersCancellation{
		MarketID: ptr.UnBox(soc.MarketId),
		OrderID:  ptr.UnBox(soc.StopOrderId),
	}
}

func (s StopOrdersCancellation) String() string {
	return fmt.Sprintf(
		"marketID(%v) orderID(%v)",
		s.MarketID,
		s.OrderID,
	)
}
