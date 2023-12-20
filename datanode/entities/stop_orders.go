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
	"errors"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	pbevents "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type (
	_StopOrder  struct{}
	StopOrderID = ID[_StopOrder]
	StopOrder   struct {
		ID                   StopOrderID
		OCOLinkID            StopOrderID
		ExpiresAt            *time.Time
		ExpiryStrategy       StopOrderExpiryStrategy
		TriggerDirection     StopOrderTriggerDirection
		Status               StopOrderStatus
		CreatedAt            time.Time
		UpdatedAt            *time.Time
		OrderID              OrderID
		TriggerPrice         *string
		TriggerPercentOffset *string
		PartyID              PartyID
		MarketID             MarketID
		VegaTime             time.Time
		SeqNum               uint64
		TxHash               TxHash
		Submission           *commandspb.OrderSubmission
		RejectionReason      StopOrderRejectionReason
		SizeOverrideSetting  int32
		SizeOverrideValue    *string
	}
)

type StopOrderKey struct {
	ID        StopOrderID
	UpdatedAt time.Time
	VegaTime  time.Time
}

var StopOrderColumns = []string{
	"id",
	"oco_link_id",
	"expires_at",
	"expiry_strategy",
	"trigger_direction",
	"status",
	"created_at",
	"updated_at",
	"order_id",
	"trigger_price",
	"trigger_percent_offset",
	"party_id",
	"market_id",
	"vega_time",
	"seq_num",
	"tx_hash",
	"submission",
	"rejection_reason",
	"size_override_setting",
	"size_override_value",
}

func (o StopOrder) ToProto() *pbevents.StopOrderEvent {
	var ocoLinkID *string
	var expiresAt, updatedAt *int64
	var expiryStrategy *vega.StopOrder_ExpiryStrategy
	var triggerPrice *vega.StopOrder_Price
	var triggerPercentOffset *vega.StopOrder_TrailingPercentOffset

	if o.OCOLinkID != "" {
		ocoLinkID = ptr.From(o.OCOLinkID.String())
	}

	if o.ExpiresAt != nil {
		expiresAt = ptr.From(o.ExpiresAt.UnixNano())
	}

	if o.ExpiryStrategy != StopOrderExpiryStrategyUnspecified {
		expiryStrategy = ptr.From(vega.StopOrder_ExpiryStrategy(o.ExpiryStrategy))
	}

	if o.TriggerPrice != nil {
		triggerPrice = &vega.StopOrder_Price{
			Price: *o.TriggerPrice,
		}
	}

	if o.TriggerPercentOffset != nil {
		triggerPercentOffset = &vega.StopOrder_TrailingPercentOffset{
			TrailingPercentOffset: *o.TriggerPercentOffset,
		}
	}

	// We cannot copy a nil value to a enum field in the database when using copy, so we only set the
	// rejection reason on the proto if the stop order is rejected. Otherwise, we will leave the proto field
	// as nil
	var rejectionReason *vega.StopOrder_RejectionReason
	if o.Status == StopOrderStatusRejected {
		rejectionReason = ptr.From(vega.StopOrder_RejectionReason(o.RejectionReason))
	}

	var sizeOVerrideValue *vega.StopOrder_SizeOverrideValue

	if o.SizeOverrideValue != nil {
		sizeOVerrideValue = &vega.StopOrder_SizeOverrideValue{
			Percentage: *o.SizeOverrideValue,
		}
	}

	stopOrder := &vega.StopOrder{
		Id:                  o.ID.String(),
		OcoLinkId:           ocoLinkID,
		ExpiresAt:           expiresAt,
		ExpiryStrategy:      expiryStrategy,
		TriggerDirection:    vega.StopOrder_TriggerDirection(o.TriggerDirection),
		Status:              vega.StopOrder_Status(o.Status),
		CreatedAt:           o.CreatedAt.UnixNano(),
		UpdatedAt:           updatedAt,
		OrderId:             o.OrderID.String(),
		PartyId:             o.PartyID.String(),
		MarketId:            o.MarketID.String(),
		RejectionReason:     rejectionReason,
		SizeOverrideSetting: vega.StopOrder_SizeOverrideSetting(o.SizeOverrideSetting),
		SizeOverrideValue:   sizeOVerrideValue,
	}

	if triggerPrice != nil {
		stopOrder.Trigger = triggerPrice
	}

	if triggerPercentOffset != nil {
		stopOrder.Trigger = triggerPercentOffset
	}

	event := &pbevents.StopOrderEvent{
		Submission: o.Submission,
		StopOrder:  stopOrder,
	}

	return event
}

func (s StopOrder) Key() StopOrderKey {
	updatedAt := s.CreatedAt
	if s.UpdatedAt != nil {
		updatedAt = *s.UpdatedAt
	}

	return StopOrderKey{
		ID:        s.ID,
		UpdatedAt: updatedAt,
		VegaTime:  s.VegaTime,
	}
}

func (s StopOrder) Cursor() *Cursor {
	cursor := StopOrderCursor{
		CreatedAt: s.CreatedAt,
		ID:        s.ID,
		VegaTime:  s.VegaTime,
	}

	return NewCursor(cursor.String())
}

func (s StopOrder) ToProtoEdge(_ ...any) (*v2.StopOrderEdge, error) {
	return &v2.StopOrderEdge{
		Node:   s.ToProto(),
		Cursor: s.Cursor().Encode(),
	}, nil
}

func StopOrderFromProto(so *pbevents.StopOrderEvent, vegaTime time.Time, seqNum uint64, txHash TxHash) (StopOrder, error) {
	var (
		ocoLinkID                          StopOrderID
		expiresAt, updatedAt               *time.Time
		expiryStrategy                     = StopOrderExpiryStrategyUnspecified
		triggerPrice, triggerPercentOffset *string
	)

	if so.StopOrder.OcoLinkId != nil {
		ocoLinkID = StopOrderID(*so.StopOrder.OcoLinkId)
	}

	if so.StopOrder.ExpiresAt != nil {
		expiresAt = ptr.From(NanosToPostgresTimestamp(*so.StopOrder.ExpiresAt))
	}

	if so.StopOrder.ExpiryStrategy != nil {
		expiryStrategy = StopOrderExpiryStrategy(*so.StopOrder.ExpiryStrategy)
	}

	if so.StopOrder.UpdatedAt != nil {
		updatedAt = ptr.From(NanosToPostgresTimestamp(*so.StopOrder.UpdatedAt))
		if updatedAt.After(vegaTime) {
			return StopOrder{}, fmt.Errorf("stop order updated time is in the future")
		}
	}

	switch so.StopOrder.Trigger.(type) {
	case *vega.StopOrder_Price:
		price := so.StopOrder.GetPrice()
		_, err := num.DecimalFromString(price)
		if err != nil {
			return StopOrder{}, fmt.Errorf("invalid stop order trigger price: %w", err)
		}

		triggerPrice = ptr.From(price)
	case *vega.StopOrder_TrailingPercentOffset:
		offset := so.StopOrder.GetTrailingPercentOffset()
		percentage, err := num.DecimalFromString(offset)
		if err != nil {
			return StopOrder{}, fmt.Errorf("invalid stop order trigger percent offset: %w", err)
		}
		if percentage.LessThan(num.DecimalZero()) || percentage.GreaterThan(num.DecimalOne()) {
			return StopOrder{}, errors.New("invalid stop order trigger percent offset, must be decimal value between 0 and 1")
		}

		triggerPercentOffset = ptr.From(offset)
	}

	// We will default to unspecified as we need to have a value in the enum field for the pgx copy command to work
	// as it calls EncodeText on the enum fields and this will fail if the value is nil
	// We will only use the rejection reason when we convert back to proto if the status of the order is rejected.
	rejectionReason := StopOrderRejectionReasonUnspecified
	if so.StopOrder.RejectionReason != nil {
		rejectionReason = StopOrderRejectionReason(*so.StopOrder.RejectionReason)
	}

	var sizeOverrideValue *string

	if so.StopOrder.SizeOverrideValue != nil && so.StopOrder.SizeOverrideValue.Percentage != "" {
		sizeOverrideValue = ptr.From(so.StopOrder.SizeOverrideValue.Percentage)
	}

	stopOrder := StopOrder{
		ID:                   StopOrderID(so.StopOrder.Id),
		OCOLinkID:            ocoLinkID,
		ExpiresAt:            expiresAt,
		ExpiryStrategy:       expiryStrategy,
		TriggerDirection:     StopOrderTriggerDirection(so.StopOrder.TriggerDirection),
		Status:               StopOrderStatus(so.StopOrder.Status),
		CreatedAt:            NanosToPostgresTimestamp(so.StopOrder.CreatedAt),
		UpdatedAt:            updatedAt,
		OrderID:              OrderID(so.StopOrder.OrderId),
		TriggerPrice:         triggerPrice,
		TriggerPercentOffset: triggerPercentOffset,
		PartyID:              PartyID(so.StopOrder.PartyId),
		MarketID:             MarketID(so.StopOrder.MarketId),
		VegaTime:             vegaTime,
		SeqNum:               seqNum,
		TxHash:               txHash,
		Submission:           so.Submission,
		RejectionReason:      rejectionReason,
		SizeOverrideSetting:  int32(so.StopOrder.SizeOverrideSetting),
		SizeOverrideValue:    sizeOverrideValue,
	}

	return stopOrder, nil
}

func (so StopOrder) ToRow() []interface{} {
	return []interface{}{
		so.ID,
		so.OCOLinkID,
		so.ExpiresAt,
		so.ExpiryStrategy,
		so.TriggerDirection,
		so.Status,
		so.CreatedAt,
		so.UpdatedAt,
		so.OrderID,
		so.TriggerPrice,
		so.TriggerPercentOffset,
		so.PartyID,
		so.MarketID,
		so.VegaTime,
		so.SeqNum,
		so.TxHash,
		so.Submission,
		so.RejectionReason,
		so.SizeOverrideSetting,
		so.SizeOverrideValue,
	}
}

type StopOrderCursor struct {
	CreatedAt time.Time   `json:"createdAt"`
	ID        StopOrderID `json:"id"`
	VegaTime  time.Time   `json:"vegaTime"`
}

func (c *StopOrderCursor) Parse(cursorString string) error {
	if cursorString == "" {
		return nil
	}
	return json.Unmarshal([]byte(cursorString), c)
}

func (c *StopOrderCursor) String() string {
	bs, err := json.Marshal(c)
	if err != nil {
		// This should never happen
		panic(fmt.Errorf("failed to marshal order stop cursor: %w", err))
	}
	return string(bs)
}
