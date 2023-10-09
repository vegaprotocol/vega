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

package steps

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func PartiesPlaceTheFollowingIcebergOrders(
	exec Execution,
	ts *stubs.TimeStub,
	table *godog.Table,
) error {
	now := ts.GetTimeNow()
	for _, r := range parseSubmitIcebergOrderTable(table) {
		row := submitIcebergOrderRow{row: r}

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Price:       row.Price(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
			IcebergOrder: &types.IcebergOrder{
				PeakSize:           row.InitialPeak(),
				MinimumVisibleSize: row.MinimumPeak(),
			},
		}
		only := row.Only()
		switch only {
		case Post:
			orderSubmission.PostOnly = true
		case Reduce:
			orderSubmission.ReduceOnly = true
		}

		resp, err := exec.SubmitOrder(context.Background(), &orderSubmission, row.Party())
		if ceerr := checkExpectedError(row, err, nil); ceerr != nil {
			return ceerr
		}

		if !row.ExpectResultingTrades() || err != nil {
			continue
		}

		actualTradeCount := int64(len(resp.Trades))
		if actualTradeCount != row.ResultingTrades() {
			return formatDiff(fmt.Sprintf("the resulting trades didn't match the expectation for order \"%v\"", row.Reference()),
				map[string]string{
					"total": i64ToS(row.ResultingTrades()),
				},
				map[string]string{
					"total": i64ToS(actualTradeCount),
				},
			)
		}
	}
	return nil
}

func PartyAddsTheFollowingIcebergOrdersToABatch(party string, exec Execution, time *stubs.TimeStub, table *godog.Table) error {
	// ensure time is set + idgen is not nil
	now := time.GetTimeNow()
	time.SetTime(now)
	for _, r := range parseAddIcebergOrderToBatchTable(table) {
		row := submitIcebergOrderRow{r}

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Price:       row.Price(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
			IcebergOrder: &types.IcebergOrder{
				PeakSize:           row.InitialPeak(),
				MinimumVisibleSize: row.MinimumPeak(),
			},
		}
		only := row.Only()
		switch only {
		case Post:
			orderSubmission.PostOnly = true
		case Reduce:
			orderSubmission.ReduceOnly = true
		}

		if err := exec.AddSubmitOrderToBatch(&orderSubmission, party); err != nil {
			return err
		}
	}
	return nil
}

func parseSubmitIcebergOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"price",
		"type",
		"tif",
		"peak size",
		"minimum visible size",
	}, []string{
		"reference",
		"error",
		"resulting trades",
		"expires in",
		"only",
	})
}

func parseAddIcebergOrderToBatchTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"market id",
		"side",
		"volume",
		"price",
		"type",
		"tif",
		"peak size",
		"minimum visible size",
	}, []string{
		"reference",
		"error",
		"expires in",
		"only",
	})
}

type submitIcebergOrderRow struct {
	row RowWrapper
}

func (r submitIcebergOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r submitIcebergOrderRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r submitIcebergOrderRow) Side() types.Side {
	return r.row.MustSide("side")
}

func (r submitIcebergOrderRow) Volume() uint64 {
	return r.row.MustU64("volume")
}

func (r submitIcebergOrderRow) Price() *num.Uint {
	return r.row.MustUint("price")
}

func (r submitIcebergOrderRow) OrderType() types.OrderType {
	return r.row.MustOrderType("type")
}

func (r submitIcebergOrderRow) TimeInForce() types.OrderTimeInForce {
	return r.row.MustTIF("tif")
}

func (r submitIcebergOrderRow) ExpirationDate(now time.Time) int64 {
	if r.OrderType() == types.OrderTypeMarket {
		return 0
	}

	if r.TimeInForce() == types.OrderTimeInForceGTT {
		return now.Add(r.row.MustDurationSec("expires in")).Local().UnixNano()
	}
	// non GTT orders don't need an expiry time
	return 0
}

func (r submitIcebergOrderRow) ExpectResultingTrades() bool {
	return r.row.HasColumn("resulting trades")
}

func (r submitIcebergOrderRow) ResultingTrades() int64 {
	return r.row.I64("resulting trades")
}

func (r submitIcebergOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r submitIcebergOrderRow) Error() string {
	return r.row.Str("error")
}

func (r submitIcebergOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r submitIcebergOrderRow) Only() Only {
	if !r.row.HasColumn("only") {
		return None
	}
	v := r.row.MustStr("only")
	t, ok := onlyTypes[v]
	if !ok {
		panic(fmt.Errorf("unsupported type %v", v))
	}
	return t
}

func (r submitIcebergOrderRow) MinimumPeak() uint64 {
	return r.row.MustU64("minimum visible size")
}

func (r submitIcebergOrderRow) InitialPeak() uint64 {
	return r.row.MustU64("peak size")
}
