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
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type FundingPaymentsEvent interface {
	events.Event
	FundingPayments() *eventspb.FundingPayments
}

type FundingPaymentsStore interface {
	Add(context.Context, []*entities.FundingPayment) error
}

type FundingPaymentSubscriber struct {
	subscriber
	store FundingPaymentsStore
}

func NewFundingPaymentsSubscriber(store FundingPaymentsStore) *FundingPaymentSubscriber {
	return &FundingPaymentSubscriber{
		store: store,
	}
}

func (ts *FundingPaymentSubscriber) Types() []events.Type {
	return []events.Type{events.FundingPaymentsEvent}
}

func (ts *FundingPaymentSubscriber) Flush(ctx context.Context) error {
	return nil
}

func (ts *FundingPaymentSubscriber) Push(ctx context.Context, evt events.Event) error {
	return ts.consume(ctx, evt.(FundingPaymentsEvent))
}

func (ts *FundingPaymentSubscriber) consume(ctx context.Context, te FundingPaymentsEvent) error {
	fps := te.FundingPayments()

	return errors.Wrap(ts.addFundingPayments(ctx, fps, entities.TxHash(te.TxHash()), ts.vegaTime, te.Sequence()), "failed to consume funding payment")
}

func (ts *FundingPaymentSubscriber) addFundingPayments(
	ctx context.Context,
	fps *eventspb.FundingPayments,
	txHash entities.TxHash,
	vegaTime time.Time,
	_ uint64,
) error {
	payments, err := entities.NewFundingPaymentsFromProto(fps, txHash, vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting event to funding payments")
	}

	return errors.Wrap(ts.store.Add(ctx, payments), "adding funding payment to store")
}

func (ts *FundingPaymentSubscriber) Name() string {
	return "FundingPaymentSubscriber"
}
