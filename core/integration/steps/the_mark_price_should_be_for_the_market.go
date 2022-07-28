// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/types/num"
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
