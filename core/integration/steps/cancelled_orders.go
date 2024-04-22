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
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"

	"github.com/cucumber/godog"
)

func TheCancelledOrdersEventContains(broker *stubs.BrokerStub, market string, table *godog.Table) error {
	allCancelled := broker.GetCancelledOrdersPerMarket()
	cancelled, ok := allCancelled[market]
	if !ok {
		return fmt.Errorf("no cancelled orders event for market %s", market)
	}
	rows := parseReferenceTable(table)
	for _, r := range rows {
		rr := referenceRow{
			r: r,
		}
		o, err := broker.GetFirstByReference(rr.Party(), rr.Reference())
		if err != nil {
			return err
		}
		if o.MarketId != market {
			return fmt.Errorf("could not find order with reference %s for party %s and market %s", rr.Reference(), rr.Party(), market)
		}
		// now check if this ID was indeed emitted as part of a cancelled orders event.
		if _, ok := cancelled[o.Id]; !ok {
			return fmt.Errorf("order with reference %s for party %s and market %s (ID %s) missing from cancelled orders event", rr.Reference(), rr.Party(), market, o.Id)
		}
	}
	return nil
}

func parseReferenceTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"reference",
		"party",
	}, []string{})
}

type referenceRow struct {
	r RowWrapper
}

func (r referenceRow) Reference() string {
	return r.r.MustStr("reference")
}

func (r referenceRow) Party() string {
	return r.r.MustStr("party")
}
