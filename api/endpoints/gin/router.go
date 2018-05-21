package gin

import (
	"github.com/gin-gonic/gin"
	"vega/api/services"
)

const indexRoute       = "/"
const ordersRoute      = "/orders"
const createOrderRoute = ordersRoute + "/create"

func NewRouter(orderService services.OrderService) *gin.Engine  {
	gin.SetMode(gin.TestMode)

	// Set up HTTP request handlers
	httpHandlers := Handlers{
		OrderService: orderService,
	}

	// Set up HTTP router
	httpRouter := gin.New()
	httpRouter.GET(indexRoute, httpHandlers.Index)
	httpRouter.POST(createOrderRoute, httpHandlers.CreateOrder)

	return httpRouter
}
