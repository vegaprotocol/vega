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
	"errors"
	"math/big"

	"code.vegaprotocol.io/vega/libs/crypto"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckWithdrawSubmission(cmd *commandspb.WithdrawSubmission) error {
	return checkWithdrawSubmission(cmd).ErrorOrNil()
}

func checkWithdrawSubmission(cmd *commandspb.WithdrawSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("withdraw_submission", ErrIsRequired)
	}

	if len(cmd.Amount) <= 0 {
		errs.AddForProperty("withdraw_submission.amount", ErrIsRequired)
	} else {
		if amount, ok := big.NewInt(0).SetString(cmd.Amount, 10); !ok {
			errs.AddForProperty("withdraw_submission.amount", ErrNotAValidInteger)
		} else if amount.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("withdraw_submission.amount", ErrIsRequired)
		}
	}

	if len(cmd.Asset) <= 0 {
		errs.AddForProperty("withdraw_submission.asset", ErrIsRequired)
	} else if !IsVegaID(cmd.Asset) {
		errs.AddForProperty("withdraw_submission.asset", ErrShouldBeAValidVegaID)
	}

	if cmd.Ext != nil {
		errs.Merge(checkWithdrawExt(cmd.Ext))
	}

	return errs
}

func checkWithdrawExt(wext *types.WithdrawExt) Errors {
	errs := NewErrors()
	switch v := wext.Ext.(type) {
	case *types.WithdrawExt_Erc20:
		if len(v.Erc20.ReceiverAddress) <= 0 {
			errs.AddForProperty(
				"withdraw_ext.erc20.received_address",
				ErrIsRequired,
			)
		} else if !crypto.EthereumIsValidAddress(v.Erc20.ReceiverAddress) {
			errs.AddForProperty("withdraw_ext.erc20.received_address", ErrIsNotValidEthereumAddress)
		}
	default:
		errs.AddForProperty("withdraw_ext.ext", errors.New("unsupported withdraw extended details"))
	}
	return errs
}
