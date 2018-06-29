package rest

import (
	"github.com/gin-gonic/gin"
	"vega/api/trading/orders"
	"github.com/satori/go.uuid"
	"vega/api/trading/trades"
)

func NewRouter(orderService orders.OrderService, tradeService trades.TradeService) *gin.Engine  {
	
	// Set up HTTP request handlers
	httpHandlers := Handlers{
		OrderService: orderService,
		TradeService: tradeService,
	}

	// Set up HTTP router
	router := gin.New()

	// Inject middleware (must be before route handler binding)
	router.Use(RequestIdMiddleware())

	// Routing mapping
	router.GET("/", httpHandlers.Index)
	router.GET("/trades", httpHandlers.GetTrades)
	router.GET("/orders/:orderId/trades", httpHandlers.GetTradesForOrder)
	router.GET("/orders", httpHandlers.GetOrders)
	router.POST("/orders", httpHandlers.CreateOrder)

	return router
}

func RequestIdMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Request-Id", uuid.NewV4().String())
		c.Next()
	}
}