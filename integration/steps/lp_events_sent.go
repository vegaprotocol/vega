package steps

import (
	"errors"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
	"github.com/cucumber/godog/gherkin"
)

func LiquidityProvisionEventsSent(broker *stubs.BrokerStub, table *gherkin.DataTable) error {
	evts := broker.GetLPEvents()
	evtByID := func(id string) *types.LiquidityProvision {
		for _, e := range evts {
			if lp := e.LiquidityProvision(); lp.Id == id {
				return &lp
			}
		}
		return nil
	}

	for _, row := range TableWrapper(*table).Parse() {
		id := row.Str("id")
		party := row.Str("party")
		market := row.Str("market")
		commitment := row.U64("commitment amount")

		if id == "id" {
			continue
		}

		// find event
		e := evtByID(id)
		if e == nil {
			return errLiquidityProvisionEventNotFound()
		}

		if e.PartyId != party || e.MarketId != market || e.CommitmentAmount != commitment {
			return errLiquidityProvisionEventNotFound()
		}
	}
	return nil
}

func errLiquidityProvisionEventNotFound() error {
	return errors.New("liquidity provision event not found")
}
