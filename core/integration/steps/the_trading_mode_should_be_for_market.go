// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package steps

import (
	"fmt"

	types "code.vegaprotocol.io/protos/vega"
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
