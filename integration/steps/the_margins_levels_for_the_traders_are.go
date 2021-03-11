package steps

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"

	"github.com/cucumber/godog/gherkin"
)

func TheMarginsLevelsForTheTradersAre(
	broker interface {
		GetMarginByPartyAndMarket(string, string) (types.MarginLevels, error)
	},
	table *gherkin.DataTable,
) error {
	for _, row := range TableWrapper(*table).Parse() {
		partyID, marketID := row.Str("trader"), row.Str("market id")
		ml, err := broker.GetMarginByPartyAndMarket(partyID, marketID)
		if err != nil {
			return err
		}

		maintenance, err := row.U64("maintenance")
		panicW(err)
		search, err := row.U64("search")
		panicW(err)
		initial, err := row.U64("initial")
		panicW(err)
		release, err := row.U64("release")
		panicW(err)

		var hasError bool
		if ml.MaintenanceMargin != maintenance {
			hasError = true
		}
		if ml.SearchLevel != search {
			hasError = true
		}
		if ml.InitialMargin != initial {
			hasError = true
		}
		if ml.CollateralReleaseLevel != release {
			hasError = true
		}
		if hasError {
			return fmt.Errorf(
				"invalid margins, expected maintenance(%v), search(%v), initial(%v), release(%v) but got maintenance(%v), search(%v), initial(%v), release(%v) (trader=%v)",
				maintenance, search, initial, release, ml.MaintenanceMargin, ml.SearchLevel, ml.InitialMargin, ml.CollateralReleaseLevel, partyID)
		}
	}
	return nil
}
