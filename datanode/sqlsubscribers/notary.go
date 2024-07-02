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

func (n *Notary) Name() string {
	return "Notary"
}
