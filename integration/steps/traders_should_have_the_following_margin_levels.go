package steps

import (
	"fmt"

	"github.com/cucumber/godog/gherkin"

	"code.vegaprotocol.io/vega/integration/stubs"
	types "code.vegaprotocol.io/vega/proto"
)

func TheTradersShouldHaveTheFollowingMarginLevels(
	broker *stubs.BrokerStub,
	table *gherkin.DataTable,
) error {
	for _, row := range parseExpectedMarginsTable(table) {
		partyID := row.MustStr("trader")
		marketID := row.MustStr("market id")
		maintenance := row.MustU64("maintenance")
		search := row.MustU64("search")
		initial := row.MustU64("initial")
		release := row.MustU64("release")

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
	return formatDiff(fmt.Sprintf("invalid margins for party \"%s\"", partyID),
		map[string]string{
			"maintenance": u64ToS(maintenance),
			"search":      u64ToS(search),
			"initial":     u64ToS(initial),
			"release":     u64ToS(release),
		},
		map[string]string{
			"maintenance": u64ToS(levels.MaintenanceMargin),
			"search":      u64ToS(levels.SearchLevel),
			"initial":     u64ToS(levels.InitialMargin),
			"release":     u64ToS(levels.CollateralReleaseLevel),
		},
	)
}

func parseExpectedMarginsTable(table *gherkin.DataTable) []RowWrapper {
	return TableWrapper(*table).StrictParse([]string{
		"trader",
		"market id",
		"maintenance",
		"search",
		"initial",
		"release",
	}, []string{})
}
