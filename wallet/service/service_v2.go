package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"

	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

var (
	ErrCouldNotReadRequestBody  = errors.New("couldn't read the HTTP request body")
	ErrRequestCannotBeBlank     = errors.New("the request can't be blank")
	ErrNoneOfRequiredHeadersSet = errors.New("the request is expected to specified the Origin or the Referer header")
)

type TraceIDKey struct{}

func (s *Service) CheckHealthV2(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.log.Debug("New request",
		logging.String("url", vfmt.Escape(r.URL.String())),
	)

	w.WriteHeader(http.StatusOK)
}

type ListMethodsV2Response struct {
	RegisteredMethods []string `json:"registeredMethods"`
}

func (s *Service) ListMethodsV2(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	s.log.Info("New request",
		logging.String("url", vfmt.Escape(r.URL.String())),
	)

	registeredMethods := s.apiV2.RegisteredMethods()

	body, _ := json.Marshal(ListMethodsV2Response{
		RegisteredMethods: registeredMethods,
	})

	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		s.log.Info("Internal error",
			logging.Int("http-status", http.StatusInternalServerError),
			logging.Error(err),
		)
		return
	}

	s.log.Info("Success",
		logging.Int("http-status", http.StatusOK),
	)
}

func (s *Service) HandleRequestV2(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	traceID := vgrand.RandomStr(64)

	s.log.Info("New request",
		logging.String("url", vfmt.Escape(r.URL.String())),
		logging.String("trace-id", traceID),
	)

	lw := newLoggedResponseWriter(w, traceID)
	defer logResponse(s.log, lw)

	hostname, err := resolveHostname(r)
	if err != nil || hostname == "" {
		s.log.Error("Could not resolve the hostname", zap.Error(err))
		lw.WriteHeader(http.StatusUnauthorized)
		return
	}

	request, errDetails := s.unmarshallRequest(traceID, r)
	if errDetails != nil {
		lw.WriteHeader(http.StatusBadRequest)
		// Failing to unmarshall the request prevent us from retrieving the
		// request ID. So, it's left empty.
		lw.WriteBody(jsonrpc.NewErrorResponse("", errDetails))
		return
	}

	response := s.apiV2.DispatchRequest(r.Context(), *request, jsonrpc.RequestMetadata{
		TraceID:  traceID,
		Hostname: hostname,
	})

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
	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)
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
		logging.String("request", vfmt.Escape(string(strReq))),
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

func resolveHostname(r *http.Request) (string, error) {
	origin := r.Header.Get("Origin")
	if origin != "" {
		parsedOrigin, err := url.Parse(origin)
		if err != nil {
			return origin, nil //nolint:nilerr
		}
		if parsedOrigin.Host != "" {
			return parsedOrigin.Host, nil
		}
		return origin, nil
	}

	// In some scenario, the Origin can be set to null by the browser for privacy
	// reasons. Since we are not trying to fingerprint or spoof anyone, we
	// attempt to parse the Referer.
	referer := r.Header.Get("Referer")
	if referer != "" {
		parsedReferer, err := url.Parse(referer)
		if err != nil {
			return "", fmt.Errorf("could not parse the Referer header: %w", err)
		}
		return parsedReferer.Host, nil
	}

	return "", ErrNoneOfRequiredHeadersSet
}

func newLoggedResponseWriter(writer http.ResponseWriter, traceID string) *loggedResponseWriter {
	writer.Header().Set("Content-Type", "application/json")

	return &loggedResponseWriter{
		writer:  writer,
		traceID: traceID,
	}
}

func logResponse(logger *zap.Logger, lw *loggedResponseWriter) {
	if lw.statusCode >= 400 && lw.statusCode <= 499 {
		logger.Error("Client error",
			logging.Int("http-status", lw.statusCode),
			logging.String("response", string(lw.response)),
			logging.String("request-id", vfmt.Escape(lw.requestID)),
			logging.String("trace-id", lw.traceID),
		)
		return
	}
	if lw.statusCode >= 500 && lw.statusCode <= 599 {
		logger.Error("Internal error",
			logging.Int("http-status", lw.statusCode),
			logging.Error(lw.internalError),
			logging.String("request-id", vfmt.Escape(lw.requestID)),
			logging.String("trace-id", lw.traceID),
		)
		return
	}
	logger.Info("Successful response",
		logging.Int("http-status", lw.statusCode),
		logging.String("response", string(lw.response)),
		logging.String("request-id", vfmt.Escape(lw.requestID)),
		logging.String("trace-id", lw.traceID),
	)
}
