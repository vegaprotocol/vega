package rest

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vega/api/trading/orders"
	"vega/api/trading/orders/models"
	"vega/api/trading/trades"
)

const ResponseKeyResult = "result"
const ResponseKeyError  = "error"
const ResponseKeyOrders = "orders"
const ResponseKeyTrades = "trades"
const ResponseResultSuccess = "success"
const ResponseResultFailure = "failure"
const ResponseResultFailureValidation = "invalid"

const DefaultMarket           = "BTC/DEC18"

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
		wasFailureWithCode(ctx, gin.H { ResponseKeyResult : ResponseResultFailureValidation, "error" : err.Error() }, http.StatusBadRequest)
	}
}

func (handlers *Handlers) CreateOrderWithModel(ctx *gin.Context, o models.Order) {
	success, err :=  handlers.OrderService.CreateOrder(ctx, o)
	if success {
		wasSuccess(ctx, gin.H { ResponseKeyResult : ResponseResultSuccess} )
	} else {
		wasFailure(ctx, gin.H { ResponseKeyResult : ResponseResultFailure, ResponseKeyError : err.Error() })
	}
}

func (handlers *Handlers) GetOrders(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	handlers.GetOrdersWithParams(ctx, market)
}

func (handlers *Handlers) GetOrdersWithParams(ctx *gin.Context, market string) {
	orders, err := handlers.OrderService.GetOrders(ctx, market)
	if err == nil {
		wasSuccess(ctx, gin.H { ResponseKeyResult : ResponseResultSuccess, ResponseKeyOrders : orders })
	} else {
		wasFailure(ctx, gin.H { ResponseKeyResult : ResponseResultFailure, ResponseKeyError : err.Error() })
	}
}

func (handlers *Handlers) GetTrades(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	handlers.GetTradesWithParams(ctx, market)
}

func (handlers *Handlers) GetTradesWithParams(ctx *gin.Context, market string) {
	trades, err := handlers.TradeService.GetTrades(ctx, market)
	if err == nil {
		wasSuccess(ctx, gin.H { ResponseKeyResult : ResponseResultSuccess, ResponseKeyTrades : trades })
	} else {
		wasFailure(ctx, gin.H { ResponseKeyResult : ResponseResultFailure, ResponseKeyError : err.Error() })
	}
}

func (handlers *Handlers) GetTradesForOrder(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)
	orderID := ctx.Param("orderId")
	handlers.GetTradesForOrderWithParams(ctx, market, orderID)
}

func (handlers *Handlers) GetTradesForOrderWithParams(ctx *gin.Context, market string, orderID string) {
	trades, err := handlers.TradeService.GetTradesForOrder(ctx, market, orderID)
	if err == nil {
		wasSuccess(ctx, gin.H { ResponseKeyResult : ResponseResultSuccess, ResponseKeyTrades : trades })
	} else {
		wasFailure(ctx, gin.H { ResponseKeyResult : ResponseResultFailure, ResponseKeyError : err.Error() })
	}
}
