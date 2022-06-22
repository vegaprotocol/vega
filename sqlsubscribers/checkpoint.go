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

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type CheckpointEvent interface {
	events.Event
	Proto() eventspb.CheckpointEvent
}

type CheckpointStore interface {
	Add(context.Context, entities.Checkpoint) error
}

type Checkpoint struct {
	subscriber
	store CheckpointStore
	log   *logging.Logger
}

func NewCheckpoint(
	store CheckpointStore,
	log *logging.Logger,
) *Checkpoint {
	np := &Checkpoint{
		store: store,
		log:   log,
	}
	return np
}

func (n *Checkpoint) Types() []events.Type {
	return []events.Type{events.CheckpointEvent}
}

func (n *Checkpoint) Push(ctx context.Context, evt events.Event) error {
	return n.consume(ctx, evt.(CheckpointEvent))
}

func (n *Checkpoint) consume(ctx context.Context, event CheckpointEvent) error {
	pnp := event.Proto()
	np, err := entities.CheckpointFromProto(&pnp)
	if err != nil {
		return errors.Wrap(err, "unable to parse checkpoint")
	}
	np.VegaTime = n.vegaTime

	if err := n.store.Add(ctx, np); err != nil {
		return errors.Wrap(err, "error adding checkpoint")
	}

	return nil
}
