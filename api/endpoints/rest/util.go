package rest

import (
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"net/http"
)

// Change c.MustBindWith() -> c.ShouldBindWith().
// Original impl here: https://github.com/gin-gonic/gin/blob/master/context.go
func bind(c *gin.Context, obj interface{}) error {
	b := binding.Default(c.Request.Method, c.ContentType())
	return c.ShouldBindWith(obj, b)
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

