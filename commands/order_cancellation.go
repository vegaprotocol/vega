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

package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckOrderCancellation(cmd *commandspb.OrderCancellation) error {
	return checkOrderCancellation(cmd).ErrorOrNil()
}

func checkOrderCancellation(cmd *commandspb.OrderCancellation) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("order_cancellation", ErrIsRequired)
	}

	if len(cmd.MarketId) > 0 && !IsVegaID(cmd.MarketId) {
		errs.AddForProperty("order_cancellation.market_id", ErrShouldBeAValidVegaID)
	}

	if len(cmd.OrderId) > 0 && !IsVegaID(cmd.OrderId) {
		errs.AddForProperty("order_cancellation.order_id", ErrShouldBeAValidVegaID)
	}

	if cmd.VaultId != nil && !IsVegaID(*cmd.VaultId) {
		errs.AddForProperty("order_cancellation.vault_id", ErrInvalidVaultID)
	}
	return errs
}
