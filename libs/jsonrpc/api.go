package jsonrpc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
)

const JSONRPC2 string = "2.0"

type API struct {
	log *zap.Logger
	// commands maps a method to a command.
	commands map[string]Command
}

func New(log *zap.Logger) *API {
	return &API{
		log:      log,
		commands: map[string]Command{},
	}
}

func (a *API) DispatchRequest(ctx context.Context, request *Request) *Response {
	traceID := traceIDFromContext(ctx)

	if err := request.Check(); err != nil {
		a.log.Info("invalid request",
			zap.String("trace-id", traceID),
			zap.Error(err))
		return invalidRequestResponse(request, err)
	}
	command, ok := a.commands[request.Method]
	if !ok {
		a.log.Info("invalid method",
			zap.String("trace-id", traceID),
			zap.String("method", request.Method))
		return unsupportedMethodResponse(request)
	}

	result, errorDetails := command.Handle(ctx, request.Params)
	if errorDetails != nil {
		a.log.Info("method failed",
			zap.String("trace-id", traceID),
			zap.Any("error", errorDetails))
		return NewErrorResponse(request.ID, errorDetails)
	}
	a.log.Info("method succeeded",
		zap.String("trace-id", traceID),
		zap.Any("error", errorDetails))
	return NewSuccessfulResponse(request.ID, result)
}

func (a *API) RegisterMethod(method string, handler Command) {
	if len(strings.Trim(method, " \t\r\n")) == 0 {
		a.log.Panic("method cannot be empty")
	}

	if handler == nil {
		a.log.Panic("handler cannot be nil")
	}

	if _, ok := a.commands[method]; ok {
		a.log.Panic(fmt.Sprintf("method %q is already registered", method))
	}

	a.commands[method] = handler
	a.log.Info("new JSON-RPC method registered", zap.String("method", method))
}

func (a *API) RegisteredMethods() []string {
	methods := make([]string, 0, len(a.commands))
	for method := range a.commands {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}

func invalidRequestResponse(request *Request, err error) *Response {
	return NewErrorResponse(request.ID, NewInvalidRequest(err))
}

func unsupportedMethodResponse(request *Request) *Response {
	return NewErrorResponse(request.ID, NewMethodNotFound(fmt.Errorf("method %q is not supported", request.Method)))
}

func traceIDFromContext(ctx context.Context) string {
	traceID := ctx.Value("trace-id")
	if traceID == nil {
		return ""
	}
	return traceID.(string)
}
