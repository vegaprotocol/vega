package entities

import (
	"fmt"

	"code.vegaprotocol.io/protos/vega"
	"github.com/jackc/pgtype"
)

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
	OrderErrorGFNOrderDuringAnAuction OrderError = vega.OrderError_ORDER_ERROR_GFN_ORDER_DURING_AN_AUCTION
	// A GFA order has got to the market when it is in continuous trading mode.
	OrderErrorGFAOrderDuringContinuousTrading OrderError = vega.OrderError_ORDER_ERROR_GFA_ORDER_DURING_CONTINUOUS_TRADING
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
)

type MarketTradingMode vega.Market_TradingMode

const (
	MarketTradingModeUnspecified       = MarketTradingMode(vega.Market_TRADING_MODE_UNSPECIFIED)
	MarketTradingModeContinuous        = MarketTradingMode(vega.Market_TRADING_MODE_CONTINUOUS)
	MarketTradingModeBatchAuction      = MarketTradingMode(vega.Market_TRADING_MODE_BATCH_AUCTION)
	MarketTradingModeOpeningAuction    = MarketTradingMode(vega.Market_TRADING_MODE_OPENING_AUCTION)
	MarketTradingModeMonitoringAuction = MarketTradingMode(vega.Market_TRADING_MODE_MONITORING_AUCTION)
)

func (m MarketTradingMode) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	var mode []byte
	switch m {
	case MarketTradingModeUnspecified:
		mode = []byte("TRADING_MODE_UNSPECIFIED")
	case MarketTradingModeContinuous:
		mode = []byte("TRADING_MODE_CONTINUOUS")
	case MarketTradingModeBatchAuction:
		mode = []byte("TRADING_MODE_BATCH_AUCTION")
	case MarketTradingModeOpeningAuction:
		mode = []byte("TRADING_MODE_OPENING_AUCTION")
	case MarketTradingModeMonitoringAuction:
		mode = []byte("TRADING_MODE_MONITORING_AUCTION")
	}

	return append(buf, mode...), nil
}

func (m *MarketTradingMode) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	switch string(src) {
	case "TRADING_MODE_UNSPECIFIED":
		*m = MarketTradingModeUnspecified
	case "TRADING_MODE_CONTINUOUS":
		*m = MarketTradingModeContinuous
	case "TRADING_MODE_BATCH_AUCTION":
		*m = MarketTradingModeBatchAuction
	case "TRADING_MODE_OPENING_AUCTION":
		*m = MarketTradingModeOpeningAuction
	case "TRADING_MODE_MONITORING_AUCTION":
		*m = MarketTradingModeMonitoringAuction
	default:
		return fmt.Errorf("unrecognized trading mode: %s", src)
	}

	return nil
}

type MarketState vega.Market_State

const (
	MarketStateUnspecified       = MarketState(vega.Market_STATE_UNSPECIFIED)
	MarketStateProposed          = MarketState(vega.Market_STATE_PROPOSED)
	MarketStateRejected          = MarketState(vega.Market_STATE_REJECTED)
	MarketStatePending           = MarketState(vega.Market_STATE_PENDING)
	MarketStateCancelled         = MarketState(vega.Market_STATE_CANCELLED)
	MarketStateActive            = MarketState(vega.Market_STATE_ACTIVE)
	MarketStateSuspended         = MarketState(vega.Market_STATE_SUSPENDED)
	MarketStateClosed            = MarketState(vega.Market_STATE_CLOSED)
	MarketStateTradingTerminated = MarketState(vega.Market_STATE_TRADING_TERMINATED)
	MarketStateSettled           = MarketState(vega.Market_STATE_SETTLED)
)

func (s MarketState) EncodeText(_ *pgtype.ConnInfo, buf []byte) ([]byte, error) {
	var state []byte
	switch s {
	case MarketStateUnspecified:
		state = []byte("STATE_UNSPECIFIED")
	case MarketStateProposed:
		state = []byte("STATE_PROPOSED")
	case MarketStateRejected:
		state = []byte("STATE_REJECTED")
	case MarketStatePending:
		state = []byte("STATE_PENDING")
	case MarketStateCancelled:
		state = []byte("STATE_CANCELLED")
	case MarketStateActive:
		state = []byte("STATE_ACTIVE")
	case MarketStateSuspended:
		state = []byte("STATE_SUSPENDED")
	case MarketStateClosed:
		state = []byte("STATE_CLOSED")
	case MarketStateTradingTerminated:
		state = []byte("STATE_TRADING_TERMINATED")
	case MarketStateSettled:
		state = []byte("STATE_SETTLED")
	}

	return append(buf, state...), nil
}

func (s *MarketState) DecodeText(_ *pgtype.ConnInfo, src []byte) error {
	switch string(src) {
	case "STATE_UNSPECIFIED":
		*s = MarketStateUnspecified
	case "STATE_PROPOSED":
		*s = MarketStateProposed
	case "STATE_REJECTED":
		*s = MarketStateRejected
	case "STATE_PENDING":
		*s = MarketStatePending
	case "STATE_CANCELLED":
		*s = MarketStateCancelled
	case "STATE_ACTIVE":
		*s = MarketStateActive
	case "STATE_SUSPENDED":
		*s = MarketStateSuspended
	case "STATE_CLOSED":
		*s = MarketStateClosed
	case "STATE_TRADING_TERMINATED":
		*s = MarketStateTradingTerminated
	case "STATE_SETTLED":
		*s = MarketStateSettled
	default:
		return fmt.Errorf("unknown state: %s", src)
	}

	return nil
}
