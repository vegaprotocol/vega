// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
				InitialPeakSize: row.InitialPeak(),
				MinimumPeakSize: row.MinimumPeak(),
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

func parseSubmitIcebergOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"price",
		"type",
		"tif",
		"initial peak",
		"minimum peak",
	}, []string{
		"reference",
		"error",
		"resulting trades",
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
	return r.row.MustU64("minimum peak")
}

func (r submitIcebergOrderRow) InitialPeak() uint64 {
	return r.row.MustU64("initial peak")
}
