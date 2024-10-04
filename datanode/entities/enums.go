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

package entities

import (
	"fmt"

	"code.vegaprotocol.io/vega/protos/vega"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/jackc/pgtype"
)

type DispatchMetric vega.DispatchMetric

const (
	DispatchMetricUnspecified       DispatchMetric = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_UNSPECIFIED)
	DispatchMetricMakerFeePaid                     = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_PAID)
	DispatchMetricMakerFeesReceived                = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED)
	DispatchMetricLPFeesReceived                   = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_LP_FEES_RECEIVED)
	DispatchMetricMarketValue                      = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_MARKET_VALUE)
	DispatchMetricAverageNotional                  = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_NOTIONAL)
	DispatchMetricRelativeReturn                   = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_RELATIVE_RETURN)
	DispatchMetricReturnVolatility                 = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_RETURN_VOLATILITY)
	DispatchMetricValidatorRanking                 = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_VALIDATOR_RANKING)
	DispatchMetricRealisedReturn                   = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_REALISED_RETURN)
	DispatchMetricEligibleEntities                 = DispatchMetric(vega.DispatchMetric_DISPATCH_METRIC_ELIGIBLE_ENTITIES)
)

func (m DispatchMetric) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	mode, ok := vega.DispatchMetric_name[int32(m)]
	if !ok {
		return buf, fmt.Errorf("unknown dispatch metric: %s", mode)
	}
	return append(buf, []byte(mode)...), nil
}

func (m *DispatchMetric) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.DispatchMetric_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown dispatch metric: %s", src)
	}

	*m = DispatchMetric(val)
	return nil
}

type Side = vega.Side

const (
	// Default value, always invalid.
	SideUnspecified Side = vega.Side_SIDE_UNSPECIFIED
	// Buy order.
	SideBuy Side = vega.Side_SIDE_BUY
	// Sell order.
	SideSell Side = vega.Side_SIDE_SELL
)

type TradeType = vega.Trade_Type

const (
	// Default value, always invalid.
	TradeTypeUnspecified TradeType = vega.Trade_TYPE_UNSPECIFIED
	// Normal trading between two parties.
	TradeTypeDefault TradeType = vega.Trade_TYPE_DEFAULT
	// Trading initiated by the network with another party on the book,
	// which helps to zero-out the positions of one or more distressed parties.
	TradeTypeNetworkCloseOutGood TradeType = vega.Trade_TYPE_NETWORK_CLOSE_OUT_GOOD
	// Trading initiated by the network with another party off the book,
	// with a distressed party in order to zero-out the position of the party.
	TradeTypeNetworkCloseOutBad TradeType = vega.Trade_TYPE_NETWORK_CLOSE_OUT_BAD
)

type PeggedReference = vega.PeggedReference

const (
	// Default value for PeggedReference, no reference given.
	PeggedReferenceUnspecified PeggedReference = vega.PeggedReference_PEGGED_REFERENCE_UNSPECIFIED
	// Mid price reference.
	PeggedReferenceMid PeggedReference = vega.PeggedReference_PEGGED_REFERENCE_MID
	// Best bid price reference.
	PeggedReferenceBestBid PeggedReference = vega.PeggedReference_PEGGED_REFERENCE_BEST_BID
	// Best ask price reference.
	PeggedReferenceBestAsk PeggedReference = vega.PeggedReference_PEGGED_REFERENCE_BEST_ASK
)

type OrderStatus = vega.Order_Status

const (
	// Default value, always invalid.
	OrderStatusUnspecified OrderStatus = vega.Order_STATUS_UNSPECIFIED
	// Used for active unfilled or partially filled orders.
	OrderStatusActive OrderStatus = vega.Order_STATUS_ACTIVE
	// Used for expired GTT orders.
	OrderStatusExpired OrderStatus = vega.Order_STATUS_EXPIRED
	// Used for orders cancelled by the party that created the order.
	OrderStatusCancelled OrderStatus = vega.Order_STATUS_CANCELLED
	// Used for unfilled FOK or IOC orders, and for orders that were stopped by the network.
	OrderStatusStopped OrderStatus = vega.Order_STATUS_STOPPED
	// Used for closed fully filled orders.
	OrderStatusFilled OrderStatus = vega.Order_STATUS_FILLED
	// Used for orders when not enough collateral was available to fill the margin requirements.
	OrderStatusRejected OrderStatus = vega.Order_STATUS_REJECTED
	// Used for closed partially filled IOC orders.
	OrderStatusPartiallyFilled OrderStatus = vega.Order_STATUS_PARTIALLY_FILLED
	// Order has been removed from the order book and has been parked, this applies to pegged orders only.
	OrderStatusParked OrderStatus = vega.Order_STATUS_PARKED
)

type OrderType = vega.Order_Type

const (
	// Default value, always invalid.
	OrderTypeUnspecified OrderType = vega.Order_TYPE_UNSPECIFIED
	// Used for Limit orders.
	OrderTypeLimit OrderType = vega.Order_TYPE_LIMIT
	// Used for Market orders.
	OrderTypeMarket OrderType = vega.Order_TYPE_MARKET
	// Used for orders where the initiating party is the network (with distressed traders).
	OrderTypeNetwork OrderType = vega.Order_TYPE_NETWORK
)

type OrderTimeInForce = vega.Order_TimeInForce

const (
	// Default value for TimeInForce, can be valid for an amend.
	OrderTimeInForceUnspecified OrderTimeInForce = vega.Order_TIME_IN_FORCE_UNSPECIFIED
	// Good until cancelled.
	OrderTimeInForceGTC OrderTimeInForce = vega.Order_TIME_IN_FORCE_GTC
	// Good until specified time.
	OrderTimeInForceGTT OrderTimeInForce = vega.Order_TIME_IN_FORCE_GTT
	// Immediate or cancel.
	OrderTimeInForceIOC OrderTimeInForce = vega.Order_TIME_IN_FORCE_IOC
	// Fill or kill.
	OrderTimeInForceFOK OrderTimeInForce = vega.Order_TIME_IN_FORCE_FOK
	// Good for auction.
	OrderTimeInForceGFA OrderTimeInForce = vega.Order_TIME_IN_FORCE_GFA
	// Good for normal.
	OrderTimeInForceGFN OrderTimeInForce = vega.Order_TIME_IN_FORCE_GFN
)

type OrderError = vega.OrderError

const (
	// Default value, no error reported.
	OrderErrorUnspecified OrderError = vega.OrderError_ORDER_ERROR_UNSPECIFIED
	// Order was submitted for a market that does not exist.
	OrderErrorInvalidMarketID OrderError = vega.OrderError_ORDER_ERROR_INVALID_MARKET_ID
	// Order was submitted with an invalid identifier.
	OrderErrorInvalidOrderID OrderError = vega.OrderError_ORDER_ERROR_INVALID_ORDER_ID
	// Order was amended with a sequence number that was not previous version + 1.
	OrderErrorOutOfSequence OrderError = vega.OrderError_ORDER_ERROR_OUT_OF_SEQUENCE
	// Order was amended with an invalid remaining size (e.g. remaining greater than total size).
	OrderErrorInvalidRemainingSize OrderError = vega.OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE
	// Node was unable to get Vega (blockchain) time.
	OrderErrorTimeFailure OrderError = vega.OrderError_ORDER_ERROR_TIME_FAILURE
	// Failed to remove an order from the book.
	OrderErrorRemovalFailure OrderError = vega.OrderError_ORDER_ERROR_REMOVAL_FAILURE
	// An order with `TimeInForce.TIME_IN_FORCE_GTT` was submitted or amended
	// with an expiration that was badly formatted or otherwise invalid.
	OrderErrorInvalidExpirationDatetime OrderError = vega.OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME
	// Order was submitted or amended with an invalid reference field.
	OrderErrorInvalidOrderReference OrderError = vega.OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE
	// Order amend was submitted for an order field that cannot not be amended (e.g. order identifier).
	OrderErrorEditNotAllowed OrderError = vega.OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED
	// Amend failure because amend details do not match original order.
	OrderErrorAmendFailure OrderError = vega.OrderError_ORDER_ERROR_AMEND_FAILURE
	// Order not found in an order book or store.
	OrderErrorNotFound OrderError = vega.OrderError_ORDER_ERROR_NOT_FOUND
	// Order was submitted with an invalid or missing party identifier.
	OrderErrorInvalidParty OrderError = vega.OrderError_ORDER_ERROR_INVALID_PARTY_ID
	// Order was submitted for a market that has closed.
	OrderErrorMarketClosed OrderError = vega.OrderError_ORDER_ERROR_MARKET_CLOSED
	// Order was submitted, but the party did not have enough collateral to cover the order.
	OrderErrorMarginCheckFailed OrderError = vega.OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED
	// Order was submitted, but the party did not have an account for this asset.
	OrderErrorMissingGeneralAccount OrderError = vega.OrderError_ORDER_ERROR_MISSING_GENERAL_ACCOUNT
	// Unspecified internal error.
	OrderErrorInternalError OrderError = vega.OrderError_ORDER_ERROR_INTERNAL_ERROR
	// Order was submitted with an invalid or missing size (e.g. 0).
	OrderErrorInvalidSize OrderError = vega.OrderError_ORDER_ERROR_INVALID_SIZE
	// Order was submitted with an invalid persistence for its type.
	OrderErrorInvalidPersistance OrderError = vega.OrderError_ORDER_ERROR_INVALID_PERSISTENCE
	// Order was submitted with an invalid type field.
	OrderErrorInvalidType OrderError = vega.OrderError_ORDER_ERROR_INVALID_TYPE
	// Order was stopped as it would have traded with another order submitted from the same party.
	OrderErrorSelfTrading OrderError = vega.OrderError_ORDER_ERROR_SELF_TRADING
	// Order was submitted, but the party did not have enough collateral to cover the fees for the order.
	OrderErrorInsufficientFundsToPayFees OrderError = vega.OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES
	// Order was submitted with an incorrect or invalid market type.
	OrderErrorIncorrectMarketType OrderError = vega.OrderError_ORDER_ERROR_INCORRECT_MARKET_TYPE
	// Order was submitted with invalid time in force.
	OrderErrorInvalidTimeInForce OrderError = vega.OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE
	// A GFN order has got to the market when it is in auction mode.
	OrderErrorCannotSendGFNOrderDuringAnAuction OrderError = vega.OrderError_ORDER_ERROR_CANNOT_SEND_GFN_ORDER_DURING_AN_AUCTION
	// A GFA order has got to the market when it is in continuous trading mode.
	OrderErrorCannotSendGFAOrderDuringContinuousTrading OrderError = vega.OrderError_ORDER_ERROR_CANNOT_SEND_GFA_ORDER_DURING_CONTINUOUS_TRADING
	// Attempt to amend order to GTT without ExpiryAt.
	OrderErrorCannotAmendToGTTWithoutExpiryAt OrderError = vega.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT
	// Attempt to amend ExpiryAt to a value before CreatedAt.
	OrderErrorExpiryAtBeforeCreatedAt OrderError = vega.OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT
	// Attempt to amend to GTC without an ExpiryAt value.
	OrderErrorCannotHaveGTCAndExpiryAt OrderError = vega.OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT
	// Amending to FOK or IOC is invalid.
	OrderErrorCannotAmendToFOKOrIOC OrderError = vega.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC
	// Amending to GFA or GFN is invalid.
	OrderErrorCannotAmendToGFAOrGFN OrderError = vega.OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN
	// Amending from GFA or GFN is invalid.
	OrderErrorCannotAmendFromGFAOrGFN OrderError = vega.OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN
	// IOC orders are not allowed during auction.
	OrderErrorCannotSendIOCOrderDuringAuction OrderError = vega.OrderError_ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION
	// FOK orders are not allowed during auction.
	OrderErrorCannotSendFOKOrderDurinAuction OrderError = vega.OrderError_ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION
	// Pegged orders must be LIMIT orders.
	OrderErrorMustBeLimitOrder OrderError = vega.OrderError_ORDER_ERROR_MUST_BE_LIMIT_ORDER
	// Pegged orders can only have TIF GTC or GTT.
	OrderErrorMustBeGTTOrGTC OrderError = vega.OrderError_ORDER_ERROR_MUST_BE_GTT_OR_GTC
	// Pegged order must have a reference price.
	OrderErrorWithoutReferencePrice OrderError = vega.OrderError_ORDER_ERROR_WITHOUT_REFERENCE_PRICE
	// Buy pegged order cannot reference best ask price.
	OrderErrorBuyCannotReferenceBestAskPrice OrderError = vega.OrderError_ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE
	// Pegged order offset must be >= 0.
	OrderErrorOffsetMustBeGreaterOrEqualToZero OrderError = vega.OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO
	// Sell pegged order cannot reference best bid price.
	OrderErrorSellCannotReferenceBestBidPrice OrderError = vega.OrderError_ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE
	// Pegged order offset must be > zero.
	OrderErrorOffsetMustBeGreaterThanZero OrderError = vega.OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO
	// The party has an insufficient balance, or does not have
	// a general account to submit the order (no deposits made
	// for the required asset).
	OrderErrorInsufficientAssetBalance OrderError = vega.OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE
	// Cannot amend a non pegged orders details.
	OrderErrorCannotAmendPeggedOrderDetailsOnNonPeggedOrder OrderError = vega.OrderError_ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER
	// We are unable to re-price a pegged order because a market price is unavailable.
	OrderErrorUnableToRepricePeggedOrder OrderError = vega.OrderError_ORDER_ERROR_UNABLE_TO_REPRICE_PEGGED_ORDER
	// It is not possible to amend the price of an existing pegged order.
	OrderErrorUnableToAmendPriceOnPeggedOrder OrderError = vega.OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER
	// An FOK, IOC, or GFN order was rejected because it resulted in trades outside the price bounds.
	OrderErrorNonPersistentOrderOutOfPriceBounds OrderError = vega.OrderError_ORDER_ERROR_NON_PERSISTENT_ORDER_OUT_OF_PRICE_BOUNDS
	OrderErrorSellOrderNotAllowed                OrderError = vega.OrderError_ORDER_ERROR_SELL_ORDER_NOT_ALLOWED
)

type PositionStatus int32

const (
	PositionStatusUnspecified  = PositionStatus(vega.PositionStatus_POSITION_STATUS_UNSPECIFIED)
	PositionStatusOrdersClosed = PositionStatus(vega.PositionStatus_POSITION_STATUS_ORDERS_CLOSED)
	PositionStatusClosedOut    = PositionStatus(vega.PositionStatus_POSITION_STATUS_CLOSED_OUT)
	PositionStatusDistressed   = PositionStatus(vega.PositionStatus_POSITION_STATUS_DISTRESSED)
)

type TransferType int

const (
	Unknown TransferType = iota
	OneOff
	Recurring
	GovernanceOneOff
	GovernanceRecurring
)

const (
	OneOffStr              = "OneOff"
	RecurringStr           = "Recurring"
	GovernanceOneOffStr    = "GovernanceOneOff"
	GovernanceRecurringStr = "GovernanceRecurring"
	UnknownStr             = "Unknown"
)

func (m TransferType) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	mode := UnknownStr
	switch m {
	case OneOff:
		mode = OneOffStr
	case Recurring:
		mode = RecurringStr
	case GovernanceOneOff:
		mode = GovernanceOneOffStr
	case GovernanceRecurring:
		mode = GovernanceRecurringStr
	}

	return append(buf, []byte(mode)...), nil
}

func (m *TransferType) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val := Unknown
	switch string(src) {
	case OneOffStr:
		val = OneOff
	case RecurringStr:
		val = Recurring
	case GovernanceOneOffStr:
		val = GovernanceOneOff
	case GovernanceRecurringStr:
		val = GovernanceRecurring
	}

	*m = val
	return nil
}

type TransferScope int32

const (
	TransferScopeUnspecified TransferScope = 1
	TransferScopeIndividual  TransferScope = 1
	TransferScopeTeam        TransferScope = 2
)

type TransferStatus eventspb.Transfer_Status

const (
	TransferStatusUnspecified = TransferStatus(eventspb.Transfer_STATUS_UNSPECIFIED)
	TransferStatusPending     = TransferStatus(eventspb.Transfer_STATUS_PENDING)
	TransferStatusDone        = TransferStatus(eventspb.Transfer_STATUS_DONE)
	TransferStatusRejected    = TransferStatus(eventspb.Transfer_STATUS_REJECTED)
	TransferStatusStopped     = TransferStatus(eventspb.Transfer_STATUS_STOPPED)
	TransferStatusCancelled   = TransferStatus(eventspb.Transfer_STATUS_CANCELLED)
)

func (m TransferStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	mode, ok := eventspb.Transfer_Status_name[int32(m)]
	if !ok {
		return buf, fmt.Errorf("unknown transfer status: %s", mode)
	}
	return append(buf, []byte(mode)...), nil
}

func (m *TransferStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := eventspb.Transfer_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown transfer status: %s", src)
	}

	*m = TransferStatus(val)
	return nil
}

type AssetStatus vega.Asset_Status

const (
	AssetStatusUnspecified    = AssetStatus(vega.Asset_STATUS_UNSPECIFIED)
	AssetStatusProposed       = AssetStatus(vega.Asset_STATUS_PROPOSED)
	AssetStatusRejected       = AssetStatus(vega.Asset_STATUS_REJECTED)
	AssetStatusPendingListing = AssetStatus(vega.Asset_STATUS_PENDING_LISTING)
	AssetStatusEnabled        = AssetStatus(vega.Asset_STATUS_ENABLED)
)

func (m AssetStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	mode, ok := vega.Asset_Status_name[int32(m)]
	if !ok {
		return buf, fmt.Errorf("unknown asset status: %s", mode)
	}
	return append(buf, []byte(mode)...), nil
}

func (m *AssetStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.Asset_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown asset status: %s", src)
	}

	*m = AssetStatus(val)
	return nil
}

type MarketTradingMode vega.Market_TradingMode

const (
	MarketTradingModeUnspecified            = MarketTradingMode(vega.Market_TRADING_MODE_UNSPECIFIED)
	MarketTradingModeContinuous             = MarketTradingMode(vega.Market_TRADING_MODE_CONTINUOUS)
	MarketTradingModeBatchAuction           = MarketTradingMode(vega.Market_TRADING_MODE_BATCH_AUCTION)
	MarketTradingModeOpeningAuction         = MarketTradingMode(vega.Market_TRADING_MODE_OPENING_AUCTION)
	MarketTradingModeMonitoringAuction      = MarketTradingMode(vega.Market_TRADING_MODE_MONITORING_AUCTION)
	MarketTradingModeNoTrading              = MarketTradingMode(vega.Market_TRADING_MODE_NO_TRADING)
	MarketTradingModeSuspendedViaGovernance = MarketTradingMode(vega.Market_TRADING_MODE_SUSPENDED_VIA_GOVERNANCE)
	MarketTradingModelLongBlockAuction      = MarketTradingMode(vega.Market_TRADING_MODE_LONG_BLOCK_AUCTION)
)

func (m MarketTradingMode) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	mode, ok := vega.Market_TradingMode_name[int32(m)]
	if !ok {
		return buf, fmt.Errorf("unknown trading mode: %s", mode)
	}
	return append(buf, []byte(mode)...), nil
}

func (m *MarketTradingMode) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.Market_TradingMode_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown trading mode: %s", src)
	}

	*m = MarketTradingMode(val)
	return nil
}

type MarketState vega.Market_State

const (
	MarketStateUnspecified            = MarketState(vega.Market_STATE_UNSPECIFIED)
	MarketStateProposed               = MarketState(vega.Market_STATE_PROPOSED)
	MarketStateRejected               = MarketState(vega.Market_STATE_REJECTED)
	MarketStatePending                = MarketState(vega.Market_STATE_PENDING)
	MarketStateCancelled              = MarketState(vega.Market_STATE_CANCELLED)
	MarketStateActive                 = MarketState(vega.Market_STATE_ACTIVE)
	MarketStateSuspended              = MarketState(vega.Market_STATE_SUSPENDED)
	MarketStateClosed                 = MarketState(vega.Market_STATE_CLOSED)
	MarketStateTradingTerminated      = MarketState(vega.Market_STATE_TRADING_TERMINATED)
	MarketStateSettled                = MarketState(vega.Market_STATE_SETTLED)
	MarketStateSuspendedViaGovernance = MarketState(vega.Market_STATE_SUSPENDED_VIA_GOVERNANCE)
)

func (s MarketState) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	state, ok := vega.Market_State_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown market state: %s", state)
	}
	return append(buf, []byte(state)...), nil
}

func (s *MarketState) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.Market_State_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown market state: %s", src)
	}

	*s = MarketState(val)

	return nil
}

type DepositStatus vega.Deposit_Status

const (
	DepositStatusUnspecified       = DepositStatus(vega.Deposit_STATUS_UNSPECIFIED)
	DepositStatusOpen              = DepositStatus(vega.Deposit_STATUS_OPEN)
	DepositStatusCancelled         = DepositStatus(vega.Deposit_STATUS_CANCELLED)
	DepositStatusFinalized         = DepositStatus(vega.Deposit_STATUS_FINALIZED)
	DepositStatusDuplicateRejected = DepositStatus(vega.Deposit_STATUS_DUPLICATE_REJECTED)
)

func (s DepositStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := vega.Deposit_Status_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown deposit state, %s", status)
	}
	return append(buf, []byte(status)...), nil
}

func (s *DepositStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.Deposit_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown deposit state: %s", src)
	}

	*s = DepositStatus(val)

	return nil
}

type WithdrawalStatus vega.Withdrawal_Status

const (
	WithdrawalStatusUnspecified = WithdrawalStatus(vega.Withdrawal_STATUS_UNSPECIFIED)
	WithdrawalStatusOpen        = WithdrawalStatus(vega.Withdrawal_STATUS_OPEN)
	WithdrawalStatusRejected    = WithdrawalStatus(vega.Withdrawal_STATUS_REJECTED)
	WithdrawalStatusFinalized   = WithdrawalStatus(vega.Withdrawal_STATUS_FINALIZED)
)

func (s WithdrawalStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := vega.Withdrawal_Status_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown withdrawal status: %s", status)
	}
	return append(buf, []byte(status)...), nil
}

func (s *WithdrawalStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.Withdrawal_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown withdrawal status: %s", src)
	}
	*s = WithdrawalStatus(val)
	return nil
}

/************************* Proposal State *****************************/

type ProposalState vega.Proposal_State

const (
	ProposalStateUnspecified        = ProposalState(vega.Proposal_STATE_UNSPECIFIED)
	ProposalStateFailed             = ProposalState(vega.Proposal_STATE_FAILED)
	ProposalStateOpen               = ProposalState(vega.Proposal_STATE_OPEN)
	ProposalStatePassed             = ProposalState(vega.Proposal_STATE_PASSED)
	ProposalStateRejected           = ProposalState(vega.Proposal_STATE_REJECTED)
	ProposalStateDeclined           = ProposalState(vega.Proposal_STATE_DECLINED)
	ProposalStateEnacted            = ProposalState(vega.Proposal_STATE_ENACTED)
	ProposalStateWaitingForNodeVote = ProposalState(vega.Proposal_STATE_WAITING_FOR_NODE_VOTE)
)

func (s ProposalState) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.Proposal_State_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown state: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *ProposalState) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.Proposal_State_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown state: %s", src)
	}
	*s = ProposalState(val)
	return nil
}

/************************* Proposal Error *****************************/

type ProposalError vega.ProposalError

const (
	ProposalErrorUnspecified                      = ProposalError(vega.ProposalError_PROPOSAL_ERROR_UNSPECIFIED)
	ProposalErrorCloseTimeTooSoon                 = ProposalError(vega.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_SOON)
	ProposalErrorCloseTimeTooLate                 = ProposalError(vega.ProposalError_PROPOSAL_ERROR_CLOSE_TIME_TOO_LATE)
	ProposalErrorEnactTimeTooSoon                 = ProposalError(vega.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_SOON)
	ProposalErrorEnactTimeTooLate                 = ProposalError(vega.ProposalError_PROPOSAL_ERROR_ENACT_TIME_TOO_LATE)
	ProposalErrorInsufficientTokens               = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_TOKENS)
	ProposalErrorInvalidInstrumentSecurity        = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_INSTRUMENT_SECURITY)
	ProposalErrorNoProduct                        = ProposalError(vega.ProposalError_PROPOSAL_ERROR_NO_PRODUCT)
	ProposalErrorUnsupportedProduct               = ProposalError(vega.ProposalError_PROPOSAL_ERROR_UNSUPPORTED_PRODUCT)
	ProposalErrorNoTradingMode                    = ProposalError(vega.ProposalError_PROPOSAL_ERROR_NO_TRADING_MODE)
	ProposalErrorUnsupportedTradingMode           = ProposalError(vega.ProposalError_PROPOSAL_ERROR_UNSUPPORTED_TRADING_MODE)
	ProposalErrorNodeValidationFailed             = ProposalError(vega.ProposalError_PROPOSAL_ERROR_NODE_VALIDATION_FAILED)
	ProposalErrorMissingBuiltinAssetField         = ProposalError(vega.ProposalError_PROPOSAL_ERROR_MISSING_BUILTIN_ASSET_FIELD)
	ProposalErrorMissingErc20ContractAddress      = ProposalError(vega.ProposalError_PROPOSAL_ERROR_MISSING_ERC20_CONTRACT_ADDRESS)
	ProposalErrorInvalidAsset                     = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_ASSET)
	ProposalErrorIncompatibleTimestamps           = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INCOMPATIBLE_TIMESTAMPS)
	ProposalErrorNoRiskParameters                 = ProposalError(vega.ProposalError_PROPOSAL_ERROR_NO_RISK_PARAMETERS)
	ProposalErrorNetworkParameterInvalidKey       = ProposalError(vega.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_KEY)
	ProposalErrorNetworkParameterInvalidValue     = ProposalError(vega.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_INVALID_VALUE)
	ProposalErrorNetworkParameterValidationFailed = ProposalError(vega.ProposalError_PROPOSAL_ERROR_NETWORK_PARAMETER_VALIDATION_FAILED)
	ProposalErrorOpeningAuctionDurationTooSmall   = ProposalError(vega.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_SMALL)
	ProposalErrorOpeningAuctionDurationTooLarge   = ProposalError(vega.ProposalError_PROPOSAL_ERROR_OPENING_AUCTION_DURATION_TOO_LARGE)
	ProposalErrorCouldNotInstantiateMarket        = ProposalError(vega.ProposalError_PROPOSAL_ERROR_COULD_NOT_INSTANTIATE_MARKET)
	ProposalErrorInvalidFutureProduct             = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_FUTURE_PRODUCT)
	ProposalErrorInvalidRiskParameter             = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_RISK_PARAMETER)
	ProposalErrorMajorityThresholdNotReached      = ProposalError(vega.ProposalError_PROPOSAL_ERROR_MAJORITY_THRESHOLD_NOT_REACHED)
	ProposalErrorParticipationThresholdNotReached = ProposalError(vega.ProposalError_PROPOSAL_ERROR_PARTICIPATION_THRESHOLD_NOT_REACHED)
	ProposalErrorInvalidAssetDetails              = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_ASSET_DETAILS)
	ProposalErrorUnknownType                      = ProposalError(vega.ProposalError_PROPOSAL_ERROR_UNKNOWN_TYPE)
	ProposalErrorUnknownRiskParameterType         = ProposalError(vega.ProposalError_PROPOSAL_ERROR_UNKNOWN_RISK_PARAMETER_TYPE)
	ProposalErrorInvalidFreeform                  = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_FREEFORM)
	ProposalErrorInsufficientEquityLikeShare      = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INSUFFICIENT_EQUITY_LIKE_SHARE)
	ProposalErrorInvalidMarket                    = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_MARKET)
	ProposalErrorTooManyMarketDecimalPlaces       = ProposalError(vega.ProposalError_PROPOSAL_ERROR_TOO_MANY_MARKET_DECIMAL_PLACES)
	ProposalErrorTooManyPriceMonitoringTriggers   = ProposalError(vega.ProposalError_PROPOSAL_ERROR_TOO_MANY_PRICE_MONITORING_TRIGGERS)
	ProposalErrorERC20AddressAlreadyInUse         = ProposalError(vega.ProposalError_PROPOSAL_ERROR_ERC20_ADDRESS_ALREADY_IN_USE)
	ProporsalErrorInvalidGovernanceTransfer       = ProposalError(vega.ProposalError_PROPOSAL_ERROR_GOVERNANCE_TRANSFER_PROPOSAL_INVALID)
	ProporsalErrorFailedGovernanceTransfer        = ProposalError(vega.ProposalError_PROPOSAL_ERROR_GOVERNANCE_TRANSFER_PROPOSAL_FAILED)
	ProporsalErrorFailedGovernanceTransferCancel  = ProposalError(vega.ProposalError_PROPOSAL_ERROR_GOVERNANCE_CANCEL_TRANSFER_PROPOSAL_INVALID)
	ProposalErrorInvalidSpot                      = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_SPOT)
	ProposalErrorSpotNotEnabled                   = ProposalError(vega.ProposalError_PROPOSAL_ERROR_SPOT_PRODUCT_DISABLED)
	ProposalErrorInvalidSuccessorMarket           = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_SUCCESSOR_MARKET)
	ProposalErrorInvalidStateUpdate               = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_MARKET_STATE_UPDATE)
	ProposalErrorInvalidSLAParams                 = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_SLA_PARAMS)
	ProposalErrorMissingSLAParams                 = ProposalError(vega.ProposalError_PROPOSAL_ERROR_MISSING_SLA_PARAMS)
	ProposalInvalidPerpetualProduct               = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_PERPETUAL_PRODUCT)
	ProposalErrorInvalidSizeDecimalPlaces         = ProposalError(vega.ProposalError_PROPOSAL_ERROR_INVALID_SIZE_DECIMAL_PLACES)
)

func (s ProposalError) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.ProposalError_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown proposal error: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *ProposalError) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.ProposalError_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown proposal error: %s", src)
	}
	*s = ProposalError(val)
	return nil
}

/************************* VoteValue *****************************/

type VoteValue vega.Vote_Value

const (
	VoteValueUnspecified = VoteValue(vega.Vote_VALUE_UNSPECIFIED)
	VoteValueNo          = VoteValue(vega.Vote_VALUE_NO)
	VoteValueYes         = VoteValue(vega.Vote_VALUE_YES)
)

func (s VoteValue) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.Vote_Value_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown vote value: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *VoteValue) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.Vote_Value_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown vote value: %s", src)
	}
	*s = VoteValue(val)
	return nil
}

/************************* NodeSignature Kind *****************************/

type NodeSignatureKind commandspb.NodeSignatureKind

const (
	NodeSignatureKindUnspecified          = NodeSignatureKind(commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_UNSPECIFIED)
	NodeSignatureKindAsset                = NodeSignatureKind(commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW)
	NodeSignatureKindAssetUpdate          = NodeSignatureKind(commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_UPDATE)
	NodeSignatureKindAssetWithdrawal      = NodeSignatureKind(commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL)
	NodeSignatureKindMultisigSignerAdded  = NodeSignatureKind(commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_ADDED)
	NodeSignatureKindMultisigSignerRemove = NodeSignatureKind(commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ERC20_MULTISIG_SIGNER_REMOVED)
)

func (s NodeSignatureKind) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := commandspb.NodeSignatureKind_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown state: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *NodeSignatureKind) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := commandspb.NodeSignatureKind_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown state: %s", src)
	}
	*s = NodeSignatureKind(val)
	return nil
}

type (
	DataSourceSpecStatus vegapb.DataSourceSpec_Status
	OracleSpecStatus     = DataSourceSpecStatus
)

const (
	OracleSpecUnspecified = DataSourceSpecStatus(vegapb.DataSourceSpec_STATUS_UNSPECIFIED)
	OracleSpecActive      = DataSourceSpecStatus(vegapb.DataSourceSpec_STATUS_ACTIVE)
	OracleSpecDeactivated = DataSourceSpecStatus(vegapb.DataSourceSpec_STATUS_DEACTIVATED)
)

func (s DataSourceSpecStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := vegapb.DataSourceSpec_Status_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown oracle spec value: %v", s)
	}
	return append(buf, []byte(status)...), nil
}

func (s *DataSourceSpecStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vegapb.DataSourceSpec_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown oracle spec status: %s", src)
	}
	*s = DataSourceSpecStatus(val)
	return nil
}

type LiquidityProvisionStatus vega.LiquidityProvision_Status

const (
	LiquidityProvisionStatusUnspecified = LiquidityProvisionStatus(vega.LiquidityProvision_STATUS_UNSPECIFIED)
	LiquidityProvisionStatusActive      = LiquidityProvisionStatus(vega.LiquidityProvision_STATUS_ACTIVE)
	LiquidityProvisionStatusStopped     = LiquidityProvisionStatus(vega.LiquidityProvision_STATUS_STOPPED)
	LiquidityProvisionStatusCancelled   = LiquidityProvisionStatus(vega.LiquidityProvision_STATUS_CANCELLED)
	LiquidityProvisionStatusRejected    = LiquidityProvisionStatus(vega.LiquidityProvision_STATUS_REJECTED)
	LiquidityProvisionStatusUndeployed  = LiquidityProvisionStatus(vega.LiquidityProvision_STATUS_UNDEPLOYED)
	LiquidityProvisionStatusPending     = LiquidityProvisionStatus(vega.LiquidityProvision_STATUS_PENDING)
)

func (s LiquidityProvisionStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := vega.LiquidityProvision_Status_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown liquidity provision status: %v", s)
	}
	return append(buf, []byte(status)...), nil
}

func (s *LiquidityProvisionStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.LiquidityProvision_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown liquidity provision status: %s", src)
	}
	*s = LiquidityProvisionStatus(val)
	return nil
}

type StakeLinkingStatus eventspb.StakeLinking_Status

const (
	StakeLinkingStatusUnspecified = StakeLinkingStatus(eventspb.StakeLinking_STATUS_UNSPECIFIED)
	StakeLinkingStatusPending     = StakeLinkingStatus(eventspb.StakeLinking_STATUS_PENDING)
	StakeLinkingStatusAccepted    = StakeLinkingStatus(eventspb.StakeLinking_STATUS_ACCEPTED)
	StakeLinkingStatusRejected    = StakeLinkingStatus(eventspb.StakeLinking_STATUS_REJECTED)
)

func (s StakeLinkingStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := eventspb.StakeLinking_Status_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown stake linking status: %v", s)
	}
	return append(buf, []byte(status)...), nil
}

func (s *StakeLinkingStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := eventspb.StakeLinking_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown stake linking status: %s", src)
	}
	*s = StakeLinkingStatus(val)
	return nil
}

type StakeLinkingType eventspb.StakeLinking_Type

const (
	StakeLinkingTypeUnspecified = StakeLinkingType(eventspb.StakeLinking_TYPE_UNSPECIFIED)
	StakeLinkingTypeLink        = StakeLinkingType(eventspb.StakeLinking_TYPE_LINK)
	StakeLinkingTypeUnlink      = StakeLinkingType(eventspb.StakeLinking_TYPE_UNLINK)
)

func (s StakeLinkingType) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := eventspb.StakeLinking_Type_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown stake linking type: %v", s)
	}
	return append(buf, []byte(status)...), nil
}

func (s *StakeLinkingType) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := eventspb.StakeLinking_Type_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown stake linking type: %s", src)
	}
	*s = StakeLinkingType(val)

	return nil
}

/************************* Node *****************************/

type NodeStatus vega.NodeStatus

const (
	NodeStatusUnspecified  = NodeStatus(vega.NodeStatus_NODE_STATUS_UNSPECIFIED)
	NodeStatusValidator    = NodeStatus(vega.NodeStatus_NODE_STATUS_VALIDATOR)
	NodeStatusNonValidator = NodeStatus(vega.NodeStatus_NODE_STATUS_NON_VALIDATOR)
)

func (ns NodeStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.NodeStatus_name[int32(ns)]
	if !ok {
		return buf, fmt.Errorf("unknown node status: %v", ns)
	}
	return append(buf, []byte(str)...), nil
}

func (ns *NodeStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.NodeStatus_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown node status: %s", src)
	}
	*ns = NodeStatus(val)
	return nil
}

type ValidatorNodeStatus vega.ValidatorNodeStatus

const (
	ValidatorNodeStatusUnspecified = ValidatorNodeStatus(vega.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_UNSPECIFIED)
	ValidatorNodeStatusTendermint  = ValidatorNodeStatus(vega.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_TENDERMINT)
	ValidatorNodeStatusErsatz      = ValidatorNodeStatus(vega.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_ERSATZ)
	ValidatorNodeStatusPending     = ValidatorNodeStatus(vega.ValidatorNodeStatus_VALIDATOR_NODE_STATUS_PENDING)
)

// ValidatorStatusRanking so we know which direction was a promotion and which was a demotion.
var ValidatorStatusRanking = map[ValidatorNodeStatus]int{
	ValidatorNodeStatusUnspecified: 0,
	ValidatorNodeStatusPending:     1,
	ValidatorNodeStatusErsatz:      2,
	ValidatorNodeStatusTendermint:  3,
}

func (ns ValidatorNodeStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.ValidatorNodeStatus_name[int32(ns)]
	if !ok {
		return buf, fmt.Errorf("unknown validator node status: %v", ns)
	}
	return append(buf, []byte(str)...), nil
}

func (ns *ValidatorNodeStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.ValidatorNodeStatus_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown validator node status: %s", src)
	}
	*ns = ValidatorNodeStatus(val)
	return nil
}

func (ns *ValidatorNodeStatus) UnmarshalJSON(src []byte) error {
	val, ok := vega.ValidatorNodeStatus_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown validator node status: %s", src)
	}
	*ns = ValidatorNodeStatus(val)
	return nil
}

/************************* Position status  *****************************/

func (p PositionStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.PositionStatus_name[int32(p)]
	if !ok {
		return buf, fmt.Errorf("unknown position status: %v", p)
	}
	return append(buf, []byte(str)...), nil
}

func (p *PositionStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.PositionStatus_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown position status: %s", string(src))
	}
	*p = PositionStatus(val)
	return nil
}

/************************* Protocol Upgrade *****************************/

type ProtocolUpgradeProposalStatus eventspb.ProtocolUpgradeProposalStatus

func (ps ProtocolUpgradeProposalStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := eventspb.ProtocolUpgradeProposalStatus_name[int32(ps)]
	if !ok {
		return buf, fmt.Errorf("unknown protocol upgrade proposal status: %v", ps)
	}
	return append(buf, []byte(str)...), nil
}

func (ps *ProtocolUpgradeProposalStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := eventspb.ProtocolUpgradeProposalStatus_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown protocol upgrade proposal status: %s", src)
	}
	*ps = ProtocolUpgradeProposalStatus(val)
	return nil
}

type StopOrderExpiryStrategy vega.StopOrder_ExpiryStrategy

const (
	StopOrderExpiryStrategyUnspecified = StopOrderExpiryStrategy(vega.StopOrder_EXPIRY_STRATEGY_UNSPECIFIED)
	StopOrderExpiryStrategyCancels     = StopOrderExpiryStrategy(vega.StopOrder_EXPIRY_STRATEGY_CANCELS)
	StopOrderExpiryStrategySubmit      = StopOrderExpiryStrategy(vega.StopOrder_EXPIRY_STRATEGY_SUBMIT)
)

func (s StopOrderExpiryStrategy) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.StopOrder_ExpiryStrategy_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown stop order expiry strategy: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *StopOrderExpiryStrategy) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.StopOrder_ExpiryStrategy_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown stop order expiry strategy: %s", src)
	}
	*s = StopOrderExpiryStrategy(val)
	return nil
}

type StopOrderTriggerDirection vega.StopOrder_TriggerDirection

const (
	StopOrderTriggerDirectionUnspecified = StopOrderTriggerDirection(vega.StopOrder_TRIGGER_DIRECTION_UNSPECIFIED)
	StopOrderTriggerDirectionRisesAbove  = StopOrderTriggerDirection(vega.StopOrder_TRIGGER_DIRECTION_RISES_ABOVE)
	StopOrderTriggerDirectionFallsBelow  = StopOrderTriggerDirection(vega.StopOrder_TRIGGER_DIRECTION_FALLS_BELOW)
)

func (s StopOrderTriggerDirection) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.StopOrder_TriggerDirection_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown stop order trigger direction: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *StopOrderTriggerDirection) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.StopOrder_TriggerDirection_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown stop order trigger direction: %s", src)
	}
	*s = StopOrderTriggerDirection(val)
	return nil
}

type StopOrderStatus vega.StopOrder_Status

const (
	StopOrderStatusUnspecified = StopOrderStatus(vega.StopOrder_STATUS_UNSPECIFIED)
	StopOrderStatusPending     = StopOrderStatus(vega.StopOrder_STATUS_PENDING)
	StopOrderStatusCancelled   = StopOrderStatus(vega.StopOrder_STATUS_CANCELLED)
	StopOrderStatusStopped     = StopOrderStatus(vega.StopOrder_STATUS_STOPPED)
	StopOrderStatusTriggered   = StopOrderStatus(vega.StopOrder_STATUS_TRIGGERED)
	StopOrderStatusExpired     = StopOrderStatus(vega.StopOrder_STATUS_EXPIRED)
	StopOrderStatusRejected    = StopOrderStatus(vega.StopOrder_STATUS_REJECTED)
)

func (s StopOrderStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.StopOrder_Status_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown stop order status: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *StopOrderStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.StopOrder_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown stop order status: %s", src)
	}
	*s = StopOrderStatus(val)
	return nil
}

type StopOrderRejectionReason vega.StopOrder_RejectionReason

const (
	StopOrderRejectionReasonUnspecified                  = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_UNSPECIFIED)
	StopOrderRejectionReasonTradingNotAllowed            = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_TRADING_NOT_ALLOWED)
	StopOrderRejectionReasonExpiryInThePast              = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_EXPIRY_IN_THE_PAST)
	StopOrderRejectionReasonMustBeReduceOnly             = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_MUST_BE_REDUCE_ONLY)
	StopOrderRejectionReasonMaxStopOrdersPerPartyReached = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_MAX_STOP_ORDERS_PER_PARTY_REACHED)
	StopOrderRejectionReasonNotAllowedWithoutAPosition   = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_STOP_ORDER_NOT_ALLOWED_WITHOUT_A_POSITION)
	StopOrderRejectionReasonNotClosingThePosition        = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_STOP_ORDER_NOT_CLOSING_THE_POSITION)
	StopOrderRejectionReasonNotAllowedDuringAuction      = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_STOP_ORDER_NOT_ALLOWED_DURING_OPENING_AUCTION)
	StopOrderRejectionReasonOCONotAllowedSameExpiryTime  = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_STOP_ORDER_CANNOT_MATCH_OCO_EXPIRY_TIMES)
	StopOrderRejectionSizeOverrideUnSupportedForSpot     = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_STOP_ORDER_SIZE_OVERRIDE_UNSUPPORTED_FOR_SPOT)
	StopOrderRejectionLinkedPercentageInvalid            = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_STOP_ORDER_LINKED_PERCENTAGE_INVALID)
	StopeOrderRejectionReasonSellOrderNotAllowed         = StopOrderRejectionReason(vega.StopOrder_REJECTION_REASON_SELL_ORDER_NOT_ALLOWED)
)

func (s StopOrderRejectionReason) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.StopOrder_RejectionReason_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown stop order status: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *StopOrderRejectionReason) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.StopOrder_RejectionReason_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown stop order status: %s", src)
	}
	*s = StopOrderRejectionReason(val)
	return nil
}

type FundingPeriodDataPointSource eventspb.FundingPeriodDataPoint_Source

const (
	FundingPeriodDataPointSourceUnspecified = FundingPeriodDataPointSource(eventspb.FundingPeriodDataPoint_SOURCE_UNSPECIFIED)
	FundingPeriodDataPointSourceExternal    = FundingPeriodDataPointSource(eventspb.FundingPeriodDataPoint_SOURCE_EXTERNAL)
	FundingPeriodDataPointSourceInternal    = FundingPeriodDataPointSource(eventspb.FundingPeriodDataPoint_SOURCE_INTERNAL)
)

func (s FundingPeriodDataPointSource) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := eventspb.FundingPeriodDataPoint_Source_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown funding period data point source: %v", s)
	}
	return append(buf, []byte(str)...), nil
}

func (s *FundingPeriodDataPointSource) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := eventspb.FundingPeriodDataPoint_Source_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown funding period data point source: %s", src)
	}
	*s = FundingPeriodDataPointSource(val)
	return nil
}

type LiquidityFeeSettingsMethod vega.LiquidityFeeSettings_Method

const (
	LiquidityFeeMethodUnspecified     = LiquidityFeeSettingsMethod(vega.LiquidityFeeSettings_METHOD_UNSPECIFIED)
	LiquidityFeeMethodMarginalCost    = LiquidityFeeSettingsMethod(vega.LiquidityFeeSettings_METHOD_MARGINAL_COST)
	LiquidityFeeMethodWeightedAverage = LiquidityFeeSettingsMethod(vega.LiquidityFeeSettings_METHOD_WEIGHTED_AVERAGE)
	LiquidityFeeMethodConstant        = LiquidityFeeSettingsMethod(vega.LiquidityFeeSettings_METHOD_CONSTANT)
)

func (s LiquidityFeeSettingsMethod) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := vega.LiquidityFeeSettings_Method_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown liquidity provision status: %v", s)
	}
	return append(buf, []byte(status)...), nil
}

func (s *LiquidityFeeSettingsMethod) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.LiquidityFeeSettings_Method_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown liquidity provision status: %s", src)
	}
	*s = LiquidityFeeSettingsMethod(val)
	return nil
}

type MarginMode vega.MarginMode

const (
	MarginModeUnspecified    = MarginMode(vega.MarginMode_MARGIN_MODE_UNSPECIFIED)
	MarginModeCrossMargin    = MarginMode(vega.MarginMode_MARGIN_MODE_CROSS_MARGIN)
	MarginModeIsolatedMargin = MarginMode(vega.MarginMode_MARGIN_MODE_ISOLATED_MARGIN)
)

func (m MarginMode) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	str, ok := vega.MarginMode_name[int32(m)]
	if !ok {
		return buf, fmt.Errorf("unknown margin mode: %v", m)
	}
	return append(buf, []byte(str)...), nil
}

func (m *MarginMode) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := vega.MarginMode_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown margin mode: %s", src)
	}
	*m = MarginMode(val)
	return nil
}

type AMMStatus eventspb.AMM_Status

const (
	AMMStatusUnspecified = AMMStatus(eventspb.AMM_STATUS_UNSPECIFIED)
	AMMStatusActive      = AMMStatus(eventspb.AMM_STATUS_ACTIVE)
	AMMStatusRejected    = AMMStatus(eventspb.AMM_STATUS_REJECTED)
	AMMStatusCancelled   = AMMStatus(eventspb.AMM_STATUS_CANCELLED)
	AMMStatusStopped     = AMMStatus(eventspb.AMM_STATUS_STOPPED)
	AMMStatusReduceOnly  = AMMStatus(eventspb.AMM_STATUS_REDUCE_ONLY)
)

func (s AMMStatus) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := eventspb.AMM_Status_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown AMM pool status: %v", s)
	}
	return append(buf, []byte(status)...), nil
}

func (s *AMMStatus) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := eventspb.AMM_Status_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown AMM pool status: %s", src)
	}
	*s = AMMStatus(val)
	return nil
}

func (s *AMMStatus) Where(fieldName *string, nextBindVar func(args *[]any, arg any) string, args ...any) (string, []any) {
	if fieldName == nil {
		return fmt.Sprintf("status = %s", nextBindVar(&args, s)), args
	}

	return fmt.Sprintf("%s = %s", *fieldName, nextBindVar(&args, s)), args
}

type AMMStatusReason eventspb.AMM_StatusReason

const (
	AMMStatusReasonUnspecified           = AMMStatusReason(eventspb.AMM_STATUS_REASON_UNSPECIFIED)
	AMMStatusReasonCancelledByParty      = AMMStatusReason(eventspb.AMM_STATUS_REASON_CANCELLED_BY_PARTY)
	AMMStatusReasonCannotFillCommitment  = AMMStatusReason(eventspb.AMM_STATUS_REASON_CANNOT_FILL_COMMITMENT)
	AMMStatusReasonPartyAlreadyOwnsAPool = AMMStatusReason(eventspb.AMM_STATUS_REASON_PARTY_ALREADY_OWNS_AMM_FOR_MARKET)
	AMMStatusReasonPartyClosedOut        = AMMStatusReason(eventspb.AMM_STATUS_REASON_PARTY_CLOSED_OUT)
	AMMStatusReasonMarketClosed          = AMMStatusReason(eventspb.AMM_STATUS_REASON_MARKET_CLOSED)
	AMMStatusReasonCommitmentTooLow      = AMMStatusReason(eventspb.AMM_STATUS_REASON_COMMITMENT_TOO_LOW)
	AMMStatusReasonCannotRebase          = AMMStatusReason(eventspb.AMM_STATUS_REASON_CANNOT_REBASE)
)

func (s AMMStatusReason) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	status, ok := eventspb.AMM_StatusReason_name[int32(s)]
	if !ok {
		return buf, fmt.Errorf("unknown AMM pool status reason: %v", s)
	}
	return append(buf, []byte(status)...), nil
}

func (s *AMMStatusReason) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	val, ok := eventspb.AMM_StatusReason_value[string(src)]
	if !ok {
		return fmt.Errorf("unknown AMM pool status reason: %s", src)
	}
	*s = AMMStatusReason(val)
	return nil
}

type ProtoEnum interface {
	GetEnums() map[int32]string
}
