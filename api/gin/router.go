package gin

import (
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine  {
	gin.SetMode(gin.TestMode)

	// Set up HTTP router and route handlers
	httpRouter := gin.New()
	httpHandlers := Handlers{}

	httpRouter.GET(httpHandlers.IndexRoute(), httpHandlers.Index)
	httpRouter.POST(httpHandlers.CreateOrderRoute(), httpHandlers.CreateOrder)

	return httpRouter
}
