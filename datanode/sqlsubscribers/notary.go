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
	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/events"

	"github.com/pkg/errors"
)

var ErrNoSignaturesForID = errors.New("no signatures for id")

type NodeSignatureEvent interface {
	events.Event
	NodeSignature() commandspb.NodeSignature
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/notary_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/sqlsubscribers NotaryStore
type NotaryStore interface {
	Add(context.Context, *entities.NodeSignature) error
}

type Notary struct {
	subscriber
	store NotaryStore
	log   *logging.Logger
}

func NewNotary(store NotaryStore, log *logging.Logger) *Notary {
	return &Notary{
		store: store,
		log:   log,
	}
}

func (w *Notary) Push(ctx context.Context, evt events.Event) error {
	return w.consume(ctx, evt.(NodeSignatureEvent))
}

func (w *Notary) consume(ctx context.Context, event NodeSignatureEvent) error {
	ns := event.NodeSignature()
	record, err := entities.NodeSignatureFromProto(&ns)
	if err != nil {
		return errors.Wrap(err, "converting node-signature proto to database entity failed")
	}

	return errors.Wrap(w.store.Add(ctx, record), "inserting node-signature to SQL store failed")
}

func (n *Notary) Types() []events.Type {
	return []events.Type{
		events.NodeSignatureEvent,
	}
}
