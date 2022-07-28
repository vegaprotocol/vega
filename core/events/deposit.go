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

type Deposit struct {
	*Base
	d proto.Deposit
}

func NewDepositEvent(ctx context.Context, d types.Deposit) *Deposit {
	return &Deposit{
		Base: newBase(ctx, DepositEvent),
		d:    *d.IntoProto(),
	}
}

func (d *Deposit) Deposit() proto.Deposit {
	return d.d
}

func (d Deposit) IsParty(id string) bool {
	return d.d.PartyId == id
}

func (d Deposit) PartyID() string { return d.d.PartyId }

func (d Deposit) Proto() proto.Deposit {
	return d.d
}

func (d Deposit) StreamMessage() *eventspb.BusEvent {
	dep := d.d
	busEvent := newBusEventFromBase(d.Base)
	busEvent.Event = &eventspb.BusEvent_Deposit{
		Deposit: &dep,
	}
	return busEvent
}

func DepositEventFromStream(ctx context.Context, be *eventspb.BusEvent) *Deposit {
	return &Deposit{
		Base: newBaseFromBusEvent(ctx, DepositEvent, be),
		d:    *be.GetDeposit(),
	}
}
