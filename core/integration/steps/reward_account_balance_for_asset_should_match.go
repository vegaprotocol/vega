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
)

func RewardAccountBalanceForAssetShouldMatch(
	broker *stubs.BrokerStub,
	accountType, asset, rawBalance string,
) error {
	balance, _ := strconv.ParseUint(rawBalance, 10, 0)
	acc, err := broker.GetRewardAccountBalance(accountType, asset)
	if err != nil {
		if balance == 0 {
			return nil
		}
		return err
	}

	if stringToU64(acc.Balance) != balance {
		return fmt.Errorf("invalid reward account balance for asset(%s) for account type(%s), expected(%d) got(%s)",
			asset, accountType, balance, acc.Balance,
		)
	}

	return nil
}
