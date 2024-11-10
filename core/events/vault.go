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

type VaultEvent struct {
	*Base
	vault *eventspb.VaultState
}

func NewVaultEvent(ctx context.Context, vs *eventspb.VaultState) *VaultEvent {
	v := &VaultEvent{
		Base:  newBase(ctx, VaultStateEvent),
		vault: vs,
	}
	return v
}

func (v *VaultEvent) VaultState() *eventspb.VaultState {
	return v.vault
}

func (v *VaultEvent) Proto() eventspb.VaultState {
	return *v.vault
}

func (v *VaultEvent) StreamMessage() *eventspb.BusEvent {
	busEvent := newBusEventFromBase(v.Base)
	busEvent.Event = &eventspb.BusEvent_VaultState{
		VaultState: v.vault,
	}

	return busEvent
}

func VaultEventFromStream(ctx context.Context, be *eventspb.BusEvent) *VaultEvent {
	ve := &VaultEvent{
		Base:  newBaseFromBusEvent(ctx, VaultStateEvent, be),
		vault: be.GetVaultState(),
	}
	return ve
}
