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
	"fmt"
	"sync"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

type FundingPaymentsEvent interface {
	events.Event
	FundingPayments() *eventspb.FundingPayments
}

type LossSocEvt interface {
	events.Event
	IsFunding() bool
	IsNegative() bool
	PartyID() string
	MarketID() string
	Amount() *num.Int
}

type FundingPaymentsStore interface {
	Add(context.Context, []*entities.FundingPayment) error
	GetByPartyAndMarket(ctx context.Context, partyID entities.PartyID, marketID entities.MarketID) (entities.FundingPayment, error)
}

type FundingPaymentSubscriber struct {
	subscriber
	store FundingPaymentsStore
	mu    *sync.Mutex
	cache map[string]entities.FundingPayment
}

func NewFundingPaymentsSubscriber(store FundingPaymentsStore) *FundingPaymentSubscriber {
	return &FundingPaymentSubscriber{
		store: store,
		mu:    &sync.Mutex{},
		cache: map[string]entities.FundingPayment{},
	}
}

func (ts *FundingPaymentSubscriber) Types() []events.Type {
	return []events.Type{events.FundingPaymentsEvent}
}

func (ts *FundingPaymentSubscriber) Flush(ctx context.Context) error {
	ts.cache = make(map[string]entities.FundingPayment, len(ts.cache))
	return nil
}

func (ts *FundingPaymentSubscriber) Push(ctx context.Context, evt events.Event) error {
	switch et := evt.(type) {
	case FundingPaymentsEvent:
		return ts.consume(ctx, et)
	case LossSocEvt:
		return ts.handleLossSoc(ctx, et)
	default:
		return fmt.Errorf("received unknown event %T (%#v) in FundingPaymentSubscriber", evt, evt)
	}
}

func (ts *FundingPaymentSubscriber) handleLossSoc(ctx context.Context, e LossSocEvt) error {
	if !e.IsFunding() {
		return nil
	}
	var err error
	partyID, marketID := entities.PartyID(e.PartyID()), entities.MarketID(e.MarketID())
	k := fmt.Sprintf("%s%s", partyID, marketID)
	ts.mu.Lock()
	defer ts.mu.Unlock()
	fp, ok := ts.cache[k]
	// loss socialisation for a party that wasn't included in the funding payment event somehow,
	// or the funding payment even in question has not been received yet.
	if !ok {
		fp, err = ts.store.GetByPartyAndMarket(ctx, partyID, marketID)
		if err != nil {
			return err
		}
		// update the tx hash and time
		fp.TxHash = entities.TxHash(e.TxHash())
		fp.VegaTime = ts.vegaTime
		// @TODO see if this even makes sense
		fp.FundingPeriodSeq++
	}
	amtD := num.DecimalFromInt(e.Amount())
	fp.LossAmount = fp.LossAmount.Add(amtD)
	// let's insert this as a new row
	ts.cache[k] = fp
	// first figure out which keys we are missing
	return errors.Wrap(ts.store.Add(ctx, []*entities.FundingPayment{&fp}), "adding funding payment to store with loss socialisation")
}

func (ts *FundingPaymentSubscriber) consume(ctx context.Context, te FundingPaymentsEvent) error {
	ts.mu.Lock()
	defer ts.mu.Unlock()
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
	ts.mu.Lock()
	defer ts.mu.Unlock()
	payments, err := entities.NewFundingPaymentsFromProto(fps, txHash, vegaTime)
	if err != nil {
		return errors.Wrap(err, "converting event to funding payments")
	}
	for _, fp := range payments {
		k := fmt.Sprintf("%s%s", fp.PartyID, fp.MarketID)
		ts.cache[k] = *fp
	}

	return errors.Wrap(ts.store.Add(ctx, payments), "adding funding payment to store")
}

func (ts *FundingPaymentSubscriber) Name() string {
	return "FundingPaymentSubscriber"
}
