package jsonrpc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
)

type InterceptorFunc func(ctx context.Context, request Request) *ErrorDetails

// Dispatcher forward the request it gets in input to the associated method.
// Despite being useful for simple use cases, it may not fit more advanced
// use cases.
type Dispatcher struct {
	log *zap.Logger
	// commands maps a method to a command.
	commands map[string]Command
	// interceptors holds the pre-checks to run before dispatching a request.
	interceptors []InterceptorFunc
}

func (a *Dispatcher) DispatchRequest(ctx context.Context, request Request) *Response {
	traceID := TraceIDFromContext(ctx)

	if err := request.Check(); err != nil {
		a.log.Info("invalid request",
			zap.String("trace-id", traceID),
			zap.Error(err))
		return NewErrorResponse(request.ID, NewInvalidRequest(err))
	}

	for _, interceptor := range a.interceptors {
		if errDetails := interceptor(ctx, request); errDetails != nil {
			return NewErrorResponse(request.ID, errDetails)
		}
	}

	command, ok := a.commands[request.Method]
	if !ok {
		a.log.Info("invalid method",
			zap.String("trace-id", traceID),
			zap.String("method", request.Method))
		return NewErrorResponse(request.ID, NewMethodNotFound(request.Method))
	}

	result, errorDetails := command.Handle(ctx, request.Params)
	if errorDetails != nil {
		a.log.Info("method failed",
			zap.String("trace-id", traceID),
			zap.Any("error", errorDetails))
		return NewErrorResponse(request.ID, errorDetails)
	}
	a.log.Info("method succeeded",
		zap.String("trace-id", traceID))
	return NewSuccessfulResponse(request.ID, result)
}

func (a *Dispatcher) RegisterMethod(method string, handler Command) {
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

func (a *Dispatcher) RegisteredMethods() []string {
	methods := make([]string, 0, len(a.commands))
	for method := range a.commands {
		methods = append(methods, method)
	}
	sort.Strings(methods)
	return methods
}

func (a *Dispatcher) AddInterceptor(interceptor InterceptorFunc) {
	a.interceptors = append(a.interceptors, interceptor)
}

func NewDispatcher(log *zap.Logger) *Dispatcher {
	return &Dispatcher{
		log:          log,
		commands:     map[string]Command{},
		interceptors: []InterceptorFunc{},
	}
}
