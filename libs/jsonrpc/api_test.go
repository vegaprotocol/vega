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

func TestAPI(t *testing.T) {
	t.Run("API only supports JSON-RPC 2.0 requests", testAPIOnlySupportsJSONRPC2Request)
	t.Run("Method is required", testMethodIsRequired)
	t.Run("Dispatching a request succeeds", testDispatchingRequestSucceeds)
	t.Run("Dispatching a request with unsatisfied policy fails", testDispatchingRequestWithUnsatisfiedPolicyFails)
	t.Run("Dispatching a request with satisfied policy fails", testDispatchingRequestWithSatisfiedPolicyFails)
	t.Run("Dispatching an unknown request fails", testDispatchingUnknownRequestFails)
	t.Run("Failed commands return an error response", testFailedCommandReturnErrorResponse)
	t.Run("Listing registered methods succeeds", testListingRegisteredMethodsSucceeds)
}

func testAPIOnlySupportsJSONRPC2Request(t *testing.T) {
	// given
	jsonrpcAPI := newAPI(t)
	ctx := context.Background()
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

	// setup
	jsonrpcAPI.command1.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	jsonrpcAPI.command2.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	response := jsonrpcAPI.DispatchRequest(ctx, request, jsonrpc.RequestMetadata{})

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.JSONRPC2, response.Version)
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
	jsonrpcAPI := newAPI(t)
	ctx := context.Background()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.JSONRPC2,
		Method:  "",
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}

	// setup
	jsonrpcAPI.command1.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	jsonrpcAPI.command2.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	response := jsonrpcAPI.DispatchRequest(ctx, request, jsonrpc.RequestMetadata{})

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.JSONRPC2, response.Version)
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
	jsonrpcAPI := newAPI(t)
	ctx := context.Background()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.JSONRPC2,
		Method:  jsonrpcAPI.method1,
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}
	expectedResult := vgrand.RandomStr(5)

	// setup
	jsonrpcAPI.command1.EXPECT().Handle(ctx, request.Params, gomock.Any()).Times(1).Return(expectedResult, nil)
	jsonrpcAPI.command2.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	response := jsonrpcAPI.DispatchRequest(ctx, request, jsonrpc.RequestMetadata{})

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.JSONRPC2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Equal(t, expectedResult, response.Result)
	assert.Nil(t, response.Error)
}

func testDispatchingRequestWithUnsatisfiedPolicyFails(t *testing.T) {
	// given
	jsonrpcAPI := newAPI(t)
	ctx := context.Background()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.JSONRPC2,
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
	jsonrpcAPI.AddDispatchPolicy(func(_ context.Context, _ jsonrpc.Request, _ jsonrpc.RequestMetadata) *jsonrpc.ErrorDetails {
		return expectedErrDetails
	})
	jsonrpcAPI.command1.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	jsonrpcAPI.command2.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	response := jsonrpcAPI.DispatchRequest(ctx, request, jsonrpc.RequestMetadata{})

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.JSONRPC2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Nil(t, response.Result)
	assert.Equal(t, expectedErrDetails, response.Error)
}

func testDispatchingRequestWithSatisfiedPolicyFails(t *testing.T) {
	// given
	jsonrpcAPI := newAPI(t)
	ctx := context.Background()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.JSONRPC2,
		Method:  jsonrpcAPI.method1,
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}
	expectedResult := vgrand.RandomStr(5)

	// setup
	jsonrpcAPI.AddDispatchPolicy(func(_ context.Context, _ jsonrpc.Request, _ jsonrpc.RequestMetadata) *jsonrpc.ErrorDetails {
		return nil
	})
	jsonrpcAPI.command1.EXPECT().Handle(ctx, request.Params, gomock.Any()).Times(1).Return(expectedResult, nil)
	jsonrpcAPI.command2.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	response := jsonrpcAPI.DispatchRequest(ctx, request, jsonrpc.RequestMetadata{})

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.JSONRPC2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Equal(t, expectedResult, response.Result)
	assert.Nil(t, response.Error)
}

func testDispatchingUnknownRequestFails(t *testing.T) {
	// given
	jsonrpcAPI := newAPI(t)
	ctx := context.Background()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.JSONRPC2,
		Method:  vgrand.RandomStr(5), // Unregistered method.
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}

	// setup
	jsonrpcAPI.command1.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	jsonrpcAPI.command2.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	response := jsonrpcAPI.DispatchRequest(ctx, request, jsonrpc.RequestMetadata{})

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.JSONRPC2, response.Version)
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
	jsonrpcAPI := newAPI(t)
	ctx := context.Background()
	params := struct {
		Name string `json:"name"`
	}{
		Name: vgrand.RandomStr(5),
	}
	request := jsonrpc.Request{
		Version: jsonrpc.JSONRPC2,
		Method:  jsonrpcAPI.method1, // Unregistered method.
		Params:  params,
		ID:      vgrand.RandomStr(5),
	}
	expectedError := &jsonrpc.ErrorDetails{
		Code:    23456,
		Message: vgrand.RandomStr(5),
		Data:    vgrand.RandomStr(5),
	}

	// setup
	jsonrpcAPI.command1.EXPECT().Handle(ctx, request.Params, gomock.Any()).Times(1).Return(nil, expectedError)
	jsonrpcAPI.command2.EXPECT().Handle(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	// when
	response := jsonrpcAPI.DispatchRequest(ctx, request, jsonrpc.RequestMetadata{})

	// then
	require.NotNil(t, response)
	assert.Equal(t, jsonrpc.JSONRPC2, response.Version)
	assert.Equal(t, request.ID, response.ID)
	assert.Nil(t, response.Result)
	assert.Equal(t, expectedError, response.Error)
}

func testListingRegisteredMethodsSucceeds(t *testing.T) {
	// given
	jsonrpcAPI := newAPI(t)

	// when
	methods := jsonrpcAPI.RegisteredMethods()

	// then
	require.NotNil(t, methods)
	expectedMethods := []string{jsonrpcAPI.method1, jsonrpcAPI.method2}
	sort.Strings(expectedMethods)
	assert.Equal(t, expectedMethods, methods)
}

type testAPI struct {
	*jsonrpc.API
	method1  string
	command1 *mocks.MockCommand
	method2  string
	command2 *mocks.MockCommand
}

func newAPI(t *testing.T) *testAPI {
	t.Helper()
	log := newTestLogger(t)
	ctrl := gomock.NewController(t)
	method1 := vgrand.RandomStr(5)
	command1 := mocks.NewMockCommand(ctrl)
	method2 := vgrand.RandomStr(5)
	command2 := mocks.NewMockCommand(ctrl)

	// setup
	jsonrpcAPI := jsonrpc.New(log)
	jsonrpcAPI.RegisterMethod(method1, command1)
	jsonrpcAPI.RegisterMethod(method2, command2)

	return &testAPI{
		API:      jsonrpcAPI,
		method1:  method1,
		command1: command1,
		method2:  method2,
		command2: command2,
	}
}

func newTestLogger(t *testing.T) *zap.Logger {
	t.Helper()
	// Change the level to debug for debugging.
	// Keep it to Panic otherwise to not pollute tests output.
	return zaptest.NewLogger(t, zaptest.Level(zap.PanicLevel))
}
