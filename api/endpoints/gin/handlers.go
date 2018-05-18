package gin

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vega/api/services"
)

type Handlers struct {
	OrderService services.OrderService
}

const indexRoute       = "/"
const ordersRoute      = "/orders"
const createOrderRoute = ordersRoute + "/create"

func (handlers *Handlers) IndexRoute() string {
	return indexRoute
}

func (handlers *Handlers) Index(c *gin.Context) {
	c.String(http.StatusOK, "V E G A")
}

func (handlers *Handlers) CreateOrderRoute() string {
	return createOrderRoute
}

func (handlers *Handlers) CreateOrder(c *gin.Context) {

	success, err :=  handlers.OrderService.CreateOrder("BTC/DEC18", "test", 0, 10, 10)

	if success {
		c.JSON(http.StatusOK, nil)
	} else {
		c.AbortWithError(http.StatusInternalServerError, err)
	}
}


