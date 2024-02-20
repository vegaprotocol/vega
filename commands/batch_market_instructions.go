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

func CheckBatchMarketInstructions(cmd *commandspb.BatchMarketInstructions) error {
	return checkBatchMarketInstructions(cmd).ErrorOrNil()
}

func checkBatchMarketInstructions(cmd *commandspb.BatchMarketInstructions) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("batch_market_instructions", ErrIsRequired)
	}

	if len(cmd.UpdateMarginMode) > 0 {
		return errs.FinalAddForProperty("batch_market_instructions.update_margin_mode", ErrIsDisabled)
	}

	// there's very little to verify here, only if the batch is not empty
	// all transaction verification is done when processing then.
	if len(cmd.Cancellations)+
		len(cmd.Amendments)+
		len(cmd.Submissions)+
		len(cmd.StopOrdersSubmission)+
		len(cmd.StopOrdersCancellation)+
		len(cmd.UpdateMarginMode) == 0 {
		return errs.FinalAddForProperty("batch_market_instructions", ErrEmptyBatchMarketInstructions)
	}

	return errs
}
