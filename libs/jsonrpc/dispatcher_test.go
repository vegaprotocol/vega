package jsonrpc_test

import (
	"context"
	"fmt"
	"sort"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/libs/jsonrpc/mocks"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

func TestDispatcher(t *testing.T) {
	t.Run("API only supports JSON-RPC 2.0 requests", testAPIOnlySupportsJSONRPC2Request)
	t.Run("Method is required", testMethodIsRequired)
	t.Run("Dispatching a request succeeds", testDispatchingRequestSucceeds)
	t.Run("Dispatching a request with unsatisfied interceptor fails", testDispatchingRequestWithUnsatisfiedInterceptorFails)
	t.Run("Dispatching a request with satisfied interceptor succeeds", testDispatchingRequestWithSatisfiedInterceptorSucceeds)
	t.Run("Dispatching an unknown request fails", testDispatchingUnknownRequestFails)
	t.Run("Failed commands return an error response", testFailedCommandReturnErrorResponse)
	t.Run("Listing registered methods succeeds", testListingRegisteredMethodsSucceeds)
}

func testAPIOnlySupportsJSONRPC2Request(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)
	ctx := testContextWithTraceID()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: "1.0",
		Method:  vgrand.RandomStr(5), // Unregistered method.
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}

	// when
	response := dispatcher.DispatchRequest(ctx, request)

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.VERSION2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Nil(t, response.Result)
	assert.Equal(t, &jsonrpc.ErrorDetails{
		Code:    jsonrpc.ErrorCodeInvalidRequest,
		Message: "Invalid Request",
		Data:    jsonrpc.ErrOnlySupportJSONRPC2.Error(),
	}, response.Error)
}

func testMethodIsRequired(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)
	ctx := testContextWithTraceID()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.VERSION2,
		Method:  "",
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}

	// when
	response := dispatcher.DispatchRequest(ctx, request)

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.VERSION2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Nil(t, response.Result)
	assert.Equal(t, &jsonrpc.ErrorDetails{
		Code:    jsonrpc.ErrorCodeInvalidRequest,
		Message: "Invalid Request",
		Data:    jsonrpc.ErrMethodIsRequired.Error(),
	}, response.Error)
}

func testDispatchingRequestSucceeds(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)
	ctx := testContextWithTraceID()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.VERSION2,
		Method:  dispatcher.method1,
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}
	expectedResult := vgrand.RandomStr(5)

	// setup
	dispatcher.command1.EXPECT().Handle(ctx, request.Params).Times(1).Return(expectedResult, nil)

	// when
	response := dispatcher.DispatchRequest(ctx, request)

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.VERSION2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Equal(t, expectedResult, response.Result)
	assert.Nil(t, response.Error)
}

func testDispatchingRequestWithUnsatisfiedInterceptorFails(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)
	ctx := testContextWithTraceID()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.VERSION2,
		Method:  vgrand.RandomStr(5), // Unregistered method.
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}
	expectedErrDetails := &jsonrpc.ErrorDetails{
		Code:    jsonrpc.ErrorCode(1234),
		Message: vgrand.RandomStr(10),
		Data:    vgrand.RandomStr(10),
	}

	// setup
	dispatcher.AddInterceptor(func(_ context.Context, _ jsonrpc.Request) *jsonrpc.ErrorDetails {
		return expectedErrDetails
	})

	// when
	response := dispatcher.DispatchRequest(ctx, request)

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.VERSION2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Nil(t, response.Result)
	assert.Equal(t, expectedErrDetails, response.Error)
}

func testDispatchingRequestWithSatisfiedInterceptorSucceeds(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)
	ctx := testContextWithTraceID()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.VERSION2,
		Method:  dispatcher.method1,
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}
	expectedResult := vgrand.RandomStr(5)

	// setup
	dispatcher.AddInterceptor(func(_ context.Context, _ jsonrpc.Request) *jsonrpc.ErrorDetails {
		return nil
	})
	dispatcher.command1.EXPECT().Handle(ctx, request.Params).Times(1).Return(expectedResult, nil)

	// when
	response := dispatcher.DispatchRequest(ctx, request)

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.VERSION2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Equal(t, expectedResult, response.Result)
	assert.Nil(t, response.Error)
}

func testDispatchingUnknownRequestFails(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)
	ctx := testContextWithTraceID()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.VERSION2,
		Method:  vgrand.RandomStr(5), // Unregistered method.
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}

	// when
	response := dispatcher.DispatchRequest(ctx, request)

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.VERSION2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Nil(t, response.Result)
	assert.Equal(t, &jsonrpc.ErrorDetails{
		Code:    jsonrpc.ErrorCodeMethodNotFound,
		Message: "Method not found",
		Data:    fmt.Sprintf("method %q is not supported", request.Method),
	}, response.Error)
}

func testFailedCommandReturnErrorResponse(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)
	ctx := testContextWithTraceID()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.VERSION2,
		Method:  dispatcher.method1, // Unregistered method.
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}
	expectedError := &jsonrpc.ErrorDetails{
		Code:    23456,
		Message: vgrand.RandomStr(5),
		Data:    vgrand.RandomStr(5),
	}

	// setup
	dispatcher.command1.EXPECT().Handle(ctx, request.Params).Times(1).Return(nil, expectedError)

	// when
	response := dispatcher.DispatchRequest(ctx, request)

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.VERSION2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Nil(t, response.Result)
	assert.Equal(t, expectedError, response.Error)
}

func testListingRegisteredMethodsSucceeds(t *testing.T) {
	// given
	dispatcher := newDispatcher(t)

	// when
	methods := dispatcher.RegisteredMethods()

	// then
	require.NotNil(t, methods)
	expectedMethods := []string{dispatcher.method1, dispatcher.method2}
	sort.Strings(expectedMethods)
	assert.Equal(t, expectedMethods, methods)
}

type testAPI struct {
	*jsonrpc.Dispatcher
	method1  string
	command1 *mocks.MockCommand
	method2  string
	command2 *mocks.MockCommand
}

func newDispatcher(t *testing.T) *testAPI {
	t.Helper()
	log := newTestLogger(t)
	ctrl := gomock.NewController(t)
	method1 := vgrand.RandomStr(5)
	command1 := mocks.NewMockCommand(ctrl)
	method2 := vgrand.RandomStr(5)
	command2 := mocks.NewMockCommand(ctrl)

	// setup
	dispatcher := jsonrpc.NewDispatcher(log)
	dispatcher.RegisterMethod(method1, command1)
	dispatcher.RegisterMethod(method2, command2)

	return &testAPI{
		Dispatcher: dispatcher,
		method1:    method1,
		command1:   command1,
		method2:    method2,
		command2:   command2,
	}
}

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	// Change the level to debug for debugging.
	// Keep it to Panic otherwise to not pollute tests output.
	return zaptest.NewLogger(t, zaptest.Level(zap.PanicLevel))
}

func testContextWithTraceID() context.Context {
	return context.WithValue(context.Background(), jsonrpc.TraceIDKey, vgrand.RandomStr(64))
}
