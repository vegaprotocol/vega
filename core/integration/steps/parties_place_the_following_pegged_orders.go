// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cucumber/godog"
)

func PartiesPlaceTheFollowingPeggedOrders(exec Execution, table *godog.Table) error {
	for _, r := range parseSubmitPeggedOrderTable(table) {
		row := submitPeggedOrderRow{row: r}

		orderSubmission := &types.OrderSubmission{
			Type:        types.OrderTypeLimit,
			TimeInForce: types.OrderTimeInForceGTC,
			Side:        row.Side(),
			MarketId:    row.MarketID(),
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
