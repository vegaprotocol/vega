package api

import (
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// ErrorMap contains a mapping between errors and Vega numeric error codes.
var ErrorMap map[error]int32

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

	ErrOrderServiceGetByMarket      = errors.New("failed to get orders for market")
	ErrOrderServiceGetByMarketAndID = errors.New("failed to get orders for market and ID")
	ErrOrderServiceGetByParty       = errors.New("failed to get orders for party")
	ErrOrderServiceGetByReference   = errors.New("failed to get orders for reference")

	ErrTradeServiceGetByParty          = errors.New("failed to get trades for party")
	ErrTradeServiceGetByMarket         = errors.New("failed to get trades for market")
	ErrTradeServiceGetPositionsByParty = errors.New("failed to get positions for party")
	ErrTradeServiceGetByOrderID        = errors.New("failed to get trades for order ID")

	ErrMarketServiceGetMarkets    = errors.New("failed to get markets")
	ErrMarketServiceGetByID       = errors.New("failed to get market for ID")
	ErrMarketServiceGetDepth      = errors.New("failed to get market depth")
	ErrMarketServiceGetMarketData = errors.New("failed to get market data")

	ErrAccountServiceGetMarketAccounts = errors.New("failed to get market accounts")
	ErrAccountServiceGetPartyAccounts  = errors.New("failed to get party accounts")

	ErrRiskServiceGetMarginLevelsByID = errors.New("failed to get margin levels")

	ErrCandleServiceGetCandles = errors.New("failed to get candles")

	ErrPartyServiceGetAll  = errors.New("failed to get parties")
	ErrPartyServiceGetByID = errors.New("failed to get party for ID")

	ErrTimeServiceGetTimeNow = errors.New("failed to get time now")

	ErrBlockchainBacklogLength = errors.New("failed to get backlog length from blockchain")
	ErrBlockchainNetworkInfo   = errors.New("failed to get network info from blockchain")
	ErrBlockchainGenesisTime   = errors.New("failed to get genesis time from blockchain")
)

func InitErrorMap() {
	em := make(map[error]int32)
	// General
	em[ErrNotMapped] = 10000
	em[ErrChainNotConnected] = 10001
	em[ErrChannelClosed] = 10002
	em[ErrEmptyMissingMarketID] = 10003
	em[ErrEmptyMissingOrderID] = 10004
	em[ErrEmptyMissingOrderReference] = 10005
	em[ErrEmptyMissingPartyID] = 10006
	em[ErrEmptyMissingSinceTimestamp] = 10007
	em[ErrStreamClosed] = 10008
	em[ErrServerShutdown] = 10009
	em[ErrStreamInternal] = 10010
	// Orders
	em[ErrOrderServiceGetByMarket] = 20001
	em[ErrOrderServiceGetByMarketAndID] = 20002
	em[ErrOrderServiceGetByParty] = 20003
	em[ErrOrderServiceGetByReference] = 20004
	// Markets
	em[ErrMarketServiceGetMarkets] = 30001
	em[ErrMarketServiceGetByID] = 30002
	em[ErrMarketServiceGetDepth] = 30003
	em[ErrMarketServiceGetMarketData] = 30004
	// Trades
	em[ErrTradeServiceGetByMarket] = 40001
	em[ErrTradeServiceGetByParty] = 40002
	em[ErrTradeServiceGetPositionsByParty] = 40003
	em[ErrTradeServiceGetByOrderID] = 40004
	// Parties
	em[ErrPartyServiceGetAll] = 50001
	em[ErrPartyServiceGetByID] = 50002
	// Candles
	em[ErrCandleServiceGetCandles] = 60001
	// Risk
	em[ErrRiskServiceGetMarginLevelsByID] = 70001
	// Accounts
	em[ErrAccountServiceGetMarketAccounts] = 80001
	em[ErrAccountServiceGetPartyAccounts] = 80002
	// Blockchain client
	em[ErrBlockchainBacklogLength] = 90001
	em[ErrBlockchainNetworkInfo] = 90002
	em[ErrBlockchainGenesisTime] = 90003

	ErrorMap = em
}

func apiError(grpcCode codes.Code, errs ...error) error {
	s := status.Newf(grpcCode, "%v error", grpcCode)

	for _, err := range errs {
		detail := types.ErrorDetail{
			Message: err.Error(),
		}
		vegaCode, found := ErrorMap[err]
		if found {
			detail.Code = vegaCode
		} else {
			detail.Code = ErrorMap[ErrNotMapped]
		}
		s, _ = s.WithDetails(&detail)
	}

	return s.Err()
}
