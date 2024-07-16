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
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/pkg/errors"
)

type WithdrawalEvent interface {
	events.Event
	Withdrawal() vega.Withdrawal
}

type WithdrawalStore interface {
	Upsert(context.Context, *entities.Withdrawal) error
}

type Withdrawal struct {
	subscriber
	store WithdrawalStore
}

func NewWithdrawal(store WithdrawalStore) *Withdrawal {
	return &Withdrawal{
		store: store,
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
	record, err := entities.WithdrawalFromProto(&withdrawal, entities.TxHash(event.TxHash()), w.vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting withdrawal proto to database entity failed")
	}

	return errors.Wrap(w.store.Upsert(ctx, record), "inserting withdrawal to SQL store failed")
}

func (w *Withdrawal) Name() string {
	return "Withdrawal"
}
