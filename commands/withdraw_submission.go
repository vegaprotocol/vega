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
	} else if !IsVegaPubkey(cmd.Asset) {
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
