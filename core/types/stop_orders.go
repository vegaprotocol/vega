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

type StopOrderSizeOverrideSetting = vega.StopOrder_SizeOverrideSetting

const (
	// Never valid.
	StopOrderSizeOverrideSettingUnspecified StopOrderSizeOverrideSetting = vega.StopOrder_SIZE_OVERRIDE_SETTING_UNSPECIFIED
	// No size override is used.
	StopOrderSizeOverrideSettingNone = vega.StopOrder_SIZE_OVERRIDE_SETTING_NONE
	// Use the position size of the trader to override the order size.
	StopOrderSizeOverrideSettingPosition = vega.StopOrder_SIZE_OVERRIDE_SETTING_POSITION
)

type StopOrderRejectionReason = vega.StopOrder_RejectionReason

const (
	// Never valid.
	StopOrderRejectionUnspecified                    StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_UNSPECIFIED
	StopOrderRejectionTradingNotAllowed              StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_TRADING_NOT_ALLOWED
	StopOrderRejectionExpiryInThePast                StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_EXPIRY_IN_THE_PAST
	StopOrderRejectionMustBeReduceOnly               StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_MUST_BE_REDUCE_ONLY
	StopOrderRejectionMaxStopOrdersPerPartyReached   StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_MAX_STOP_ORDERS_PER_PARTY_REACHED
	StopOrderRejectionNotAllowedWithoutAPosition     StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_STOP_ORDER_NOT_ALLOWED_WITHOUT_A_POSITION
	StopOrderRejectionNotClosingThePosition          StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_STOP_ORDER_NOT_CLOSING_THE_POSITION
	StopOrderRejectionLinkedPercentageInvalid        StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_STOP_ORDER_LINKED_PERCENTAGE_INVALID
	StopOrderRejectionNotAllowedDuringOpeningAuction StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_STOP_ORDER_NOT_ALLOWED_DURING_OPENING_AUCTION
	StopOrderRejectionOCONotAllowedSameExpiryTime    StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_STOP_ORDER_CANNOT_MATCH_OCO_EXPIRY_TIMES
	StopOrderRejectionSizeOverrideUnsupportedForSpot StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_STOP_ORDER_SIZE_OVERRIDE_UNSUPPORTED_FOR_SPOT
	StopOrderRejectionSellOrderNotAllowed            StopOrderRejectionReason = vega.StopOrder_REJECTION_REASON_SELL_ORDER_NOT_ALLOWED
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

type StopOrderSizeOverrideValue struct {
	PercentageSize num.Decimal
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

func (s *StopOrderTrigger) IsTrailingPercentOffset() bool {
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
	OrderSubmission     *OrderSubmission
	Expiry              *StopOrderExpiry
	Trigger             *StopOrderTrigger
	SizeOverrideSetting StopOrderSizeOverrideSetting
	SizeOverrideValue   *StopOrderSizeOverrideValue
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

	var sizeOverrideValue *StopOrderSizeOverrideValue
	var sizeOverrideSetting StopOrderSizeOverrideSetting = vega.StopOrder_SIZE_OVERRIDE_SETTING_UNSPECIFIED

	if psetup.SizeOverrideValue != nil {
		value, err := num.DecimalFromString(psetup.SizeOverrideValue.Percentage)
		if err != nil {
			return nil, err
		}
		sizeOverrideValue = &StopOrderSizeOverrideValue{PercentageSize: value}
	}

	if psetup.SizeOverrideSetting != nil {
		sizeOverrideSetting = *psetup.SizeOverrideSetting
	}
	return &StopOrderSetup{
		OrderSubmission:     orderSubmission,
		Expiry:              expiry,
		Trigger:             trigger,
		SizeOverrideSetting: sizeOverrideSetting,
		SizeOverrideValue:   sizeOverrideValue,
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
	fallsBelowParty, risesAboveParty, fallsBelowID, risesAboveID string,
	now time.Time,
) (fallsBelow, risesAbove *StopOrder) {
	if s.RisesAbove != nil {
		risesAbove = &StopOrder{
			ID:                  risesAboveID,
			Party:               risesAboveParty,
			Market:              s.RisesAbove.OrderSubmission.MarketID,
			OrderSubmission:     s.RisesAbove.OrderSubmission,
			OCOLinkID:           fallsBelowID,
			Expiry:              s.RisesAbove.Expiry,
			Trigger:             s.RisesAbove.Trigger,
			Status:              StopOrderStatusPending,
			CreatedAt:           now,
			UpdatedAt:           now,
			SizeOverrideSetting: s.RisesAbove.SizeOverrideSetting,
			SizeOverrideValue:   s.RisesAbove.SizeOverrideValue,
		}
	}

	if s.FallsBelow != nil {
		fallsBelow = &StopOrder{
			ID:                  fallsBelowID,
			Party:               fallsBelowParty,
			Market:              s.FallsBelow.OrderSubmission.MarketID,
			OrderSubmission:     s.FallsBelow.OrderSubmission,
			OCOLinkID:           risesAboveID,
			Expiry:              s.FallsBelow.Expiry,
			Trigger:             s.FallsBelow.Trigger,
			Status:              StopOrderStatusPending,
			CreatedAt:           now,
			UpdatedAt:           now,
			SizeOverrideSetting: s.FallsBelow.SizeOverrideSetting,
			SizeOverrideValue:   s.FallsBelow.SizeOverrideValue,
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
	ID                  string
	Party               string
	Market              string
	OrderSubmission     *OrderSubmission
	OrderID             string
	OCOLinkID           string
	Expiry              *StopOrderExpiry
	Trigger             *StopOrderTrigger
	Status              StopOrderStatus
	CreatedAt           time.Time
	UpdatedAt           time.Time
	RejectionReason     *StopOrderRejectionReason
	SizeOverrideSetting StopOrderSizeOverrideSetting
	SizeOverrideValue   *StopOrderSizeOverrideValue
}

func (s *StopOrder) String() string {
	rejectionReason := "nil"
	if s.RejectionReason != nil {
		rejectionReason = s.RejectionReason.String()
	}
	sizeOverrideValue := "nil"
	if s.SizeOverrideValue != nil {
		sizeOverrideValue = s.SizeOverrideValue.PercentageSize.String()
	}

	return fmt.Sprintf(
		"id(%v) party(%v) market(%v) orderSubmission(%v) orderId(%v) ocoLink(%v) expiry(%v) trigger(%v) status(%v) createdAt(%v) updatedAt(%v) rejectionReason(%v) sizeOverrideSetting(%v) sizeOverrideValue(%v)",
		s.ID,
		s.Party,
		s.Market,
		s.OrderSubmission.String(),
		s.OrderID,
		s.OCOLinkID,
		s.Expiry.String(),
		s.Trigger.String(),
		s.Status,
		s.CreatedAt.UTC(),
		s.UpdatedAt.UTC(),
		rejectionReason,
		s.SizeOverrideSetting,
		sizeOverrideValue,
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

	var sizeOverride *StopOrderSizeOverrideValue
	if p.StopOrder.SizeOverrideSetting == StopOrderSizeOverrideSettingPosition {
		value, err := num.DecimalFromString(p.StopOrder.SizeOverrideValue.GetPercentage())
		if err != nil {
			panic(err)
		}
		sizeOverride = &StopOrderSizeOverrideValue{PercentageSize: value}
	}

	return &StopOrder{
		ID:                  p.StopOrder.Id,
		Party:               p.StopOrder.PartyId,
		Market:              p.StopOrder.MarketId,
		OrderID:             p.StopOrder.OrderId,
		OCOLinkID:           ptr.UnBox(p.StopOrder.OcoLinkId),
		Status:              p.StopOrder.Status,
		CreatedAt:           time.Unix(0, p.StopOrder.CreatedAt),
		UpdatedAt:           time.Unix(0, ptr.UnBox(p.StopOrder.UpdatedAt)),
		OrderSubmission:     sub,
		Trigger:             trigger,
		Expiry:              expiry,
		RejectionReason:     p.StopOrder.RejectionReason,
		SizeOverrideSetting: p.StopOrder.SizeOverrideSetting,
		SizeOverrideValue:   sizeOverride,
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

	var sizeOverrideValue *vega.StopOrder_SizeOverrideValue
	if s.SizeOverrideSetting == StopOrderSizeOverrideSettingPosition {
		sizeOverrideValue = &vega.StopOrder_SizeOverrideValue{Percentage: s.SizeOverrideValue.PercentageSize.String()}
	}

	ev := &eventspb.StopOrderEvent{
		Submission: s.OrderSubmission.IntoProto(),
		StopOrder: &vega.StopOrder{
			Id:                  s.ID,
			PartyId:             s.Party,
			MarketId:            s.Market,
			OrderId:             s.OrderID,
			OcoLinkId:           ocoLinkID,
			Status:              s.Status,
			CreatedAt:           s.CreatedAt.UnixNano(),
			UpdatedAt:           updatedAt,
			TriggerDirection:    s.Trigger.Direction,
			RejectionReason:     s.RejectionReason,
			SizeOverrideSetting: s.SizeOverrideSetting,
			SizeOverrideValue:   sizeOverrideValue,
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
	case s.Trigger.IsTrailingPercentOffset():
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
