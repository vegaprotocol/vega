package api

import (
	"github.com/pkg/errors"

	types "code.vegaprotocol.io/vega/proto"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// API Errors and descriptions.
var (
	// ErrChainNotConnected signals to the user that he cannot access a given endpoint
	// which require the chain, but the chain is actually offline
	ErrChainNotConnected = errors.New("chain not connected")
	// ErrChannelClosed signals that the channel streaming data is closed
	ErrChannelClosed = errors.New("channel closed")
	// ErrEmptyMissingMarketID signals to the caller that the request expected a
	// market id but the field is missing or empty
	ErrEmptyMissingMarketID = errors.New("empty or missing market ID")
	// ErrEmptyMissingOrderID signals to the caller that the request expected an
	// order id but the field is missing or empty
	ErrEmptyMissingOrderID = errors.New("empty or missing order ID")
	// ErrEmptyMissingOrderReference signals to the caller that the request expected an
	// order reference but the field is missing or empty
	ErrEmptyMissingOrderReference = errors.New("empty or missing order reference")
	// ErrEmptyMissingPartyID signals to the caller that the request expected a
	// party id but the field is missing or empty
	ErrEmptyMissingPartyID = errors.New("empty or missing party ID")
	// ErrEmptyMissingSinceTimestamp signals to the caller that the request expected a
	// timestamp but the field is missing or empty
	ErrEmptyMissingSinceTimestamp = errors.New("empty or missing since-timestamp")
	// ErrServerShutdown signals to the client that the server  is shutting down
	ErrServerShutdown = errors.New("server shutdown")
	// ErrStreamClosed signals to the users that the grpc stream is closing
	ErrStreamClosed = errors.New("stream closed")
	// ErrStreamInternal signals to the users that the grpc stream has an internal problem
	ErrStreamInternal = errors.New("internal stream failure")
	// ErrNotMapped is when an error cannot be found in the current error map/lookup table
	ErrNotMapped = errors.New("error not found in error lookup table")
	// ErrAuthDisabled signals to the caller that the authentication is disabled
	ErrAuthDisabled = errors.New("auth disabled")
	// ErrInvalidCredentials signals that the credentials specified by the client are invalid
	ErrInvalidCredentials = errors.New("invalid credentials")
	// ErrMissingToken signals that a token was required but is missing with this request
	ErrMissingToken = errors.New("missing token")
	// ErrInvalidToken signals that a token was not valid for this request
	ErrInvalidToken = errors.New("invalid token")
	// ErrInvalidMarketID signals that the market ID does not exists
	ErrInvalidMarketID = errors.New("invalid market ID")
	// ErrMissingOrder signals that the actual payload is expected to contains an order
	ErrMissingOrder = errors.New("missing order in request payload")
	// ErrMissingTraderID signals that the payload is expected to contain a trader id
	ErrMissingTraderID = errors.New("missing trader id")
	// ErrMissingPartyID signals that the payload is expected to contain a party id
	ErrMissingPartyID = errors.New("missing party id")
	// ErrMalformedRequest signals that the request was malformed
	ErrMalformedRequest = errors.New("malformed request")
	// ErrInvalidWithdrawAmount signals that the amount of money to withdraw is invalid
	// usually the party specified an amount of 0
	ErrInvalidWithdrawAmount = errors.New("invalid withdraw amount (must be > 0)")
	// ErrMissingAsset signals that an asset was required but not specified
	ErrMissingAsset = errors.New("missing asset")
	// ErrSubmitOrder is returned when submitting an order fails for some reason.
	ErrSubmitOrder = errors.New("submit order failure")
	// ErrAmendOrder is returned when amending an order fails for some reason.
	ErrAmendOrder = errors.New("amend order failure")
	// ErrCancelOrder is returned when cancelling an order fails for some reason.
	ErrCancelOrder = errors.New("cancel order failure")
	// OrderService...
	ErrOrderServiceGetByMarket      = errors.New("failed to get orders for market")
	ErrOrderServiceGetByMarketAndID = errors.New("failed to get orders for market and ID")
	ErrOrderServiceGetByParty       = errors.New("failed to get orders for party")
	ErrOrderServiceGetByReference   = errors.New("failed to get orders for reference")
	ErrMissingOrderIDParameter      = errors.New("missing orderID parameter")
	ErrMissingReferenceIDParameter  = errors.New("missing referenceID parameter")
	ErrOrderAndReferenceMismatch    = errors.New("referenceID and orderID do not match up")
	ErrOrderNotFound                = errors.New("order not found")
	// TradeService...
	ErrTradeServiceGetByParty          = errors.New("failed to get trades for party")
	ErrTradeServiceGetByMarket         = errors.New("failed to get trades for market")
	ErrTradeServiceGetPositionsByParty = errors.New("failed to get positions for party")
	ErrTradeServiceGetByOrderID        = errors.New("failed to get trades for order ID")
	// MarketService...
	ErrMarketServiceGetMarkets    = errors.New("failed to get markets")
	ErrMarketServiceGetByID       = errors.New("failed to get market for ID")
	ErrMarketServiceGetDepth      = errors.New("failed to get market depth")
	ErrMarketServiceGetMarketData = errors.New("failed to get market data")
	// AccountService...
	ErrAccountServiceGetMarketAccounts = errors.New("failed to get market accounts")
	// AccountService...
	ErrAccountServiceGetFeeInfrastructureAccounts = errors.New("failed to get fee infrastructure accounts")
	ErrAccountServiceGetPartyAccounts             = errors.New("failed to get party accounts")
	// RiskService...
	ErrRiskServiceGetMarginLevelsByID = errors.New("failed to get margin levels")
	// CandleService...
	ErrCandleServiceGetCandles = errors.New("failed to get candles")
	// PartyService...
	ErrPartyServiceGetAll  = errors.New("failed to get parties")
	ErrPartyServiceGetByID = errors.New("failed to get party for ID")
	// TimeService...
	ErrTimeServiceGetTimeNow = errors.New("failed to get time now")
	// Blockchain...
	ErrBlockchainBacklogLength = errors.New("failed to get backlog length from blockchain")
	ErrBlockchainNetworkInfo   = errors.New("failed to get network info from blockchain")
	ErrBlockchainGenesisTime   = errors.New("failed to get genesis time from blockchain")
	ErrBlockchainChainID       = errors.New("failed to get chain ID from blockchain")
	// Governance...
	// ErrPrepareProposal is returned when preparation of a governance proposal fails for some reason.
	ErrPrepareProposal = errors.New("failed to prepare a proposal")
	ErrPrepareVote     = errors.New("failed to prepare vote")
	// ErrMissingProposalID returned if proposal with this id is missing
	ErrMissingProposalID = errors.New("missing proposal id")
	// ErrMissingProposalReference returned if proposal with this reference is not found
	ErrMissingProposalReference = errors.New("failed to find proposal with the reference")
)

// errorMap contains a mapping between errors and Vega numeric error codes.
var errorMap = map[error]int32{
	// General
	ErrNotMapped:                  10000,
	ErrChainNotConnected:          10001,
	ErrChannelClosed:              10002,
	ErrEmptyMissingMarketID:       10003,
	ErrEmptyMissingOrderID:        10004,
	ErrEmptyMissingOrderReference: 10005,
	ErrEmptyMissingPartyID:        10006,
	ErrEmptyMissingSinceTimestamp: 10007,
	ErrStreamClosed:               10008,
	ErrServerShutdown:             10009,
	ErrStreamInternal:             10010,
	ErrInvalidMarketID:            10011,
	ErrMissingOrder:               10012,
	ErrMissingTraderID:            10013,
	ErrMissingPartyID:             10014,
	ErrMalformedRequest:           10015,
	ErrInvalidWithdrawAmount:      10016,
	ErrMissingAsset:               10017,
	ErrSubmitOrder:                10018,
	ErrAmendOrder:                 10019,
	ErrCancelOrder:                10020,
	ErrAuthDisabled:               10021,
	ErrInvalidCredentials:         10022,
	ErrMissingToken:               10023,
	ErrInvalidToken:               10024,
	// Orders
	ErrOrderServiceGetByMarket:      20001,
	ErrOrderServiceGetByMarketAndID: 20002,
	ErrOrderServiceGetByParty:       20003,
	ErrOrderServiceGetByReference:   20004,
	// Markets
	ErrMarketServiceGetMarkets:    30001,
	ErrMarketServiceGetByID:       30002,
	ErrMarketServiceGetDepth:      30003,
	ErrMarketServiceGetMarketData: 30004,
	// Trades
	ErrTradeServiceGetByMarket:         40001,
	ErrTradeServiceGetByParty:          40002,
	ErrTradeServiceGetPositionsByParty: 40003,
	ErrTradeServiceGetByOrderID:        40004,
	// Parties
	ErrPartyServiceGetAll:  50001,
	ErrPartyServiceGetByID: 50002,
	// Candles
	ErrCandleServiceGetCandles: 60001,
	// Risk
	ErrRiskServiceGetMarginLevelsByID: 70001,
	// Accounts
	ErrAccountServiceGetMarketAccounts: 80001,
	ErrAccountServiceGetPartyAccounts:  80002,
	// Blockchain client
	ErrBlockchainBacklogLength: 90001,
	ErrBlockchainNetworkInfo:   90002,
	ErrBlockchainGenesisTime:   90003,
	// End of mapping
}

// ErrorMap returns a map of error to code, which is a mapping between
// API errors and Vega API specific numeric codes.
func ErrorMap() map[error]int32 {
	return errorMap
}

// apiError is a helper function to build the Vega specific Error Details that
// can be returned by gRPC API and therefore also REST, GraphQL will be mapped too.
// It takes a standardised grpcCode, a Vega specific apiError, and optionally one
// or more internal errors (error from the core, rather than API).
func apiError(grpcCode codes.Code, apiError error, innerErrors ...error) error {
	s := status.Newf(grpcCode, "%v error", grpcCode)
	// Create the API specific error detail for error e.g. missing party ID
	detail := types.ErrorDetail{
		Message: apiError.Error(),
	}
	// Lookup the API specific error in the table, return not found/not mapped
	// if a code has not yet been added to the map, can happen if developer misses
	// a step, periodic checking/ownership of API package can keep this up to date.
	vegaCode, found := errorMap[apiError]
	if found {
		detail.Code = vegaCode
	} else {
		detail.Code = errorMap[ErrNotMapped]
	}
	// If there is an inner error (and possibly in the future, a config to turn this
	// level of detail on/off) then process and append to inner.
	first := true
	for _, err := range innerErrors {
		if !first {
			detail.Inner += ", "
		}
		detail.Inner += err.Error()
		first = false
	}
	// Pack the Vega domain specific errorDetails into the status returned by gRPC domain.
	s, _ = s.WithDetails(&detail)
	return s.Err()
}
