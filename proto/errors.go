package proto

var (
	ErrInvalidMarketID           = OrderError_INVALID_MARKET_ID
	ErrInvalidOrderID            = OrderError_INVALID_ORDER_ID
	ErrOrderOutOfSequence        = OrderError_ORDER_OUT_OF_SEQUENCE
	ErrInvalidRemainingSize      = OrderError_INVALID_REMAINING_SIZE
	ErrVegaTimeFailure           = OrderError_TIME_FAILURE
	ErrOrderRemovalFailure       = OrderError_ORDER_REMOVAL_FAILURE
	ErrInvalidExpirationDatetime = OrderError_INVALID_EXPIRATION_DATETIME
	ErrInvalidOrderReference     = OrderError_INVALID_ORDER_REFERENCE
	ErrEditNotAllowed            = OrderError_EDIT_NOT_ALLOWED
	ErrOrderAmendFailure         = OrderError_ORDER_AMEND_FAILURE
	ErrOrderNotFound             = OrderError_ORDER_NOT_FOUND
)

func IsOrderError(err error) (OrderError, bool) {
	oerr, ok := err.(OrderError)
	return oerr, ok
}

func (err OrderError) Error() string {
	switch err {
	case OrderError_NONE:
		return "none"
	case OrderError_INVALID_MARKET_ID:
		return "OrderError: Invalid Market ID"
	case OrderError_INVALID_ORDER_ID:
		return "OrderError: Invalid Order ID"
	case OrderError_ORDER_OUT_OF_SEQUENCE:
		return "OrderError: Order Out Of Sequence"
	case OrderError_INVALID_REMAINING_SIZE:
		return "OrderError: Invalid Remaining Size"
	case OrderError_TIME_FAILURE:
		return "OrderError: Vega Time failure"
	case OrderError_ORDER_REMOVAL_FAILURE:
		return "OrderError: Order Removal Failure"
	case OrderError_INVALID_EXPIRATION_DATETIME:
		return "OrderError: Invalid Expiration Datetime"
	case OrderError_INVALID_ORDER_REFERENCE:
		return "OrderError: Invalid Order Reference"
	case OrderError_EDIT_NOT_ALLOWED:
		return "OrderError: Edit Not Allowed"
	case OrderError_ORDER_AMEND_FAILURE:
		return "OrderError: Order Amend Failure"
	case OrderError_ORDER_NOT_FOUND:
		return "OrderError: Order Not Found"
	case OrderError_INVALID_PARTY_ID:
		return "OrderError: Invalid Party ID"
	case OrderError_MARKET_CLOSED:
		return "OrderError: Market Closed"
	case OrderError_MARGIN_CHECK_FAILED:
		return "OrderError: Margin Check Failed"
	case OrderError_VEGA_INTERNAL_ERROR:
		return "OrderError: Vega Internal Error"
	default:
		return "invalid OrderError"
	}
}
