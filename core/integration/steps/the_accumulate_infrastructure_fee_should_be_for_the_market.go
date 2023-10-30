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

func TheAccumulatedInfrastructureFeesShouldBeForTheMarket(
	broker *stubs.BrokerStub,
	amountStr, asset string,
) error {
	amount, err := U64(amountStr)
	if err != nil {
		return fmt.Errorf("invalid amount: %w", err)
	}

	acc, err := broker.GetMarketInfrastructureFeePoolAccount(asset)
	if err != nil {
		return err
	}

	if stringToU64(acc.Balance) != amount {
		return errInvalidAmountInInfraFee(asset, amount, stringToU64(acc.Balance))
	}

	return nil
}

func errInvalidAmountInInfraFee(asset string, expected, got uint64) error {
	return fmt.Errorf("invalid amount in infrastructure fee pool for asset %s want %d got %d", asset, expected, got)
}
