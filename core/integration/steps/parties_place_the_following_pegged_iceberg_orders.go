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

func PartiesPlaceTheFollowingPeggedIcebergOrders(
	exec Execution,
	ts *stubs.TimeStub,
	table *godog.Table,
) error {
	now := ts.GetTimeNow()
	for _, r := range parseSubmitPeggedIcebergOrderTable(table) {
		row := submitPeggedIcebergOrderRow{row: r}

		typ := types.OrderTypeLimit
		if row.HasOrderType() {
			typ = row.OrderType()
		}
		tif := types.OrderTimeInForceGTC
		if row.HasTimeInForce() {
			tif = row.TimeInForce()
		}

		orderSubmission := types.OrderSubmission{
			MarketID:    row.MarketID(),
			Side:        row.Side(),
			Size:        row.Volume(),
			ExpiresAt:   row.ExpirationDate(now),
			Type:        typ,
			TimeInForce: tif,
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
		"peak size",
		"minimum visible size",
		"pegged reference",
		"offset",
	}, []string{
		"type",
		"tif",
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

func (r submitPeggedIcebergOrderRow) HasOrderType() bool {
	return r.row.HasColumn("type")
}

func (r submitPeggedIcebergOrderRow) HasTimeInForce() bool {
	return r.row.HasColumn("tif")
}

func (r submitPeggedIcebergOrderRow) OrderType() types.OrderType {
	return r.row.MustOrderType("type")
}

func (r submitPeggedIcebergOrderRow) TimeInForce() types.OrderTimeInForce {
	return r.row.MustTIF("tif")
}

func (r submitPeggedIcebergOrderRow) ExpirationDate(now time.Time) int64 {
	if r.HasOrderType() && r.OrderType() == types.OrderTypeMarket {
		return 0
	}

	if r.HasTimeInForce() && r.TimeInForce() == types.OrderTimeInForceGTT {
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
