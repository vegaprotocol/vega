package gin

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

type Handlers struct {
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
	message := "ORDER created"
	c.String(http.StatusOK, message)
}


