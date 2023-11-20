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
	"fmt"

	"code.vegaprotocol.io/vega/libs/num"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckUpdateMarginMode(cmd *commandspb.UpdateMarginMode) error {
	return checkUpdateMarginMode(cmd).ErrorOrNil()
}

func checkUpdateMarginMode(cmd *commandspb.UpdateMarginMode) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("update_margin_mode", ErrIsRequired)
	}

	if cmd.Mode == commandspb.UpdateMarginMode_MODE_CROSS_UNSPECIFIED {
		errs.AddForProperty("update_margin_mode.margin_mode", ErrIsNotValid)
	}

	if cmd.Mode == commandspb.UpdateMarginMode_MODE_CROSS_MARGIN && cmd.MarginFactor != nil {
		errs.AddForProperty("update_margin_mode.margin_factor", fmt.Errorf("margin factor must not be defined when margin mode is cross margin"))
	}

	if cmd.Mode == commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN && (cmd.MarginFactor == nil || len(*cmd.MarginFactor) == 0) {
		errs.AddForProperty("update_margin_mode.margin_factor", fmt.Errorf("margin factor must be defined when margin mode is isolated margin"))
	}

	if cmd.Mode == commandspb.UpdateMarginMode_MODE_ISOLATED_MARGIN && cmd.MarginFactor != nil && len(*cmd.MarginFactor) > 0 {
		if factor, err := num.DecimalFromString(*cmd.MarginFactor); err != nil {
			errs.AddForProperty("update_margin_mode.margin_factor", ErrIsNotValidNumber)
		} else if factor.LessThanOrEqual(num.DecimalZero()) || factor.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty("update_margin_mode.margin_factor", ErrMustBeBetween01)
		}
	}
	if len(cmd.MarketId) == 0 {
		errs.AddForProperty("update_margin_mode.market_id", ErrIsRequired)
	}

	return errs
}
