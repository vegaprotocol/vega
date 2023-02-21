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
	"fmt"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	proto "code.vegaprotocol.io/vega/protos/vega"
)

func PartiesShouldHaveTheFollowingPositionStatus(
	broker *stubs.BrokerStub,
	market string,
	table *godog.Table,
) error {
	return positionEvents(broker, table, market, false)
}

func PartiesShouldHaveTheFollowingPositionStatusAgg(
	broker *stubs.BrokerStub,
	market string,
	table *godog.Table,
) error {
	return positionEvents(broker, table, market, true)
}

func positionEvents(broker *stubs.BrokerStub, table *godog.Table, market string, aggregate bool) error {
	orders := broker.GetDistressedOrders()
	closed := broker.GetSettleDistressed()
	// get a status map by party
	partyStatus := map[string]proto.PositionStatus{}
	for _, o := range orders {
		if o.MarketID() != market {
			continue
		}
		for _, p := range p.Parties() {
			partyStatus[p] = proto.PositionStatus_POSITION_STATUS_ORDERS_CLOSED
		}
	}
	for _, c := range closed {
		if c.MarketID() != market {
			continue
		}
		partyStatus[c.PartyID()] = proto.PositionStatus_POSITION_STATUS_CLOSED_OUT
	}
	losSoc := broker.GetLossSocializationEvents()
	partyLSA := map[string]int64{}
	for _, le := range lossSoc {
		if le.MarketID() != market {
			continue
		}
		amt := le.AmountLost()
		party := le.PartyID()
		// aggregate if required
		if aggregate {
			amt += partyLSA[party]
		}
		partyLSA[party] = amt
	}
	for _, r := range parsePositionStatusRow(table) {
		row := positionStatusRow{row: r}
		party := row.Party()
		if row.HasStatus() {
			exp := row.Status()
			// status is not unspecified (ie we expect to see an event with this information
			if exp == proto.PositionStatus_POSITION_STATUS_UNSPECIFIED {
				continue
			}
			status, ok := partyStatus[party]
			if !ok {
				return fmt.Errorf("no status update found for party %s", party)
			}
			if status != row.Status() {
				return fmt.Errorf("expected status %s for party %s, instead got %s", row.Status(), party, status)
			}
		}
		if row.HasAmount() {
			exp := row.Amount()
			got, ok := partyLSA[party]
			if !ok && exp != 0 {
				return fmt.Errorf("expected a loss socialisation amount of %d, none found", exp)
			}
			if got != exp {
				return fmt.Errorf("expected a loss socialisation amount of %d, instead saw %d", exp, got)
			}
		}
	}
	return nil
}

func parsePositionStatusRow(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
	}, []string{
		"status",
		"amount",
	})
}

type positionStatusRow struct {
	row RowWrapper
}

func (p positionStatusRow) HasStatus() bool {
	return r.row.HasColumn("status")
}

func (p positionStatusRow) HasAmount() bool {
	return r.row.HasColumn("amount")
}

func (p positionStatusRow) Party() string {
	return r.row.MustStr("party")
}

func (p positionStatusRow) Status() proto.PositionStatus {
	return r.row.MustPositionStatus("status")
}

func (p positionStatusRow) Amount() int64 {
	r.row.MustI64("amount")
}
