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
	"errors"
	"strconv"
	"strings"

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

		buyShape := row.Str("buy shape")
		sellShape := row.Str("sell shape")

		e := evtByID(id)
		if e == nil {
			return errLiquidityProvisionEventNotFound()
		}

		if e.PartyId != party || e.MarketId != market || e.CommitmentAmount != commitment || e.Status != status {
			return errLiquidityProvisionEventNotFound()
		}

		bs, err := strconv.Atoi(buyShape)
		if len(strings.TrimSpace(buyShape)) > 0 && (err != nil || bs != len(e.Buys)) {
			return errLiquidityProvisionEventNotFound()
		}
		ss, err := strconv.Atoi(sellShape)
		if len(strings.TrimSpace(sellShape)) > 0 && (err != nil || ss != len(e.Sells)) {
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
	}, []string{
		"buy shape",
		"sell shape",
	})
}

func errLiquidityProvisionEventNotFound() error {
	return errors.New("liquidity provision event not found")
}
