package api

import (
	types "code.vegaprotocol.io/vega/proto"
	"github.com/pkg/errors"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Errors
var (
	ErrAccountService = errors.New("account service")
	ErrBlockchain     = errors.New("blockchain")
	ErrCandleService  = errors.New("candle service")
	ErrMarketService  = errors.New("market service")
	ErrOrderService   = errors.New("order service")
	ErrPartyService   = errors.New("party service")
	ErrRiskService    = errors.New("risk service")
	ErrTimeService    = errors.New("time service")
	ErrTradeService   = errors.New("trade service")
)

// ErrorMap contains a mapping between errors and Vega numeric error codes.
var ErrorMap map[error]int32

func initErrorMap() {
	em := make(map[error]int32)

	em[ErrAccountService] = 1000

	em[ErrBlockchain] = 1100
	em[ErrChainNotConnected] = 1101

	em[ErrCandleService] = 1200

	em[ErrMarketService] = 1300

	em[ErrOrderService] = 1400

	em[ErrPartyService] = 1500

	em[ErrRiskService] = 1600

	em[ErrTimeService] = 1700

	em[ErrTradeService] = 1800

	ErrorMap = em
}

func apiError(code codes.Code, errs ...error) error {
	s := status.Newf(code, "API call failed: %v", code)

	for _, err := range errs {
		detail := types.ErrorDetail{
			Message: err.Error(),
		}
		vcode, found := ErrorMap[err]
		if found {
			detail.Code = vcode
		}
		s, _ = s.WithDetails(&detail)
	}

	return s.Err()
}
