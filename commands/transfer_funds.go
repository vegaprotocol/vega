package commands

import (
	"errors"
	"math/big"

	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

var (
	ErrMustBeAfterStartEpoch = errors.New("must be after start_epoch")
	ErrUnknownAsset          = errors.New("unknown asset")
)

func CheckTransferInstruction(cmd *commandspb.TransferInstruction) error {
	return checkTransferInstruction(cmd).ErrorOrNil()
}

func checkTransferInstruction(cmd *commandspb.TransferInstruction) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("transfer_instruction", ErrIsRequired)
	}

	if len(cmd.Amount) <= 0 {
		errs.AddForProperty("transfer_instruction.amount", ErrIsRequired)
	} else {
		if amount, ok := big.NewInt(0).SetString(cmd.Amount, 10); !ok {
			errs.AddForProperty("transfer_instruction.amount", ErrNotAValidInteger)
		} else {
			if amount.Cmp(big.NewInt(0)) == 0 {
				errs.AddForProperty("transfer_instruction.amount", ErrIsRequired)
			}
			if amount.Cmp(big.NewInt(0)) == -1 {
				errs.AddForProperty("transfer_instruction.amount", ErrMustBePositive)
			}
		}
	}

	if len(cmd.To) <= 0 {
		errs.AddForProperty("transfer_instruction.to", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.To) {
		errs.AddForProperty("transfer_instruction_to", ErrShouldBeAValidVegaPubkey)
	}

	if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		errs.AddForProperty("transfer_instruction.to_account_type", ErrIsNotValid)
	} else if _, ok := vega.AccountType_name[int32(cmd.ToAccountType)]; !ok {
		errs.AddForProperty("transfer_instruction.to_account_type", ErrIsNotValid)
	}

	if cmd.FromAccountType == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		errs.AddForProperty("transfer_instruction.from_account_type", ErrIsNotValid)
	} else if _, ok := vega.AccountType_name[int32(cmd.FromAccountType)]; !ok {
		errs.AddForProperty("transfer_instruction.from_account_type", ErrIsNotValid)
	}

	if len(cmd.Asset) <= 0 {
		errs.AddForProperty("transfer_instruction.asset", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.Asset) {
		errs.AddForProperty("transfer_instruction.asset", ErrShouldBeAValidVegaID)
	}

	// arbitrary 100 char length for now
	if len(cmd.Reference) > 100 {
		errs.AddForProperty("transfer_instruction.reference", ErrMustBeLessThan100Chars)
	}

	if cmd.Kind == nil {
		errs.AddForProperty("transfer_instruction.kind", ErrIsRequired)
	} else {
		switch k := cmd.Kind.(type) {
		case *commandspb.TransferInstruction_OneOff:
			if k.OneOff.GetDeliverOn() < 0 {
				errs.AddForProperty("transfer_instruction.kind.deliver_on", ErrMustBePositiveOrZero)
			}
			// do not allow for one off transfer instruction to one of the metric based accounts
			if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
				errs.AddForProperty("transfer_instruction.account.to", errors.New("transfer instructions to metric-based reward accounts must be recurring transfer instructions that specify a distribution metric"))
			}
		case *commandspb.TransferInstruction_Recurring:
			if k.Recurring.EndEpoch != nil && *k.Recurring.EndEpoch <= 0 {
				errs.AddForProperty("transfer_instruction.kind.end_epoch", ErrMustBePositive)
			}
			if k.Recurring.StartEpoch == 0 {
				errs.AddForProperty("transfer_instruction.kind.start_epoch", ErrMustBePositive)
			}
			if k.Recurring.EndEpoch != nil && k.Recurring.StartEpoch > *k.Recurring.EndEpoch {
				errs.AddForProperty("transfer_instruction.kind.end_epoch", ErrMustBeAfterStartEpoch)
			}
			if f, ok := big.NewFloat(0).SetString(k.Recurring.Factor); !ok {
				errs.AddForProperty("transfer_instruction.kind.factor", ErrNotAValidFloat)
			} else {
				if f.Cmp(big.NewFloat(0)) <= 0 {
					errs.AddForProperty("transfer_instruction.kind.factor", ErrMustBePositive)
				}
			}
			// dispatch strategy only makes sense for reward pools
			if k.Recurring.DispatchStrategy != nil {
				// check account type is one of the relevant reward accounts
				if !(cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES ||
					cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES ||
					cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES ||
					cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS) {
					errs.AddForProperty("transfer_instruction.kind.dispatch_strategy", ErrIsNotValid)
				}
				// check asset for metric is passed unless it's a market proposer reward
				if len(k.Recurring.DispatchStrategy.AssetForMetric) <= 0 && cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
					errs.AddForProperty("transfer_instruction.kind.dispatch_strategy.asset_for_metric", ErrUnknownAsset)
				}
				if len(k.Recurring.DispatchStrategy.AssetForMetric) > 0 && !IsVegaPubkey(k.Recurring.DispatchStrategy.AssetForMetric) {
					errs.AddForProperty("transfer_instruction.kind.dispatch_strategy.asset_for_metric", ErrShouldBeAValidVegaID)
				}
				// check that that the metric makes sense for the account type
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED {
					errs.AddForProperty("transfer_instruction.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED {
					errs.AddForProperty("transfer_instruction.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID {
					errs.AddForProperty("transfer_instruction.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE {
					errs.AddForProperty("transfer_instruction.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
			}

		default:
			errs.AddForProperty("transfer_instruction.kind", ErrIsNotSupported)
		}
	}

	return errs
}
