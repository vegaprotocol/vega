package rest

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vega/api"
	"vega/proto"
	"strconv"
)

const ResponseKeyResult = "result"
const ResponseKeyError = "error"
const ResponseKeyOrders = "orders"
const ResponseKeyTrades = "trades"
const ResponseResultSuccess = "success"
const ResponseResultFailure = "failure"
const ResponseResultFailureValidation = "invalid"

const DefaultMarket = "BTC/DEC18"
const LimitMax = uint64(18446744073709551615)

type Handlers struct {
	OrderService api.OrderService
	TradeService api.TradeService
}

func (handlers *Handlers) Index(c *gin.Context) {
	c.String(http.StatusOK, "V E G A")
}

func (handlers *Handlers) CreateOrder(ctx *gin.Context) {
	var o msg.Order

	if err := bind(ctx, &o); err == nil {
		handlers.CreateOrderWithModel(ctx, o)
	} else {
		wasFailureWithCode(ctx, gin.H{ResponseKeyResult: ResponseResultFailureValidation, "error": err.Error()}, http.StatusBadRequest)
	}
}

func (handlers *Handlers) CreateOrderWithModel(ctx *gin.Context, o msg.Order) {
	success, err := handlers.OrderService.CreateOrder(ctx, o)
	if success {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess})
	} else {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}

func (handlers *Handlers) GetOrders(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	limit := ctx.DefaultQuery("limit", "")
	party := ctx.DefaultQuery("party", "")
	handlers.GetOrdersWithParams(ctx, market, party, handlers.stringToUint64(limit, LimitMax))
}

func (handlers *Handlers) GetOrdersWithParams(ctx *gin.Context, market string, party string, limit uint64) {
	orders, err := handlers.OrderService.GetOrders(ctx, market, "", limit)
	if err == nil {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess, ResponseKeyOrders: orders})
	} else {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}

func (handlers *Handlers) GetTrades(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	limit := ctx.DefaultQuery("limit", "")
	handlers.GetTradesWithParams(ctx, market, handlers.stringToUint64(limit, LimitMax))
}

func (handlers *Handlers) GetTradesWithParams(ctx *gin.Context, market string, limit uint64) {
	trades, err := handlers.TradeService.GetTrades(ctx, market, limit)
	if err == nil {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess, ResponseKeyTrades: trades})
	} else {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}

func (handlers *Handlers) GetTradesForOrder(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	orderId := ctx.Param("orderId")
	limit := ctx.DefaultQuery("limit", "")
	handlers.GetTradesForOrderWithParams(ctx, market, orderId, handlers.stringToUint64(limit, LimitMax))
}

func (handlers *Handlers) GetTradesForOrderWithParams(ctx *gin.Context, market string, orderId string, limit uint64) {
	trades, err := handlers.TradeService.GetTradesForOrder(ctx, market, orderId, limit)
	if err == nil {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess, ResponseKeyTrades: trades})
	} else {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}

func (handlers *Handlers) stringToUint64(str string, defaultValue uint64) uint64 {
	i, err := strconv.Atoi(str)
	if i < 0 || err != nil {
		// todo log error when we have structured logging
		return defaultValue
	}
	return uint64(i)
}
