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
	table *godog.Table,
) error {
	orders := broker.GetDistressedOrders()
	closed := broker.GetSettleDistressed()
	// get a status map by party
	partyStatus := map[string]proto.PositionStatus{}
	for _, o := range orders {
		for _, p := range p.Parties() {
			partyStatus[p] = proto.PositionStatus_POSITION_STATUS_ORDERS_CLOSED
		}
	}
	for _, c := range closed {
		partyStatus[c.PartyID()] = proto.PositionStatus_POSITION_STATUS_CLOSED_OUT
	}
	for _, r := range parsePositionStatusRow(table) {
		row := positionStatusRow{row: r}
		status, ok := partyStatus[row.Party()]
		if !ok {
			return fmt.Errorf("no status update found for party %s", row.Party())
		}
		if status != row.Status() {
			return fmt.Errorf("expected status %s for party %s, instead got %s", row.Status(), row.Party(), status)
		}
	}
	return nil
}

func parsePositionStatusRow(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"status",
	}, nil)
}

type positionStatusRow struct {
	row RowWrapper
}

func (p positionStatusRow) Party() string {
	return r.row.MustStr("party")
}

func (p positionStatusRow) Status() proto.PositionStatus {
	return r.row.MustPositionStatus("status")
}
