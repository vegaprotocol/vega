package jsonrpc

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync/atomic"

	"go.uber.org/zap"
)

const JSONRPC2 string = "2.0"

type API struct {
	log *zap.Logger
	// commands maps a method to a command.
	commands map[string]Command
	// isProcessingRequest tells if the API is processing a request or not.
	// It enforces a sequential and non-concurrent processing.
	isProcessingRequest uint32

	// processOnlyOneRequest tells if the API should allow multiple requests
	// in parallel.
	// This is a hot-fix, as this should be handled by the pipeline itself.
	processOnlyOneRequest bool
}

func New(log *zap.Logger, processOnlyOneRequest bool) *API {
	return &API{
		log:                   log,
		commands:              map[string]Command{},
		isProcessingRequest:   0,
		processOnlyOneRequest: processOnlyOneRequest,
	}
}

func (a *API) DispatchRequest(ctx context.Context, request *Request) *Response {
	traceID := traceIDFromContext(ctx)

	if a.processOnlyOneRequest {
		// We reject all incoming request as long as there is a request being
		// processed.
		if !atomic.CompareAndSwapUint32(&a.isProcessingRequest, 0, 1) {
			a.log.Info("request rejected because another one is being processed",
				zap.String("trace-id", traceID))
			return requestAlreadyBeingProcessed(request)
		}
		defer atomic.SwapUint32(&a.isProcessingRequest, 0)
	}

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

func requestAlreadyBeingProcessed(request *Request) *Response {
	return NewErrorResponse(request.ID, NewServerError(ErrorCodeRequestAlreadyBeingProcessed, ErrRequestAlreadyBeingProcessed))
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
