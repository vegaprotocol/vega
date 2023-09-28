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
	"time"

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

type FundingPaymentsEvent interface {
	events.Event
	FundingPayments() *eventspb.FundingPayments
}

type FundingPaymentsStore interface {
	Add([]*entities.FundingPayment) error
	Flush(ctx context.Context) error
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
	return ts.store.Flush(ctx)
}

func (ts *FundingPaymentSubscriber) Push(ctx context.Context, evt events.Event) error {
	return ts.consume(evt.(FundingPaymentsEvent))
}

func (ts *FundingPaymentSubscriber) consume(te FundingPaymentsEvent) error {
	fps := te.FundingPayments()

	return errors.Wrap(ts.addFundingPayments(fps, entities.TxHash(te.TxHash()), ts.vegaTime, te.Sequence()), "failed to consume funding payment")
}

func (ts *FundingPaymentSubscriber) addFundingPayments(fps *eventspb.FundingPayments, txHash entities.TxHash, vegaTime time.Time, blockSeqNumber uint64) error {
	payments, err := entities.NewFundingPaymentsFromProto(fps, txHash, vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting event to funding payments")
	}

	return errors.Wrap(ts.store.Add(payments), "adding funding payment to store")
}
