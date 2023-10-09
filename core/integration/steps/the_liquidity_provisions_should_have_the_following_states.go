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
	"errors"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	types "code.vegaprotocol.io/vega/protos/vega"
)

func TheLiquidityProvisionsShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *godog.Table) error {
	evts := broker.GetLPEvents()
	evtByID := func(id string) *types.LiquidityProvision {
		found := &types.LiquidityProvision{}
		for _, e := range evts {
			if lp := e.LiquidityProvision(); lp.Reference == id {
				found = lp
			}
		}
		return found
	}

	for _, row := range parseLiquidityProvisionStatesTable(table) {
		id := row.MustStr("id")
		party := row.MustStr("party")
		market := row.MustStr("market")
		commitment := row.MustStr("commitment amount")
		status := row.MustLiquidityStatus("status")

		e := evtByID(id)
		if e == nil {
			return errLiquidityProvisionEventNotFound()
		}

		if e.PartyId != party || e.MarketId != market || e.CommitmentAmount != commitment || e.Status != status {
			return errLiquidityProvisionEventNotFound()
		}
	}
	return nil
}

func parseLiquidityProvisionStatesTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"id",
		"party",
		"market",
		"commitment amount",
		"status",
	}, []string{})
}

func errLiquidityProvisionEventNotFound() error {
	return errors.New("liquidity provision event not found")
}
