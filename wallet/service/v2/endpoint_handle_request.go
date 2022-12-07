package v2

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"

	vfmt "code.vegaprotocol.io/vega/libs/fmt"
	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/wallet/api"
	"github.com/julienschmidt/httprouter"
	"go.uber.org/zap"
)

func (a *API) HandleRequest(w http.ResponseWriter, httpRequest *http.Request, _ httprouter.Params) {
	traceID := vgrand.RandomStr(64)
	ctx := context.WithValue(httpRequest.Context(), jsonrpc.TraceIDKey, traceID)

	a.log.Info("New request",
		logging.String("url", vfmt.Escape(httpRequest.URL.String())),
		logging.String("trace-id", traceID),
	)

	lw := newResponseWriter(w, traceID)
	defer logResponse(a.log, lw)

	rpcRequest, errDetails := a.unmarshallRequest(traceID, httpRequest)
	if errDetails != nil {
		lw.SetStatusCode(http.StatusBadRequest)
		// Failing to unmarshall the request prevent us from retrieving the
		// request ID. So, it's left empty.
		lw.WriteJSONRPCResponse(jsonrpc.NewErrorResponse("", errDetails))
		return
	}

	response := a.processJSONRPCRequest(ctx, traceID, lw, httpRequest, rpcRequest)

	// If the request doesn't have an ID, it's a notification. Notifications do
	// not send content back, even if an error occurred.
	if rpcRequest.IsNotification() {
		lw.SetStatusCode(http.StatusNoContent)
		return
	}

	if response.Error == nil {
		lw.SetStatusCode(http.StatusOK)
	} else {
		if response.Error.Code == api.ErrorCodeAuthenticationFailure {
			lw.SetStatusCode(401)
		} else if response.Error.IsInternalError() {
			lw.SetStatusCode(http.StatusInternalServerError)
		} else {
			lw.SetStatusCode(http.StatusBadRequest)
		}
	}
	lw.WriteJSONRPCResponse(response)
}

func (a *API) unmarshallRequest(traceID string, r *http.Request) (jsonrpc.Request, *jsonrpc.ErrorDetails) {
	defer func() {
		_ = r.Body.Close()
	}()

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return jsonrpc.Request{}, jsonrpc.NewParseError(ErrCouldNotReadRequestBody)
	}

	if len(body) == 0 {
		return jsonrpc.Request{}, jsonrpc.NewParseError(ErrRequestCannotBeBlank)
	}

	request := jsonrpc.Request{}
	if err := json.Unmarshal(body, &request); err != nil {
		a.log.Error("Request could not be parsed",
			logging.String("trace-id", traceID),
			logging.Error(err),
		)

		var syntaxError *json.SyntaxError
		var unmarshallTypeError *json.UnmarshalTypeError
		if errors.As(err, &syntaxError) || errors.As(err, &unmarshallTypeError) || errors.As(err, &unmarshallTypeError) {
			return jsonrpc.Request{}, jsonrpc.NewParseError(err)
		}

		return jsonrpc.Request{}, jsonrpc.NewInvalidRequest(err)
	}

	strReq, _ := json.Marshal(&request)
	a.log.Info("Request successfully parsed",
		logging.String("request", vfmt.Escape(string(strReq))),
		logging.String("trace-id", traceID),
	)

	return request, nil
}

func (a *API) processJSONRPCRequest(ctx context.Context, traceID string, lw *responseWriter, httpRequest *http.Request, rpcRequest jsonrpc.Request) *jsonrpc.Response {
	if err := rpcRequest.Check(); err != nil {
		a.log.Info("invalid RPC request",
			zap.String("trace-id", traceID),
			zap.Error(err))
		return jsonrpc.NewErrorResponse(rpcRequest.ID, jsonrpc.NewInvalidRequest(err))
	}

	// We add this pre-check so users stop asking why they can't access the
	// administrative endpoints.
	if strings.HasPrefix(rpcRequest.Method, "admin.") {
		a.log.Debug("attempt to call administrative endpoint rejected",
			zap.String("trace-id", traceID),
			zap.String("method", vfmt.Escape(rpcRequest.Method)))
		return jsonrpc.NewErrorResponse(rpcRequest.ID, jsonrpc.NewUnsupportedMethod(ErrAdminEndpointsNotExposed))
	}

	command, ok := a.commands[rpcRequest.Method]
	if !ok {
		a.log.Debug("unknown RPC method",
			zap.String("trace-id", traceID),
			zap.String("method", vfmt.Escape(rpcRequest.Method)))
		return jsonrpc.NewErrorResponse(rpcRequest.ID, jsonrpc.NewMethodNotFound(rpcRequest.Method))
	}

	result, errDetails := command(ctx, lw, httpRequest, rpcRequest)
	if errDetails != nil {
		a.log.Info("RPC request failed",
			zap.String("trace-id", traceID),
			zap.Error(errDetails))

		return jsonrpc.NewErrorResponse(rpcRequest.ID, errDetails)
	}

	a.log.Info("RPC request succeeded",
		zap.String("trace-id", traceID))

	return jsonrpc.NewSuccessfulResponse(rpcRequest.ID, result)
}

func logResponse(logger *zap.Logger, lw *responseWriter) {
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
