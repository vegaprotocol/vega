package rest

import (
	"net/http"
	"vega/api/trading/orders"
	"vega/api/trading/orders/models"
	"vega/api/trading/trades"

	"github.com/gin-gonic/gin"
)

const ResponseKeyResult = "result"
const ResponseKeyError = "error"
const ResponseKeyOrders = "orders"
const ResponseKeyTrades = "trades"
const ResponseResultSuccess = "success"
const ResponseResultFailure = "failure"
const ResponseResultFailureValidation = "invalid"

const DefaultMarket = "BTC/DEC18"
const LimitMax = uint64(9223372036854775807)

type Handlers struct {
	OrderService orders.OrderService
	TradeService trades.TradeService
}

func (handlers *Handlers) Index(c *gin.Context) {
	c.String(http.StatusOK, "V E G A")
}

func (handlers *Handlers) CreateOrder(ctx *gin.Context) {
	var o models.Order

	if err := bind(ctx, &o); err == nil {
		handlers.CreateOrderWithModel(ctx, o)
	} else {
		wasFailureWithCode(ctx, gin.H{ResponseKeyResult: ResponseResultFailureValidation, "error": err.Error()}, http.StatusBadRequest)
	}
}

func (handlers *Handlers) CreateOrderWithModel(ctx *gin.Context, o models.Order) {
	success, err := handlers.OrderService.CreateOrder(ctx, o)
	if success {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess})
	} else {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}

func (handlers *Handlers) GetOrders(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	//limit := ctx.DefaultQuery("limit", LimitMax)
	handlers.GetOrdersWithParams(ctx, market, LimitMax)
}

func (handlers *Handlers) GetOrdersWithParams(ctx *gin.Context, market string, limit uint64) {
	orders, err := handlers.OrderService.GetOrders(ctx, market, limit)
	if err == nil {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess, ResponseKeyOrders: orders})
	} else {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}

func (handlers *Handlers) GetTrades(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	//limit := ctx.DefaultQuery("limit", LimitMax)
	handlers.GetTradesWithParams(ctx, market, LimitMax)
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
	//limit := ctx.DefaultQuery("limit", LimitMax)
	handlers.GetTradesForOrderWithParams(ctx, market, orderId, LimitMax)
}

func (handlers *Handlers) GetTradesForOrderWithParams(ctx *gin.Context, market string, orderId string, limit uint64) {
	trades, err := handlers.TradeService.GetTradesForOrder(ctx, market, orderId, limit)
	if err == nil {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess, ResponseKeyTrades: trades})
	} else {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}
