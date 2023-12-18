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

	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

type OrderAmendmentError struct {
	OrderAmendment types.OrderAmendment
	OrderReference string
	Err            error
}

func (o OrderAmendmentError) Error() string {
	return fmt.Sprintf("%v: %v", o.OrderAmendment, o.Err)
}

func PartiesAmendTheFollowingOrders(
	broker *stubs.BrokerStub,
	exec Execution,
	table *godog.Table,
) error {
	for _, r := range parseAmendOrderTable(table) {
		row := amendOrderRow{row: r}

		o, err := broker.GetByReference(row.Party(), row.Reference())
		if err != nil {
			return errOrderNotFound(row.Reference(), row.Party(), err)
		}

		amend := types.OrderAmendment{
			OrderID:      o.Id,
			MarketID:     o.MarketId,
			Price:        row.Price(),
			SizeDelta:    row.SizeDelta(),
			Size:         row.Size(),
			ExpiresAt:    row.ExpirationDate(),
			TimeInForce:  row.TimeInForce(),
			PeggedOffset: row.PeggedOffset(),
		}

		_, err = exec.AmendOrder(context.Background(), &amend, o.PartyId)
		if err := checkExpectedError(row, err, nil); err != nil {
			return err
		}
	}

	return nil
}

type amendOrderRow struct {
	row RowWrapper
}

func parseAmendOrderTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"reference",
		"tif",
	}, []string{
		"price",
		"size delta",
		"size",
		"error",
		"expiration date",
		"pegged offset",
	})
}

func (r amendOrderRow) Party() string {
	return r.row.MustStr("party")
}

func (r amendOrderRow) Reference() string {
	return r.row.MustStr("reference")
}

func (r amendOrderRow) Price() *num.Uint {
	return r.row.MaybeUint("price")
}

func (r amendOrderRow) SizeDelta() int64 {
	if !r.row.HasColumn("size delta") {
		return 0
	}
	return r.row.MustI64("size delta")
}

func (r amendOrderRow) Size() *uint64 {
	return r.row.MaybeU64("size")
}

func (r amendOrderRow) TimeInForce() types.OrderTimeInForce {
	return r.row.MustTIF("tif")
}

func (r amendOrderRow) ExpirationDate() *int64 {
	if !r.row.HasColumn("expiration date") {
		return nil
	}

	timeNano := r.row.MustTime("expiration date").UnixNano()
	if timeNano == 0 {
		return nil
	}

	return &timeNano
}

func (r amendOrderRow) Error() string {
	return r.row.Str("error")
}

func (r amendOrderRow) ExpectError() bool {
	return r.row.HasColumn("error")
}

func (r amendOrderRow) PeggedOffset() *num.Uint {
	return r.row.MaybeUint("pegged offset")
}
