package proto

var (
	ErrInvalidMarketID                 = OrderError_ORDER_ERROR_INVALID_MARKET_ID
	ErrInvalidOrderID                  = OrderError_ORDER_ERROR_INVALID_ORDER_ID
	ErrOrderOutOfSequence              = OrderError_ORDER_ERROR_OUT_OF_SEQUENCE
	ErrInvalidRemainingSize            = OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE
	ErrVegaTimeFailure                 = OrderError_ORDER_ERROR_TIME_FAILURE
	ErrOrderRemovalFailure             = OrderError_ORDER_ERROR_REMOVAL_FAILURE
	ErrInvalidExpirationDatetime       = OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME
	ErrInvalidOrderReference           = OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE
	ErrEditNotAllowed                  = OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED
	ErrOrderAmendFailure               = OrderError_ORDER_ERROR_AMEND_FAILURE
	ErrOrderNotFound                   = OrderError_ORDER_ERROR_NOT_FOUND
	ErrInvalidPartyID                  = OrderError_ORDER_ERROR_INVALID_PARTY_ID
	ErrInvalidSize                     = OrderError_ORDER_ERROR_INVALID_SIZE
	ErrInvalidPersistence              = OrderError_ORDER_ERROR_INVALID_PERSISTENCE
	ErrInvalidType                     = OrderError_ORDER_ERROR_INVALID_TYPE
	ErrInsufficientFundsToPayFees      = OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES
	ErrInvalidTimeInForce              = OrderError_ORDER_ERROR_INVALID_TIME_IN_FORCE
	ErrCannotAmendToGTTWithoutExpiryAt = OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT
	ErrExpiryAtBeforeCreatedAt         = OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT
	ErrCannotHaveGTCAndExpiryAt        = OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT
	ErrCannotAmendToFOKAndIOC          = OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC
	ErrCannotAmendToGFAOrGFN           = OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN
	ErrCannotAmendFromGFAOrGFN         = OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN
)

func IsOrderError(err error) (OrderError, bool) {
	oerr, ok := err.(OrderError)
	return oerr, ok
}

func (err OrderError) Error() string {
	switch err {
	case OrderError_ORDER_ERROR_NONE:
		return "none"
	case OrderError_ORDER_ERROR_INVALID_MARKET_ID:
		return "OrderError: Invalid Market ID"
	case OrderError_ORDER_ERROR_INVALID_ORDER_ID:
		return "OrderError: Invalid Order ID"
	case OrderError_ORDER_ERROR_OUT_OF_SEQUENCE:
		return "OrderError: Order Out Of Sequence"
	case OrderError_ORDER_ERROR_INVALID_REMAINING_SIZE:
		return "OrderError: Invalid Remaining Size"
	case OrderError_ORDER_ERROR_TIME_FAILURE:
		return "OrderError: Vega Time failure"
	case OrderError_ORDER_ERROR_REMOVAL_FAILURE:
		return "OrderError: Order Removal Failure"
	case OrderError_ORDER_ERROR_INVALID_EXPIRATION_DATETIME:
		return "OrderError: Invalid Expiration Datetime"
	case OrderError_ORDER_ERROR_INVALID_ORDER_REFERENCE:
		return "OrderError: Invalid Order Reference"
	case OrderError_ORDER_ERROR_EDIT_NOT_ALLOWED:
		return "OrderError: Edit Not Allowed"
	case OrderError_ORDER_ERROR_AMEND_FAILURE:
		return "OrderError: Order Amend Failure"
	case OrderError_ORDER_ERROR_NOT_FOUND:
		return "OrderError: Order Not Found"
	case OrderError_ORDER_ERROR_INVALID_PARTY_ID:
		return "OrderError: Invalid Party ID"
	case OrderError_ORDER_ERROR_MARKET_CLOSED:
		return "OrderError: Market Closed"
	case OrderError_ORDER_ERROR_MARGIN_CHECK_FAILED:
		return "OrderError: Margin Check Failed"
	case OrderError_ORDER_ERROR_MISSING_GENERAL_ACCOUNT:
		return "OrderError: Missing General Account"
	case OrderError_ORDER_ERROR_INTERNAL_ERROR:
		return "OrderError: Internal Error"
	case OrderError_ORDER_ERROR_INVALID_SIZE:
		return "OrderError: Invalid Size"
	case OrderError_ORDER_ERROR_INVALID_PERSISTENCE:
		return "OrderError: Invalid Persistence"
	case OrderError_ORDER_ERROR_INSUFFICIENT_FUNDS_TO_PAY_FEES:
		return "OrderError: Insufficient funds to pay fees"
	case OrderError_ORDER_ERROR_SELF_TRADING:
		return "OrderError: Self trading"
	case OrderError_ORDER_ERROR_INVALID_TYPE:
		return "OrderError: Invalid Type"
	case OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GTT_WITHOUT_EXPIRYAT:
		return "OrderError: Cannot amend order to GTT without an expiryAt field"
	case OrderError_ORDER_ERROR_EXPIRYAT_BEFORE_CREATEDAT:
		return "OrderError: ExpiryAt field must not be before CreatedAt"
	case OrderError_ORDER_ERROR_CANNOT_HAVE_GTC_AND_EXPIRYAT:
		return "OrderError: Cannot set ExpiryAt and GTC"
	case OrderError_ORDER_ERROR_CANNOT_AMEND_TO_FOK_OR_IOC:
		return "OrderError: Cannot amend TIF to FOK or IOC"
	case OrderError_ORDER_ERROR_CANNOT_AMEND_TO_GFA_OR_GFN:
		return "OrderError: Cannot amend TIF to GFA or GFN"
	case OrderError_ORDER_ERROR_CANNOT_AMEND_FROM_GFA_OR_GFN:
		return "OrderError: Cannot amend TIF from GFA or GFN"
	default:
		return "invalid OrderError"
	}
}
