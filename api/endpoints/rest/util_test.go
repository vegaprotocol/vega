package rest

import (
	"net/http"
	"testing"
	"github.com/gin-gonic/gin"
	"net/http/httptest"
	"github.com/stretchr/testify/assert"
)

func TestHandlers_WasSuccessJson(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	wasSuccess(context, gin.H { "chuck" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t,"{\"chuck\":\"rhodes\"}", w.Body.String())
}

func TestHandlers_WasFailureJson(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	wasFailure(context, gin.H { "wendy" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"{\"wendy\":\"rhodes\"}", w.Body.String())

}

func TestHandlers_SendResponseJson(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context, _ := gin.CreateTestContext(w)

	sendResponse(context, gin.H { "robert" : "axelrod" }, http.StatusBadGateway)

	assert.Equal(t, w.Code, http.StatusBadGateway)
	assert.Equal(t,"{\"robert\":\"axelrod\"}", w.Body.String())
}

func TestHandlers_WasSuccessXml(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context := buildContextWithDefaultRouteRequest(w,true)

	wasSuccess(context, gin.H { "chuck" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusOK)
	assert.Equal(t,"<map><chuck>rhodes</chuck></map>", w.Body.String())
}

func TestHandlers_WasFailureXml(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context := buildContextWithDefaultRouteRequest(w,true)

	wasFailure(context, gin.H { "wendy" : "rhodes" })

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"<map><wendy>rhodes</wendy></map>", w.Body.String())

}

func TestHandlers_SendResponseXml(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context := buildContextWithDefaultRouteRequest(w,true)

	sendResponse(context, gin.H { "robert" : "axelrod" }, http.StatusBadGateway)

	assert.Equal(t, w.Code, http.StatusBadGateway)
	assert.Equal(t,"<map><robert>axelrod</robert></map>", w.Body.String())
}

func TestHandlers_SendResponseInvalidRequestStatusCode(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	context := buildContextWithDefaultRouteRequest(w,true)

	sendResponse(context, gin.H { "test" : "test" }, -1)

	assert.Equal(t, w.Code, http.StatusInternalServerError)
	assert.Equal(t,"<map><test>test</test></map>", w.Body.String())
}

// Helpers
func buildContextWithDefaultRouteRequest(w http.ResponseWriter, requestXml bool) *gin.Context {

	context, _ := gin.CreateTestContext(w)
	context.Request, _ = http.NewRequest(http.MethodGet, "/", nil)
	if requestXml {
		context.Request.Header.Add("Accept", "application/xml")
	}
	return context
}