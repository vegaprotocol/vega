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

	types "code.vegaprotocol.io/vega/protos/vega"
)

func TheTradingModeShouldBeForMarket(
	engine Execution,
	market, tradingModeStr string,
) error {
	tradingMode, err := TradingMode(tradingModeStr)
	panicW("trading mode", err)

	marketData, err := engine.GetMarketData(market)
	if err != nil {
		return errMarketDataNotFound(market, err)
	}

	if marketData.MarketTradingMode != tradingMode {
		return errMismatchedTradingMode(market, tradingMode, marketData.MarketTradingMode)
	}
	return nil
}

func errMismatchedTradingMode(market string, expectedTradingMode, gotTradingMode types.Market_TradingMode) error {
	return formatDiff(
		fmt.Sprintf("unexpected market trading mode for market \"%s\"", market),
		map[string]string{
			"trading mode": expectedTradingMode.String(),
		},
		map[string]string{
			"trading mode": gotTradingMode.String(),
		},
	)
}
