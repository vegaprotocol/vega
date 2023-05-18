package commands

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

const ReferenceMaxLen int = 100

func CheckProposalSubmission(cmd *commandspb.ProposalSubmission) error {
	return checkProposalSubmission(cmd).ErrorOrNil()
}

func checkProposalSubmission(cmd *commandspb.ProposalSubmission) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("proposal_submission", ErrIsRequired)
	}

	if len(cmd.Reference) > ReferenceMaxLen {
		errs.AddForProperty("proposal_submission.reference", ErrReferenceTooLong)
	}

	if cmd.Rationale == nil {
		errs.AddForProperty("proposal_submission.rationale", ErrIsRequired)
	} else {
		if cmd.Rationale != nil {
			if len(strings.Trim(cmd.Rationale.Description, " \n\r\t")) == 0 {
				errs.AddForProperty("proposal_submission.rationale.description", ErrIsRequired)
			} else if len(cmd.Rationale.Description) > 20000 {
				errs.AddForProperty("proposal_submission.rationale.description", ErrMustNotExceed20000Chars)
			}
			if len(strings.Trim(cmd.Rationale.Title, " \n\r\t")) == 0 {
				errs.AddForProperty("proposal_submission.rationale.title", ErrIsRequired)
			} else if len(cmd.Rationale.Title) > 100 {
				errs.AddForProperty("proposal_submission.rationale.title", ErrMustBeLessThan100Chars)
			}
		}
	}

	if cmd.Terms == nil {
		return errs.FinalAddForProperty("proposal_submission.terms", ErrIsRequired)
	}

	if cmd.Terms.ClosingTimestamp <= 0 {
		errs.AddForProperty("proposal_submission.terms.closing_timestamp", ErrMustBePositive)
	}

	if cmd.Terms.ValidationTimestamp < 0 {
		errs.AddForProperty("proposal_submission.terms.validation_timestamp", ErrMustBePositiveOrZero)
	}

	if cmd.Terms.ValidationTimestamp >= cmd.Terms.ClosingTimestamp {
		errs.AddForProperty("proposal_submission.terms.validation_timestamp",
			errors.New("cannot be after or equal to closing time"),
		)
	}

	// check for enactment timestamp
	switch cmd.Terms.Change.(type) {
	case *protoTypes.ProposalTerms_NewFreeform:
		if cmd.Terms.EnactmentTimestamp != 0 {
			errs.AddForProperty("proposal_submission.terms.enactment_timestamp", ErrIsNotSupported)
		}
	default:
		if cmd.Terms.EnactmentTimestamp <= 0 {
			errs.AddForProperty("proposal_submission.terms.enactment_timestamp", ErrMustBePositive)
		}

		if cmd.Terms.ClosingTimestamp > cmd.Terms.EnactmentTimestamp {
			errs.AddForProperty("proposal_submission.terms.closing_timestamp",
				errors.New("cannot be after enactment time"),
			)
		}
	}

	// check for validation timestamp
	switch cmd.Terms.Change.(type) {
	case *protoTypes.ProposalTerms_NewAsset:
		if cmd.Terms.ValidationTimestamp == 0 {
			errs.AddForProperty("proposal_submission.terms.validation_timestamp", ErrMustBePositive)
		}
		if cmd.Terms.ValidationTimestamp > cmd.Terms.ClosingTimestamp {
			errs.AddForProperty("proposal_submission.terms.validation_timestamp",
				errors.New("cannot be after closing time"),
			)
		}
	default:
		if cmd.Terms.ValidationTimestamp != 0 {
			errs.AddForProperty("proposal_submission.terms.validation_timestamp", ErrIsNotSupported)
		}
	}

	errs.Merge(checkProposalChanges(cmd.Terms))

	return errs
}

func checkProposalChanges(terms *protoTypes.ProposalTerms) Errors {
	errs := NewErrors()

	if terms.Change == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change", ErrIsRequired)
	}

	switch c := terms.Change.(type) {
	case *protoTypes.ProposalTerms_NewMarket:
		errs.Merge(checkNewMarketChanges(c))
	case *protoTypes.ProposalTerms_UpdateMarket:
		errs.Merge(checkUpdateMarketChanges(c))
	case *protoTypes.ProposalTerms_UpdateNetworkParameter:
		errs.Merge(checkNetworkParameterUpdateChanges(c))
	case *protoTypes.ProposalTerms_NewAsset:
		errs.Merge(checkNewAssetChanges(c))
	case *protoTypes.ProposalTerms_UpdateAsset:
		errs.Merge(checkUpdateAssetChanges(c))
	case *protoTypes.ProposalTerms_NewFreeform:
		errs.Merge(CheckNewFreeformChanges(c))
	case *protoTypes.ProposalTerms_SuccessorMarket:
		// @TODO validate successor proposal
		return errs
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change", ErrIsNotValid)
	}

	return errs
}

func checkNetworkParameterUpdateChanges(change *protoTypes.ProposalTerms_UpdateNetworkParameter) Errors {
	errs := NewErrors()

	if change.UpdateNetworkParameter == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_network_parameter", ErrIsRequired)
	}

	if change.UpdateNetworkParameter.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_network_parameter.changes", ErrIsRequired)
	}

	parameter := change.UpdateNetworkParameter.Changes

	if len(parameter.Key) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_network_parameter.changes.key", ErrIsRequired)
	}

	if len(parameter.Value) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_network_parameter.changes.value", ErrIsRequired)
	}

	return errs
}

func checkNewAssetChanges(change *protoTypes.ProposalTerms_NewAsset) Errors {
	errs := NewErrors()

	if change.NewAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset", ErrIsRequired)
	}

	if change.NewAsset.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes", ErrIsRequired)
	}

	if len(change.NewAsset.Changes.Name) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.name", ErrIsRequired)
	}
	if len(change.NewAsset.Changes.Symbol) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.symbol", ErrIsRequired)
	}

	if len(change.NewAsset.Changes.Quantum) <= 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.quantum", ErrIsRequired)
	} else if quantum, err := num.DecimalFromString(change.NewAsset.Changes.Quantum); err != nil {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.quantum", ErrIsNotValidNumber)
	} else if quantum.LessThanOrEqual(num.DecimalZero()) {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.quantum", ErrMustBePositive)
	}

	if change.NewAsset.Changes.Source == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source", ErrIsRequired)
	}

	switch s := change.NewAsset.Changes.Source.(type) {
	case *protoTypes.AssetDetails_BuiltinAsset:
		errs.Merge(checkBuiltinAssetSource(s))
	case *protoTypes.AssetDetails_Erc20:
		errs.Merge(checkERC20AssetSource(s))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source", ErrIsNotValid)
	}

	return errs
}

func CheckNewFreeformChanges(change *protoTypes.ProposalTerms_NewFreeform) Errors {
	errs := NewErrors()

	if change.NewFreeform == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_freeform", ErrIsRequired)
	}
	return errs
}

func checkBuiltinAssetSource(s *protoTypes.AssetDetails_BuiltinAsset) Errors {
	errs := NewErrors()

	if s.BuiltinAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset", ErrIsRequired)
	}

	asset := s.BuiltinAsset

	if len(asset.MaxFaucetAmountMint) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrIsRequired)
	} else {
		if maxFaucetAmount, ok := big.NewInt(0).SetString(asset.MaxFaucetAmountMint, 10); !ok {
			return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrIsNotValidNumber)
		} else if maxFaucetAmount.Cmp(big.NewInt(0)) <= 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrMustBePositive)
		}
	}

	return errs
}

func checkERC20AssetSource(s *protoTypes.AssetDetails_Erc20) Errors {
	errs := NewErrors()

	if s.Erc20 == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20", ErrIsRequired)
	}

	asset := s.Erc20

	if len(asset.ContractAddress) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.contract_address", ErrIsRequired)
	}
	if len(asset.LifetimeLimit) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit", ErrIsRequired)
	} else {
		if lifetimeLimit, ok := big.NewInt(0).SetString(asset.LifetimeLimit, 10); !ok {
			errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit", ErrIsNotValidNumber)
		} else {
			if lifetimeLimit.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.lifetime_limit", ErrMustBePositive)
			}
		}
	}
	if len(asset.WithdrawThreshold) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold", ErrIsRequired)
	} else {
		if withdrawThreshold, ok := big.NewInt(0).SetString(asset.WithdrawThreshold, 10); !ok {
			errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold", ErrIsNotValidNumber)
		} else {
			if withdrawThreshold.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.withdraw_threshold", ErrMustBePositive)
			}
		}
	}

	return errs
}

func checkUpdateAssetChanges(change *protoTypes.ProposalTerms_UpdateAsset) Errors {
	errs := NewErrors()

	if change.UpdateAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_asset", ErrIsRequired)
	}

	if len(change.UpdateAsset.AssetId) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_asset.asset_id", ErrIsRequired)
	} else if !IsVegaPubkey(change.UpdateAsset.AssetId) {
		errs.AddForProperty("proposal_submission.terms.change.update_asset.asset_id", ErrShouldBeAValidVegaID)
	}

	if change.UpdateAsset.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_asset.changes", ErrIsRequired)
	}

	if len(change.UpdateAsset.Changes.Quantum) <= 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.quantum", ErrIsRequired)
	} else if quantum, err := num.DecimalFromString(change.UpdateAsset.Changes.Quantum); err != nil {
		errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.quantum", ErrIsNotValidNumber)
	} else if quantum.LessThanOrEqual(num.DecimalZero()) {
		errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.quantum", ErrMustBePositive)
	}

	if change.UpdateAsset.Changes.Source == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_asset.changes.source", ErrIsRequired)
	}

	switch s := change.UpdateAsset.Changes.Source.(type) {
	case *protoTypes.AssetDetailsUpdate_Erc20:
		errs.Merge(checkERC20UpdateAssetSource(s))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_asset.changes.source", ErrIsNotValid)
	}

	return errs
}

func checkERC20UpdateAssetSource(s *protoTypes.AssetDetailsUpdate_Erc20) Errors {
	errs := NewErrors()

	if s.Erc20 == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_asset.changes.source.erc20", ErrIsRequired)
	}

	asset := s.Erc20

	if len(asset.LifetimeLimit) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.source.erc20.lifetime_limit", ErrIsRequired)
	} else {
		if lifetimeLimit, ok := big.NewInt(0).SetString(asset.LifetimeLimit, 10); !ok {
			errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.source.erc20.lifetime_limit", ErrIsNotValidNumber)
		} else {
			if lifetimeLimit.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.source.erc20.lifetime_limit", ErrMustBePositive)
			}
		}
	}

	if len(asset.WithdrawThreshold) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.source.erc20.withdraw_threshold", ErrIsRequired)
	} else {
		if withdrawThreshold, ok := big.NewInt(0).SetString(asset.WithdrawThreshold, 10); !ok {
			errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.source.erc20.withdraw_threshold", ErrIsNotValidNumber)
		} else {
			if withdrawThreshold.Cmp(big.NewInt(0)) <= 0 {
				errs.AddForProperty("proposal_submission.terms.change.update_asset.changes.source.erc20.withdraw_threshold", ErrMustBePositive)
			}
		}
	}

	return errs
}

func checkNewMarketChanges(change *protoTypes.ProposalTerms_NewMarket) Errors {
	errs := NewErrors()

	if change.NewMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market", ErrIsRequired)
	}

	if change.NewMarket.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes", ErrIsRequired)
	}

	changes := change.NewMarket.Changes

	if changes.DecimalPlaces >= 150 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.decimal_places", ErrMustBeLessThan150)
	}

	if changes.PositionDecimalPlaces >= 7 || changes.PositionDecimalPlaces <= -7 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.position_decimal_places", ErrMustBeWithinRange7)
	}

	lppr, err := num.DecimalFromString(changes.LpPriceRange)
	if err != nil {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.lp_price_range", ErrIsNotValidNumber)
	} else if lppr.IsNegative() || lppr.IsZero() {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.lp_price_range", ErrMustBePositive)
	} else if lppr.GreaterThan(num.DecimalFromInt64(100)) {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.lp_price_range", ErrMustBeAtMost100)
	}

	if len(changes.LinearSlippageFactor) > 0 {
		linearSlippage, err := num.DecimalFromString(changes.LinearSlippageFactor)
		if err != nil {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.linear_slippage_factor", ErrIsNotValidNumber)
		} else if linearSlippage.IsNegative() {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.linear_slippage_factor", ErrMustBePositiveOrZero)
		} else if linearSlippage.GreaterThan(num.DecimalFromInt64(1000000)) {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.linear_slippage_factor", ErrMustBeAtMost1M)
		}
	}

	if len(changes.QuadraticSlippageFactor) > 0 {
		squaredSlippage, err := num.DecimalFromString(changes.QuadraticSlippageFactor)
		if err != nil {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor", ErrIsNotValidNumber)
		} else if squaredSlippage.IsNegative() {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor", ErrMustBePositiveOrZero)
		} else if squaredSlippage.GreaterThan(num.DecimalFromInt64(1000000)) {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.quadratic_slippage_factor", ErrMustBeAtMost1M)
		}
	}

	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "proposal_submission.terms.change.new_market.changes"))
	errs.Merge(checkLiquidityMonitoring(changes.LiquidityMonitoringParameters, "proposal_submission.terms.change.new_market.changes"))
	errs.Merge(checkNewInstrument(changes.Instrument))
	errs.Merge(checkNewRiskParameters(changes))

	return errs
}

func checkUpdateMarketChanges(change *protoTypes.ProposalTerms_UpdateMarket) Errors {
	errs := NewErrors()

	if change.UpdateMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market", ErrIsRequired)
	}

	if len(change.UpdateMarket.MarketId) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_market.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(change.UpdateMarket.MarketId) {
		errs.AddForProperty("proposal_submission.terms.change.update_market.market_id", ErrShouldBeAValidVegaID)
	}

	if change.UpdateMarket.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes", ErrIsRequired)
	}

	changes := change.UpdateMarket.Changes
	lppr, err := num.DecimalFromString(changes.LpPriceRange)
	if err != nil {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.lp_price_range", ErrIsNotValidNumber)
	} else if lppr.IsNegative() || lppr.IsZero() {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.lp_price_range", ErrMustBePositive)
	} else if lppr.GreaterThan(num.DecimalFromInt64(100)) {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.lp_price_range", ErrMustBeAtMost100)
	}

	if len(changes.LinearSlippageFactor) > 0 {
		linearSlippage, err := num.DecimalFromString(changes.LinearSlippageFactor)
		if err != nil {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.linear_slippage_factor", ErrIsNotValidNumber)
		} else if linearSlippage.IsNegative() {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.linear_slippage_factor", ErrMustBePositiveOrZero)
		} else if linearSlippage.GreaterThan(num.DecimalFromInt64(1000000)) {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.linear_slippage_factor", ErrMustBeAtMost1M)
		}
	}

	if len(changes.QuadraticSlippageFactor) > 0 {
		squaredSlippage, err := num.DecimalFromString(changes.QuadraticSlippageFactor)
		if err != nil {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.quadratic_slippage_factor", ErrIsNotValidNumber)
		} else if squaredSlippage.IsNegative() {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.quadratic_slippage_factor", ErrMustBePositiveOrZero)
		} else if squaredSlippage.GreaterThan(num.DecimalFromInt64(1000000)) {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.quadratic_slippage_factor", ErrMustBeAtMost1M)
		}
	}

	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "proposal_submission.terms.change.update_market.changes"))
	errs.Merge(checkLiquidityMonitoring(changes.LiquidityMonitoringParameters, "proposal_submission.terms.change.update_market.changes"))
	errs.Merge(checkUpdateInstrument(changes.Instrument))
	errs.Merge(checkUpdateRiskParameters(changes))

	return errs
}

func checkPriceMonitoring(parameters *protoTypes.PriceMonitoringParameters, parentProperty string) Errors {
	errs := NewErrors()

	if parameters == nil || len(parameters.Triggers) == 0 {
		return errs
	}

	if len(parameters.Triggers) > 5 {
		errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers", parentProperty), errors.New("maximum 5 triggers allowed"))
	}

	for i, trigger := range parameters.Triggers {
		if trigger.Horizon <= 0 {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.horizon", parentProperty, i), ErrMustBePositive)
		}
		if trigger.AuctionExtension <= 0 {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.auction_extension", parentProperty, i), ErrMustBePositive)
		}

		probability, err := strconv.ParseFloat(trigger.Probability, 64)
		if err != nil {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.probability", parentProperty, i),
				errors.New("must be numeric and be between 0 (exclusive) and 1 (exclusive)"),
			)
		}

		if probability <= 0.9 || probability >= 1 {
			errs.AddForProperty(fmt.Sprintf("%s.price_monitoring_parameters.triggers.%d.probability", parentProperty, i),
				errors.New("should be between 0.9 (exclusive) and 1 (exclusive)"),
			)
		}
	}

	return errs
}

func checkLiquidityMonitoring(parameters *protoTypes.LiquidityMonitoringParameters, parentProperty string) Errors {
	errs := NewErrors()

	if parameters == nil {
		return errs
	}

	if len(parameters.TriggeringRatio) == 0 {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters.triggering_ratio", parentProperty), ErrIsNotValidNumber)
	}

	tr, err := num.DecimalFromString(parameters.TriggeringRatio)
	if err != nil {
		errs.AddForProperty(
			fmt.Sprintf("%s.liquidity_monitoring_parameters.triggering_ratio", parentProperty),
			fmt.Errorf("error parsing triggering ratio value: %s", err.Error()),
		)
	}
	if tr.IsNegative() || tr.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty(
			fmt.Sprintf("%s.liquidity_monitoring_parameters.triggering_ratio", parentProperty),
			errors.New("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	if parameters.TargetStakeParameters == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters.target_stake_parameters", parentProperty), ErrIsRequired)
	}

	if parameters.TargetStakeParameters.TimeWindow <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters.target_stake_parameters.time_window", parentProperty), ErrMustBePositive)
	}
	if parameters.TargetStakeParameters.ScalingFactor <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.liquidity_monitoring_parameters.target_stake_parameters.scaling_factor", parentProperty), ErrMustBePositive)
	}

	return errs
}

func checkNewInstrument(instrument *protoTypes.InstrumentConfiguration) Errors {
	errs := NewErrors()

	if instrument == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.instrument", ErrIsRequired)
	}

	if len(instrument.Name) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.name", ErrIsRequired)
	}
	if len(instrument.Code) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.code", ErrIsRequired)
	}

	if instrument.Product == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product", ErrIsRequired)
	}

	switch product := instrument.Product.(type) {
	case *protoTypes.InstrumentConfiguration_Future:
		errs.Merge(checkNewFuture(product.Future))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product", ErrIsNotValid)
	}

	return errs
}

func checkUpdateInstrument(instrument *protoTypes.UpdateInstrumentConfiguration) Errors {
	errs := NewErrors()

	if instrument == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.instrument", ErrIsRequired)
	}

	if len(instrument.Code) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.code", ErrIsRequired)
	}

	if instrument.Product == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product", ErrIsRequired)
	}

	switch product := instrument.Product.(type) {
	case *protoTypes.UpdateInstrumentConfiguration_Future:
		errs.Merge(checkUpdateFuture(product.Future))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product", ErrIsNotValid)
	}

	return errs
}

func checkNewFuture(future *protoTypes.FutureProduct) Errors {
	errs := NewErrors()

	if future == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future", ErrIsRequired)
	}

	if len(future.SettlementAsset) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.settlement_asset", ErrIsRequired)
	}
	if len(future.QuoteName) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.quote_name", ErrIsRequired)
	}

	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForSettlementData, "data_source_spec_for_settlement_data", "proposal_submission.terms.change.new_market.changes.instrument.product.future", true))
	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForTradingTermination, "data_source_spec_for_trading_termination", "proposal_submission.terms.change.new_market.changes.instrument.product.future", false))
	errs.Merge(checkNewOracleBinding(future))

	return errs
}

func checkUpdateFuture(future *protoTypes.UpdateFutureProduct) Errors {
	errs := NewErrors()

	if future == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.future", ErrIsRequired)
	}

	if len(future.QuoteName) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.future.quote_name", ErrIsRequired)
	}

	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForSettlementData, "data_source_spec_for_settlement_data", "proposal_submission.terms.change.update_market.changes.instrument.product.future", true))
	errs.Merge(checkDataSourceSpec(future.DataSourceSpecForTradingTermination, "data_source_spec_for_trading_termination", "proposal_submission.terms.change.update_market.changes.instrument.product.future", false))
	errs.Merge(checkUpdateOracleBinding(future))

	return errs
}

func checkDataSourceSpec(spec *vegapb.DataSourceDefinition, name string, parentProperty string, tryToSettle bool) Errors {
	errs := NewErrors()
	if spec == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.%s", parentProperty, name), ErrIsRequired)
	}

	if spec.SourceType == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.%s", parentProperty, name+".source_type"), ErrIsRequired)
	}

	switch tp := spec.SourceType.(type) {
	case *vegapb.DataSourceDefinition_Internal:
		if tryToSettle {
			return errs.FinalAddForProperty(fmt.Sprintf("%s.%s", parentProperty, name), ErrIsNotValid)
		}

		t := tp.Internal.GetTime()
		if t == nil {
			return errs.FinalAddForProperty(fmt.Sprintf("%s.%s.internal", parentProperty, name), ErrIsRequired)
		}

		if len(t.Conditions) == 0 {
			errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions", parentProperty, name), ErrIsRequired)
		}

		if len(t.Conditions) != 0 {
			for j, condition := range t.Conditions {
				if len(condition.Value) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions.%d.value", parentProperty, name, j), ErrIsRequired)
				}
				if condition.Operator == datapb.Condition_OPERATOR_UNSPECIFIED {
					errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions.%d.operator", parentProperty, name, j), ErrIsRequired)
				}

				if _, ok := datapb.Condition_Operator_name[int32(condition.Operator)]; !ok {
					errs.AddForProperty(fmt.Sprintf("%s.%s.internal.time.conditions.%d.operator", parentProperty, name, j), ErrIsNotValid)
				}
			}
		}

	case *vegapb.DataSourceDefinition_External:
		// If data source type is external - check if the signers are present first.
		o := tp.External.GetOracle()

		signers := o.Signers
		if len(signers) == 0 {
			errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers", parentProperty, name), ErrIsRequired)
		}

		for i, key := range signers {
			signer := types.SignerFromProto(key)
			if signer.IsEmpty() {
				errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers.%d", parentProperty, name, i), ErrIsNotValid)
			} else if pubkey := signer.GetSignerPubKey(); pubkey != nil && !crypto.IsValidVegaPubKey(pubkey.Key) {
				errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers.%d", parentProperty, name, i), ErrIsNotValidVegaPubkey)
			} else if address := signer.GetSignerETHAddress(); address != nil && !crypto.EthereumIsValidAddress(address.Address) {
				errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers.%d", parentProperty, name, i), ErrIsNotValidEthereumAddress)
			}
		}

		filters := o.Filters
		errs.Merge(checkDataSourceSpecFilters(filters, fmt.Sprintf("%s.external.oracle", name), parentProperty))
	}

	return errs
}

func checkDataSourceSpecFilters(filters []*datapb.Filter, name string, parentProperty string) Errors {
	errs := NewErrors()

	if len(filters) == 0 {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.%s.filters", parentProperty, name), ErrIsRequired)
	}

	for i, filter := range filters {
		if filter.Key == nil {
			errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key", parentProperty, name, i), ErrIsNotValid)
		} else {
			if len(filter.Key.Name) == 0 {
				errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key.name", parentProperty, name, i), ErrIsRequired)
			}
			if filter.Key.Type == datapb.PropertyKey_TYPE_UNSPECIFIED {
				errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key.type", parentProperty, name, i), ErrIsRequired)
			}
			if _, ok := datapb.PropertyKey_Type_name[int32(filter.Key.Type)]; !ok {
				errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.key.type", parentProperty, name, i), ErrIsNotValid)
			}
		}

		if len(filter.Conditions) != 0 {
			for j, condition := range filter.Conditions {
				if len(condition.Value) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.conditions.%d.value", parentProperty, name, i, j), ErrIsRequired)
				}
				if condition.Operator == datapb.Condition_OPERATOR_UNSPECIFIED {
					errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.conditions.%d.operator", parentProperty, name, i, j), ErrIsRequired)
				}
				if _, ok := datapb.Condition_Operator_name[int32(condition.Operator)]; !ok {
					errs.AddForProperty(fmt.Sprintf("%s.%s.filters.%d.conditions.%d.operator", parentProperty, name, i, j), ErrIsNotValid)
				}
			}
		}
	}

	return errs
}

func isBindingMatchingSpec(spec *vegapb.DataSourceDefinition, bindingProperty string) bool {
	bindingPropertyFound := false
	filters := []*datapb.Filter{}
	if spec != nil {
		filters = spec.GetFilters()
	}
	if spec != nil && filters != nil {
		for _, filter := range filters {
			if filter.Key != nil && filter.Key.Name == bindingProperty {
				bindingPropertyFound = true
			}
		}
	}
	return bindingPropertyFound
}

func checkNewOracleBinding(future *protoTypes.FutureProduct) Errors {
	errs := NewErrors()
	if future.DataSourceSpecBinding != nil {
		if len(future.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(future.DataSourceSpecForSettlementData, future.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}

		if len(future.DataSourceSpecBinding.TradingTerminationProperty) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsRequired)
		} else {
			if future.DataSourceSpecForTradingTermination == nil || future.DataSourceSpecForTradingTermination.GetExternal() != nil && !isBindingMatchingSpec(future.DataSourceSpecForTradingTermination, future.DataSourceSpecBinding.TradingTerminationProperty) {
				errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.data_source_spec_binding", ErrIsRequired)
	}

	return errs
}

func checkUpdateOracleBinding(future *protoTypes.UpdateFutureProduct) Errors {
	errs := NewErrors()
	if future.DataSourceSpecBinding != nil {
		if len(future.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(future.DataSourceSpecForSettlementData, future.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}

		if len(future.DataSourceSpecBinding.TradingTerminationProperty) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(future.DataSourceSpecForTradingTermination, future.DataSourceSpecBinding.TradingTerminationProperty) {
				errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding.trading_termination_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.future.data_source_spec_binding", ErrIsRequired)
	}

	return errs
}

func checkNewRiskParameters(config *protoTypes.NewMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.NewMarketConfiguration_Simple:
		errs.Merge(checkNewSimpleParameters(parameters))
	case *protoTypes.NewMarketConfiguration_LogNormal:
		errs.Merge(checkNewLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters", ErrIsNotValid)
	}

	return errs
}

func checkUpdateRiskParameters(config *protoTypes.UpdateMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.UpdateMarketConfiguration_Simple:
		errs.Merge(checkUpdateSimpleParameters(parameters))
	case *protoTypes.UpdateMarketConfiguration_LogNormal:
		errs.Merge(checkUpdateLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters", ErrIsNotValid)
	}

	return errs
}

func checkNewSimpleParameters(params *protoTypes.NewMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.simple.probability_of_trading",
			fmt.Errorf("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	return errs
}

func checkUpdateSimpleParameters(params *protoTypes.UpdateMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.simple.probability_of_trading",
			fmt.Errorf("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	return errs
}

func checkNewLogNormalRiskParameters(params *protoTypes.NewMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter < 1e-8 || params.LogNormal.RiskAversionParameter > 0.1 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.risk_aversion_parameter", errors.New("must be between [1e-8, 0.1]"))
	}

	if params.LogNormal.Tau < 1e-8 || params.LogNormal.Tau > 1 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.tau", errors.New("must be between [1e-8, 1]"))
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Mu < -1e-6 || params.LogNormal.Params.Mu > 1e-6 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.mu", errors.New("must be between [-1e-6,1e-6]"))
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma < 1e-3 || params.LogNormal.Params.Sigma > 50 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.sigma", errors.New("must be between [1e-3,50]"))
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.R < -1 || params.LogNormal.Params.R > 1 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params.r", errors.New("must be between [-1,1]"))
	}

	return errs
}

func checkUpdateLogNormalRiskParameters(params *protoTypes.UpdateMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter <= 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.risk_aversion_parameter", ErrMustBePositive)
	}

	if params.LogNormal.Tau <= 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.tau", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma <= 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.sigma", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	return errs
}
