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
	pbevents "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type (
	StopOrderEvent interface {
		events.Event
		StopOrder() *pbevents.StopOrderEvent
	}

	StopOrderStore interface {
		Add(entities.StopOrder) error
		Flush(ctx context.Context) error
	}

	StopOrder struct {
		subscriber
		store StopOrderStore
	}
)

func NewStopOrder(store StopOrderStore) *StopOrder {
	return &StopOrder{
		store: store,
	}
}

func (so *StopOrder) Types() []events.Type {
	return []events.Type{
		events.StopOrderEvent,
	}
}

func (so *StopOrder) Push(ctx context.Context, evt events.Event) error {
	return so.consume(evt.(StopOrderEvent), evt.Sequence())
}

func (so *StopOrder) Flush(ctx context.Context) error {
	return so.store.Flush(ctx)
}

func (so *StopOrder) consume(evt StopOrderEvent, seqNum uint64) error {
	protoOrder := evt.StopOrder()
	stop, err := entities.StopOrderFromProto(protoOrder, so.vegaTime, seqNum, entities.TxHash(evt.TxHash()))
	if err != nil {
		return errors.Wrap(err, "deserializing stop order")
	}
	return errors.Wrap(so.store.Add(stop), "adding stop order")
}
