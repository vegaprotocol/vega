package rest

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vega/api/trading/orders"
	"vega/api/trading/orders/models"
	"vega/api/trading/trades"
)

const ResultSuccess           = "success"
const ResultFailure           = "failure"
const ResultFailureValidation = "invalid"

const DefaultMarket           = "BTC/DEC18"

type Handlers struct {
	OrderService orders.OrderService
	TradeService trades.TradeService
}

func (handlers *Handlers) Index(c *gin.Context) {
	c.String(http.StatusOK, "V E G A")
}

func (handlers *Handlers) CreateOrder(c *gin.Context) {
	var o models.Order

	if err := bind(c, &o); err == nil {
		handlers.CreateOrderWithModel(c, o)
	} else {
		wasFailureWithCode(c, gin.H { "result" : ResultFailureValidation, "error" : err.Error() }, http.StatusBadRequest)
	}
}

func (handlers *Handlers) CreateOrderWithModel(c *gin.Context, o models.Order) {
	success, err :=  handlers.OrderService.CreateOrder(o)
	if success {
		wasSuccess(c, gin.H { "result" : ResultSuccess } )
	} else {
		wasFailure(c, gin.H { "result" : ResultFailure, "error" : err.Error() })
	}
}

func (handlers *Handlers) GetOrders(c *gin.Context) {
	market := c.DefaultQuery("market", DefaultMarket)
	handlers.GetOrdersWithParams(c, market)
}

func (handlers *Handlers) GetOrdersWithParams(c *gin.Context, market string) {
	orders, err := handlers.OrderService.GetOrders(market)
	if err == nil {
		wasSuccess(c, gin.H { "result" : ResultSuccess, "orders" : orders })
	} else {
		wasFailure(c, gin.H { "result" : ResultFailure, "error" : err.Error() })
	}
}

func (handlers *Handlers) GetTrades(c *gin.Context) {
	market := c.DefaultQuery("market", DefaultMarket)
	handlers.GetTradesWithParams(c, market)
}

func (handlers *Handlers) GetTradesWithParams(c *gin.Context, market string) {
	trades, err := handlers.TradeService.GetTrades(market)
	if err == nil {
		wasSuccess(c, gin.H { "result" : ResultSuccess, "trades" : trades })
	} else {
		wasFailure(c, gin.H { "result" : ResultFailure, "error" : err.Error() })
	}
}