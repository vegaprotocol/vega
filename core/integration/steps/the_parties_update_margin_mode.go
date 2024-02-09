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
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/cucumber/godog"
)

func ThePartiesUpdateMarginMode(
	execution Execution,
	table *godog.Table,
) error {
	for _, r := range parseUpdateMarginModeTable(table) {
		party := r.MustStr("party")
		market := r.MustStr("market")
		var marginMode types.MarginMode
		if r.MustStr("margin_mode") == "cross margin" {
			marginMode = types.MarginModeCrossMargin
		} else if r.MustStr("margin_mode") == "isolated margin" {
			marginMode = types.MarginModeIsolatedMargin
		} else {
			panic(fmt.Errorf("invalid margin mode"))
		}
		factor := num.DecimalZero()
		if r.HasColumn("margin_factor") && marginMode == types.MarginModeIsolatedMargin {
			factor = num.MustDecimalFromString(r.MustStr("margin_factor"))
		}
		err := execution.UpdateMarginMode(context.Background(), party, market, marginMode, factor)
		if r.HasColumn("error") && len(r.Str("error")) > 0 && (err == nil || err != nil && r.Str("error") != err.Error()) {
			gotError := ""
			if err != nil {
				gotError = err.Error()
			}
			return fmt.Errorf("invalid error expected %v got %v", r.Str("error"), gotError)
		}
	}

	return nil
}

func parseUpdateMarginModeTable(table *godog.Table) []RowWrapper {
	return StrictParseTable(table, []string{
		"party",
		"market",
		"margin_mode",
	}, []string{
		"margin_factor",
		"error",
	})
}
