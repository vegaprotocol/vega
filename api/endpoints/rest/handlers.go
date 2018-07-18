package rest

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"vega/api"
	"vega/msg"

	"github.com/gin-gonic/gin"
)

const ResponseKeyResult = "result"
const ResponseKeyError = "error"
const ResponseKeyOrders = "orders"
const ResponseKeyTrades = "trades"
const ResponseKeyCandles = "candles"
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
	success, err := handlers.OrderService.CreateOrder(ctx, &o)
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

func (handlers *Handlers) GetCandleChart(ctx *gin.Context) {
	market := ctx.DefaultQuery("market", DefaultMarket)

	sinceStr := ctx.DefaultQuery("since", "2018-07-09T12:00:00Z")

	fmt.Printf("sinceStr: %s\n", sinceStr)
	since, err := time.Parse(time.RFC3339, sinceStr)
	if err != nil {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}


	currentTime := time.Now().UTC()
	since = currentTime.Add(time.Duration(-604800) * time.Second)


	fmt.Printf("%+v, %+v", since, currentTime)
	fmt.Println()

	intervalStr := ctx.DefaultQuery("interval", "2")
	fmt.Printf("intervalStr: %s\n", intervalStr)
	interval, err := strconv.ParseUint(intervalStr, 10, 64)
	if err != nil {
		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}

	candles, err := handlers.TradeService.GetCandlesChart(ctx, market, since, interval)
	if err == nil {
		wasSuccess(ctx, gin.H{ResponseKeyResult: ResponseResultSuccess, ResponseKeyCandles: candles})
	} else {

		fmt.Errorf("err %v", err)

		wasFailure(ctx, gin.H{ResponseKeyResult: ResponseResultFailure, ResponseKeyError: err.Error()})
	}
}
