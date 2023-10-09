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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func PartiesPlaceTheFollowingPeggedOrders(exec Execution, table *godog.Table) error {
	for _, r := range parseSubmitPeggedOrderTable(table) {
		row := submitPeggedOrderRow{row: r}

		orderSubmission := &types.OrderSubmission{
			Type:        types.OrderTypeLimit,
			TimeInForce: types.OrderTimeInForceGTC,
			Side:        row.Side(),
			MarketID:    row.MarketID(),
			Size:        row.Volume(),
			Reference:   row.Reference(),
			PeggedOrder: &types.PeggedOrder{
				Reference: row.PeggedReference(),
				Offset:    row.Offset(),
			},
		}
		_, err := exec.SubmitOrder(context.Background(), orderSubmission, row.Party())
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}
	}
	return nil
}

func parseSubmitPeggedOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"side",
		"volume",
		"pegged reference",
		"offset",
	}, []string{
		"error",
		"reference",
	})
}

type submitPeggedOrderRow struct {
	row RowWrapper
}

func (r submitPeggedOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r submitPeggedOrderRow) MarketID() string {
	return r.row.MustStr("market id")
}

func (r submitPeggedOrderRow) Side() types.Side {
	return r.row.MustSide("side")
}

func (r submitPeggedOrderRow) PeggedReference() types.PeggedReference {
	return r.row.MustPeggedReference("pegged reference")
}

func (r submitPeggedOrderRow) Volume() uint64 {
	return r.row.MustU64("volume")
}

func (r submitPeggedOrderRow) Offset() *num.Uint {
	return r.row.MustUint("offset")
}

func (r submitPeggedOrderRow) Error() string {
	return r.row.Str("error")
}

func (r submitPeggedOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r submitPeggedOrderRow) Reference() string {
	return r.row.Str("reference")
}
