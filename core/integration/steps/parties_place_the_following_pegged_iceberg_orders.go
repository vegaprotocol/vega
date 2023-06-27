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

func PartiesPlaceTheFollowingPeggedIcebergOrders(
	exec Execution,
	ts *stubs.TimeStub,
	table *godog.Table,
) error {
	now := ts.GetTimeNow()
	for _, r := range parseSubmitPeggedIcebergOrderTable(table) {
		row := submitPeggedIcebergOrderRow{row: r}

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        row.OrderType(),
			TimeInForce: row.TimeInForce(),
			Reference:   row.Reference(),
			IcebergOrder: &types.IcebergOrder{
				PeakSize:           row.InitialPeak(),
				MinimumVisibleSize: row.MinimumPeak(),
			},
			PeggedOrder: &types.PeggedOrder{
				Reference: row.PeggedReference(),
				Offset:    row.Offset(),
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

func parseSubmitPeggedIcebergOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"type",
		"tif",
		"peak size",
		"minimum visible size",
		"pegged reference",
		"offset",
	}, []string{
		"reference",
		"error",
		"resulting trades",
		"expires in",
		"only",
	})
}

type submitPeggedIcebergOrderRow struct {
	row RowWrapper
}

func (r submitPeggedIcebergOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r submitPeggedIcebergOrderRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r submitPeggedIcebergOrderRow) Side() types.Side {
	return r.row.MustSide("side")
}

func (r submitPeggedIcebergOrderRow) Volume() uint64 {
	return r.row.MustU64("volume")
}

func (r submitPeggedIcebergOrderRow) OrderType() types.OrderType {
	return r.row.MustOrderType("type")
}

func (r submitPeggedIcebergOrderRow) TimeInForce() types.OrderTimeInForce {
	return r.row.MustTIF("tif")
}

func (r submitPeggedIcebergOrderRow) ExpirationDate(now time.Time) int64 {
	if r.OrderType() == types.OrderTypeMarket {
		return 0
	}

	if r.TimeInForce() == types.OrderTimeInForceGTT {
		return now.Add(r.row.MustDurationSec("expires in")).Local().UnixNano()
	}
	// non GTT orders don't need an expiry time
	return 0
}

func (r submitPeggedIcebergOrderRow) ExpectResultingTrades() bool {
	return r.row.HasColumn("resulting trades")
}

func (r submitPeggedIcebergOrderRow) ResultingTrades() int64 {
	return r.row.I64("resulting trades")
}

func (r submitPeggedIcebergOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r submitPeggedIcebergOrderRow) Error() string {
	return r.row.Str("error")
}

func (r submitPeggedIcebergOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r submitPeggedIcebergOrderRow) Only() Only {
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

func (r submitPeggedIcebergOrderRow) MinimumPeak() uint64 {
	return r.row.MustU64("minimum visible size")
}

func (r submitPeggedIcebergOrderRow) InitialPeak() uint64 {
	return r.row.MustU64("peak size")
}

func (r submitPeggedIcebergOrderRow) PeggedReference() types.PeggedReference {
	return r.row.MustPeggedReference("pegged reference")
}

func (r submitPeggedIcebergOrderRow) Offset() *num.Uint {
	return r.row.Uint("offset")
}
