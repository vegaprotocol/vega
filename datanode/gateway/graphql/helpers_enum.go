// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql

import (
	"fmt"

	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	types "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
	oraclesv1 "code.vegaprotocol.io/vega/protos/vega/oracles/v1"
)

func convertAssetStatusFromProto(s types.Asset_Status) (AssetStatus, error) {
	switch s {
	case types.Asset_STATUS_PROPOSED:
		return AssetStatusProposed, nil
	case types.Asset_STATUS_REJECTED:
		return AssetStatusRejected, nil
	case types.Asset_STATUS_PENDING_LISTING:
		return AssetStatusPendingListing, nil
	case types.Asset_STATUS_ENABLED:
		return AssetStatusEnabled, nil
	default:
		err := fmt.Errorf("failed to convert AssetStatus from Proto to GraphQL: %v", s)
		return AssetStatus(""), err
	}
}

func convertTransferStatusFromProto(x eventspb.Transfer_Status) (TransferStatus, error) {
	switch x {
	case eventspb.Transfer_STATUS_PENDING:
		return TransferStatusPending, nil
	case eventspb.Transfer_STATUS_DONE:
		return TransferStatusDone, nil
	case eventspb.Transfer_STATUS_STOPPED:
		return TransferStatusStopped, nil
	case eventspb.Transfer_STATUS_REJECTED:
		return TransferStatusRejected, nil
	case eventspb.Transfer_STATUS_CANCELLED:
		return TransferStatusCancelled, nil
	default:
		err := fmt.Errorf("failed to convert TransferStatus from GraphQL to Proto: %v", x)
		return TransferStatusDone, err
	}
}

func convertStakeLinkingTypeFromProto(
	s eventspb.StakeLinking_Type,
) (StakeLinkingType, error) {
	switch s {
	case eventspb.StakeLinking_TYPE_LINK:
		return StakeLinkingTypeLink, nil
	case eventspb.StakeLinking_TYPE_UNLINK:
		return StakeLinkingTypeUnlink, nil
	default:
		err := fmt.Errorf("failed to convert StakeLinkingType from Proto to GraphQL: %v", s)
		return StakeLinkingType(""), err
	}
}

func convertStakeLinkingStatusFromProto(
	s eventspb.StakeLinking_Status,
) (StakeLinkingStatus, error) {
	switch s {
	case eventspb.StakeLinking_STATUS_PENDING:
		return StakeLinkingStatusPending, nil
	case eventspb.StakeLinking_STATUS_ACCEPTED:
		return StakeLinkingStatusAccepted, nil
	case eventspb.StakeLinking_STATUS_REJECTED:
		return StakeLinkingStatusRejected, nil
	default:
		err := fmt.Errorf("failed to convert StakeLinkingStatus from Proto to GraphQL: %v", s)
		return StakeLinkingStatus(""), err
	}
}

func convertOracleSpecStatusFromProto(s oraclesv1.OracleSpec_Status) (OracleSpecStatus, error) {
	switch s {
	case oraclesv1.OracleSpec_STATUS_ACTIVE:
		return OracleSpecStatusStatusActive, nil
	case oraclesv1.OracleSpec_STATUS_DEACTIVATED:
		return OracleSpecStatusStatusUnused, nil
	default:
		err := fmt.Errorf("failed to convert OracleSpecStatus from Proto to GraphQL: %v", s)
		return OracleSpecStatusStatusUnused, err
	}
}

func convertPropertyKeyTypeFromProto(t oraclesv1.PropertyKey_Type) (PropertyKeyType, error) {
	switch t {
	case oraclesv1.PropertyKey_TYPE_EMPTY:
		return PropertyKeyTypeTypeEmpty, nil
	case oraclesv1.PropertyKey_TYPE_INTEGER:
		return PropertyKeyTypeTypeInteger, nil
	case oraclesv1.PropertyKey_TYPE_DECIMAL:
		return PropertyKeyTypeTypeDecimal, nil
	case oraclesv1.PropertyKey_TYPE_BOOLEAN:
		return PropertyKeyTypeTypeBoolean, nil
	case oraclesv1.PropertyKey_TYPE_TIMESTAMP:
		return PropertyKeyTypeTypeTimestamp, nil
	case oraclesv1.PropertyKey_TYPE_STRING:
		return PropertyKeyTypeTypeString, nil
	default:
		err := fmt.Errorf("failed to convert PropertyKeyType from Proto to GraphQL: %v", t)
		return PropertyKeyTypeTypeEmpty, err
	}
}

func convertConditionOperatorFromProto(o oraclesv1.Condition_Operator) (ConditionOperator, error) {
	switch o {
	case oraclesv1.Condition_OPERATOR_EQUALS:
		return ConditionOperatorOperatorEquals, nil
	case oraclesv1.Condition_OPERATOR_GREATER_THAN:
		return ConditionOperatorOperatorGreaterThan, nil
	case oraclesv1.Condition_OPERATOR_GREATER_THAN_OR_EQUAL:
		return ConditionOperatorOperatorGreaterThanOrEqual, nil
	case oraclesv1.Condition_OPERATOR_LESS_THAN:
		return ConditionOperatorOperatorLessThan, nil
	case oraclesv1.Condition_OPERATOR_LESS_THAN_OR_EQUAL:
		return ConditionOperatorOperatorLessThanOrEqual, nil
	default:
		err := fmt.Errorf("failed to convert ConditionOperator from Proto to GraphQL: %v", o)
		return ConditionOperatorOperatorEquals, err
	}
}

func convertLiquidityProvisionStatusFromProto(x types.LiquidityProvision_Status) (LiquidityProvisionStatus, error) {
	switch x {
	case types.LiquidityProvision_STATUS_ACTIVE:
		return LiquidityProvisionStatusActive, nil
	case types.LiquidityProvision_STATUS_STOPPED:
		return LiquidityProvisionStatusStopped, nil
	case types.LiquidityProvision_STATUS_CANCELLED:
		return LiquidityProvisionStatusCancelled, nil
	case types.LiquidityProvision_STATUS_REJECTED:
		return LiquidityProvisionStatusRejected, nil
	case types.LiquidityProvision_STATUS_UNDEPLOYED:
		return LiquidityProvisionStatusUndeployed, nil
	case types.LiquidityProvision_STATUS_PENDING:
		return LiquidityProvisionStatusPending, nil
	default:
		err := fmt.Errorf("failed to convert LiquidityProvisionStatus from GraphQL to Proto: %v", x)
		return LiquidityProvisionStatusActive, err
	}
}

func convertDepositStatusFromProto(x types.Deposit_Status) (DepositStatus, error) {
	switch x {
	case types.Deposit_STATUS_OPEN:
		return DepositStatusOpen, nil
	case types.Deposit_STATUS_CANCELLED:
		return DepositStatusCancelled, nil
	case types.Deposit_STATUS_FINALIZED:
		return DepositStatusFinalized, nil
	default:
		err := fmt.Errorf("failed to convert DepositStatus from GraphQL to Proto: %v", x)
		return DepositStatusOpen, err
	}
}

func convertWithdrawalStatusFromProto(x types.Withdrawal_Status) (WithdrawalStatus, error) {
	switch x {
	case types.Withdrawal_STATUS_OPEN:
		return WithdrawalStatusOpen, nil
	case types.Withdrawal_STATUS_REJECTED:
		return WithdrawalStatusRejected, nil
	case types.Withdrawal_STATUS_FINALIZED:
		return WithdrawalStatusFinalized, nil
	default:
		err := fmt.Errorf("failed to convert WithdrawalStatus from GraphQL to Proto: %v", x)
		return WithdrawalStatusOpen, err
	}
}

func convertNodeSignatureKindFromProto(x commandspb.NodeSignatureKind) (NodeSignatureKind, error) {
	switch x {
	case commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW:
		return NodeSignatureKindAssetNew, nil
	case commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL:
		return NodeSignatureKindAssetWithdrawal, nil
	default:
		err := fmt.Errorf("failed to convert NodeSignatureKind from proto to graphql: %v", x)
		return NodeSignatureKindAssetNew, err
	}
}

// convertIntervalToProto converts a GraphQL enum to a Proto enum.
func convertIntervalToProto(x Interval) (types.Interval, error) {
	switch x {
	case IntervalI1m:
		return types.Interval_INTERVAL_I1M, nil
	case IntervalI5m:
		return types.Interval_INTERVAL_I5M, nil
	case IntervalI15m:
		return types.Interval_INTERVAL_I15M, nil
	case IntervalI1h:
		return types.Interval_INTERVAL_I1H, nil
	case IntervalI6h:
		return types.Interval_INTERVAL_I6H, nil
	case IntervalI1d:
		return types.Interval_INTERVAL_I1D, nil
	default:
		err := fmt.Errorf("failed to convert Interval from GraphQL to Proto: %v", x)
		return types.Interval_INTERVAL_UNSPECIFIED, err
	}
}

func convertDataNodeIntervalToProto(interval string) (types.Interval, error) {
	switch interval {
	case "1 minute":
		return types.Interval_INTERVAL_I1M, nil
	case "5 minutes":
		return types.Interval_INTERVAL_I5M, nil
	case "15 minutes":
		return types.Interval_INTERVAL_I15M, nil
	case "1 hour":
		return types.Interval_INTERVAL_I1H, nil
	case "6 hours":
		return types.Interval_INTERVAL_I6H, nil
	case "1 day":
		return types.Interval_INTERVAL_I1D, nil
	default:
		err := fmt.Errorf("failed to convert Interval from GraphQL to Proto: %v", interval)
		return types.Interval_INTERVAL_UNSPECIFIED, err
	}
}

// convertIntervalFromProto converts a Proto enum to a GraphQL enum.
func convertIntervalFromProto(x types.Interval) (Interval, error) {
	switch x {
	case types.Interval_INTERVAL_I1M:
		return IntervalI1m, nil
	case types.Interval_INTERVAL_I5M:
		return IntervalI5m, nil
	case types.Interval_INTERVAL_I15M:
		return IntervalI15m, nil
	case types.Interval_INTERVAL_I1H:
		return IntervalI1h, nil
	case types.Interval_INTERVAL_I6H:
		return IntervalI6h, nil
	case types.Interval_INTERVAL_I1D:
		return IntervalI1d, nil
	default:
		err := fmt.Errorf("failed to convert Interval from Proto to GraphQL: %v", x)
		return IntervalI15m, err
	}
}

// convertOrderStatusToProto converts a GraphQL enum to a Proto enum.
func convertOrderStatusToProto(x OrderStatus) (types.Order_Status, error) {
	switch x {
	case OrderStatusActive:
		return types.Order_STATUS_ACTIVE, nil
	case OrderStatusExpired:
		return types.Order_STATUS_EXPIRED, nil
	case OrderStatusCancelled:
		return types.Order_STATUS_CANCELLED, nil
	case OrderStatusStopped:
		return types.Order_STATUS_STOPPED, nil
	case OrderStatusFilled:
		return types.Order_STATUS_FILLED, nil
	case OrderStatusRejected:
		return types.Order_STATUS_REJECTED, nil
	case OrderStatusPartiallyFilled:
		return types.Order_STATUS_PARTIALLY_FILLED, nil
	case OrderStatusParked:
		return types.Order_STATUS_PARKED, nil
	default:
		err := fmt.Errorf("failed to convert OrderStatus from GraphQL to Proto: %v", x)
		return types.Order_STATUS_UNSPECIFIED, err
	}
}

// convertOrderStatusFromProto converts a Proto enum to a GraphQL enum.
func convertOrderStatusFromProto(x types.Order_Status) (OrderStatus, error) {
	switch x {
	case types.Order_STATUS_ACTIVE:
		return OrderStatusActive, nil
	case types.Order_STATUS_EXPIRED:
		return OrderStatusExpired, nil
	case types.Order_STATUS_CANCELLED:
		return OrderStatusCancelled, nil
	case types.Order_STATUS_STOPPED:
		return OrderStatusStopped, nil
	case types.Order_STATUS_FILLED:
		return OrderStatusFilled, nil
	case types.Order_STATUS_REJECTED:
		return OrderStatusRejected, nil
	case types.Order_STATUS_PARTIALLY_FILLED:
		return OrderStatusPartiallyFilled, nil
	case types.Order_STATUS_PARKED:
		return OrderStatusParked, nil
	default:
		err := fmt.Errorf("failed to convert OrderStatus from Proto to GraphQL: %v", x)
		return OrderStatusActive, err
	}
}

// convertOrderTypeToProto converts a GraphQL enum to a Proto enum.
func convertOrderTypeToProto(x OrderType) (types.Order_Type, error) {
	switch x {
	case OrderTypeLimit:
		return types.Order_TYPE_LIMIT, nil
	case OrderTypeMarket:
		return types.Order_TYPE_MARKET, nil
	case OrderTypeNetwork:
		return types.Order_TYPE_NETWORK, nil
	default:
		err := fmt.Errorf("failed to convert OrderType from GraphQL to Proto: %v", x)
		return types.Order_TYPE_UNSPECIFIED, err
	}
}

// convertOrderTypeFromProto converts a Proto enum to a GraphQL enum.
func convertOrderTypeFromProto(x types.Order_Type) (OrderType, error) {
	switch x {
	case types.Order_TYPE_LIMIT:
		return OrderTypeLimit, nil
	case types.Order_TYPE_MARKET:
		return OrderTypeMarket, nil
	case types.Order_TYPE_NETWORK:
		return OrderTypeNetwork, nil
	default:
		err := fmt.Errorf("failed to convert OrderType from Proto to GraphQL: %v", x)
		return OrderTypeLimit, err
	}
}

// convertMarketStateFromProto converts a Proto enum to a GraphQL enum.
func convertMarketTradingModeFromProto(ms types.Market_TradingMode) (MarketTradingMode, error) {
	switch ms {
	case types.Market_TRADING_MODE_OPENING_AUCTION:
		return MarketTradingModeOpeningAuction, nil
	case types.Market_TRADING_MODE_BATCH_AUCTION:
		return MarketTradingModeBatchAuction, nil
	case types.Market_TRADING_MODE_MONITORING_AUCTION:
		return MarketTradingModeMonitoringAuction, nil
	case types.Market_TRADING_MODE_CONTINUOUS:
		return MarketTradingModeContinuous, nil
	case types.Market_TRADING_MODE_NO_TRADING:
		return MarketTradingModeNoTrading, nil
	default:
		err := fmt.Errorf("failed to convert MarketTradingMode from Proto to GraphQL: %v", ms)
		return MarketTradingModeContinuous, err
	}
}

// convertMarketStateFromProto converts a Proto enum to a GraphQL enum.
func convertMarketStateFromProto(ms types.Market_State) (MarketState, error) {
	switch ms {
	case types.Market_STATE_PROPOSED:
		return MarketStateProposed, nil
	case types.Market_STATE_REJECTED:
		return MarketStateRejected, nil
	case types.Market_STATE_PENDING:
		return MarketStatePending, nil
	case types.Market_STATE_CANCELLED:
		return MarketStateCancelled, nil
	case types.Market_STATE_ACTIVE:
		return MarketStateActive, nil
	case types.Market_STATE_SUSPENDED:
		return MarketStateSuspended, nil
	case types.Market_STATE_CLOSED:
		return MarketStateClosed, nil
	case types.Market_STATE_TRADING_TERMINATED:
		return MarketStateTradingTerminated, nil
	case types.Market_STATE_SETTLED:
		return MarketStateSettled, nil
	default:
		err := fmt.Errorf("failed to convert MarketMode from Proto to GraphQL: %v", ms)
		return MarketStateActive, err
	}
}

// convertAuctionTriggerFromProto converts a proto enum to GQL enum.
func convertAuctionTriggerFromProto(t types.AuctionTrigger) (AuctionTrigger, error) {
	switch t {
	case types.AuctionTrigger_AUCTION_TRIGGER_UNSPECIFIED:
		return AuctionTriggerUnspecified, nil
	case types.AuctionTrigger_AUCTION_TRIGGER_BATCH:
		return AuctionTriggerBatch, nil
	case types.AuctionTrigger_AUCTION_TRIGGER_OPENING:
		return AuctionTriggerOpening, nil
	case types.AuctionTrigger_AUCTION_TRIGGER_PRICE:
		return AuctionTriggerPrice, nil
	case types.AuctionTrigger_AUCTION_TRIGGER_LIQUIDITY:
		return AuctionTriggerLiquidity, nil
	}
	err := fmt.Errorf("failed to convert AuctionTrigger from proto to GQL: %v", t)
	return AuctionTriggerUnspecified, err
}

// convertProposalStateToProto converts a GraphQL enum to a Proto enum.
func convertProposalStateToProto(x ProposalState) (types.Proposal_State, error) {
	switch x {
	case ProposalStateFailed:
		return types.Proposal_STATE_FAILED, nil
	case ProposalStateOpen:
		return types.Proposal_STATE_OPEN, nil
	case ProposalStatePassed:
		return types.Proposal_STATE_PASSED, nil
	case ProposalStateRejected:
		return types.Proposal_STATE_REJECTED, nil
	case ProposalStateDeclined:
		return types.Proposal_STATE_DECLINED, nil
	case ProposalStateEnacted:
		return types.Proposal_STATE_ENACTED, nil
	case ProposalStateWaitingForNodeVote:
		return types.Proposal_STATE_WAITING_FOR_NODE_VOTE, nil
	default:
		err := fmt.Errorf("failed to convert ProposalState from GraphQL to Proto: %v", x)
		return types.Proposal_STATE_UNSPECIFIED, err
	}
}

// convertProposalStateToProto converts a GraphQL enum to a Proto enum.
func convertProposalTypeToProto(x ProposalType) v2.ListGovernanceDataRequest_Type {
	switch x {
	case ProposalTypeNewMarket:
		return v2.ListGovernanceDataRequest_TYPE_NEW_MARKET
	case ProposalTypeUpdateMarket:
		return v2.ListGovernanceDataRequest_TYPE_UPDATE_MARKET
	case ProposalTypeNetworkParameters:
		return v2.ListGovernanceDataRequest_TYPE_NETWORK_PARAMETERS
	case ProposalTypeNewAsset:
		return v2.ListGovernanceDataRequest_TYPE_NEW_ASSET
	case ProposalTypeNewFreeForm:
		return v2.ListGovernanceDataRequest_TYPE_NEW_FREE_FORM
	default:
		return v2.ListGovernanceDataRequest_TYPE_ALL
	}
}

// convertProposalStateFromProto converts a Proto enum to a GraphQL enum.
func convertProposalStateFromProto(x types.Proposal_State) (ProposalState, error) {
	switch x {
	case types.Proposal_STATE_FAILED:
		return ProposalStateFailed, nil
	case types.Proposal_STATE_OPEN:
		return ProposalStateOpen, nil
	case types.Proposal_STATE_PASSED:
		return ProposalStatePassed, nil
	case types.Proposal_STATE_REJECTED:
		return ProposalStateRejected, nil
	case types.Proposal_STATE_DECLINED:
		return ProposalStateDeclined, nil
	case types.Proposal_STATE_ENACTED:
		return ProposalStateEnacted, nil
	case types.Proposal_STATE_WAITING_FOR_NODE_VOTE:
		return ProposalStateWaitingForNodeVote, nil
	default:
		err := fmt.Errorf("failed to convert ProposalState from Proto to GraphQL: %v", x)
		return ProposalStateFailed, err
	}
}

func convertProposalRejectionReasonFromProto(x types.ProposalError) (ProposalRejectionReason, error) {
	switch x {
	case types.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON:
		return ProposalRejectionReasonCloseTimeTooSoon, nil
	case types.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE:
		return ProposalRejectionReasonCloseTimeTooLate, nil
	case types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON:
		return ProposalRejectionReasonEnactTimeTooSoon, nil
	case types.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE:
		return ProposalRejectionReasonEnactTimeTooLate, nil
	case types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS:
		return ProposalRejectionReasonInsufficientTokens, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY:
		return ProposalRejectionReasonInvalidInstrumentSecurity, nil
	case types.ProposalError_PROPOSAL_ERROR_NO_PRODUCT:
		return ProposalRejectionReasonNoProduct, nil
	case types.ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT:
		return ProposalRejectionReasonUnsupportedProduct, nil
	case types.ProposalError_PROPOSAL_ERROR_NO_TRADING_MODE:
		return ProposalRejectionReasonNoTradingMode, nil
	case types.ProposalError_PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE:
		return ProposalRejectionReasonUnsupportedTradingMode, nil
	case types.ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED:
		return ProposalRejectionReasonNodeValidationFailed, nil
	case types.ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD:
		return ProposalRejectionReasonMissingBuiltinAssetField, nil
	case types.ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS:
		return ProposalRejectionReasonMissingERC20ContractAddress, nil
	case types.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS:
		return ProposalRejectionReasonIncompatibleTimestamps, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_ASSET:
		return ProposalRejectionReasonInvalidAsset, nil
	case types.ProposalError_PROPOSAL_ERROR_NO_RISK_PARAMETERS:
		return ProposalRejectionReasonNoRiskParameters, nil
	case types.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY:
		return ProposalRejectionReasonNetworkParameterInvalidKey, nil
	case types.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE:
		return ProposalRejectionReasonNetworkParameterInvalidValue, nil
	case types.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED:
		return ProposalRejectionReasonNetworkParameterValidationFailed, nil
	case types.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL:
		return ProposalRejectionReasonOpeningAuctionDurationTooSmall, nil
	case types.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE:
		return ProposalRejectionReasonOpeningAuctionDurationTooLarge, nil
	case types.ProposalError_PROPOSAL_ERROR_MARKET_MISSING_LIQUIDITY_COMMITMENT:
		return ProposalRejectionReasonMarketMissingLiquidityCommitment, nil
	case types.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET:
		return ProposalRejectionReasonCouldNotInstantiateMarket, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT:
		return ProposalRejectionReasonInvalidFutureProduct, nil
	case types.ProposalError_PROPOSAL_ERROR_MISSING_COMMITMENT_AMOUNT:
		return ProposalRejectionReasonMissingCommitmentAmount, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_FEE_AMOUNT:
		return ProposalRejectionReasonInvalidFeeAmount, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_SHAPE:
		return ProposalRejectionReasonInvalidShape, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_RISK_PARAMETER:
		return ProposalRejectionReasonInvalidRiskParameter, nil
	case types.ProposalError_PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED:
		return ProposalRejectionReasonMajorityThresholdNotReached, nil
	case types.ProposalError_PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED:
		return ProposalRejectionReasonParticipationThresholdNotReached, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS:
		return ProposalRejectionReasonInvalidAssetDetails, nil
	case types.ProposalError_PROPOSAL_ERROR_TOO_MANY_PRICE_MONITORING_TRIGGERS:
		return ProposalRejectionReasonTooManyPriceMonitoringTriggers, nil
	case types.ProposalError_PROPOSAL_ERROR_TOO_MANY_MARKET_DECIMAL_PLACES:
		return ProposalRejectionReasonTooManyMarketDecimalPlaces, nil
	case types.ProposalError_PROPOSAL_ERROR_INVALID_MARKET:
		return ProposalRejectionReasonInvalidMarket, nil
	case types.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_EQUITY_LIKE_SHARE:
		return ProposalRejectionReasonInsufficientEquityLikeShare, nil
	default:
		err := fmt.Errorf("failed to convert ProposalRejectionReason from Proto to GraphQL: %v", x)
		return "", err
	}
}

// convertRejectionReasonFromProto converts a Proto enum to a GraphQL enum.
func convertOrderRejectionReasonFromProto(x types.OrderError) (OrderRejectionReason, error) {
	switch x {
	case types.OrderError_ORDER_ERROR_INVALID_MARKET_ID:
		return OrderRejectionReasonInvalidMarketID, nil
	case types.OrderError_ORDER_ERROR_INVALID_ORDER_ID:
		return OrderRejectionReasonInvalidOrderID, nil
	case types.OrderError_ORDER_ERROR_OUT_OF_SEQUENCE:
		return OrderRejectionReasonOrderOutOfSequence, nil
	case types.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE:
		return OrderRejectionReasonInvalidRemainingSize, nil
	case types.OrderError_ORDER_ERROR_TIME_FAILURE:
		return OrderRejectionReasonTimeFailure, nil
	case types.OrderError_ORDER_ERROR_REMOVAL_FAILURE:
		return OrderRejectionReasonOrderRemovalFailure, nil
	case types.OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME:
		return OrderRejectionReasonInvalidExpirationTime, nil
	case types.OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE:
		return OrderRejectionReasonInvalidOrderReference, nil
	case types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED:
		return OrderRejectionReasonEditNotAllowed, nil
	case types.OrderError_ORDER_ERROR_AMEND_FAILURE:
		return OrderRejectionReasonOrderAmendFailure, nil
	case types.OrderError_ORDER_ERROR_NOT_FOUND:
		return OrderRejectionReasonOrderNotFound, nil
	case types.OrderError_ORDER_ERROR_INVALID_PARTY_ID:
		return OrderRejectionReasonInvalidPartyID, nil
	case types.OrderError_ORDER_ERROR_MARKET_CLOSED:
		return OrderRejectionReasonMarketClosed, nil
	case types.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED:
		return OrderRejectionReasonMarginCheckFailed, nil
	case types.OrderError_ORDER_ERROR_SELF_TRADING:
		return OrderRejectionReasonSelfTrading, nil
	case types.OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES:
		return OrderRejectionReasonInsufficientFundsToPayFees, nil
	case types.OrderError_ORDER_ERROR_INTERNAL_ERROR:
		return OrderRejectionReasonInternalError, nil
	case types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT:
		return OrderRejectionReasonAmendToGTTWithoutExpiryAt, nil
	case types.OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT:
		return OrderRejectionReasonExpiryAtBeforeCreatedAt, nil
	case types.OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT:
		return OrderRejectionReasonGTCWithExpiryAtNotValid, nil
	case types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC:
		return OrderRejectionReasonCannotAmendToFOKOrIoc, nil
	case types.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN:
		return OrderRejectionReasonCannotAmendToGFAOrGfn, nil
	case types.OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN:
		return OrderRejectionReasonCannotAmendFromGFAOrGfn, nil
	case types.OrderError_ORDER_ERROR_INCORRECT_MARKET_TYPE:
		return OrderRejectionReasonInvalidMarketType, nil
	case types.OrderError_ORDER_ERROR_GFA_ORDER_DURING_CONTINUOUS_TRADING:
		return OrderRejectionReasonGFAOrderDuringContinuousTrading, nil
	case types.OrderError_ORDER_ERROR_GFN_ORDER_DURING_AN_AUCTION:
		return OrderRejectionReasonGFNOrderDuringAuction, nil
	case types.OrderError_ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION:
		return OrderRejectionReasonIOCOrderDuringAuction, nil
	case types.OrderError_ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION:
		return OrderRejectionReasonFOKOrderDuringAuction, nil
	case types.OrderError_ORDER_ERROR_MUST_BE_LIMIT_ORDER:
		return OrderRejectionReasonPeggedOrderMustBeLimitOrder, nil
	case types.OrderError_ORDER_ERROR_MUST_BE_GTT_OR_GTC:
		return OrderRejectionReasonPeggedOrderMustBeGTTOrGtc, nil
	case types.OrderError_ORDER_ERROR_WITHOUT_REFERENCE_PRICE:
		return OrderRejectionReasonPeggedOrderWithoutReferencePrice, nil
	case types.OrderError_ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE:
		return OrderRejectionReasonPeggedOrderBuyCannotReferenceBestAskPrice, nil
	case types.OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO:
		return OrderRejectionReasonPeggedOrderOffsetMustBeGreaterOrEqualToZero, nil
	case types.OrderError_ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE:
		return OrderRejectionReasonPeggedOrderSellCannotReferenceBestBidPrice, nil
	case types.OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO:
		return OrderRejectionReasonPeggedOrderOffsetMustBeGreaterThanZero, nil
	case types.OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE:
		return OrderRejectionReasonInsufficientAssetBalance, nil
	case types.OrderError_ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER:
		return OrderRejectionReasonCannotAmendPeggedOrderDetailsOnNonPeggedOrder, nil
	case types.OrderError_ORDER_ERROR_UNABLE_TO_REPRICE_PEGGED_ORDER:
		return OrderRejectionReasonUnableToRepricePeggedOrder, nil
	case types.OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE:
		return OrderRejectionReasonInvalidTimeInForce, nil
	case types.OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER:
		return OrderRejectionReasonUnableToAmendPeggedOrderPrice, nil
	case types.OrderError_ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS:
		return OrderRejectionReasonNonPersistentOrderExceedsPriceBounds, nil
	default:
		err := fmt.Errorf("failed to convert OrderRejectionReason from Proto to GraphQL: %v", x)
		return OrderRejectionReasonInternalError, err
	}
}

// convertSideToProto converts a GraphQL enum to a Proto enum.
func convertSideToProto(x Side) (types.Side, error) {
	switch x {
	case SideBuy:
		return types.Side_SIDE_BUY, nil
	case SideSell:
		return types.Side_SIDE_SELL, nil
	default:
		err := fmt.Errorf("failed to convert Side from GraphQL to Proto: %v", x)
		return types.Side_SIDE_UNSPECIFIED, err
	}
}

// convertSideFromProto converts a Proto enum to a GraphQL enum.
func convertSideFromProto(x types.Side) (Side, error) {
	switch x {
	case types.Side_SIDE_BUY:
		return SideBuy, nil
	case types.Side_SIDE_SELL:
		return SideSell, nil
	default:
		err := fmt.Errorf("failed to convert Side from Proto to GraphQL: %v", x)
		return SideBuy, err
	}
}

func convertPeggedReferenceFromProto(x types.PeggedReference) (PeggedReference, error) {
	switch x {
	case types.PeggedReference_PEGGED_REFERENCE_MID:
		return PeggedReferenceMid, nil
	case types.PeggedReference_PEGGED_REFERENCE_BEST_BID:
		return PeggedReferenceBestBid, nil
	case types.PeggedReference_PEGGED_REFERENCE_BEST_ASK:
		return PeggedReferenceBestAsk, nil
	default:
		err := fmt.Errorf("failed to convert PeggedReference from Proto to GraphQL: %v", x)
		return PeggedReferenceMid, err
	}
}

// convertOrderTimeInForceToProto converts a GraphQL enum to a Proto enum.
func convertOrderTimeInForceToProto(x OrderTimeInForce) (types.Order_TimeInForce, error) {
	switch x {
	case OrderTimeInForceFok:
		return types.Order_TIME_IN_FORCE_FOK, nil
	case OrderTimeInForceIoc:
		return types.Order_TIME_IN_FORCE_IOC, nil
	case OrderTimeInForceGtc:
		return types.Order_TIME_IN_FORCE_GTC, nil
	case OrderTimeInForceGtt:
		return types.Order_TIME_IN_FORCE_GTT, nil
	case OrderTimeInForceGfa:
		return types.Order_TIME_IN_FORCE_GFA, nil
	case OrderTimeInForceGfn:
		return types.Order_TIME_IN_FORCE_GFN, nil
	default:
		err := fmt.Errorf("failed to convert OrderTimeInForce from GraphQL to Proto: %v", x)
		return types.Order_TIME_IN_FORCE_UNSPECIFIED, err
	}
}

// convertOrderTimeInForceFromProto converts a Proto enum to a GraphQL enum.
func convertOrderTimeInForceFromProto(x types.Order_TimeInForce) (OrderTimeInForce, error) {
	switch x {
	case types.Order_TIME_IN_FORCE_FOK:
		return OrderTimeInForceFok, nil
	case types.Order_TIME_IN_FORCE_IOC:
		return OrderTimeInForceIoc, nil
	case types.Order_TIME_IN_FORCE_GTC:
		return OrderTimeInForceGtc, nil
	case types.Order_TIME_IN_FORCE_GTT:
		return OrderTimeInForceGtt, nil
	case types.Order_TIME_IN_FORCE_GFA:
		return OrderTimeInForceGfa, nil
	case types.Order_TIME_IN_FORCE_GFN:
		return OrderTimeInForceGfn, nil
	default:
		err := fmt.Errorf("failed to convert OrderTimeInForce from Proto to GraphQL: %v", x)
		return OrderTimeInForceGtc, err
	}
}

// convertTradeTypeFromProto converts a Proto enum to a GraphQL enum.
func convertTradeTypeFromProto(x types.Trade_Type) (TradeType, error) {
	switch x {
	case types.Trade_TYPE_DEFAULT:
		return TradeTypeDefault, nil
	case types.Trade_TYPE_NETWORK_CLOSE_OUT_BAD:
		return TradeTypeNetworkCloseOutBad, nil
	case types.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD:
		return TradeTypeNetworkCloseOutGood, nil
	default:
		err := fmt.Errorf("failed to convert TradeType from Proto to GraphQL: %v", x)
		return TradeTypeDefault, err
	}
}

// convertVoteValueFromProto converts a Proto enum to a GraphQL enum.
func convertVoteValueFromProto(x types.Vote_Value) (VoteValue, error) {
	switch x {
	case types.Vote_VALUE_NO:
		return VoteValueNo, nil
	case types.Vote_VALUE_YES:
		return VoteValueYes, nil
	default:
		err := fmt.Errorf("failed to convert VoteValue from Proto to GraphQL: %v", x)
		return VoteValueNo, err
	}
}
