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
	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type DepositEvent interface {
	events.Event
	Deposit() vega.Deposit
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/deposits_mock.go -package mocks code.vegaprotocol.io/data-node/sqlsubscribers DepositStore
type DepositStore interface {
	Upsert(context.Context, *entities.Deposit) error
}

type Deposit struct {
	subscriber
	store DepositStore
	log   *logging.Logger
}

func NewDeposit(store DepositStore, log *logging.Logger) *Deposit {
	return &Deposit{
		store: store,
		log:   log,
	}
}

func (d *Deposit) Types() []events.Type {
	return []events.Type{events.DepositEvent}
}

func (d *Deposit) Push(ctx context.Context, evt events.Event) error {
	return d.consume(ctx, evt.(DepositEvent))
}

func (d *Deposit) consume(ctx context.Context, event DepositEvent) error {
	deposit := event.Deposit()
	record, err := entities.DepositFromProto(&deposit, d.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting deposit proto to database entity failed")
	}

	return errors.Wrap(d.store.Upsert(ctx, record), "inserting deposit to SQL store failed")
}
