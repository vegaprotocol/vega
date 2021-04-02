package steps

import (
	"errors"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func LiquidityProvisionEventsSent(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	evts := broker.GetLPEvents()
	evtByID := func(id string) *types.LiquidityProvision {
		found := &types.LiquidityProvision{}
		for _, e := range evts {
			if lp := e.LiquidityProvision(); lp.Id == id {
				found = &lp
			}
		}
		return found
	}

	for _, row := range TableWrapper(*table).Parse() {
		id := row.MustStr("id")
		party := row.MustStr("party")
		market := row.MustStr("market")
		commitment := row.MustU64("commitment amount")
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

func errLiquidityProvisionEventNotFound() error {
	return errors.New("liquidity provision event not found")
}
