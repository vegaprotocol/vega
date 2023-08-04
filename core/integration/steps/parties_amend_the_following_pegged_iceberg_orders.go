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
	"time"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func PartiesAmendTheFollowingPeggedIcebergOrders(
	broker *stubs.BrokerStub,
	exec Execution,
	ts *stubs.TimeStub,
	table *godog.Table,
) error {
	now := ts.GetTimeNow()
	for _, r := range parseAmendPeggedIcebergOrderTable(table) {
		row := amendPeggedIcebergOrderRow{row: r}

		o, err := broker.GetByReference(row.Party(), row.Reference())
		if err != nil {
			return errOrderNotFound(row.Reference(), row.Party(), err)
		}

		tif := o.TimeInForce
		if row.HasTimeInForce() {
			tif = row.TimeInForce()
		}
		var offset *num.Uint
		if row.HasOffset() {
			offset = row.Offset()
		}
		var pegRef types.PeggedReference
		if row.HasPeggedReference() {
			pegRef = row.PeggedReference()
		}

		amend := types.OrderAmendment{
			OrderID:         o.Id,
			MarketID:        o.MarketId,
			SizeDelta:       row.SizeDelta(),
			TimeInForce:     tif,
			ExpiresAt:       row.ExpirationDate(now),
			PeggedOffset:    offset,
			PeggedReference: pegRef,
		}

		_, err = exec.AmendOrder(context.Background(), &amend, o.PartyId)
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}

	}
	return nil
}

func parseAmendPeggedIcebergOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"reference",
		"size delta",
	}, []string{
		"pegged reference",
		"offset",
		"tif",
		"error",
		"expires in",
	})
}

type amendPeggedIcebergOrderRow struct {
	row RowWrapper
}

func (r amendPeggedIcebergOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r amendPeggedIcebergOrderRow) SizeDelta() int64 {
	return r.row.MustI64("size delta")
}

func (r amendPeggedIcebergOrderRow) HasTimeInForce() bool {
	return r.row.HasColumn("tif")
}

func (r amendPeggedIcebergOrderRow) TimeInForce() types.OrderTimeInForce {
	return r.row.MustTIF("tif")
}

func (r amendPeggedIcebergOrderRow) ExpirationDate(now time.Time) *int64 {
	if r.HasTimeInForce() && r.TimeInForce() == types.OrderTimeInForceGTT {
		ed := now.Add(r.row.MustDurationSec("expires in")).Local().UnixNano()
		return &ed
	}
	// non GTT orders don't need an expiry time
	return nil
}

func (r amendPeggedIcebergOrderRow) Reference() string {
	return r.row.Str("reference")
}

func (r amendPeggedIcebergOrderRow) Error() string {
	return r.row.Str("error")
}

func (r amendPeggedIcebergOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r amendPeggedIcebergOrderRow) HasPeggedReference() bool {
	return r.row.HasColumn("pegged reference")
}

func (r amendPeggedIcebergOrderRow) PeggedReference() types.PeggedReference {
	return r.row.MustPeggedReference("pegged reference")
}

func (r amendPeggedIcebergOrderRow) HasOffset() bool {
	return r.row.HasColumn("offset")
}

func (r amendPeggedIcebergOrderRow) Offset() *num.Uint {
	return r.row.Uint("offset")
}
