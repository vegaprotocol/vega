package gin

import (
	"net/http"
	"github.com/gin-gonic/gin"
)

func NewRouter() *gin.Engine  {

	// Set up HTTP handlers
	router := gin.New()
	router.GET("/", func(c *gin.Context) {

		message := "V E G A"
		c.String(http.StatusOK, message)
	})

	return router
}
