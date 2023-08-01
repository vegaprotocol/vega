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

func CheckTransfer(cmd *commandspb.Transfer) error {
	return checkTransfer(cmd).ErrorOrNil()
}

func checkTransfer(cmd *commandspb.Transfer) (e Errors) {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("transfer", ErrIsRequired)
	}

	if len(cmd.Amount) <= 0 {
		errs.AddForProperty("transfer.amount", ErrIsRequired)
	} else {
		if amount, ok := big.NewInt(0).SetString(cmd.Amount, 10); !ok {
			errs.AddForProperty("transfer.amount", ErrNotAValidInteger)
		} else {
			if amount.Cmp(big.NewInt(0)) == 0 {
				errs.AddForProperty("transfer.amount", ErrIsRequired)
			}
			if amount.Cmp(big.NewInt(0)) == -1 {
				errs.AddForProperty("transfer.amount", ErrMustBePositive)
			}
		}
	}

	if len(cmd.To) <= 0 {
		errs.AddForProperty("transfer.to", ErrIsRequired)
	} else if !IsVegaPublicKey(cmd.To) {
		errs.AddForProperty("transfer_to", ErrShouldBeAValidVegaPublicKey)
	}

	if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		errs.AddForProperty("transfer.to_account_type", ErrIsNotValid)
	} else if _, ok := vega.AccountType_name[int32(cmd.ToAccountType)]; !ok {
		errs.AddForProperty("transfer.to_account_type", ErrIsNotValid)
	}

	// if the transfer is to a reward account, it must have the to set to 0
	if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD && cmd.To != "0000000000000000000000000000000000000000000000000000000000000000" {
		errs.AddForProperty("transfer_to", ErrIsNotValid)
	}

	if cmd.FromAccountType != vega.AccountType_ACCOUNT_TYPE_GENERAL {
		errs.AddForProperty("transfer.from_account_type", ErrIsNotValid)
	}

	if len(cmd.Asset) <= 0 {
		errs.AddForProperty("transfer.asset", ErrIsRequired)
	} else if !IsVegaID(cmd.Asset) {
		errs.AddForProperty("transfer.asset", ErrShouldBeAValidVegaID)
	}

	// arbitrary 100 char length for now
	if len(cmd.Reference) > 100 {
		errs.AddForProperty("transfer.reference", ErrMustBeLessThan100Chars)
	}

	if cmd.Kind == nil {
		errs.AddForProperty("transfer.kind", ErrIsRequired)
	} else {
		switch k := cmd.Kind.(type) {
		case *commandspb.Transfer_OneOff:
			if cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD && cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_GENERAL && cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
				errs.AddForProperty("transfer.to_account_type", errors.New("account type is not valid for one off transfer"))
			}
			if k.OneOff.GetDeliverOn() < 0 {
				errs.AddForProperty("transfer.kind.deliver_on", ErrMustBePositiveOrZero)
			}
			// do not allow for one off transfer to one of the metric based accounts
			if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
				errs.AddForProperty("transfer.account.to", errors.New("transfers to metric-based reward accounts must be recurring transfers that specify a distribution metric"))
			}
		case *commandspb.Transfer_Recurring:
			if k.Recurring.EndEpoch != nil && *k.Recurring.EndEpoch <= 0 {
				errs.AddForProperty("transfer.kind.end_epoch", ErrMustBePositive)
			}
			if k.Recurring.StartEpoch == 0 {
				errs.AddForProperty("transfer.kind.start_epoch", ErrMustBePositive)
			}
			if k.Recurring.EndEpoch != nil && k.Recurring.StartEpoch > *k.Recurring.EndEpoch {
				errs.AddForProperty("transfer.kind.end_epoch", ErrMustBeAfterStartEpoch)
			}
			if f, ok := big.NewFloat(0).SetString(k.Recurring.Factor); !ok {
				errs.AddForProperty("transfer.kind.factor", ErrNotAValidFloat)
			} else {
				if f.Cmp(big.NewFloat(0)) <= 0 {
					errs.AddForProperty("transfer.kind.factor", ErrMustBePositive)
				}
			}
			// dispatch strategy only makes sense for reward pools
			if k.Recurring.DispatchStrategy != nil {
				// check account type is one of the relevant reward accounts
				if cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES &&
					cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES &&
					cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES &&
					cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
					errs.AddForProperty("transfer.kind.dispatch_strategy", ErrIsNotValid)
				}
				// check asset for metric is passed unless it's a market proposer reward
				if len(k.Recurring.DispatchStrategy.AssetForMetric) <= 0 && cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
					errs.AddForProperty("transfer.kind.dispatch_strategy.asset_for_metric", ErrUnknownAsset)
				}
				if len(k.Recurring.DispatchStrategy.AssetForMetric) > 0 && !IsVegaID(k.Recurring.DispatchStrategy.AssetForMetric) {
					errs.AddForProperty("transfer.kind.dispatch_strategy.asset_for_metric", ErrShouldBeAValidVegaID)
				}
				// check that that the metric makes sense for the account type
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED {
					errs.AddForProperty("transfer.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED {
					errs.AddForProperty("transfer.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID {
					errs.AddForProperty("transfer.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
				if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS && k.Recurring.DispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE {
					errs.AddForProperty("transfer.kind.dispatch_strategy.dispatch_metric", ErrIsNotValid)
				}
			}

		default:
			errs.AddForProperty("transfer.kind", ErrIsNotSupported)
		}
	}

	return errs
}
