// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/data-node/datanode/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
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
	log   *logging.Logger
}

func NewNetworkParameter(
	store NetworkParameterStore,
	log *logging.Logger,
) *NetworkParameter {
	np := &NetworkParameter{
		store: store,
		log:   log,
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
	np, err := entities.NetworkParameterFromProto(&pnp)
	if err != nil {
		return errors.Wrap(err, "unable to parse network parameter")
	}
	np.VegaTime = n.vegaTime

	return errors.Wrap(n.store.Add(ctx, np), "error adding networkParameter")
}
