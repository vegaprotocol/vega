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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type NetworkParameterEvent interface {
	events.Event
	NetworkParameter() vega.NetworkParameter
}

type NetworkParameterStore interface {
	Add(context.Context, entities.NetworkParameter) error
}

type NetworkParameter struct {
	subscriber
	store NetworkParameterStore
}

func NewNetworkParameter(store NetworkParameterStore) *NetworkParameter {
	np := &NetworkParameter{
		store: store,
	}
	return np
}

func (n *NetworkParameter) Types() []events.Type {
	return []events.Type{events.NetworkParameterEvent}
}

func (n *NetworkParameter) Push(ctx context.Context, evt events.Event) error {
	return n.consume(ctx, evt.(NetworkParameterEvent))
}

func (n *NetworkParameter) consume(ctx context.Context, event NetworkParameterEvent) error {
	pnp := event.NetworkParameter()
	np, err := entities.NetworkParameterFromProto(&pnp, entities.TxHash(event.TxHash()))
	if err != nil {
		return errors.Wrap(err, "unable to parse network parameter")
	}
	np.VegaTime = n.vegaTime

	return errors.Wrap(n.store.Add(ctx, np), "error adding networkParameter")
}
