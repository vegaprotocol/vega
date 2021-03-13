//lint:file-ignore ST1003 Ignore underscores in names, this is straigh copied from the proto package to ease introducing the domain types

package types

import "code.vegaprotocol.io/vega/proto"

type Order = proto.Order
type OrderConfirmation = proto.OrderConfirmation
type OrderAmendment = proto.OrderAmendment
type OrderCancellation = proto.OrderCancellation
type OrderCancellationConfirmation = proto.OrderCancellationConfirmation
type PeggedOrder = proto.PeggedOrder
type Trade = proto.Trade
type Fee = proto.Fee

type Trade_Type = proto.Trade_Type

const (
	// Default value, always invalid
	Trade_TYPE_UNSPECIFIED Trade_Type = 0
	// Normal trading between two parties
	Trade_TYPE_DEFAULT Trade_Type = 1
	// Trading initiated by the network with another party on the book,
	// which helps to zero-out the positions of one or more distressed parties
	Trade_TYPE_NETWORK_CLOSE_OUT_GOOD Trade_Type = 2
	// Trading initiated by the network with another party off the book,
	// with a distressed party in order to zero-out the position of the party
	Trade_TYPE_NETWORK_CLOSE_OUT_BAD Trade_Type = 3
)

type PeggedReference = proto.PeggedReference

const (
	// Default value for PeggedReference, no reference given
	PeggedReference_PEGGED_REFERENCE_UNSPECIFIED PeggedReference = 0
	// Mid price reference
	PeggedReference_PEGGED_REFERENCE_MID PeggedReference = 1
	// Best bid price reference
	PeggedReference_PEGGED_REFERENCE_BEST_BID PeggedReference = 2
	// Best ask price reference
	PeggedReference_PEGGED_REFERENCE_BEST_ASK PeggedReference = 3
)

type Order_Status = proto.Order_Status

const (
	// Default value, always invalid
	Order_STATUS_UNSPECIFIED Order_Status = 0
	// Used for active unfilled or partially filled orders
	Order_STATUS_ACTIVE Order_Status = 1
	// Used for expired GTT orders
	Order_STATUS_EXPIRED Order_Status = 2
	// Used for orders cancelled by the party that created the order
	Order_STATUS_CANCELLED Order_Status = 3
	// Used for unfilled FOK or IOC orders, and for orders that were stopped by the network
	Order_STATUS_STOPPED Order_Status = 4
	// Used for closed fully filled orders
	Order_STATUS_FILLED Order_Status = 5
	// Used for orders when not enough collateral was available to fill the margin requirements
	Order_STATUS_REJECTED Order_Status = 6
	// Used for closed partially filled IOC orders
	Order_STATUS_PARTIALLY_FILLED Order_Status = 7
	// Order has been removed from the order book and has been parked, this applies to pegged orders only
	Order_STATUS_PARKED Order_Status = 8
)

type Side = proto.Side

const (
	// Default value, always invalid
	Side_SIDE_UNSPECIFIED Side = 0
	// Buy order
	Side_SIDE_BUY Side = 1
	// Sell order
	Side_SIDE_SELL Side = 2
)

type Order_Type = proto.Order_Type

const (
	// Default value, always invalid
	Order_TYPE_UNSPECIFIED Order_Type = 0
	// Used for Limit orders
	Order_TYPE_LIMIT Order_Type = 1
	// Used for Market orders
	Order_TYPE_MARKET Order_Type = 2
	// Used for orders where the initiating party is the network (with distressed traders)
	Order_TYPE_NETWORK Order_Type = 3
)

type Order_TimeInForce = proto.Order_TimeInForce

const (
	// Default value for TimeInForce, can be valid for an amend
	Order_TIME_IN_FORCE_UNSPECIFIED Order_TimeInForce = 0
	// Good until cancelled
	Order_TIME_IN_FORCE_GTC Order_TimeInForce = 1
	// Good until specified time
	Order_TIME_IN_FORCE_GTT Order_TimeInForce = 2
	// Immediate or cancel
	Order_TIME_IN_FORCE_IOC Order_TimeInForce = 3
	// Fill or kill
	Order_TIME_IN_FORCE_FOK Order_TimeInForce = 4
	// Good for auction
	Order_TIME_IN_FORCE_GFA Order_TimeInForce = 5
	// Good for normal
	Order_TIME_IN_FORCE_GFN Order_TimeInForce = 6
)

type OrderError = proto.OrderError

const (
	// Default value, no error reported
	OrderError_ORDER_ERROR_UNSPECIFIED OrderError = 0
	// Order was submitted for a market that does not exist
	OrderError_ORDER_ERROR_INVALID_MARKET_ID OrderError = 1
	// Order was submitted with an invalid identifier
	OrderError_ORDER_ERROR_INVALID_ORDER_ID OrderError = 2
	// Order was amended with a sequence number that was not previous version + 1
	OrderError_ORDER_ERROR_OUT_OF_SEQUENCE OrderError = 3
	// Order was amended with an invalid remaining size (e.g. remaining greater than total size)
	OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE OrderError = 4
	// Node was unable to get Vega (blockchain) time
	OrderError_ORDER_ERROR_TIME_FAILURE OrderError = 5
	// Failed to remove an order from the book
	OrderError_ORDER_ERROR_REMOVAL_FAILURE OrderError = 6
	// An order with `TimeInForce.TIME_IN_FORCE_GTT` was submitted or amended
	// with an expiration that was badly formatted or otherwise invalid
	OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME OrderError = 7
	// Order was submitted or amended with an invalid reference field
	OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE OrderError = 8
	// Order amend was submitted for an order field that cannot not be amended (e.g. order identifier)
	OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED OrderError = 9
	// Amend failure because amend details do not match original order
	OrderError_ORDER_ERROR_AMEND_FAILURE OrderError = 10
	// Order not found in an order book or store
	OrderError_ORDER_ERROR_NOT_FOUND OrderError = 11
	// Order was submitted with an invalid or missing party identifier
	OrderError_ORDER_ERROR_INVALID_PARTY_ID OrderError = 12
	// Order was submitted for a market that has closed
	OrderError_ORDER_ERROR_MARKET_CLOSED OrderError = 13
	// Order was submitted, but the party did not have enough collateral to cover the order
	OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED OrderError = 14
	// Order was submitted, but the party did not have an account for this asset
	OrderError_ORDER_ERROR_MISSING_GENERAL_ACCOUNT OrderError = 15
	// Unspecified internal error
	OrderError_ORDER_ERROR_INTERNAL_ERROR OrderError = 16
	// Order was submitted with an invalid or missing size (e.g. 0)
	OrderError_ORDER_ERROR_INVALID_SIZE OrderError = 17
	// Order was submitted with an invalid persistence for its type
	OrderError_ORDER_ERROR_INVALID_PERSISTENCE OrderError = 18
	// Order was submitted with an invalid type field
	OrderError_ORDER_ERROR_INVALID_TYPE OrderError = 19
	// Order was stopped as it would have traded with another order submitted from the same party
	OrderError_ORDER_ERROR_SELF_TRADING OrderError = 20
	// Order was submitted, but the party did not have enough collateral to cover the fees for the order
	OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES OrderError = 21
	// Order was submitted with an incorrect or invalid market type
	OrderError_ORDER_ERROR_INCORRECT_MARKET_TYPE OrderError = 22
	// Order was submitted with invalid time in force
	OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE OrderError = 23
	// A GFN order has got to the market when it is in auction mode
	OrderError_ORDER_ERROR_GFN_ORDER_DURING_AN_AUCTION OrderError = 24
	// A GFA order has got to the market when it is in continuous trading mode
	OrderError_ORDER_ERROR_GFA_ORDER_DURING_CONTINUOUS_TRADING OrderError = 25
	// Attempt to amend order to GTT without ExpiryAt
	OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT OrderError = 26
	// Attempt to amend ExpiryAt to a value before CreatedAt
	OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT OrderError = 27
	// Attempt to amend to GTC without an ExpiryAt value
	OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT OrderError = 28
	// Amending to FOK or IOC is invalid
	OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC OrderError = 29
	// Amending to GFA or GFN is invalid
	OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN OrderError = 30
	// Amending from GFA or GFN is invalid
	OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN OrderError = 31
	// IOC orders are not allowed during auction
	OrderError_ORDER_ERROR_CANNOT_SEND_IOC_ORDER_DURING_AUCTION OrderError = 32
	// FOK orders are not allowed during auction
	OrderError_ORDER_ERROR_CANNOT_SEND_FOK_ORDER_DURING_AUCTION OrderError = 33
	// Pegged orders must be LIMIT orders
	OrderError_ORDER_ERROR_MUST_BE_LIMIT_ORDER OrderError = 34
	// Pegged orders can only have TIF GTC or GTT
	OrderError_ORDER_ERROR_MUST_BE_GTT_OR_GTC OrderError = 35
	// Pegged order must have a reference price
	OrderError_ORDER_ERROR_WITHOUT_REFERENCE_PRICE OrderError = 36
	// Buy pegged order cannot reference best ask price
	OrderError_ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE OrderError = 37
	// Pegged order offset must be <= 0
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_OR_EQUAL_TO_ZERO OrderError = 38
	// Pegged order offset must be < 0
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_THAN_ZERO OrderError = 39
	// Pegged order offset must be >= 0
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO OrderError = 40
	// Sell pegged order cannot reference best bid price
	OrderError_ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE OrderError = 41
	// Pegged order offset must be > zero
	OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO OrderError = 42
	// The party has an insufficient balance, or does not have
	// a general account to submit the order (no deposits made
	// for the required asset)
	OrderError_ORDER_ERROR_INSUFFICIENT_ASSET_BALANCE OrderError = 43
	// Cannot amend a non pegged orders details
	OrderError_ORDER_ERROR_CANNOT_AMEND_PEGGED_ORDER_DETAILS_ON_NON_PEGGED_ORDER OrderError = 44
	// We are unable to re-price a pegged order because a market price is unavailable
	OrderError_ORDER_ERROR_UNABLE_TO_REPRICE_PEGGED_ORDER OrderError = 45
	// It is not possible to amend the price of an existing pegged order
	OrderError_ORDER_ERROR_UNABLE_TO_AMEND_PRICE_ON_PEGGED_ORDER OrderError = 46
)

var (
	ErrInvalidMarketID                             = OrderError_ORDER_ERROR_INVALID_MARKET_ID
	ErrInvalidOrderID                              = OrderError_ORDER_ERROR_INVALID_ORDER_ID
	ErrOrderOutOfSequence                          = OrderError_ORDER_ERROR_OUT_OF_SEQUENCE
	ErrInvalidRemainingSize                        = OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE
	ErrOrderRemovalFailure                         = OrderError_ORDER_ERROR_REMOVAL_FAILURE
	ErrInvalidExpirationDatetime                   = OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME
	ErrEditNotAllowed                              = OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED
	ErrOrderAmendFailure                           = OrderError_ORDER_ERROR_AMEND_FAILURE
	ErrOrderNotFound                               = OrderError_ORDER_ERROR_NOT_FOUND
	ErrInvalidPartyID                              = OrderError_ORDER_ERROR_INVALID_PARTY_ID
	ErrInvalidSize                                 = OrderError_ORDER_ERROR_INVALID_SIZE
	ErrInvalidPersistence                          = OrderError_ORDER_ERROR_INVALID_PERSISTENCE
	ErrInvalidType                                 = OrderError_ORDER_ERROR_INVALID_TYPE
	ErrInvalidTimeInForce                          = OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE
	ErrPeggedOrderMustBeLimitOrder                 = OrderError_ORDER_ERROR_MUST_BE_LIMIT_ORDER
	ErrPeggedOrderMustBeGTTOrGTC                   = OrderError_ORDER_ERROR_MUST_BE_GTT_OR_GTC
	ErrPeggedOrderWithoutReferencePrice            = OrderError_ORDER_ERROR_WITHOUT_REFERENCE_PRICE
	ErrPeggedOrderBuyCannotReferenceBestAskPrice   = OrderError_ORDER_ERROR_BUY_CANNOT_REFERENCE_BEST_ASK_PRICE
	ErrPeggedOrderOffsetMustBeLessOrEqualToZero    = OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_OR_EQUAL_TO_ZERO
	ErrPeggedOrderOffsetMustBeLessThanZero         = OrderError_ORDER_ERROR_OFFSET_MUST_BE_LESS_THAN_ZERO
	ErrPeggedOrderOffsetMustBeGreaterOrEqualToZero = OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_OR_EQUAL_TO_ZERO
	ErrPeggedOrderSellCannotReferenceBestBidPrice  = OrderError_ORDER_ERROR_SELL_CANNOT_REFERENCE_BEST_BID_PRICE
	ErrPeggedOrderOffsetMustBeGreaterThanZero      = OrderError_ORDER_ERROR_OFFSET_MUST_BE_GREATER_THAN_ZERO
)
