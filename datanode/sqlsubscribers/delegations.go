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
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
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
	log   *logging.Logger
}

func NewDelegation(
	store DelegationStore,
	log *logging.Logger,
) *Delegation {
	t := &Delegation{
		store: store,
		log:   log,
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
	delegation, err := entities.DelegationFromEventProto(&protoDBE)
	if err != nil {
		return errors.Wrap(err, "unable to parse delegation")
	}

	delegation.VegaTime = ds.vegaTime

	if err := ds.store.Add(ctx, delegation); err != nil {
		return errors.Wrap(err, "error adding delegation")
	}

	return nil
}
