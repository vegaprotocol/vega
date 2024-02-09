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

package sqlsubscribers

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type PartyEvent interface {
	events.Event
	Party() types.Party
}

type PartyProfileUpdatedEvent interface {
	events.Event
	PartyProfileUpdated() *eventspb.PartyProfileUpdated
}

type PartyStore interface {
	Add(context.Context, entities.Party) error
	UpdateProfile(ctx context.Context, updated *entities.PartyProfile) error
}

type Party struct {
	subscriber
	store PartyStore
}

func NewParty(store PartyStore) *Party {
	ps := &Party{
		store: store,
	}
	return ps
}

func (ps *Party) Types() []events.Type {
	return []events.Type{events.PartyEvent, events.PartyProfileUpdatedEvent}
}

func (ps *Party) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case PartyEvent:
		return ps.consumeNewParty(ctx, e)
	case PartyProfileUpdatedEvent:
		return ps.consumePartyProfileUpdated(ctx, e)
	default:
		return nil
	}
}

func (ps *Party) consumeNewParty(ctx context.Context, event PartyEvent) error {
	pp := event.Party().IntoProto()
	p := entities.PartyFromProto(pp, entities.TxHash(event.TxHash()))
	vt := ps.vegaTime
	p.VegaTime = &vt

	return errors.Wrap(ps.store.Add(ctx, p), "adding party")
}

func (ps *Party) consumePartyProfileUpdated(ctx context.Context, e PartyProfileUpdatedEvent) error {
	updateEvent := e.PartyProfileUpdated()
	updated := entities.PartyProfileFromProto(updateEvent.UpdatedProfile)

	return errors.Wrap(ps.store.UpdateProfile(ctx, updated), "updating party profile")
}
