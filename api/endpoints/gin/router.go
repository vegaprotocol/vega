package gin

import (
	"github.com/gin-gonic/gin"
	"vega/api/services"
)

func NewRouter(tradingService services.TradingService) *gin.Engine  {
	gin.SetMode(gin.TestMode)

	// Set up HTTP router and route handlers
	httpRouter := gin.New()
	httpHandlers := Handlers{
		TradingService: tradingService,
	}

	httpRouter.GET(httpHandlers.IndexRoute(), httpHandlers.Index)
	httpRouter.GET(httpHandlers.CreateOrderRoute(), httpHandlers.CreateOrder)

	return httpRouter
}
