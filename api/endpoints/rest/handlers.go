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

	if err := c.Bind(&o); err == nil {
		handlers.CreateOrderWithModel(c, o)
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

func wasSuccess(c *gin.Context, data gin.H) {
	sendResponse(c, data, http.StatusOK)
}

func wasFailure(c *gin.Context, data gin.H) {
	wasFailureWithCode(c, data, http.StatusInternalServerError)
}

func wasFailureWithCode(c *gin.Context, data gin.H, httpStatusCode int) {
	sendResponse(c, data, httpStatusCode)
}

// Render one of XML or JSON based on the 'Accept' header of the request
// If the header doesn't specify this, JSON is rendered
func sendResponse(c *gin.Context, data gin.H, httpStatusCode int) {
	var acceptType = "application/unknown"

	if httpStatusCode < 1 {
		// Sanity check http status code (passed as int)
		httpStatusCode = http.StatusInternalServerError
	}

	if c.Request != nil && c.Request.Header != nil {
		// Default return type is JSON
		acceptType = c.Request.Header.Get("Accept")
	}
	
	switch acceptType {
		case "application/xml":
			c.XML(httpStatusCode, data)
		default:
			c.JSON(httpStatusCode, data)
	}
}


