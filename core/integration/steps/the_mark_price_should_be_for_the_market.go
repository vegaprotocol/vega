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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
)

func TheMarkPriceForTheMarketIs(
	exec Execution,
	market, markPriceStr string,
) error {
	markPrice := parseMarkPrice(markPriceStr)

	marketData, err := exec.GetMarketData(market)
	if err != nil {
		return errMarkPriceNotFound(market, err)
	}

	if marketData.MarkPrice.NEQ(markPrice) {
		return errWrongMarkPrice(market, markPrice, marketData)
	}

	return nil
}

func parseMarkPrice(markPriceStr string) *num.Uint {
	markPrice, err := U64(markPriceStr)
	panicW("mark price", err)
	return num.NewUint(markPrice)
}

func errWrongMarkPrice(market string, markPrice *num.Uint, marketData types.MarketData) error {
	return fmt.Errorf("wrong mark price for market(%v), expected(%v) got(%v)",
		market, markPrice, marketData.MarkPrice,
	)
}

func errMarkPriceNotFound(market string, err error) error {
	return fmt.Errorf("unable to get mark price for market(%v), err(%v)", market, err)
}
