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
	"fmt"
	"math/big"

	"code.vegaprotocol.io/vega/libs/num"
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
		errs.AddForProperty("transfer.to", ErrShouldBeAValidVegaPublicKey)
	}

	if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		errs.AddForProperty("transfer.to_account_type", ErrIsNotValid)
	} else if _, ok := vega.AccountType_name[int32(cmd.ToAccountType)]; !ok {
		errs.AddForProperty("transfer.to_account_type", ErrIsNotValid)
	}

	// if the transfer is to a reward account, it must have the to set to 0
	if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD && cmd.To != "0000000000000000000000000000000000000000000000000000000000000000" {
		errs.AddForProperty("transfer.to_account_type", ErrIsNotValid)
	}

	if cmd.FromAccountType != vega.AccountType_ACCOUNT_TYPE_GENERAL &&
		cmd.FromAccountType != vega.AccountType_ACCOUNT_TYPE_VESTED_REWARDS {
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
			if cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD && cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_GENERAL && cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_UNSPECIFIED && cmd.ToAccountType != vega.AccountType_ACCOUNT_TYPE_NETWORK_TREASURY {
				errs.AddForProperty("transfer.to_account_type", errors.New("account type is not valid for one off transfer"))
			}
			if k.OneOff.GetDeliverOn() < 0 {
				errs.AddForProperty("transfer.kind.deliver_on", ErrMustBePositiveOrZero)
			}
			// do not allow for one off transfer to one of the metric based accounts
			if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING {
				errs.AddForProperty("transfer.account.to", errors.New("transfers to metric-based reward accounts must be recurring transfers that specify a distribution metric"))
			}
		case *commandspb.Transfer_Recurring:
			if cmd.FromAccountType == vega.AccountType_ACCOUNT_TYPE_VESTED_REWARDS {
				errs.AddForProperty("transfer.from_account_type", errors.New("account type is not valid for one recurring transfer"))
			}
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
			if cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY ||
				cmd.ToAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING {
				if k.Recurring.DispatchStrategy == nil {
					errs.AddForProperty("transfer.kind.dispatch_strategy", ErrIsRequired)
				}
			}
			// dispatch strategy only makes sense for reward pools
			if k.Recurring.DispatchStrategy != nil {
				validateDispatchStrategy(cmd.ToAccountType, k.Recurring.DispatchStrategy, errs, "transfer.kind.dispatch_strategy", "transfer.account.to")
			}

		default:
			errs.AddForProperty("transfer.kind", ErrIsNotSupported)
		}
	}

	return errs
}

func mismatchingAccountTypeError(tp vega.AccountType, metric vega.DispatchMetric) error {
	return errors.New("cannot set toAccountType to " + tp.String() + " when dispatch metric is set to " + metric.String())
}

func validateDispatchStrategy(toAccountType vega.AccountType, dispatchStrategy *vega.DispatchStrategy, errs Errors, prefix string, destinationPrefixErr string) {
	// check account type is one of the relevant reward accounts
	if toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES &&
		toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES &&
		toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES &&
		toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS &&
		toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION &&
		toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN &&
		toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY &&
		toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING {
		errs.AddForProperty(destinationPrefixErr, ErrIsNotValid)
	}
	// check asset for metric is passed unless it's a market proposer reward
	if len(dispatchStrategy.AssetForMetric) <= 0 && toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS && toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING {
		errs.AddForProperty(prefix+".asset_for_metric", ErrIsRequired)
	}
	if len(dispatchStrategy.AssetForMetric) > 0 && !IsVegaID(dispatchStrategy.AssetForMetric) {
		errs.AddForProperty(prefix+".asset_for_metric", ErrShouldBeAValidVegaID)
	}
	if len(dispatchStrategy.AssetForMetric) > 0 && toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING {
		errs.AddForProperty(prefix+".asset_for_metric", errors.New("not be specified when to_account type is VALIDATOR_RANKING"))
	}
	// check that that the metric makes sense for the account type
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_LP_RECEIVED_FEES && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_RECEIVED_FEES && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MAKER_PAID_FEES && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_AVERAGE_POSITION && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_RELATIVE_RETURN && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_RETURN_VOLATILITY && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING && dispatchStrategy.Metric != vega.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING {
		errs.AddForProperty(prefix+".dispatch_metric", mismatchingAccountTypeError(toAccountType, dispatchStrategy.Metric))
	}
	if dispatchStrategy.EntityScope == vega.EntityScope_ENTITY_SCOPE_UNSPECIFIED {
		errs.AddForProperty(prefix+".entity_scope", ErrIsRequired)
	}
	if dispatchStrategy.EntityScope == vega.EntityScope_ENTITY_SCOPE_TEAMS && toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
		errs.AddForProperty(prefix+".entity_scope", errors.New(vega.EntityScope_ENTITY_SCOPE_TEAMS.String()+" is not allowed for "+toAccountType.String()))
	}
	if dispatchStrategy.EntityScope == vega.EntityScope_ENTITY_SCOPE_TEAMS && len(dispatchStrategy.NTopPerformers) == 0 {
		errs.AddForProperty(prefix+".n_top_performers", ErrIsRequired)
	}

	if dispatchStrategy.EntityScope != vega.EntityScope_ENTITY_SCOPE_TEAMS && len(dispatchStrategy.NTopPerformers) != 0 {
		errs.AddForProperty(prefix+".n_top_performers", errors.New("must not be set when entity scope is not "+vega.EntityScope_ENTITY_SCOPE_TEAMS.String()))
	}

	if dispatchStrategy.EntityScope == vega.EntityScope_ENTITY_SCOPE_TEAMS && len(dispatchStrategy.NTopPerformers) > 0 {
		nTopPerformers, err := num.DecimalFromString(dispatchStrategy.NTopPerformers)
		if err != nil {
			errs.AddForProperty(prefix+".n_top_performers", ErrIsNotValidNumber)
		} else if nTopPerformers.LessThanOrEqual(num.DecimalZero()) {
			errs.AddForProperty(prefix+".n_top_performers", ErrMustBeBetween01)
		} else if nTopPerformers.GreaterThan(num.DecimalOne()) {
			errs.AddForProperty(prefix+".n_top_performers", ErrMustBeBetween01)
		}
	}
	if dispatchStrategy.EntityScope == vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS && dispatchStrategy.IndividualScope == vega.IndividualScope_INDIVIDUAL_SCOPE_UNSPECIFIED {
		errs.AddForProperty(prefix+".individual_scope", ErrIsRequired)
	}

	if dispatchStrategy.EntityScope == vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS && len(dispatchStrategy.TeamScope) > 0 {
		errs.AddForProperty(prefix+".team_scope", errors.New("should not be set when entity_scope is set to "+dispatchStrategy.EntityScope.String()))
	}

	if dispatchStrategy.EntityScope != vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS && dispatchStrategy.IndividualScope != vega.IndividualScope_INDIVIDUAL_SCOPE_UNSPECIFIED {
		errs.AddForProperty(prefix+".individual_scope", errors.New("should not be set when entity_scope is set to "+dispatchStrategy.EntityScope.String()))
	}
	if dispatchStrategy.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_UNSPECIFIED && toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
		errs.AddForProperty(prefix+".distribution_strategy", ErrIsRequired)
	}
	if dispatchStrategy.DistributionStrategy != vega.DistributionStrategy_DISTRIBUTION_STRATEGY_UNSPECIFIED && toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
		errs.AddForProperty(prefix+".distribution_strategy", errors.New("should not be set when to_account is set to "+toAccountType.String()))
	}
	if len(dispatchStrategy.StakingRequirement) > 0 {
		if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING || toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
			errs.AddForProperty(prefix+".staking_requirement", errors.New("should not be set if to_account is set to "+toAccountType.String()))
		} else if staking, ok := big.NewInt(0).SetString(dispatchStrategy.StakingRequirement, 10); !ok {
			errs.AddForProperty(prefix+".staking_requirement", ErrNotAValidInteger)
		} else if staking.Cmp(big.NewInt(0)) < 0 {
			errs.AddForProperty(prefix+".staking_requirement", ErrMustBePositiveOrZero)
		}
	}
	if len(dispatchStrategy.NotionalTimeWeightedAveragePositionRequirement) > 0 {
		if toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_VALIDATOR_RANKING || toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
			errs.AddForProperty(prefix+".notional_time_weighted_average_position_requirement", errors.New("should not be set if to_account is set to "+toAccountType.String()))
		} else if notional, ok := big.NewInt(0).SetString(dispatchStrategy.NotionalTimeWeightedAveragePositionRequirement, 10); !ok {
			errs.AddForProperty(prefix+".notional_time_weighted_average_position_requirement", ErrNotAValidInteger)
		} else if notional.Cmp(big.NewInt(0)) < 0 {
			errs.AddForProperty(prefix+".notional_time_weighted_average_position_requirement", ErrMustBePositiveOrZero)
		}
	}
	if dispatchStrategy.WindowLength > 0 && toAccountType == vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
		errs.AddForProperty(prefix+".window_length", errors.New("should not be set for "+vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS.String()))
	}
	if dispatchStrategy.WindowLength == 0 && toAccountType != vega.AccountType_ACCOUNT_TYPE_REWARD_MARKET_PROPOSERS {
		errs.AddForProperty(prefix+".window_length", errors.New("must bet between 1 and 100"))
	}
	if dispatchStrategy.WindowLength > 100 {
		errs.AddForProperty(prefix+".window_length", ErrMustBeAtMost100)
	}
	if len(dispatchStrategy.RankTable) == 0 && dispatchStrategy.DistributionStrategy == vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK {
		errs.AddForProperty(prefix+".rank_table", ErrMustBePositive)
	}
	if len(dispatchStrategy.RankTable) > 0 && dispatchStrategy.DistributionStrategy != vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK {
		errs.AddForProperty(prefix+".rank_table", errors.New("should not be set for distribution strategy "+dispatchStrategy.DistributionStrategy.String()))
	}
	if len(dispatchStrategy.RankTable) > 500 {
		errs.AddForProperty(prefix+".rank_table", ErrMustBeAtMost500)
	}
	if len(dispatchStrategy.RankTable) > 1 {
		for i := 1; i < len(dispatchStrategy.RankTable); i++ {
			if dispatchStrategy.RankTable[i].StartRank <= dispatchStrategy.RankTable[i-1].StartRank {
				errs.AddForProperty(fmt.Sprintf(prefix+".rank_table.%i.start_rank", i), fmt.Errorf("must be greater than start_rank of element #%d", i-1))
				break
			}
		}
	}
	if dispatchStrategy.CapRewardFeeMultiple != nil && len(*dispatchStrategy.CapRewardFeeMultiple) > 0 {
		cap, err := num.DecimalFromString(*dispatchStrategy.CapRewardFeeMultiple)
		if err != nil {
			errs.AddForProperty(prefix+".cap_reward_fee_multiple", ErrIsNotValidNumber)
		} else {
			if cap.LessThanOrEqual(num.DecimalZero()) {
				errs.AddForProperty(prefix+".cap_reward_fee_multiple", ErrMustBePositive)
			}
		}
	}
}
