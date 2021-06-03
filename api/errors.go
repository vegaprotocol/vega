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
	// ErrInvalidMarketID signals that the market ID does not exists
	ErrInvalidMarketID = errors.New("invalid market ID")
	// ErrMissingOrder signals that the actual payload is expected to contains an order
	ErrMissingOrder = errors.New("missing order in request payload")
	// ErrMissingPartyID signals that the payload is expected to contain a party id
	ErrMissingPartyID = errors.New("missing party id")
	// ErrMalformedRequest signals that the request was malformed
	ErrMalformedRequest = errors.New("malformed request")
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
	// ErrPrepareWithdraw is return when a withdraw request was invalid
	ErrPrepareWithdraw = errors.New("failed to prepare withdrawal")
	// ErrPrepareProposal is returned when preparation of a governance proposal fails for some reason.
	ErrPrepareProposal = errors.New("failed to prepare a proposal")
	ErrPrepareVote     = errors.New("failed to prepare vote")
	// ErrMissingProposalID returned if proposal with this id is missing
	ErrMissingProposalID = errors.New("missing proposal id")
	// ErrMissingProposalReference returned if proposal with this reference is not found
	ErrMissingProposalReference = errors.New("failed to find proposal with the reference")
	// ErrMissingWithdrawalID is returned when the ID is missing from the request
	ErrMissingWithdrawalID = errors.New("missing withdrawal ID")
	// ErrMissingOracleSpecID is returned when the ID is missing from the request
	ErrMissingOracleSpecID = errors.New("missing oracle spec ID")
	// ErrMissingDepositID is returned when the ID is missing from the request
	ErrMissingDepositID = errors.New("missing deposit ID")
)

// errorMap contains a mapping between errors and Vega numeric error codes.
var errorMap = map[string]int32{
	// General
	ErrNotMapped.Error():                  10000,
	ErrChainNotConnected.Error():          10001,
	ErrChannelClosed.Error():              10002,
	ErrEmptyMissingMarketID.Error():       10003,
	ErrEmptyMissingOrderID.Error():        10004,
	ErrEmptyMissingOrderReference.Error(): 10005,
	ErrEmptyMissingPartyID.Error():        10006,
	ErrEmptyMissingSinceTimestamp.Error(): 10007,
	ErrStreamClosed.Error():               10008,
	ErrServerShutdown.Error():             10009,
	ErrStreamInternal.Error():             10010,
	ErrInvalidMarketID.Error():            10011,
	ErrMissingOrder.Error():               10012,
	ErrMissingPartyID.Error():             10014,
	ErrMalformedRequest.Error():           10015,
	ErrMissingAsset.Error():               10017,
	ErrSubmitOrder.Error():                10018,
	ErrAmendOrder.Error():                 10019,
	ErrCancelOrder.Error():                10020,
	// Orders
	ErrOrderServiceGetByMarket.Error():      20001,
	ErrOrderServiceGetByMarketAndID.Error(): 20002,
	ErrOrderServiceGetByParty.Error():       20003,
	ErrOrderServiceGetByReference.Error():   20004,
	// Markets
	ErrMarketServiceGetMarkets.Error():    30001,
	ErrMarketServiceGetByID.Error():       30002,
	ErrMarketServiceGetDepth.Error():      30003,
	ErrMarketServiceGetMarketData.Error(): 30004,
	// Trades
	ErrTradeServiceGetByMarket.Error():         40001,
	ErrTradeServiceGetByParty.Error():          40002,
	ErrTradeServiceGetPositionsByParty.Error(): 40003,
	ErrTradeServiceGetByOrderID.Error():        40004,
	// Parties
	ErrPartyServiceGetAll.Error():  50001,
	ErrPartyServiceGetByID.Error(): 50002,
	// Candles
	ErrCandleServiceGetCandles.Error(): 60001,
	// Risk
	ErrRiskServiceGetMarginLevelsByID.Error(): 70001,
	// Accounts
	ErrAccountServiceGetMarketAccounts.Error(): 80001,
	ErrAccountServiceGetPartyAccounts.Error():  80002,
	ErrMissingWithdrawalID.Error():             80003,
	ErrMissingDepositID.Error():                80004,
	// Blockchain client
	ErrBlockchainBacklogLength.Error(): 90001,
	ErrBlockchainNetworkInfo.Error():   90002,
	ErrBlockchainGenesisTime.Error():   90003,
	// End of mapping
}

// ErrorMap returns a map of error to code, which is a mapping between
// API errors and Vega API specific numeric codes.
func ErrorMap() map[string]int32 {
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
	vegaCode, found := errorMap[apiError.Error()]
	if found {
		detail.Code = vegaCode
	} else {
		detail.Code = errorMap[ErrNotMapped.Error()]
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
