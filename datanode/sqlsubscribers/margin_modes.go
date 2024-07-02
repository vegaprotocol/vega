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
	"code.vegaprotocol.io/vega/datanode/entities"

	"github.com/pkg/errors"
)

type MarginModeStore interface {
	UpdatePartyMarginMode(ctx context.Context, update entities.PartyMarginMode) error
}

type MarginModes struct {
	subscriber
	store MarginModeStore
}

func (t *MarginModes) Types() []events.Type {
	return []events.Type{
		events.PartyMarginModeUpdatedEvent,
	}
}

func (t *MarginModes) Push(ctx context.Context, evt events.Event) error {
	switch e := evt.(type) {
	case PartyMarginModeUpdatedEvent:
		return t.consumePartyMarginModeUpdatedEvent(ctx, e)
	default:
		return nil
	}
}

func (t *MarginModes) consumePartyMarginModeUpdatedEvent(ctx context.Context, e PartyMarginModeUpdatedEvent) error {
	mode := entities.PartyMarginModeFromProto(e.PartyMarginModeUpdated())
	return errors.Wrap(t.store.UpdatePartyMarginMode(ctx, mode), "update party margin mode")
}

func NewMarginModes(store MarginModeStore) *MarginModes {
	return &MarginModes{
		store: store,
	}
}

func (t *MarginModes) Name() string {
	return "MarginModes"
}
