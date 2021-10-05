package steps

import (
	"errors"

	"github.com/cucumber/godog"

	types "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/integration/stubs"
)

func TheLiquidityProvisionsShouldHaveTheFollowingStates(broker *stubs.BrokerStub, table *godog.Table) error {
	evts := broker.GetLPEvents()
	evtByID := func(id string) *types.LiquidityProvision {
		found := &types.LiquidityProvision{}
		for _, e := range evts {
			if lp := e.LiquidityProvision(); lp.Id == id {
				found = lp
			}
		}
		return found
	}

	for _, row := range parseLiquidityProvisionStatesTable(table) {
		id := row.MustStr("id")
		party := row.MustStr("party")
		market := row.MustStr("market")
		commitment := row.MustU64("commitment amount")
		status := row.MustLiquidityStatus("status")

		e := evtByID(id)
		if e == nil {
			return errLiquidityProvisionEventNotFound()
		}

		if e.PartyId != party || e.MarketId != market || stringToU64(e.CommitmentAmount) != commitment || e.Status != status {
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
