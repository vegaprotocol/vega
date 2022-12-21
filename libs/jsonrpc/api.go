package jsonrpc

import (
	"context"
	"fmt"
	"sort"
	"strings"

	"go.uber.org/zap"
)

const JSONRPC2 string = "2.0"

type DispatchPolicyFunc func(ctx context.Context, request Request, metadata RequestMetadata) *ErrorDetails

type API struct {
	log *zap.Logger
	// commands maps a method to a command.
	commands map[string]Command
	// dispatchPolicies holds the pre-checks to run before dispatching a request.
	dispatchPolicies []DispatchPolicyFunc
}

func New(log *zap.Logger) *API {
	return &API{
		log:              log,
		commands:         map[string]Command{},
		dispatchPolicies: []DispatchPolicyFunc{},
	}
}

func (a *API) DispatchRequest(ctx context.Context, request Request, metadata RequestMetadata) *Response {
	if err := request.Check(); err != nil {
		a.log.Info("invalid request",
			zap.String("trace-id", metadata.TraceID),
			zap.Error(err))
		return NewErrorResponse(request.ID, NewInvalidRequest(err))
	}

	for _, dispatchPolicy := range a.dispatchPolicies {
		if errDetails := dispatchPolicy(ctx, request, metadata); errDetails != nil {
			return NewErrorResponse(request.ID, errDetails)
		}
	}

	command, ok := a.commands[request.Method]
	if !ok {
		a.log.Info("invalid method",
			zap.String("trace-id", metadata.TraceID),
			zap.String("method", request.Method))
		return NewErrorResponse(request.ID, NewMethodNotFound(fmt.Errorf("method %q is not supported", request.Method)))
	}

	result, errorDetails := command.Handle(ctx, request.Params, metadata)
	if errorDetails != nil {
		a.log.Info("method failed",
			zap.String("trace-id", metadata.TraceID),
			zap.Any("error", errorDetails))
		return NewErrorResponse(request.ID, errorDetails)
	}
	a.log.Info("method succeeded",
		zap.String("trace-id", metadata.TraceID),
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

func (a *API) AddDispatchPolicy(dispatchPolicy DispatchPolicyFunc) {
	a.dispatchPolicies = append(a.dispatchPolicies, dispatchPolicy)
}
