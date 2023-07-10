package commands

import (
	"errors"
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
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
	case *protoTypes.ProposalTerms_NewSpotMarket:
		errs.Merge(checkNewSpotMarketChanges(c))
	case *protoTypes.ProposalTerms_UpdateSpotMarket:
		errs.Merge(checkUpdateSpotMarketChanges(c))
	case *protoTypes.ProposalTerms_UpdateNetworkParameter:
		errs.Merge(checkNetworkParameterUpdateChanges(c))
	case *protoTypes.ProposalTerms_NewAsset:
		errs.Merge(checkNewAssetChanges(c))
	case *protoTypes.ProposalTerms_UpdateAsset:
		errs.Merge(checkUpdateAssetChanges(c))
	case *protoTypes.ProposalTerms_NewFreeform:
		errs.Merge(CheckNewFreeformChanges(c))
	case *protoTypes.ProposalTerms_NewTransfer:
		errs.Merge((checkNewTransferChanges(c)))
	case *protoTypes.ProposalTerms_CancelTransfer:
		errs.Merge((checkCancelTransferChanges(c)))
	case *protoTypes.ProposalTerms_UpdateMarketState:
		errs.Merge((checkMarketUpdateState(c)))
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

func checkCancelTransferChanges(change *protoTypes.ProposalTerms_CancelTransfer) Errors {
	errs := NewErrors()
	if change.CancelTransfer == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.cancel_transfer", ErrIsRequired)
	}

	if change.CancelTransfer.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.cancel_transfer.changes", ErrIsRequired)
	}

	changes := change.CancelTransfer.Changes
	if len(changes.TransferId) == 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.cancel_transfer.changes.transferId", ErrIsRequired)
	}
	return errs
}

func checkMarketUpdateState(change *protoTypes.ProposalTerms_UpdateMarketState) Errors {
	errs := NewErrors()
	if change.UpdateMarketState == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state", ErrIsRequired)
	}
	if change.UpdateMarketState.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state.changes", ErrIsRequired)
	}
	changes := change.UpdateMarketState.Changes
	if len(changes.MarketId) == 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state.changes.marketId", ErrIsRequired)
	}
	if changes.UpdateType == 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state.changes.updateType", ErrIsRequired)
	}
	// if the update type is not terminate, price must be empty
	if changes.UpdateType != vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE && changes.Price != nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state.changes.price", ErrMustBeEmpty)
	}

	// if termination and price is provided it must be a valid uint
	if changes.UpdateType == vega.MarketStateUpdateType_MARKET_STATE_UPDATE_TYPE_TERMINATE && changes.Price != nil && len(*changes.Price) > 0 {
		n, overflow := num.UintFromString(*changes.Price, 10)
		if overflow || n.IsNegative() {
			return errs.FinalAddForProperty("proposal_submission.terms.change.update_market_state.changes.price", ErrIsNotValid)
		}
	}
	return errs
}

func checkNewTransferChanges(change *protoTypes.ProposalTerms_NewTransfer) Errors {
	errs := NewErrors()
	if change.NewTransfer == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer", ErrIsRequired)
	}

	if change.NewTransfer.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes", ErrIsRequired)
	}

	changes := change.NewTransfer.Changes
	if changes.SourceType == protoTypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.source_type", ErrIsRequired)
	}

	// source account type may be one of the following:
	if changes.SourceType != protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE &&
		changes.SourceType != protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.source_type", ErrIsNotValid)
	}

	if changes.DestinationType == protoTypes.AccountType_ACCOUNT_TYPE_UNSPECIFIED {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.destination_type", ErrIsRequired)
	}

	if changes.SourceType == protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD && changes.DestinationType == protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.destination_type", ErrIsNotValid)
	}

	// destination account type may be one of the following:
	if changes.DestinationType != protoTypes.AccountType_ACCOUNT_TYPE_GENERAL &&
		changes.DestinationType != protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD &&
		changes.DestinationType != protoTypes.AccountType_ACCOUNT_TYPE_INSURANCE {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.destination_type", ErrIsNotValid)
	}

	if changes.SourceType == protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD && len(changes.Source) > 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.source", ErrIsNotValid)
	}

	if changes.DestinationType == protoTypes.AccountType_ACCOUNT_TYPE_GLOBAL_REWARD && len(changes.Destination) > 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.destination", ErrIsNotValid)
	}

	if changes.SourceType == changes.DestinationType && changes.Source == changes.Destination {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.destination", ErrIsNotValid)
	}

	if changes.TransferType == protoTypes.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_UNSPECIFIED {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.transfer_type", ErrIsRequired)
	}

	if len(changes.Amount) == 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.amount", ErrIsRequired)
	}

	n, overflow := num.UintFromString(changes.Amount, 10)
	if overflow || n.IsNegative() {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.amount", ErrIsNotValid)
	}

	if len(changes.Asset) == 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.asset", ErrIsRequired)
	}

	if len(changes.FractionOfBalance) == 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance", ErrIsRequired)
	}

	fraction, err := num.DecimalFromString(changes.FractionOfBalance)
	if err != nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance", ErrIsNotValid)
	}
	if !fraction.IsPositive() {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance", ErrMustBePositive)
	}

	if fraction.GreaterThan(num.DecimalOne()) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.fraction_of_balance", ErrMustBeLTE1)
	}

	if recurring := changes.GetRecurring(); recurring != nil {
		if recurring.EndEpoch != nil && *recurring.EndEpoch < recurring.StartEpoch {
			return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.recurring.end_epoch", ErrIsNotValid)
		}
	}

	if changes.GetRecurring() == nil && changes.GetOneOff() == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_transfer.changes.kind", ErrIsRequired)
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

func checkNewSpotMarketChanges(change *protoTypes.ProposalTerms_NewSpotMarket) Errors {
	errs := NewErrors()

	if change.NewSpotMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market", ErrIsRequired)
	}

	if change.NewSpotMarket.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes", ErrIsRequired)
	}

	changes := change.NewSpotMarket.Changes
	isCorrectProduct := false

	if changes.Instrument == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument", ErrIsRequired)
	}

	if changes.Instrument.Product == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument.product", ErrIsRequired)
	}

	switch changes.Instrument.Product.(type) {
	case *protoTypes.InstrumentConfiguration_Spot:
		isCorrectProduct = true
	default:
		isCorrectProduct = false
	}

	if !isCorrectProduct {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument.product", ErrIsMismatching)
	}
	if changes.DecimalPlaces >= 150 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.decimal_places", ErrMustBeLessThan150)
	}

	if changes.PositionDecimalPlaces >= 7 || changes.PositionDecimalPlaces <= -7 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.position_decimal_places", ErrMustBeWithinRange7)
	}
	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "proposal_submission.terms.change.new_spot_market.changes"))
	errs.Merge(checkTargetStakeParams(changes.TargetStakeParameters, "proposal_submission.terms.change.new_spot_market.changes"))
	errs.Merge(checkNewInstrument(changes.Instrument, "proposal_submission.terms.change.new_spot_market.changes.instrument"))
	errs.Merge(checkNewSpotRiskParameters(changes))
	errs.Merge(checkSLAParams(changes.SlaParams, "proposal_submission.terms.change.new_spot_market.changes.sla_params"))
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
	if successor := changes.Successor; successor != nil {
		if len(successor.InsurancePoolFraction) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.successor.insurance_pool_fraction", ErrIsRequired)
		} else {
			if ipf, err := num.DecimalFromString(successor.InsurancePoolFraction); err != nil {
				errs.AddForProperty("proposal_submission.terms.change.new_market.changes.successor.insurance_pool_fraction", ErrIsNotValidNumber)
			} else if ipf.IsNegative() || ipf.GreaterThan(num.DecimalFromInt64(1)) {
				errs.AddForProperty("proposal_submission.terms.change.new_market.changes.successor.insurance_pool_fraction", ErrMustBeWithinRange01)
			}
		}
	}

	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "proposal_submission.terms.change.new_market.changes"))
	errs.Merge(checkLiquidityMonitoring(changes.LiquidityMonitoringParameters, "proposal_submission.terms.change.new_market.changes"))
	errs.Merge(checkNewInstrument(changes.Instrument, "proposal_submission.terms.change.new_market.changes.instrument"))
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

func checkUpdateSpotMarketChanges(change *protoTypes.ProposalTerms_UpdateSpotMarket) Errors {
	errs := NewErrors()

	if change.UpdateSpotMarket == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market", ErrIsRequired)
	}

	if len(change.UpdateSpotMarket.MarketId) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_spot_market.market_id", ErrIsRequired)
	} else if !IsVegaPubkey(change.UpdateSpotMarket.MarketId) {
		errs.AddForProperty("proposal_submission.terms.change.update_spot_market.market_id", ErrShouldBeAValidVegaID)
	}

	if change.UpdateSpotMarket.Changes == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes", ErrIsRequired)
	}

	changes := change.UpdateSpotMarket.Changes
	errs.Merge(checkPriceMonitoring(changes.PriceMonitoringParameters, "proposal_submission.terms.change.update_spot_market.changes"))
	errs.Merge(checkTargetStakeParams(changes.TargetStakeParameters, "proposal_submission.terms.change.update_spot_market.changes"))
	errs.Merge(checkUpdateSpotRiskParameters(changes))
	errs.Merge(checkSLAParams(changes.SlaParams, "proposal_submission.terms.change.update_spot_market.changes.sla_params"))
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

func checkTargetStakeParams(targetStakeParameters *protoTypes.TargetStakeParameters, parentProperty string) Errors {
	errs := NewErrors()
	if targetStakeParameters == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.target_stake_parameters", parentProperty), ErrIsRequired)
	}

	if targetStakeParameters.TimeWindow <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.target_stake_parameters.time_window", parentProperty), ErrMustBePositive)
	}
	if targetStakeParameters.ScalingFactor <= 0 {
		errs.AddForProperty(fmt.Sprintf("%s.target_stake_parameters.scaling_factor", parentProperty), ErrMustBePositive)
	}
	return errs
}

func checkNewInstrument(instrument *protoTypes.InstrumentConfiguration, parent string) Errors {
	errs := NewErrors()

	if instrument == nil {
		return errs.FinalAddForProperty(parent, ErrIsRequired)
	}

	if len(instrument.Name) == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.name", parent), ErrIsRequired)
	}
	if len(instrument.Code) == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.code", parent), ErrIsRequired)
	}

	if instrument.Product == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.product", parent), ErrIsRequired)
	}

	switch product := instrument.Product.(type) {
	case *protoTypes.InstrumentConfiguration_Future:
		errs.Merge(checkNewFuture(product.Future))
	case *protoTypes.InstrumentConfiguration_Perps:
		errs.Merge(checkNewPerps(product.Perps))
	case *protoTypes.InstrumentConfiguration_Spot:
		errs.Merge(checkNewSpot(product.Spot))
	default:
		return errs.FinalAddForProperty(fmt.Sprintf("%s.product", parent), ErrIsNotValid)
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

func checkNewPerps(perps *protoTypes.PerpsProduct) Errors {
	errs := NewErrors()

	if perps == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.perps", ErrIsRequired)
	}

	if len(perps.SettlementAsset) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.perps.settlement_asset", ErrIsRequired)
	}
	if len(perps.QuoteName) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.perps.quote_name", ErrIsRequired)
	}

	errs.Merge(checkDataSourceSpec(perps.DataSourceSpecForSettlementData, "data_source_spec_for_settlement_data", "proposal_submission.terms.change.new_market.changes.instrument.product.perps", true))
	errs.Merge(checkNewPerpsOracleBinding(perps))

	return errs
}

func checkNewSpot(spot *protoTypes.SpotProduct) Errors {
	errs := NewErrors()

	if spot == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot", ErrIsRequired)
	}

	if len(spot.BaseAsset) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.base_asset", ErrIsRequired)
	}
	if len(spot.QuoteAsset) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.quote_asset", ErrIsRequired)
	}
	if spot.BaseAsset == spot.QuoteAsset {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.quote_asset", ErrIsNotValid)
	}
	if len(spot.Name) == 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.instrument.product.spot.name", ErrIsRequired)
	}
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

func checkUpdatePerps(future *protoTypes.UpdateFutureProduct) Errors {
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
		if t != nil {
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
		} else {
			return errs.FinalAddForProperty(fmt.Sprintf("%s.%s.internal.time", parentProperty, name), ErrIsRequired)
		}

	case *vegapb.DataSourceDefinition_External:

		switch tp.External.SourceType.(type) {
		case *vegapb.DataSourceDefinitionExternal_Oracle:
			// If data source type is external - check if the signers are present first.
			o := tp.External.GetOracle()
			if o != nil {
				signers := o.Signers
				if len(signers) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle.signers", parentProperty, name), ErrIsRequired)
				}

				for i, key := range signers {
					signer := dstypes.SignerFromProto(key)
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
			} else {
				errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle", parentProperty, name), ErrIsRequired)
			}
		case *vegapb.DataSourceDefinitionExternal_EthOracle:
			ethOracle := tp.External.GetEthOracle()

			if ethOracle != nil {
				if !crypto.EthereumIsValidAddress(ethOracle.Address) {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.address", parentProperty, name), ErrIsNotValidEthereumAddress)
				}

				filters := ethOracle.Filters
				errs.Merge(checkDataSourceSpecFilters(filters, fmt.Sprintf("%s.external.ethoracle", name), parentProperty))

				if len(ethOracle.Abi) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.abi", parentProperty, name), ErrIsRequired)
				}

				if len(strings.TrimSpace(ethOracle.Method)) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.method", parentProperty, name), ErrIsRequired)
				}

				if len(ethOracle.Normalisers) == 0 {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.normalisers", parentProperty, name), ErrIsRequired)
				}

				if ethOracle.Trigger == nil {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.trigger", parentProperty, name), ErrIsRequired)
				}

				if ethOracle.Trigger != nil &&
					ethOracle.Trigger.GetTimeTrigger() != nil &&
					ethOracle.Trigger.GetTimeTrigger().Initial == nil &&
					ethOracle.Trigger.GetTimeTrigger().Every == nil {
					errs.AddForProperty(fmt.Sprintf("%s.%s.external.ethoracle.trigger.timetrigger.(initial|every)", parentProperty, name), ErrIsRequired)
				}
			} else {
				errs.AddForProperty(fmt.Sprintf("%s.%s.external.oracle", parentProperty, name), ErrIsRequired)
			}
		}
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
	if spec == nil {
		return false
	}

	switch specType := spec.SourceType.(type) {
	case *vegapb.DataSourceDefinition_External:
		switch specType.External.SourceType.(type) {
		case *vegapb.DataSourceDefinitionExternal_Oracle:
			return isBindingMatchingSpecFilters(spec, bindingProperty)
		case *vegapb.DataSourceDefinitionExternal_EthOracle:
			ethOracle := specType.External.GetEthOracle()

			isNormaliser := false

			for _, v := range ethOracle.Normalisers {
				if v.Name == bindingProperty {
					isNormaliser = true
				}
			}

			return isNormaliser || isBindingMatchingSpecFilters(spec, bindingProperty)
		}

	case *vegapb.DataSourceDefinition_Internal:
		return isBindingMatchingSpecFilters(spec, bindingProperty)
	}

	return isBindingMatchingSpecFilters(spec, bindingProperty)
}

// This is the legacy oracles way of checking that the spec has a property matching the binding property by iterating
// over the filters, but is it not possible to not have filters, or a filter that does not match the oracle property so
// this would break?
func isBindingMatchingSpecFilters(spec *vegapb.DataSourceDefinition, bindingProperty string) bool {
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

func checkNewPerpsOracleBinding(perps *protoTypes.PerpsProduct) Errors {
	errs := NewErrors()

	if perps.DataSourceSpecBinding != nil {
		if len(perps.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(perps.DataSourceSpecForSettlementData, perps.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("proposal_submission.terms.change.new_market.changes.instrument.product.perps.data_source_spec_binding", ErrIsRequired)
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

func checkUpdatePerpsOracleBinding(perps *protoTypes.UpdatePerpsProduct) Errors {
	errs := NewErrors()
	if perps.DataSourceSpecBinding != nil {
		if len(perps.DataSourceSpecBinding.SettlementDataProperty) == 0 {
			errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsRequired)
		} else {
			if !isBindingMatchingSpec(perps.DataSourceSpecForSettlementData, perps.DataSourceSpecBinding.SettlementDataProperty) {
				errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.perps.data_source_spec_binding.settlement_data_property", ErrIsMismatching)
			}
		}
	} else {
		errs.AddForProperty("proposal_submission.terms.change.update_market.changes.instrument.product.perps.data_source_spec_binding", ErrIsRequired)
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

func checkSLAParams(config *protoTypes.LiquiditySLAParameters, parent string) Errors {
	errs := NewErrors()
	if config == nil {
		return errs.FinalAddForProperty(fmt.Sprintf("%s.sla_params", parent), ErrIsRequired)
	}

	lppr, err := num.DecimalFromString(config.PriceRange)
	if err != nil {
		errs.AddForProperty(fmt.Sprintf("%s.price_range", parent), ErrIsNotValidNumber)
	} else if lppr.IsNegative() || lppr.IsZero() {
		errs.AddForProperty(fmt.Sprintf("%s.price_range", parent), ErrMustBePositive)
	} else if lppr.GreaterThan(num.DecimalFromInt64(100)) {
		errs.AddForProperty(fmt.Sprintf("%s.price_range", parent), ErrMustBeAtMost100)
	}

	commitmentMinTimeFraction, err := num.DecimalFromString(config.CommitmentMinTimeFraction)
	if err != nil {
		errs.AddForProperty(fmt.Sprintf("%s.commitment_min_time_fraction", parent), ErrIsNotValidNumber)
	} else if commitmentMinTimeFraction.IsNegative() || commitmentMinTimeFraction.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty(fmt.Sprintf("%s.commitment_min_time_fraction", parent), ErrMustBeWithinRange01)
	}

	if config.ProvidersFeeCalculationTimeStep == 0 {
		errs.AddForProperty(fmt.Sprintf("%s.providers.fee.calculation_time_step", parent), ErrMustBePositive)
	}

	slaCompetitionFactor, err := num.DecimalFromString(config.SlaCompetitionFactor)
	if err != nil {
		errs.AddForProperty(fmt.Sprintf("%s.sla_competition_factor", parent), ErrIsNotValidNumber)
	} else if slaCompetitionFactor.IsNegative() || slaCompetitionFactor.GreaterThan(num.DecimalOne()) {
		errs.AddForProperty(fmt.Sprintf("%s.sla_competition_factor", parent), ErrMustBeWithinRange01)
	}

	if config.PerformanceHysteresisEpochs < 1 {
		errs.AddForProperty(fmt.Sprintf("%s.performance_hysteresis_epochs", parent), ErrMustBePositive)
	}

	return errs
}

func checkNewSpotRiskParameters(config *protoTypes.NewSpotMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.NewSpotMarketConfiguration_Simple:
		errs.Merge(checkNewSpotSimpleParameters(parameters))
	case *protoTypes.NewSpotMarketConfiguration_LogNormal:
		errs.Merge(checkNewSpotLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters", ErrIsNotValid)
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

func checkUpdateSpotRiskParameters(config *protoTypes.UpdateSpotMarketConfiguration) Errors {
	errs := NewErrors()

	if config.RiskParameters == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters", ErrIsRequired)
	}

	switch parameters := config.RiskParameters.(type) {
	case *protoTypes.UpdateSpotMarketConfiguration_Simple:
		errs.Merge(checkUpdateSpotSimpleParameters(parameters))
	case *protoTypes.UpdateSpotMarketConfiguration_LogNormal:
		errs.Merge(checkUpdateSpotLogNormalRiskParameters(parameters))
	default:
		errs.AddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters", ErrIsNotValid)
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

func checkNewSpotSimpleParameters(params *protoTypes.NewSpotMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.simple.probability_of_trading",
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

func checkUpdateSpotSimpleParameters(params *protoTypes.UpdateSpotMarketConfiguration_Simple) Errors {
	errs := NewErrors()

	if params.Simple == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple", ErrIsRequired)
	}

	if params.Simple.MinMoveDown > 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.min_move_down", ErrMustBeNegativeOrZero)
	}

	if params.Simple.MaxMoveUp < 0 {
		errs.AddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.max_move_up", ErrMustBePositiveOrZero)
	}

	if params.Simple.ProbabilityOfTrading < 0 || params.Simple.ProbabilityOfTrading > 1 {
		errs.AddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.simple.probability_of_trading",
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

func checkNewSpotLogNormalRiskParameters(params *protoTypes.NewSpotMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter < 1e-8 || params.LogNormal.RiskAversionParameter > 0.1 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter", errors.New("must be between [1e-8, 0.1]"))
	}

	if params.LogNormal.Tau < 1e-8 || params.LogNormal.Tau > 1 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.tau", errors.New("must be between [1e-8, 1]"))
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Mu < -1e-6 || params.LogNormal.Params.Mu > 1e-6 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.mu", errors.New("must be between [-1e-6,1e-6]"))
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma < 1e-3 || params.LogNormal.Params.Sigma > 50 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.sigma", errors.New("must be between [1e-3,50]"))
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.R < -1 || params.LogNormal.Params.R > 1 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.new_spot_market.changes.risk_parameters.log_normal.params.r", errors.New("must be between [-1,1]"))
	}

	return errs
}

func checkUpdateSpotLogNormalRiskParameters(params *protoTypes.UpdateSpotMarketConfiguration_LogNormal) Errors {
	errs := NewErrors()

	if params.LogNormal == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal", ErrIsRequired)
	}

	if params.LogNormal.Params == nil {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params", ErrIsRequired)
	}

	if params.LogNormal.RiskAversionParameter <= 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.risk_aversion_parameter", ErrMustBePositive)
	}

	if params.LogNormal.Tau <= 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.tau", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.Mu) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.mu", ErrIsNotValidNumber)
	}

	if math.IsNaN(params.LogNormal.Params.Sigma) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.sigma", ErrIsNotValidNumber)
	}

	if params.LogNormal.Params.Sigma <= 0 {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.sigma", ErrMustBePositive)
	}

	if math.IsNaN(params.LogNormal.Params.R) {
		return errs.FinalAddForProperty("proposal_submission.terms.change.update_spot_market.changes.risk_parameters.log_normal.params.r", ErrIsNotValidNumber)
	}

	return errs
}
