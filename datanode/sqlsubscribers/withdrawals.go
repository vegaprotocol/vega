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

type WithdrawalEvent interface {
	events.Event
	Withdrawal() vega.Withdrawal
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/withdrawals_mock.go -package mocks code.vegaprotocol.io/data-node/datanode/sqlsubscribers WithdrawalStore
type WithdrawalStore interface {
	Upsert(context.Context, *entities.Withdrawal) error
}

type Withdrawal struct {
	subscriber
	store WithdrawalStore
	log   *logging.Logger
}

func NewWithdrawal(store WithdrawalStore, log *logging.Logger) *Withdrawal {
	return &Withdrawal{
		store: store,
		log:   log,
	}
}

func (w *Withdrawal) Types() []events.Type {
	return []events.Type{events.WithdrawalEvent}
}

func (w *Withdrawal) Push(ctx context.Context, evt events.Event) error {
	return w.consume(ctx, evt.(WithdrawalEvent))
}

func (w *Withdrawal) consume(ctx context.Context, event WithdrawalEvent) error {
	withdrawal := event.Withdrawal()
	record, err := entities.WithdrawalFromProto(&withdrawal, w.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting withdrawal proto to database entity failed")
	}

	return errors.Wrap(w.store.Upsert(ctx, record), "inserting withdrawal to SQL store failed")
}
