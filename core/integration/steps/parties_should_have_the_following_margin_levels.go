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
	"fmt"

	"github.com/cucumber/godog"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	types "code.vegaprotocol.io/vega/protos/vega"
)

func ThePartiesShouldHaveTheFollowingMarginLevels(
	broker *stubs.BrokerStub,
	table *godog.Table,
) error {
	for _, row := range parseExpectedMarginsTable(table) {
		partyID := row.MustStr("party")
		marketID := row.MustStr("market id")
		maintenance := row.MustU64("maintenance")
		search, hasSearch := row.U64B("search")
		initial, hasInitial := row.U64B("initial")
		release, hasRelease := row.U64B("release")

		levels, err := broker.GetMarginByPartyAndMarket(partyID, marketID)
		if err != nil {
			return errCannotGetMarginLevelsForPartyAndMarket(partyID, marketID, err)
		}

		var hasError bool
		if stringToU64(levels.MaintenanceMargin) != maintenance {
			hasError = true
		}
		if hasSearch && stringToU64(levels.SearchLevel) != search {
			hasError = true
		}
		if hasInitial && stringToU64(levels.InitialMargin) != initial {
			hasError = true
		}
		if hasRelease && stringToU64(levels.CollateralReleaseLevel) != release {
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
			"maintenance": levels.MaintenanceMargin,
			"search":      levels.SearchLevel,
			"initial":     levels.InitialMargin,
			"release":     levels.CollateralReleaseLevel,
		},
	)
}

func parseExpectedMarginsTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market id",
		"maintenance",
	}, []string{
		"search",
		"initial",
		"release",
	},
	)
}
