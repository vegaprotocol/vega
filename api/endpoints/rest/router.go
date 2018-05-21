package rest

import (
	"github.com/gin-gonic/gin"
	"vega/api/trading/orders"
)

func NewRouter(orderService orders.OrderService) *gin.Engine  {
	gin.SetMode(gin.TestMode)

	// Set up HTTP request handlers
	httpHandlers := Handlers{
		OrderService: orderService,
	}

	// Set up HTTP router
	httpRouter := gin.New()
	httpRouter.GET("/", httpHandlers.Index)
	httpRouter.POST("/orders/create", httpHandlers.CreateOrder)

	return httpRouter
}
