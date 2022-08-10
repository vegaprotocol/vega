package service

import (
	"context"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var (
	ErrCouldNotReadRequestBody = errors.New("couldn't read the HTTP request body")
	ErrRequestCannotBeBlank    = errors.New("request can't be blank")
)

func (s *Service) HandleRequestV2(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	traceID := vgrand.RandomStr(64)
	tracedCtx := context.WithValue(r.Context(), "trace-id", traceID)

	lw := newLoggedResponseWriter(w, traceID)
	defer log(s.log, lw)

	request, err := s.unmarshallRequest(traceID, r)
	if err != nil {
		lw.WriteHeader(http.StatusBadRequest)
		// Failing to unmarshall the request prevent us from retrieving the
		// request ID. So, it's left empty.
		lw.WriteBody(jsonrpc.NewErrorResponse("", err))
		return
	}

	response := s.apiV2.DispatchRequest(tracedCtx, request)

	// If the request doesn't have an ID, it's a notification. Notifications do
	// not send content back, even if an error occurred.
	if request.IsNotification() {
		lw.WriteHeader(http.StatusNoContent)
		return
	}

	if response.Error == nil {
		lw.WriteHeader(http.StatusOK)
	} else {
		if response.Error.IsInternalError() {
			lw.WriteHeader(http.StatusInternalServerError)
		} else {
			lw.WriteHeader(http.StatusBadRequest)
		}
	}
	lw.WriteBody(response)
}

func (s *Service) unmarshallRequest(traceID string, r *http.Request) (*jsonrpc.Request, *jsonrpc.ErrorDetails) {
	s.log.Info("Incoming request",
		logging.String("url", r.URL.String()),
		logging.String("trace-id", traceID),
	)

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		return nil, jsonrpc.NewParseError(ErrCouldNotReadRequestBody)
	}

	request := &jsonrpc.Request{}
	if len(body) == 0 {
		return nil, jsonrpc.NewParseError(ErrRequestCannotBeBlank)
	}

	if err := json.Unmarshal(body, request); err != nil {
		var syntaxError *json.SyntaxError
		if errors.As(err, &syntaxError) {
			return nil, jsonrpc.NewParseError(err)
		}
		return nil, jsonrpc.NewInvalidRequest(err)
	}

	strReq, _ := json.Marshal(request)
	s.log.Info("Request successfully parsed",
		logging.String("request", string(strReq)),
		logging.String("trace-id", traceID),
	)

	return request, nil
}

// loggedResponseWriter is a wrapper used to provide clean logging capabilities
// to the http.ResponseWriter interface, without polluting the handler's code.
type loggedResponseWriter struct {
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

func (lw *loggedResponseWriter) WriteHeader(statusCode int) {
	lw.statusCode = statusCode
	lw.writer.WriteHeader(statusCode)
}

func (lw *loggedResponseWriter) WriteBody(response *jsonrpc.Response) {
	lw.requestID = response.ID

	marshaledResponse, err := json.Marshal(response)
	if err != nil {
		lw.WriteHeader(http.StatusInternalServerError)
		lw.response = nil
		lw.internalError = err
		return
	}

	if _, err = lw.writer.Write(marshaledResponse); err != nil {
		lw.WriteHeader(http.StatusInternalServerError)
		lw.response = nil
		lw.internalError = err
		return
	}
}

func newLoggedResponseWriter(writer http.ResponseWriter, traceID string) *loggedResponseWriter {
	writer.Header().Set("Content-Type", "application/json")

	return &loggedResponseWriter{
		writer:  writer,
		traceID: traceID,
	}
}

func log(logger *zap.Logger, lw *loggedResponseWriter) {
	if lw.statusCode >= 400 && lw.statusCode <= 499 {
		logger.Error("Client error",
			logging.Int("http-status", lw.statusCode),
			logging.String("response", string(lw.response)),
			logging.String("request-id", lw.requestID),
			logging.String("trace-id", lw.traceID),
		)
		return
	}
	if lw.statusCode >= 500 && lw.statusCode <= 599 {
		logger.Error("Internal error",
			logging.Int("http-status", lw.statusCode),
			logging.Error(lw.internalError),
			logging.String("request-id", lw.requestID),
			logging.String("trace-id", lw.traceID),
		)
		return
	}
	logger.Info("Successful response",
		logging.Int("http-status", lw.statusCode),
		logging.String("response", string(lw.response)),
		logging.String("request-id", lw.requestID),
		logging.String("trace-id", lw.traceID),
	)
}
