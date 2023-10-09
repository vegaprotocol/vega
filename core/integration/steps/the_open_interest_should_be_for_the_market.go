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
)

func TheOpenInterestShouldBeForTheMarket(engine Execution, marketID string, wantOpenInterest string) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	if fmt.Sprintf("%d", marketData.OpenInterest) != wantOpenInterest {
		return errUnexpectedOpenInterest(marketData, wantOpenInterest)
	}

	return nil
}

func errUnexpectedOpenInterest(md types.MarketData, wantOpenInterest string) error {
	return fmt.Errorf("unexpected open interest for market %s got %d, want %s", md.Market, md.OpenInterest, wantOpenInterest)
}
