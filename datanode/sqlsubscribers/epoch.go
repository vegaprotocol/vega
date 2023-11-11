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
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type EpochUpdateEvent interface {
	events.Event
	Proto() eventspb.EpochEvent
}

type EpochStore interface {
	Add(context.Context, entities.Epoch) error
}

type Epoch struct {
	subscriber
	store EpochStore
}

func NewEpoch(store EpochStore) *Epoch {
	t := &Epoch{
		store: store,
	}
	return t
}

func (es *Epoch) Types() []events.Type {
	return []events.Type{events.EpochUpdate}
}

func (es *Epoch) Push(ctx context.Context, evt events.Event) error {
	return es.consume(ctx, evt.(EpochUpdateEvent))
}

func (es *Epoch) consume(ctx context.Context, event EpochUpdateEvent) error {
	epochUpdateEvent := event.Proto()
	epoch := entities.EpochFromProto(epochUpdateEvent, entities.TxHash(event.TxHash()))
	epoch.VegaTime = es.vegaTime

	return errors.Wrap(es.store.Add(ctx, epoch), "error adding epoch update")
}
