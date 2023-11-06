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

package sqlstore_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/sqlstore"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/ptr"

	"github.com/stretchr/testify/assert"
)

type fundingPaymentTestStore struct {
	fp *sqlstore.FundingPayments
}

func newFundingPaymentTestStore(t *testing.T) *fundingPaymentTestStore {
	t.Helper()
	return &fundingPaymentTestStore{
		fp: sqlstore.NewFundingPayments(connectionSource),
	}
}

func TestFundingPayments(t *testing.T) {
	ctx := tempTransaction(t)
	store := newFundingPaymentTestStore(t)

	now := time.Now()

	fundingPayments := []*entities.FundingPayment{
		{
			PartyID: entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0"),

			MarketID:         entities.MarketID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			FundingPeriodSeq: 1,
			Amount:           num.MustDecimalFromString("-100"),
			VegaTime:         now,
			TxHash:           "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0",
		},
		{
			PartyID: entities.PartyID("947a700141e3d175304ee176d0beecf9ee9f462e09330e33c386952caf21f679"),

			MarketID:         entities.MarketID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			FundingPeriodSeq: 1,
			Amount:           num.MustDecimalFromString("100"),
			VegaTime:         now,
			TxHash:           "f1e520d7612de709503d493a3335a4aa8e8b3125b5d5661b7bed7f509b67bf53",
		},
		{
			PartyID: entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0"),

			MarketID:         entities.MarketID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			FundingPeriodSeq: 2,
			Amount:           num.MustDecimalFromString("42"),
			VegaTime:         now.Add(5 * time.Second),
			TxHash:           "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0",
		},
		{
			PartyID: entities.PartyID("947a700141e3d175304ee176d0beecf9ee9f462e09330e33c386952caf21f679"),

			MarketID:         entities.MarketID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			FundingPeriodSeq: 2,
			Amount:           num.MustDecimalFromString("-42"),
			VegaTime:         now.Add(5 * time.Second),
			TxHash:           "f1e520d7612de709503d493a3335a4aa8e8b3125b5d5661b7bed7f509b67bf53",
		},
		{
			PartyID: entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0"),

			MarketID:         entities.MarketID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			FundingPeriodSeq: 3,
			Amount:           num.MustDecimalFromString("25"),
			VegaTime:         now.Add(10 * time.Second),
			TxHash:           "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0",
		},
		{
			PartyID: entities.PartyID("947a700141e3d175304ee176d0beecf9ee9f462e09330e33c386952caf21f679"),

			MarketID:         entities.MarketID("46d66ea0a00609615e04aaf6b41e5e9f552650535ed85059444d68bb6456852a"),
			FundingPeriodSeq: 3,
			Amount:           num.MustDecimalFromString("-25"),
			VegaTime:         now.Add(10 * time.Second),
			TxHash:           "f1e520d7612de709503d493a3335a4aa8e8b3125b5d5661b7bed7f509b67bf53",
		},
		{
			PartyID: entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0"),

			MarketID:         entities.MarketID("f1e520d7612de709503d493a3335a4aa8e8b3125b5d5661b7bed7f509b67bf53"),
			FundingPeriodSeq: 1,
			Amount:           num.MustDecimalFromString("2400"),
			VegaTime:         now.Add(10 * time.Second),
			TxHash:           "09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0",
		},
	}

	t.Run("can insert successfully", func(t *testing.T) {
		assert.NoError(t, store.fp.Add(ctx, fundingPayments))
	})

	t.Run("can get for a party, no market", func(t *testing.T) {
		pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
		partyID := entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0")
		// get only the first party,
		// and ensure the ordering is correct
		payments, _, err := store.fp.List(
			ctx,
			partyID,
			nil,
			pagination,
		)

		assert.NoError(t, err)
		assert.Len(t, payments, 4)

		// expected in this order
		// newest first
		amounts := []num.Decimal{
			num.MustDecimalFromString("2400"),
			num.MustDecimalFromString("25"),
			num.MustDecimalFromString("42"),
			num.MustDecimalFromString("-100"),
		}

		for i, p := range payments {
			assert.Equal(t, p.PartyID, partyID)
			assert.Equal(t, p.Amount, amounts[i])
		}
	})

	t.Run("can get for a party and a market", func(t *testing.T) {
		pagination, _ := entities.NewCursorPagination(nil, nil, nil, nil, true)
		partyID := entities.PartyID("09d82547b823da327af14727d02936db75c33cffe8e09341a9fc729fe53865e0")
		marketID := entities.MarketID("f1e520d7612de709503d493a3335a4aa8e8b3125b5d5661b7bed7f509b67bf53")
		// get only the first party,
		// and ensure the ordering is correct
		payments, _, err := store.fp.List(
			ctx,
			partyID,
			ptr.From(marketID),
			pagination,
		)

		assert.NoError(t, err)
		assert.Len(t, payments, 1)

		// expected in this order
		// newest first
		amounts := []num.Decimal{
			num.MustDecimalFromString("2400"),
		}

		for i, p := range payments {
			assert.Equal(t, p.PartyID, partyID)
			assert.Equal(t, p.MarketID, marketID)
			assert.Equal(t, p.Amount, amounts[i])
		}
	})
}
