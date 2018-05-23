package rest

import (
	"net/http"
	"github.com/gin-gonic/gin"
	"vega/api/trading/orders"
	"vega/api/trading/orders/models"
)

const ResultSuccess           = "success"
const ResultFailure           = "failure"
const ResultFailureValidation = "invalid"

type Handlers struct {
	OrderService orders.OrderService
}

func (handlers *Handlers) Index(c *gin.Context) {
	c.String(http.StatusOK, "V E G A")
}

func (handlers *Handlers) CreateOrder(c *gin.Context) {
	var o models.Order

	if err := bind(c, &o); err == nil {
		handlers.CreateOrderWithModel(c, o)
	} else {
		wasFailureWithCode(c, gin.H { "result" : ResultFailureValidation, "error" : err.Error() }, http.StatusBadRequest)
	}
}

func (handlers *Handlers) CreateOrderWithModel(c *gin.Context, o models.Order) {
	//fmt.Printf("HandleCreateOrderWithModel, got %+v\n", o)
	success, err :=  handlers.OrderService.CreateOrder(o)

	if success {
		wasSuccess(c, gin.H { "result" : ResultSuccess } )
	} else {
		wasFailure(c, gin.H { "result" : ResultFailure, "error" : err.Error() })
	}
}
