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

package events

import (
	"context"

	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type RedemptionEvent struct {
	*Base
	redemption *eventspb.RedemptionRequest
}

func NewRedemptionEvent(ctx context.Context, rr *eventspb.RedemptionRequest) *RedemptionEvent {
	re := &RedemptionEvent{
		Base:       newBase(ctx, RedemptionRequestEvent),
		redemption: rr,
	}
	return re
}

func (re *RedemptionEvent) RedemptionRequest() *eventspb.RedemptionRequest {
	return re.redemption
}

func (re *RedemptionEvent) Proto() eventspb.RedemptionRequest {
	return *re.redemption
}

func (re *RedemptionEvent) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(re.Base)
	busEvent.Event = &eventspb.BusEvent_RedemptionRequest{
		RedemptionRequest: re.redemption,
	}

	return busEvent
}

func RedemptionEventFromStream(ctx context.Context, be *eventspb.BusEvent) *RedemptionEvent {
	re := &RedemptionEvent{
		Base:       newBaseFromBusEvent(ctx, RedemptionRequestEvent, be),
		redemption: be.GetRedemptionRequest(),
	}
	return re
}
