package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TheMarginsLevelsForTheTradersAre(
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		partyID := row.Str("trader")
		marketID := row.Str("market id")
		maintenance := row.U64("maintenance")
		search := row.U64("search")
		initial := row.U64("initial")
		release := row.U64("release")

		levels, err := broker.GetMarginByPartyAndMarket(partyID, marketID)
		if err != nil {
			return errCannotGetMarginLevelsForPartyAndMarket(partyID, marketID, err)
		}

		var hasError bool
		if levels.MaintenanceMargin != maintenance {
			hasError = true
		}
		if levels.SearchLevel != search {
			hasError = true
		}
		if levels.InitialMargin != initial {
			hasError = true
		}
		if levels.CollateralReleaseLevel != release {
			hasError = true
		}
		if hasError {
			return errInvalidMargins(maintenance, search, initial, release, levels, partyID)
		}
	}
	return nil
}

func errCannotGetMarginLevelsForPartyAndMarket(partyID, market string, err error) error {
	return fmt.Errorf("couldn't get margin levels for party(%s) and market(%s): %s", partyID, market, err.Error())
}

func errInvalidMargins(
	maintenance, search, initial, release uint64,
	levels types.MarginLevels,
	partyID string,
) error {
	return fmt.Errorf(
		"invalid margins, expected maintenance(%v), search(%v), initial(%v), release(%v) but got maintenance(%v), search(%v), initial(%v), release(%v) (trader=%v)",
		maintenance, search, initial, release, levels.MaintenanceMargin, levels.SearchLevel, levels.InitialMargin, levels.CollateralReleaseLevel, partyID)
}
