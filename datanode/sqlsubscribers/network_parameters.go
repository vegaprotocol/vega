// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package sqlsubscribers

import (
	"context"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/protos/vega"
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
