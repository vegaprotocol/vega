package v2

import (
	"encoding/json"
	"net/http"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
)

// responseWriter is a wrapper used to easily track response information written
// to the HTTP response writer, without polluting the handler's code.
type responseWriter struct {
	writer http.ResponseWriter
	// traceID holds a unique identifier of the incoming request. It's used to
	// trace everything related to that request. It's used on a "technical"
	// level for tracing the request through multiple components, and services.
	// It shouldn't be confused with the requestID that is an optional
	// identifier set by the client.
	traceID string
	// statusCode holds the latest status code set on the writer.
	statusCode int
	// internalError holds the error that came up during processing of the
	// request or the response.
	internalError error
	// response holds the body of the response.
	response []byte
	// requestID is the identifier the client set in the request. It's used by
	// the client to track its requests. This identifier can be empty. It
	// shouldn't be confused with the traceID.
	requestID string
}

func (lw *responseWriter) SetStatusCode(statusCode int) {
	lw.statusCode = statusCode
	lw.writer.WriteHeader(statusCode)
	if lw.statusCode == 401 {
		lw.writer.Header().Set("WWW-Authenticate", "VWT")
	}
}

func (lw *responseWriter) SetAuthorization(vwt VWT) {
	lw.writer.Header().Set("Authorization", vwt.String())
}

func (lw *responseWriter) WriteHTTPResponse(response *Response) {
	lw.writer.Header().Set("Content-Type", "application/json")

	marshaledResponse, err := json.Marshal(response)
	if err != nil {
		lw.SetStatusCode(http.StatusInternalServerError)
		lw.response = nil
		lw.internalError = err
		return
	}

	if _, err = lw.writer.Write(marshaledResponse); err != nil {
		lw.SetStatusCode(http.StatusInternalServerError)
		lw.response = nil
		lw.internalError = err
		return
	}
}

func (lw *responseWriter) WriteJSONRPCResponse(response *jsonrpc.Response) {
	lw.requestID = response.ID

	lw.writer.Header().Set("Content-Type", "application/json-rpc")

	marshaledResponse, err := json.Marshal(response)
	if err != nil {
		lw.SetStatusCode(http.StatusInternalServerError)
		lw.response = nil
		lw.internalError = err
		return
	}

	if _, err = lw.writer.Write(marshaledResponse); err != nil {
		lw.SetStatusCode(http.StatusInternalServerError)
		lw.response = nil
		lw.internalError = err
		return
	}
}

func newResponseWriter(writer http.ResponseWriter, traceID string) *responseWriter {
	writer.Header().Set("Content-Type", "application/json")

	return &responseWriter{
		writer:  writer,
		traceID: traceID,
	}
}
