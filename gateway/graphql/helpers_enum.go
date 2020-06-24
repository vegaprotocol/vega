package gql

import (
	"fmt"

	types "code.vegaprotocol.io/vega/proto"
)

// convertAccountTypeToProto converts a GraphQL enum to a Proto enum
func convertAccountTypeToProto(x AccountType) (types.AccountType, error) {
	switch x {
	case AccountTypeGeneral:
		return types.AccountType_ACCOUNT_TYPE_GENERAL, nil
	case AccountTypeInsurance:
		return types.AccountType_ACCOUNT_TYPE_INSURANCE, nil
	case AccountTypeMargin:
		return types.AccountType_ACCOUNT_TYPE_MARGIN, nil
	case AccountTypeSettlement:
		return types.AccountType_ACCOUNT_TYPE_SETTLEMENT, nil
	default:
		err := fmt.Errorf("failed to convert AccountType from GraphQL to Proto: %v", x)
		return types.AccountType_ACCOUNT_TYPE_UNSPECIFIED, err
	}
}

// convertAccountTypeFromProto converts a Proto enum to a GraphQL enum
func convertAccountTypeFromProto(x types.AccountType) (AccountType, error) {
	switch x {
	case types.AccountType_ACCOUNT_TYPE_GENERAL:
		return AccountTypeGeneral, nil
	case types.AccountType_ACCOUNT_TYPE_INSURANCE:
		return AccountTypeInsurance, nil
	case types.AccountType_ACCOUNT_TYPE_MARGIN:
		return AccountTypeMargin, nil
	case types.AccountType_ACCOUNT_TYPE_SETTLEMENT:
		return AccountTypeSettlement, nil
	default:
		err := fmt.Errorf("failed to convert AccountType from Proto to GraphQL: %v", x)
		return AccountTypeGeneral, err
	}
}

// convertIntervalToProto converts a GraphQL enum to a Proto enum
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

// convertIntervalFromProto converts a Proto enum to a GraphQL enum
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

// convertOrderStatusToProto converts a GraphQL enum to a Proto enum
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
	default:
		err := fmt.Errorf("failed to convert OrderStatus from GraphQL to Proto: %v", x)
		return types.Order_STATUS_INVALID, err
	}
}

// convertOrderStatusFromProto converts a Proto enum to a GraphQL enum
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
	default:
		err := fmt.Errorf("failed to convert OrderStatus from Proto to GraphQL: %v", x)
		return OrderStatusActive, err
	}
}

// convertOrderTypeToProto converts a GraphQL enum to a Proto enum
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

// convertOrderTypeFromProto converts a Proto enum to a GraphQL enum
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

// convertProposalStateToProto converts a GraphQL enum to a Proto enum
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
	default:
		err := fmt.Errorf("failed to convert ProposalState from GraphQL to Proto: %v", x)
		return types.Proposal_STATE_UNSPECIFIED, err
	}
}

// convertProposalStateFromProto converts a Proto enum to a GraphQL enum
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
	default:
		err := fmt.Errorf("failed to convert ProposalState from Proto to GraphQL: %v", x)
		return ProposalStateFailed, err
	}
}

// convertRejectionReasonToProto converts a GraphQL enum to a Proto enum
func convertRejectionReasonToProto(x RejectionReason) (types.OrderError, error) {
	switch x {
	case RejectionReasonInvalidMarketID:
		return types.OrderError_ORDER_ERROR_INVALID_MARKET_ID, nil
	case RejectionReasonInvalidOrderID:
		return types.OrderError_ORDER_ERROR_INVALID_ORDER_ID, nil
	case RejectionReasonOrderOutOfSequence:
		return types.OrderError_ORDER_ERROR_OUT_OF_SEQUENCE, nil
	case RejectionReasonInvalidRemainingSize:
		return types.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE, nil
	case RejectionReasonTimeFailure:
		return types.OrderError_ORDER_ERROR_TIME_FAILURE, nil
	case RejectionReasonOrderRemovalFailure:
		return types.OrderError_ORDER_ERROR_REMOVAL_FAILURE, nil
	case RejectionReasonInvalidExpirationTime:
		return types.OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME, nil
	case RejectionReasonInvalidOrderReference:
		return types.OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE, nil
	case RejectionReasonEditNotAllowed:
		return types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED, nil
	case RejectionReasonOrderAmendFailure:
		return types.OrderError_ORDER_ERROR_AMEND_FAILURE, nil
	case RejectionReasonOrderNotFound:
		return types.OrderError_ORDER_ERROR_NOT_FOUND, nil
	case RejectionReasonInvalidPartyID:
		return types.OrderError_ORDER_ERROR_INVALID_PARTY_ID, nil
	case RejectionReasonMarketClosed:
		return types.OrderError_ORDER_ERROR_MARKET_CLOSED, nil
	case RejectionReasonMarginCheckFailed:
		return types.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED, nil
	case RejectionReasonInternalError:
		return types.OrderError_ORDER_ERROR_INTERNAL_ERROR, nil
	default:
		err := fmt.Errorf("failed to convert RejectionReason from GraphQL to Proto: %v", x)
		return types.OrderError_ORDER_ERROR_INTERNAL_ERROR, err
	}
}

// convertRejectionReasonFromProto converts a Proto enum to a GraphQL enum
func convertRejectionReasonFromProto(x types.OrderError) (RejectionReason, error) {
	switch x {
	case types.OrderError_ORDER_ERROR_INVALID_MARKET_ID:
		return RejectionReasonInvalidMarketID, nil
	case types.OrderError_ORDER_ERROR_INVALID_ORDER_ID:
		return RejectionReasonInvalidOrderID, nil
	case types.OrderError_ORDER_ERROR_OUT_OF_SEQUENCE:
		return RejectionReasonOrderOutOfSequence, nil
	case types.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE:
		return RejectionReasonInvalidRemainingSize, nil
	case types.OrderError_ORDER_ERROR_TIME_FAILURE:
		return RejectionReasonTimeFailure, nil
	case types.OrderError_ORDER_ERROR_REMOVAL_FAILURE:
		return RejectionReasonOrderRemovalFailure, nil
	case types.OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME:
		return RejectionReasonInvalidExpirationTime, nil
	case types.OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE:
		return RejectionReasonInvalidOrderReference, nil
	case types.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED:
		return RejectionReasonEditNotAllowed, nil
	case types.OrderError_ORDER_ERROR_AMEND_FAILURE:
		return RejectionReasonOrderAmendFailure, nil
	case types.OrderError_ORDER_ERROR_NOT_FOUND:
		return RejectionReasonOrderNotFound, nil
	case types.OrderError_ORDER_ERROR_INVALID_PARTY_ID:
		return RejectionReasonInvalidPartyID, nil
	case types.OrderError_ORDER_ERROR_MARKET_CLOSED:
		return RejectionReasonMarketClosed, nil
	case types.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED:
		return RejectionReasonMarginCheckFailed, nil
	case types.OrderError_ORDER_ERROR_INTERNAL_ERROR:
		return RejectionReasonInternalError, nil
	default:
		err := fmt.Errorf("failed to convert RejectionReason from Proto to GraphQL: %v", x)
		return RejectionReasonInternalError, err
	}
}

// convertSideToProto converts a GraphQL enum to a Proto enum
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

// convertSideFromProto converts a Proto enum to a GraphQL enum
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

// convertOrderTimeInForceToProto converts a GraphQL enum to a Proto enum
func convertOrderTimeInForceToProto(x OrderTimeInForce) (types.Order_TimeInForce, error) {
	switch x {
	case OrderTimeInForceFok:
		return types.Order_TIF_FOK, nil
	case OrderTimeInForceIoc:
		return types.Order_TIF_IOC, nil
	case OrderTimeInForceGtc:
		return types.Order_TIF_GTC, nil
	case OrderTimeInForceGtt:
		return types.Order_TIF_GTT, nil
	default:
		err := fmt.Errorf("failed to convert OrderTimeInForce from GraphQL to Proto: %v", x)
		return types.Order_TIF_UNSPECIFIED, err
	}
}

// convertOrderTimeInForceFromProto converts a Proto enum to a GraphQL enum
func convertOrderTimeInForceFromProto(x types.Order_TimeInForce) (OrderTimeInForce, error) {
	switch x {
	case types.Order_TIF_FOK:
		return OrderTimeInForceFok, nil
	case types.Order_TIF_IOC:
		return OrderTimeInForceIoc, nil
	case types.Order_TIF_GTC:
		return OrderTimeInForceGtc, nil
	case types.Order_TIF_GTT:
		return OrderTimeInForceGtt, nil
	default:
		err := fmt.Errorf("failed to convert OrderTimeInForce from Proto to GraphQL: %v", x)
		return OrderTimeInForceGtc, err
	}
}

// convertTradeTypeToProto converts a GraphQL enum to a Proto enum
func convertTradeTypeToProto(x TradeType) (types.Trade_Type, error) {
	switch x {
	case TradeTypeDefault:
		return types.Trade_TYPE_DEFAULT, nil
	case TradeTypeNetworkCloseOutBad:
		return types.Trade_TYPE_NETWORK_CLOSE_OUT_BAD, nil
	case TradeTypeNetworkCloseOutGood:
		return types.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD, nil
	default:
		err := fmt.Errorf("failed to convert TradeType from GraphQL to Proto: %v", x)
		return types.Trade_TYPE_UNSPECIFIED, err
	}
}

// convertTradeTypeFromProto converts a Proto enum to a GraphQL enum
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

// convertVoteValueToProto converts a GraphQL enum to a Proto enum
func convertVoteValueToProto(x VoteValue) (types.Vote_Value, error) {
	switch x {
	case VoteValueNo:
		return types.Vote_VALUE_NO, nil
	case VoteValueYes:
		return types.Vote_VALUE_YES, nil
	default:
		err := fmt.Errorf("failed to convert VoteValue from GraphQL to Proto: %v", x)
		return types.Vote_VALUE_UNSPECIFIED, err
	}
}

// convertVoteValueFromProto converts a Proto enum to a GraphQL enum
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
