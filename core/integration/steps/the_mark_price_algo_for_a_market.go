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

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func TheMarkPriceAlgoShouldBeForMarket(
	broker *stubs.BrokerStub,
	market, expectedMarkPriceAlgo string,
) error {
	actualMarkPriceAlgo := broker.GetMarkPriceSettings(market).CompositePriceType.String()

	if actualMarkPriceAlgo != expectedMarkPriceAlgo {
		return errMismatchedMarkPriceAlgo(market, expectedMarkPriceAlgo, actualMarkPriceAlgo)
	}
	return nil
}

func errMismatchedMarkPriceAlgo(market, expectedAlgo, actualAlgo string) error {
	return formatDiff(
		fmt.Sprintf("unexpected mark price algo for market \"%s\"", market),
		map[string]string{
			"mark price algo": expectedAlgo,
		},
		map[string]string{
			"mark price algo": actualAlgo,
		},
	)
}
