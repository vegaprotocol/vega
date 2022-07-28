// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events

import (
	"context"

	proto "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/types"
)

type Withdrawal struct {
	*Base
	w proto.Withdrawal
}

func NewWithdrawalEvent(ctx context.Context, w types.Withdrawal) *Withdrawal {
	return &Withdrawal{
		Base: newBase(ctx, WithdrawalEvent),
		w:    *w.IntoProto(),
	}
}

func (w *Withdrawal) Withdrawal() proto.Withdrawal {
	return w.w
}

func (w Withdrawal) IsParty(id string) bool {
	return w.w.PartyId == id
}

func (w Withdrawal) PartyID() string { return w.w.PartyId }

func (w Withdrawal) Proto() proto.Withdrawal {
	return w.w
}

func (w Withdrawal) StreamMessage() *eventspb.BusEvent {
	wit := w.w

	busEvent := newBusEventFromBase(w.Base)
	busEvent.Event = &eventspb.BusEvent_Withdrawal{
		Withdrawal: &wit,
	}

	return busEvent
}

func WithdrawalEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Withdrawal {
	return &Withdrawal{
		Base: newBaseFromBusEvent(ctx, WithdrawalEvent, be),
		w:    *be.GetWithdrawal(),
	}
}
