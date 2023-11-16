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

type DelegationBalanceEvent interface {
	events.Event
	Proto() eventspb.DelegationBalanceEvent
}

type DelegationStore interface {
	Add(context.Context, entities.Delegation) error
}

type Delegation struct {
	subscriber
	store DelegationStore
}

func NewDelegation(store DelegationStore) *Delegation {
	t := &Delegation{
		store: store,
	}
	return t
}

func (ds *Delegation) Types() []events.Type {
	return []events.Type{events.DelegationBalanceEvent}
}

func (ds *Delegation) Push(ctx context.Context, evt events.Event) error {
	return ds.consume(ctx, evt.(DelegationBalanceEvent))
}

func (ds *Delegation) consume(ctx context.Context, event DelegationBalanceEvent) error {
	protoDBE := event.Proto()
	delegation, err := entities.DelegationFromEventProto(&protoDBE, entities.TxHash(event.TxHash()))
	if err != nil {
		return errors.Wrap(err, "unable to parse delegation")
	}

	delegation.VegaTime = ds.vegaTime
	delegation.SeqNum = event.Sequence()

	if err := ds.store.Add(ctx, delegation); err != nil {
		return errors.Wrap(err, "error adding delegation")
	}

	return nil
}
