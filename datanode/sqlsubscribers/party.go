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
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"github.com/pkg/errors"
)

type PartyEvent interface {
	events.Event
	Party() types.Party
}

type PartyStore interface {
	Add(context.Context, entities.Party) error
}

type Party struct {
	subscriber
	store PartyStore
	log   *logging.Logger
}

func NewParty(
	store PartyStore,
	log *logging.Logger,
) *Party {
	ps := &Party{
		store: store,
		log:   log,
	}
	return ps
}

func (ps *Party) Types() []events.Type {
	return []events.Type{events.PartyEvent}
}

func (ps *Party) Push(ctx context.Context, evt events.Event) error {
	return ps.consume(ctx, evt.(PartyEvent))
}

func (ps *Party) consume(ctx context.Context, event PartyEvent) error {
	pp := event.Party()
	p := entities.PartyFromProto(&pp)
	vt := ps.vegaTime
	p.VegaTime = &vt

	return errors.Wrap(ps.store.Add(ctx, p), "error adding party:%w")
}
