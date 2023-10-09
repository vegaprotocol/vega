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

func TheSuppliedStakeShouldBeForTheMarket(engine Execution, marketID string, wantSuppliedStake string) error {
	marketData, err := engine.GetMarketData(marketID)
	if err != nil {
		return errMarketDataNotFound(marketID, err)
	}

	if marketData.SuppliedStake != wantSuppliedStake {
		return errUnexpectedSuppliedStake(marketData, wantSuppliedStake)
	}

	return nil
}

func errUnexpectedSuppliedStake(md types.MarketData, wantSuppliedStake string) error {
	return fmt.Errorf("unexpected supplied stake for market %s got %s, want %s", md.Market, md.SuppliedStake, wantSuppliedStake)
}
