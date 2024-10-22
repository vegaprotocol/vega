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

package sqlsubscribers_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers"
	"code.vegaprotocol.io/vega/datanode/sqlsubscribers/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type fpSub struct {
	*sqlsubscribers.FundingPaymentSubscriber
	ctrl  *gomock.Controller
	store *mocks.MockFundingPaymentsStore
}

func TestBasicInterface(t *testing.T) {
	sub := getFundingPaymentsSub(t)
	ctx := context.Background()
	defer sub.ctrl.Finish()
	types := sub.Types()
	require.Equal(t, 2, len(types))
	require.EqualValues(t, []events.Type{
		events.FundingPaymentsEvent,
		events.LossSocializationEvent,
	}, types)
	require.NoError(t, sub.Flush(ctx))
}

func TestUnmatchedLossSocCases(t *testing.T) {
	t.Run("simple case getting the previous record and adding a new one with amount lost", testUnmatchedLossSoc)
	t.Run("excess from loss socialisation sets amount lost to zero", testUnmatchedZeroOutLoss)
	t.Run("excess from loss socialisation increases amount", testUnmatchedIncreasesAmount)
	t.Run("excess from loss socialisation decreases loss amount", testUnmatchedDecreasesLoss)
}

func TestCachedFundingPaymentsAndLossSocialisation(t *testing.T) {
	t.Run("expected flow: funding payment followed by loss socialisation events", processFundingPaymentThenLossSocialisation)
}

func processFundingPaymentThenLossSocialisation(t *testing.T) {
	sub := getFundingPaymentsSub(t)
	defer sub.ctrl.Finish()
	ctx := context.Background()
	now := time.Now()
	party, market := "partyID", "marketID"
	won := uint64(1000)
	seq := uint64(123)
	loss := num.NewUint(100)
	// first, send the funding payment events: party wins 1000
	fEvt := events.NewFundingPaymentsEvent(ctx, market, seq, []events.Transfer{
		getTransferEvent(party, market, num.NewUint(won), false),
	})
	var got entities.FundingPayment
	sub.store.EXPECT().Add(gomock.Any(), gomock.Any()).Times(1).Do(func(_ context.Context, data []*entities.FundingPayment) {
		profit := num.DecimalFromFloat(float64(won))
		require.Equal(t, 1, len(data))
		require.True(t, profit.Equal(data[0].Amount))
		require.True(t, data[0].LossSocialisationAmount.IsZero())
		require.Equal(t, seq, data[0].FundingPeriodSeq)
		got = *data[0]
	}).Return(nil)

	// step 1: send the funding payment events, see if we get the data we expect.
	sub.Push(ctx, fEvt)

	// now create the loss socialisation event.
	evt := events.NewLossSocializationEvent(ctx, party, market, loss, true, now.Unix(), types.LossTypeFunding)
	// make sure we process this event
	require.True(t, evt.IsFunding())

	// should be in cache, so no need to expect a call here
	// sub.store.EXPECT().GetByPartyAndMarket(gomock.Any(), party, market).Times(0)
	// use the DoAndReturn to make sure the entitiy is updated correctly
	sub.store.EXPECT().Add(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, data []*entities.FundingPayment) error {
		require.Equal(t, 1, len(data))
		require.Equal(t, got.PartyID, data[0].PartyID)
		require.Equal(t, got.MarketID, data[0].MarketID)
		require.Equal(t, got.FundingPeriodSeq, data[0].FundingPeriodSeq)
		require.True(t, data[0].LossSocialisationAmount.IsNegative())
		require.True(t, data[0].LossSocialisationAmount.IsInteger())
		require.Equal(t, loss.String(), data[0].LossSocialisationAmount.Abs().String())
		// amounts are untouched
		require.True(t, got.Amount.Equal(data[0].Amount))
		return nil
	})

	sub.Push(ctx, evt)
	// make sure to reset the cache
	sub.Flush(ctx)
}

func testUnmatchedLossSoc(t *testing.T) {
	sub := getFundingPaymentsSub(t)
	defer sub.ctrl.Finish()
	ctx := context.Background()
	last := time.Now().Add(-1 * time.Second)
	party, market, nowTS := "partyID", "marketID", last.Add(time.Second).Unix()
	loss := num.NewUint(100)
	get := entities.FundingPayment{
		PartyID:                 entities.PartyID(party),
		MarketID:                entities.MarketID(market),
		FundingPeriodSeq:        123,
		VegaTime:                last,
		Amount:                  num.DecimalZero(),
		LossSocialisationAmount: num.DecimalZero(),
	}
	evt := events.NewLossSocializationEvent(ctx, party, market, loss, true, nowTS, types.LossTypeFunding)
	// make sure we process this event
	require.True(t, evt.IsFunding())

	sub.store.EXPECT().GetByPartyAndMarket(gomock.Any(), party, market).Times(1).Return(get, nil)
	// use the DoAndReturn to make sure the entitiy is updated correctly
	sub.store.EXPECT().Add(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, data []*entities.FundingPayment) error {
		require.Equal(t, 1, len(data))
		require.Equal(t, get.PartyID, data[0].PartyID)
		require.Equal(t, get.MarketID, data[0].MarketID)
		require.Equal(t, get.FundingPeriodSeq, data[0].FundingPeriodSeq)
		require.True(t, data[0].LossSocialisationAmount.IsNegative())
		require.True(t, data[0].LossSocialisationAmount.IsInteger())
		require.Equal(t, loss.String(), data[0].LossSocialisationAmount.Abs().String())
		return nil
	})

	sub.Push(ctx, evt)
	// make sure to reset the cache
	sub.Flush(ctx)
}

func testUnmatchedZeroOutLoss(t *testing.T) {
	sub := getFundingPaymentsSub(t)
	defer sub.ctrl.Finish()
	ctx := context.Background()
	last := time.Now().Add(-1 * time.Second)
	party, market, nowTS := "partyID", "marketID", last.Add(time.Second).Unix()
	loss := num.NewUint(100)
	get := entities.FundingPayment{
		PartyID:                 entities.PartyID(party),
		MarketID:                entities.MarketID(market),
		FundingPeriodSeq:        123,
		VegaTime:                last,
		Amount:                  num.DecimalZero(),
		LossSocialisationAmount: num.DecimalFromUint(loss).Neg(),
	}
	evt := events.NewLossSocializationEvent(ctx, party, market, loss, false, nowTS, types.LossTypeFunding)
	// make sure we process this event
	require.True(t, evt.IsFunding())

	sub.store.EXPECT().GetByPartyAndMarket(gomock.Any(), party, market).Times(1).Return(get, nil)
	// use the DoAndReturn to make sure the entitiy is updated correctly
	sub.store.EXPECT().Add(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, data []*entities.FundingPayment) error {
		require.Equal(t, 1, len(data))
		require.Equal(t, get.PartyID, data[0].PartyID)
		require.Equal(t, get.MarketID, data[0].MarketID)
		require.Equal(t, get.FundingPeriodSeq, data[0].FundingPeriodSeq)
		require.True(t, data[0].LossSocialisationAmount.IsZero())
		return nil
	})

	sub.Push(ctx, evt)
	// make sure to reset the cache
	sub.Flush(ctx)
}

func testUnmatchedIncreasesAmount(t *testing.T) {
	sub := getFundingPaymentsSub(t)
	defer sub.ctrl.Finish()
	ctx := context.Background()
	last := time.Now().Add(-1 * time.Second)
	party, market, nowTS := "partyID", "marketID", last.Add(time.Second).Unix()
	loss := num.NewUint(11)
	get := entities.FundingPayment{
		PartyID:                 entities.PartyID(party),
		MarketID:                entities.MarketID(market),
		FundingPeriodSeq:        123,
		VegaTime:                last,
		Amount:                  num.DecimalFromFloat(1000),
		LossSocialisationAmount: num.DecimalFromFloat(1).Neg(),
	}
	evt := events.NewLossSocializationEvent(ctx, party, market, loss, false, nowTS, types.LossTypeFunding)
	// make sure we process this event
	require.True(t, evt.IsFunding())

	sub.store.EXPECT().GetByPartyAndMarket(gomock.Any(), party, market).Times(1).Return(get, nil)
	// use the DoAndReturn to make sure the entitiy is updated correctly
	sub.store.EXPECT().Add(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, data []*entities.FundingPayment) error {
		excess := num.DecimalFromUint(loss).Add(get.LossSocialisationAmount)
		require.Equal(t, 1, len(data))
		require.Equal(t, get.PartyID, data[0].PartyID)
		require.Equal(t, get.MarketID, data[0].MarketID)
		require.Equal(t, get.FundingPeriodSeq, data[0].FundingPeriodSeq)
		require.True(t, data[0].LossSocialisationAmount.IsZero()) // loss is zeroed out
		// the new amount equals the excess in additional payout (so amount + extra pay - loss amount)
		require.True(t, get.Amount.Add(excess).Equal(data[0].Amount))
		return nil
	})

	sub.Push(ctx, evt)
	// make sure to reset the cache
	sub.Flush(ctx)
}

func testUnmatchedDecreasesLoss(t *testing.T) {
	sub := getFundingPaymentsSub(t)
	defer sub.ctrl.Finish()
	ctx := context.Background()
	last := time.Now().Add(-1 * time.Second)
	party, market, nowTS := "partyID", "marketID", last.Add(time.Second).Unix()
	loss := num.NewUint(10)
	get := entities.FundingPayment{
		PartyID:                 entities.PartyID(party),
		MarketID:                entities.MarketID(market),
		FundingPeriodSeq:        123,
		VegaTime:                last,
		Amount:                  num.DecimalFromFloat(1000),
		LossSocialisationAmount: num.DecimalFromFloat(100).Neg(),
	}
	evt := events.NewLossSocializationEvent(ctx, party, market, loss, false, nowTS, types.LossTypeFunding)
	// make sure we process this event
	require.True(t, evt.IsFunding())

	sub.store.EXPECT().GetByPartyAndMarket(gomock.Any(), party, market).Times(1).Return(get, nil)
	// use the DoAndReturn to make sure the entitiy is updated correctly
	sub.store.EXPECT().Add(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, data []*entities.FundingPayment) error {
		require.Equal(t, 1, len(data))
		require.Equal(t, get.PartyID, data[0].PartyID)
		require.Equal(t, get.MarketID, data[0].MarketID)
		require.Equal(t, get.FundingPeriodSeq, data[0].FundingPeriodSeq)
		// loss soc has been reduced by the amount from the event
		require.True(t, data[0].LossSocialisationAmount.Equal(get.LossSocialisationAmount.Add(num.DecimalFromUint(loss)))) // loss is zeroed out
		// amount paid hasn't changed
		require.True(t, get.Amount.Equal(data[0].Amount))
		return nil
	})

	sub.Push(ctx, evt)
	// make sure to reset the cache
	sub.Flush(ctx)
}

func getFundingPaymentsSub(t *testing.T) *fpSub {
	t.Helper()
	ctrl := gomock.NewController(t)
	store := mocks.NewMockFundingPaymentsStore(ctrl)
	sub := sqlsubscribers.NewFundingPaymentsSubscriber(store)
	return &fpSub{
		FundingPaymentSubscriber: sub,
		ctrl:                     ctrl,
		store:                    store,
	}
}

type mpStub struct {
	party  string
	market string
	amount *num.Uint
	loss   bool
}

func (m *mpStub) Party() string {
	return m.party
}

func (m *mpStub) Size() int64 {
	return 0
}

func (m *mpStub) Buy() int64 {
	return 0
}

func (m *mpStub) Sell() int64 {
	return 0
}

func (m *mpStub) Price() *num.Uint {
	return num.UintZero()
}

func (m *mpStub) BuySumProduct() *num.Uint {
	return num.UintZero()
}

func (m *mpStub) SellSumProduct() *num.Uint {
	return num.UintZero()
}

func (m *mpStub) VWBuy() *num.Uint {
	return num.UintZero()
}

func (m *mpStub) VWSell() *num.Uint {
	return num.UintZero()
}

func (m *mpStub) AverageEntryPrice() *num.Uint {
	return num.UintZero()
}

func (m *mpStub) Transfer() *types.Transfer {
	ret := &types.Transfer{
		Owner: m.party,
		Amount: &types.FinancialAmount{
			Asset:  "testasset",
			Amount: m.amount.Clone(),
		},
		Type:       types.TransferTypePerpFundingWin,
		MinAmount:  num.UintZero(),
		Market:     "market",
		TransferID: ptr.From("test"),
	}
	if m.loss {
		ret.Type = types.TransferTypePerpFundingLoss
	}
	return ret
}

func getTransferEvent(party, market string, amount *num.Uint, loss bool) events.Transfer {
	if amount == nil {
		amount = num.UintZero()
	}
	mp := mpStub{
		party:  party,
		market: market,
		amount: amount.Clone(),
		loss:   loss,
	}
	return &mp
}
