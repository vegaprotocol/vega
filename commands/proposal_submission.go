package commands

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	oraclespb "code.vegaprotocol.io/vega/proto/oracles/v1"
)

func CheckProposalSubmission(cmd *commandspb.ProposalSubmission) error {
	return checkProposalSubmission(cmd).ErrorOrNil()
}

func checkProposalSubmission(cmd *commandspb.ProposalSubmission) Errors {
	errs := NewErrors()

	if cmd.Terms == nil {
		return errs.FinalAddForProperty("proposal_submission.terms", ErrIsRequired)
	}

	if cmd.Terms.ClosingTimestamp <= 0 {
		errs.AddForProperty("proposal_submission.terms.closing_timestamp", ErrMustBePositive)
	}
	if cmd.Terms.EnactmentTimestamp <= 0 {
		errs.AddForProperty("proposal_submission.terms.enactment_timestamp", ErrMustBePositive)
	}
	if cmd.Terms.ValidationTimestamp < 0 {
		errs.AddForProperty("proposal_submission.terms.validation_timestamp", ErrMustBePositiveOrZero)
	}

	if cmd.Terms.ClosingTimestamp > cmd.Terms.EnactmentTimestamp {
		errs.AddForProperty("proposal_submission.terms.closing_timestamp",
			errors.New("cannot be after enactment time"),
		)
	}

	if cmd.Terms.ValidationTimestamp >= cmd.Terms.ClosingTimestamp {
		errs.AddForProperty("proposal_submission.terms.validation_timestamp",
			errors.New("cannot be after or equal to closing time"),
		)
	}

	errs.Merge(checkProposalChanges(cmd.Terms))

	return errs
}

func checkProposalChanges(terms *types.ProposalTerms) Errors {
	errs := NewErrors()

	if terms.Change == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change", ErrIsRequired)
	}

	switch c := terms.Change.(type) {
	case *types.ProposalTerms_NewMarket:
		errs.Merge(checkNewMarketChanges(c))
	case *types.ProposalTerms_UpdateNetworkParameter:
		errs.Merge(checkNetworkParameterUpdateChanges(c))
	case *types.ProposalTerms_NewAsset:
		errs.Merge(checkNewAssetChanges(c))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change", ErrIsNotValid)
	}

	return errs
}

func checkNetworkParameterUpdateChanges(change *types.ProposalTerms_UpdateNetworkParameter) Errors {
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

func checkNewAssetChanges(change *types.ProposalTerms_NewAsset) Errors {
	errs := NewErrors()

	if change.NewAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset", ErrIsRequired)
	}

	if change.NewAsset.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes", ErrIsRequired)
	}

	if change.NewAsset.Changes.Source == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source", ErrIsRequired)
	}

	switch s := change.NewAsset.Changes.Source.(type) {
	case *types.AssetSource_BuiltinAsset:
		errs.Merge(checkBuiltinAssetSource(s))
	case *types.AssetSource_Erc20:
		errs.Merge(checkERC20AssetSource(s))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source", ErrIsNotValid)
	}

	return errs
}

func checkBuiltinAssetSource(s *types.AssetSource_BuiltinAsset) Errors {
	errs := NewErrors()

	if s.BuiltinAsset == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset", ErrIsRequired)
	}

	asset := s.BuiltinAsset

	if len(asset.Name) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.name", ErrIsRequired)
	}
	if len(asset.Symbol) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.symbol", ErrIsRequired)
	}
	if asset.Decimals == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.decimals", ErrIsRequired)
	}
	if len(asset.TotalSupply) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.total_supply", ErrIsRequired)
	}
	if len(asset.MaxFaucetAmountMint) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrIsRequired)
	}

	totalSupply, err := strconv.ParseUint(asset.TotalSupply, 10, 64)
	if err != nil {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.total_supply", ErrIsNotValidNumber)
	} else if totalSupply == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.total_supply", ErrMustBePositive)
	}

	maxFaucetAmount, err := strconv.ParseUint(asset.MaxFaucetAmountMint, 10, 64)
	if err != nil {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrIsNotValidNumber)
	} else if maxFaucetAmount == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.builtin_asset.max_faucet_amount_mint", ErrMustBePositive)
	}

	return errs
}

func checkERC20AssetSource(s *types.AssetSource_Erc20) Errors {
	errs := NewErrors()

	if s.Erc20 == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20", ErrIsRequired)
	}

	asset := s.Erc20

	if len(asset.ContractAddress) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_asset.changes.source.erc20.contract_address", ErrIsRequired)
	}

	return errs
}

func checkNewMarketChanges(change *types.ProposalTerms_NewMarket) Errors {
	errs := NewErrors()

	if change.NewMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market", ErrIsRequired)
	}
	if change.NewMarket.Changes == nil {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes", ErrIsRequired)
	} else {
		changes := change.NewMarket.Changes

		if changes.DecimalPlaces <= 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.decimal_places", ErrMustBePositive)
		} else if changes.DecimalPlaces >= 150 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.decimal_places", ErrMustBeLessThan150)
		}

		errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters))
		errs.Merge(checkLiquidityMonitoring(changes.LiquidityMonitoringParameters))
		errs.Merge(checkInstrument(changes.Instrument))
		errs.Merge(checkTradingMode(changes))
		errs.Merge(checkRiskParameters(changes))
	}

	errs.Merge(checkLiquidityCommitment(change.NewMarket.LiquidityCommitment))

	return errs
}

func checkPriceMonitoring(parameters *types.PriceMonitoringParameters) Errors {
	errs := NewErrors()

	if parameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters", ErrIsRequired)
	}

	if len(parameters.Triggers) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers", ErrIsRequired)
	}

	for i, trigger := range parameters.Triggers {
		if trigger.Horizon <= 0 {
			errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.%d.horizon", i), ErrMustBePositive)
		}
		if trigger.AuctionExtension <= 0 {
			errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.%d.auction_extension", i), ErrMustBePositive)
		}
		if trigger.Probability <= 0 || trigger.Probability >= 1 {
			errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.price_monitoring_parameters.triggers.%d.probability", i),
				errors.New("should be between 0 (exclusive) and 1 (exclusive)"),
			)
		}
	}

	return errs
}

func checkLiquidityMonitoring(parameters *types.LiquidityMonitoringParameters) Errors {
	errs := NewErrors()

	if parameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters", ErrIsRequired)
	}

	if parameters.TriggeringRatio < 0 || parameters.TriggeringRatio > 1 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.triggering_ratio",
			errors.New("should be between 0 (inclusive) and 1 (inclusive)"),
		)
	}

	if parameters.TargetStakeParameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters", ErrIsRequired)
	}

	if parameters.TargetStakeParameters.TimeWindow <= 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters.time_window", ErrMustBePositive)
	}
	if parameters.TargetStakeParameters.ScalingFactor <= 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.liquidity_monitoring_parameters.target_stake_parameters.scaling_factor", ErrMustBePositive)
	}

	return errs
}

func checkInstrument(instrument *types.InstrumentConfiguration) Errors {
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
	case *types.InstrumentConfiguration_Future:
		errs.Merge(checkFuture(product.Future))
	default:
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product", ErrIsNotValid)
	}

	return errs
}

func checkFuture(future *types.FutureProduct) Errors {
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

	if len(future.Maturity) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.maturity", ErrIsRequired)
	}
	_, err := time.Parse(time.RFC3339, future.Maturity)
	if err != nil {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.maturity", ErrMustBeValidDate)
	}

	errs.Merge(checkOracleSpec(future))

	return errs
}

func checkOracleSpec(future *types.FutureProduct) Errors {
	errs := NewErrors()

	if future.OracleSpec == nil {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec", ErrIsRequired)
	} else {
		if len(future.OracleSpec.PubKeys) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.pub_keys", ErrIsRequired)
		}
		for i, key := range future.OracleSpec.PubKeys {
			if len(strings.TrimSpace(key)) == 0 {
				errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.pub_keys.%d", i), ErrIsNotValid)
			}
		}
		if len(future.OracleSpec.Filters) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters", ErrIsRequired)
		} else {
			for i, filter := range future.OracleSpec.Filters {
				if filter.Key == nil {
					errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.%d.key", i), ErrIsNotValid)
				} else {
					if len(filter.Key.Name) == 0 {
						errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.%d.key.name", i), ErrIsRequired)
					}
					if filter.Key.Type == oraclespb.PropertyKey_TYPE_UNSPECIFIED {
						errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.%d.key.type", i), ErrIsRequired)
					}
				}

				if len(filter.Conditions) != 0 {
					for j, condition := range filter.Conditions {
						if len(condition.Value) == 0 {
							errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.%d.conditions.%d.value", i, j), ErrIsRequired)
						}
						if condition.Operator == oraclespb.Condition_OPERATOR_UNSPECIFIED {
							errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec.filters.%d.conditions.%d.operator", i, j), ErrIsRequired)
						}
					}
				}
			}
		}
	}

	if future.OracleSpecBinding == nil {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding", ErrIsRequired)
	} else {
		if len(future.OracleSpecBinding.SettlementPriceProperty) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.future.oracle_spec_binding.settlement_price_property", ErrIsRequired)
		}
	}

	return errs
}

func checkTradingMode(config *types.NewMarketConfiguration) Errors {
	errs := NewErrors()

	if config.TradingMode == nil {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.trading_mode", ErrIsRequired)
	}

	switch mode := config.TradingMode.(type) {
	case *types.NewMarketConfiguration_Continuous:
		errs.Merge(checkContinuousTradingMode(mode))
	case *types.NewMarketConfiguration_Discrete:
		errs.Merge(checkDiscreteTradingMode(mode))
	default:
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.trading_mode", ErrIsNotValid)
	}

	return errs
}

func checkContinuousTradingMode(mode *types.NewMarketConfiguration_Continuous) Errors {
	errs := NewErrors()

	if mode.Continuous == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.trading_mode.continuous", ErrIsRequired)
	}

	return errs
}

func checkDiscreteTradingMode(mode *types.NewMarketConfiguration_Discrete) Errors {
	errs := NewErrors()

	if mode.Discrete == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.trading_mode.discrete", ErrIsRequired)
	}

	if mode.Discrete.DurationNs <= 0 || mode.Discrete.DurationNs >= 2592000000000000 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.trading_mode.discrete.duration_ns",
			fmt.Errorf("should be between 0 (excluded) and 2592000000000000 (excluded)"))
	}

	return errs
}

func checkRiskParameters(config *types.NewMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *types.NewMarketConfiguration_Simple:
		errs.Merge(checkSimpleParameters(parameters))
	case *types.NewMarketConfiguration_LogNormal:
		errs.Merge(checkLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters", ErrIsNotValid)
	}

	return errs
}

func checkSimpleParameters(params *types.NewMarketConfiguration_Simple) Errors {
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

func checkLogNormalRiskParameters(params *types.NewMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	return errs
}

func checkLiquidityCommitment(commitment *types.NewMarketCommitment) Errors {
	errs := NewErrors()

	if commitment == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.liquidity_commitment", ErrIsRequired)
	}

	if commitment.CommitmentAmount == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.liquidity_commitment.commitment_amount", ErrMustBePositive)
	}
	if len(commitment.Fee) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.liquidity_commitment.fee", ErrIsRequired)
	}
	fee, err := strconv.ParseFloat(commitment.Fee, 64)
	if err != nil {
		errs.AddForProperty("proposal_submission.terms.change.new_market.liquidity_commitment.fee", ErrIsNotValidNumber)
	} else if fee < 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.liquidity_commitment.fee", ErrMustBePositiveOrZero)
	}

	errs.Merge(checkShape(commitment.Buys, types.Side_SIDE_BUY))
	errs.Merge(checkShape(commitment.Sells, types.Side_SIDE_SELL))

	return errs
}

func checkShape(orders []*types.LiquidityOrder, side types.Side) Errors {
	errs := NewErrors()

	humanizedSide := "buys"
	if side == types.Side_SIDE_SELL {
		humanizedSide = "sells"
	}

	if len(orders) == 0 {
		errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s", humanizedSide), ErrIsRequired)
	}

	for i, order := range orders {
		if order.Reference == types.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED {
			errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.reference.%d", humanizedSide, i), ErrIsRequired)
		}
		if _, ok := types.PeggedReference_name[int32(order.Reference)]; !ok {
			errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.reference.%d", humanizedSide, i), ErrIsNotValid)
		}

		if order.Proportion == 0 {
			errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.proportion.%d", humanizedSide, i), ErrIsRequired)
		}

		if side == types.Side_SIDE_BUY {
			switch order.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.reference.%d", humanizedSide, i),
					errors.New("cannot have a reference of type BEST_ASK when on BUY side"),
				)
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				if order.Offset > 0 {
					errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.offset.%d", humanizedSide, i), ErrMustBeNegativeOrZero)
				}
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if order.Offset >= 0 {
					errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.offset.%d", humanizedSide, i), ErrMustBeNegative)
				}
			}
		} else {
			switch order.Reference {
			case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
				errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.reference.%d", humanizedSide, i),
					errors.New("cannot have a reference of type BEST_BID when on SELL side"),
				)
			case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
				if order.Offset < 0 {
					errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.offset.%d", humanizedSide, i), ErrMustBePositiveOrZero)
				}
			case types.PeggedReference_PEGGED_REFERENCE_MID:
				if order.Offset <= 0 {
					errs.AddForProperty(fmt.Sprintf("proposal_submission.terms.change.new_asset.liquidity_commitment.%s.offset.%d", humanizedSide, i), ErrMustBePositive)
				}
			}
		}
	}

	return errs
}
