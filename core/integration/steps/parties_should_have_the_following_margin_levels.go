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
	"strconv"

	"code.vegaprotocol.io/vega/core/integration/stubs"
	types "code.vegaprotocol.io/vega/protos/vega"

	"github.com/cucumber/godog"
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
		order, hasOrder := row.U64B("order")
		marginMode := row.Str("margin mode")
		marginFactor := row.Str("margin factor")

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
		if hasOrder && stringToU64(levels.OrderMargin) != order {
			hasError = true
		}
		if row.HasColumn("margin mode") {
			if marginMode == "cross margin" && levels.MarginMode != types.MarginMode_MARGIN_MODE_CROSS_MARGIN {
				hasError = true
			} else if marginMode == "isolated margin" && levels.MarginMode != types.MarginMode_MARGIN_MODE_ISOLATED_MARGIN {
				hasError = true
			} else if marginMode != "cross margin" && marginMode != "isolated margin" {
				hasError = true
			}
		}
		if row.HasColumn("margin factor") {
			expected, err := strconv.ParseFloat(marginFactor, 64)
			if err != nil {
				return fmt.Errorf("can't parse the expected margin factor ('%s') into float: %v", marginFactor, err)
			}
			actual, err := strconv.ParseFloat(levels.MarginFactor, 64)
			if err != nil {
				return fmt.Errorf("can't parse the actual margin factor ('%s') into float: %v", levels.MarginFactor, err)
			}
			if actual != expected {
				hasError = true
			}
		}
		if hasError {
			return errInvalidMargins(maintenance, search, initial, release, order, levels, partyID, marginMode, marginFactor)
		}
	}
	return nil
}

func errCannotGetMarginLevelsForPartyAndMarket(partyID, market string, err error) error {
	return fmt.Errorf("couldn't get margin levels for party(%s) and market(%s): %s", partyID, market, err.Error())
}

func errInvalidMargins(
	maintenance, search, initial, release, order uint64,
	levels types.MarginLevels,
	partyID string,
	marginMode string,
	marginFactor string,
) error {
	return formatDiff(fmt.Sprintf("invalid margins for party \"%s\"", partyID),
		map[string]string{
			"maintenance":   u64ToS(maintenance),
			"search":        u64ToS(search),
			"initial":       u64ToS(initial),
			"release":       u64ToS(release),
			"order":         u64ToS(order),
			"margin mode":   marginMode,
			"margin factor": marginFactor,
		},
		map[string]string{
			"maintenance":   levels.MaintenanceMargin,
			"search":        levels.SearchLevel,
			"initial":       levels.InitialMargin,
			"release":       levels.CollateralReleaseLevel,
			"order":         levels.OrderMargin,
			"margin mode":   levels.MarginMode.String(),
			"margin factor": levels.MarginFactor,
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
		"order",
		"margin mode",
		"margin factor",
	},
	)
}
