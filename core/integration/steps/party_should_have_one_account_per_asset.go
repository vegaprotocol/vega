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
	types "code.vegaprotocol.io/vega/protos/vega"
)

func PartyShouldHaveOneAccountPerAsset(
	broker *stubs.BrokerStub,
	owner string,
) error {
	assets := map[string]struct{}{}

	accounts := broker.GetAccounts()

	for _, acc := range accounts {
		if acc.Owner == owner && acc.Type == types.AccountType_ACCOUNT_TYPE_GENERAL {
			if _, ok := assets[acc.Asset]; ok {
				return errMultipleGeneralAccountForAsset(owner, acc)
			}
			assets[acc.Asset] = struct{}{}
		}
	}
	return nil
}

func errMultipleGeneralAccountForAsset(owner string, acc types.Account) error {
	return fmt.Errorf("party=%v have multiple account for asset=%v", owner, acc.Asset)
}
