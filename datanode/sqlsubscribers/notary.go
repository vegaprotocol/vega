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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/pkg/errors"
)

var ErrNoSignaturesForID = errors.New("no signatures for id")

type NodeSignatureEvent interface {
	events.Event
	NodeSignature() commandspb.NodeSignature
}

type NotaryStore interface {
	Add(context.Context, *entities.NodeSignature) error
}

type Notary struct {
	subscriber
	store NotaryStore
}

func NewNotary(store NotaryStore) *Notary {
	return &Notary{
		store: store,
	}
}

func (n *Notary) Push(ctx context.Context, evt events.Event) error {
	return n.consume(ctx, evt.(NodeSignatureEvent))
}

func (n *Notary) consume(ctx context.Context, event NodeSignatureEvent) error {
	ns := event.NodeSignature()
	record, err := entities.NodeSignatureFromProto(&ns, entities.TxHash(event.TxHash()), n.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting node-signature proto to database entity failed")
	}

	return errors.Wrap(n.store.Add(ctx, record), "inserting node-signature to SQL store failed")
}

func (n *Notary) Types() []events.Type {
	return []events.Type{
		events.NodeSignatureEvent,
	}
}
