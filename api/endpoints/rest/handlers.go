package rest

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vega/api/trading/orders"
	"fmt"
)

type Handlers struct {
	OrderService orders.OrderService
}

func (handlers *Handlers) Index(c *gin.Context) {
	c.String(http.StatusOK, "V E G A")
}

func (handlers *Handlers) CreateOrder(c *gin.Context) {
	var o orders.Order

	if err := bind(c, &o); err == nil {
		handlers.CreateOrderWithModel(c, o)
	} else {
		wasFailureWithCode(c, gin.H { "result" : "failure_validation", "error" : err.Error() }, http.StatusBadRequest)
	}
}

func (handlers *Handlers) CreateOrderWithModel(c *gin.Context, o orders.Order) {
	fmt.Printf("HandleCreateOrder, got %+v\n", o)

	success, err :=  handlers.OrderService.CreateOrder(o.Market, o.Party, o.Side, o.Price, o.Size)

	if success {
		wasSuccess(c, gin.H { "result" : "success" } )
	} else {
		wasFailure(c, gin.H { "result" : "failure", "error" : err.Error() })
	}
}
