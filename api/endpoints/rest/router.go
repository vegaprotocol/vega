package rest

import (
	"github.com/gin-gonic/gin"
	"vega/api/trading/orders"
	"github.com/satori/go.uuid"
)

func NewRouter(orderService orders.OrderService) *gin.Engine  {
	
	// Set up HTTP request handlers
	httpHandlers := Handlers{
		OrderService: orderService,
	}

	// Set up HTTP router
	router := gin.New()

	// Inject middleware (must be before route handler binding)
	router.Use(RequestIdMiddleware())
	
	router.GET("/", httpHandlers.Index)
	router.POST("/orders", httpHandlers.CreateOrder)
	router.GET("/orders", httpHandlers.GetOrders)

	// Perhaps we'll do this in the future:
	// https://stackoverflow.com/a/42968011

	return router
}

func RequestIdMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("X-Request-Id", uuid.NewV4().String())
		c.Next()
	}
}