package service_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/libs/ptr"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	v2 "code.vegaprotocol.io/vega/wallet/service/v2"
	"code.vegaprotocol.io/vega/wallet/service/v2/connections"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/golang/mock/gomock"
	"github.com/mitchellh/mapstructure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServiceV2(t *testing.T) {
	t.Run("GET /api/v2/health", testServiceV2_GetHealth)
	t.Run("GET /api/v2/methods", testServiceV2_GetMethods)
	t.Run("POST /api/v2/requests", testServiceV2_PostRequests)
}

func testServiceV2_GetHealth(t *testing.T) {
	t.Run("Checking health succeeds", testServiceV2_GetHealth_CheckingHealthSucceeds)
}

func testServiceV2_GetHealth_CheckingHealthSucceeds(t *testing.T) {
	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, response := s.serveHTTP(t, buildRequest(t, http.MethodGet, "/api/v2/health", "", nil))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	assert.Empty(t, response)
}

func testServiceV2_GetMethods(t *testing.T) {
	t.Run("Listing methods succeeds", testServiceV2_GetMethods_ListingMethodsSucceeds)
}

func testServiceV2_GetMethods_ListingMethodsSucceeds(t *testing.T) {
	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodGet, "/api/v2/methods", "", nil))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	response := intoGetMethodsResponse(t, rawResponse)
	assert.Equal(t, []string{
		"client.check_transaction",
		"client.connect_wallet",
		"client.disconnect_wallet",
		"client.get_chain_id",
		"client.list_keys",
		"client.send_transaction",
		"client.sign_transaction",
	}, response.Result.RegisteredMethods)
}

type getMethodsResponse struct {
	Result struct {
		RegisteredMethods []string `json:"registeredMethods"`
	} `json:"result,omitempty"`
}

func intoGetMethodsResponse(t *testing.T, response []byte) *getMethodsResponse {
	t.Helper()
	resp := &getMethodsResponse{}
	if err := json.Unmarshal(response, resp); err != nil {
		t.Fatalf("couldn't unmarshal response from /api/v2/methods: %v", err)
	}
	return resp
}

func testServiceV2_PostRequests(t *testing.T) {
	t.Run("Posting a malformed request fails", testServiceV2_PostRequests_MalformedRequestFails)
	t.Run("Posting an invalid request fails", testServiceV2_PostRequests_InvalidRequestFails)
	t.Run("Posting a request calling an admin method fails", testServiceV2_PostRequests_CallingAdminMethodFails)
	t.Run("Posting a request calling an unknown method fails", testServiceV2_PostRequests_CallingUnknownMethodFails)
	t.Run("`client.get_chain_id` succeeds", testServiceV2_PostRequests_GetChainIDSucceeds)
	t.Run("`client.get_chain_id` as notification returns nothing", testServiceV2_PostRequests_GetChainIDAsNotificationReturnsNothing)
	t.Run("`client.get_chain_id` getting error fails", testServiceV2_PostRequests_GetChainIDGettingErrorFails)
	t.Run("`client.get_chain_id` getting internal error fails", testServiceV2_PostRequests_GetChainIDGettingInternalErrorFails)
	t.Run("`client.connect_wallet` succeeds", testServiceV2_PostRequests_ConnectWalletSucceeds)
	t.Run("`client.connect_wallet` without origin fails", testServiceV2_PostRequests_ConnectWalletWithoutOriginFails)
	t.Run("`client.connect_wallet` as notification returns nothing", testServiceV2_PostRequests_ConnectWalletAsNotificationReturnsNothing)
	t.Run("`client.connect_wallet` getting error fails", testServiceV2_PostRequests_ConnectWalletGettingErrorFails)
	t.Run("`client.connect_wallet` getting internal error fails", testServiceV2_PostRequests_ConnectWalletGettingInternalErrorFails)
	t.Run("`client.disconnect_wallet` succeeds", testServiceV2_PostRequests_DisconnectWalletSucceeds)
	t.Run("`client.list_keys` succeeds", testServiceV2_PostRequests_ListKeysSucceeds)
	t.Run("`client.list_keys` without origin fails", testServiceV2_PostRequests_ListKeysWithoutOriginFails)
	t.Run("`client.list_keys` without token fails", testServiceV2_PostRequests_ListKeysWithoutTokenFails)
	t.Run("`client.list_keys` with unknown token fails", testServiceV2_PostRequests_ListKeysWithUnknownTokenFails)
	t.Run("`client.list_keys` with origin not matching the original hostname fails", testServiceV2_PostRequests_ListKeysWithMismatchingHostnameFails)
	t.Run("`client.list_keys` as notification returns nothing", testServiceV2_PostRequests_ListKeysAsNotificationReturnsNothing)
	t.Run("`client.list_keys` getting error fails", testServiceV2_PostRequests_ListKeysGettingErrorFails)
	t.Run("`client.list_keys` getting internal error fails", testServiceV2_PostRequests_ListKeysGettingInternalErrorFails)
	t.Run("`client.list_keys` with expired long-living token fails", testServiceV2_PostRequests_ListKeysWithExpiredLongLivingTokenFails)
	t.Run("`client.list_keys` with long-living token succeeds", testServiceV2_PostRequests_ListKeysWithLongLivingTokenSucceeds)
	t.Run("`client.send_transaction` succeeds", testServiceV2_PostRequests_SendTransactionSucceeds)
	t.Run("`client.send_transaction` without origin fails", testServiceV2_PostRequests_SendTransactionWithoutOriginFails)
	t.Run("`client.send_transaction` without token fails", testServiceV2_PostRequests_SendTransactionWithoutTokenFails)
	t.Run("`client.send_transaction` with unknown token fails", testServiceV2_PostRequests_SendTransactionWithUnknownTokenFails)
	t.Run("`client.send_transaction` with origin not matching the original hostname fails", testServiceV2_PostRequests_SendTransactionWithMismatchingHostnameFails)
	t.Run("`client.send_transaction` as notification returns nothing", testServiceV2_PostRequests_SendTransactionAsNotificationReturnsNothing)
	t.Run("`client.send_transaction` getting error fails", testServiceV2_PostRequests_SendTransactionGettingErrorFails)
	t.Run("`client.send_transaction` getting internal error fails", testServiceV2_PostRequests_SendTransactionGettingInternalErrorFails)
	t.Run("`client.send_transaction` with expired long-living token fails", testServiceV2_PostRequests_SendTransactionWithExpiredLongLivingTokenFails)
	t.Run("`client.send_transaction` with long-living token succeeds", testServiceV2_PostRequests_SendTransactionWithLongLivingTokenSucceeds)
}

func testServiceV2_PostRequests_MalformedRequestFails(t *testing.T) {
	// given
	reqBody := `"not-a-valid-json-rpc-request"`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	// Since the request can't even be unmarshall, we expect no ID
	rpcErr := intoJSONRPCError(t, rawResponse, "")
	assert.Equal(t, "Parse error", rpcErr.Message)
	assert.Equal(t, jsonrpc.ErrorCodeParseError, rpcErr.Code)
	assert.NotEmpty(t, rpcErr.Data)
}

func testServiceV2_PostRequests_InvalidRequestFails(t *testing.T) {
	// setup
	s := getTestServiceV2(t)

	tcs := []struct {
		name    string
		reqBody string
		error   error
	}{
		{
			name:    "with invalid version",
			reqBody: `{"jsonrpc": "1000", "method": "hack_the_world", "id": "123456789"}`,
			error:   jsonrpc.ErrOnlySupportJSONRPC2,
		}, {
			name:    "without method specified",
			reqBody: `{"jsonrpc": "2.0", "method": "", "id": "123456789"}`,
			error:   jsonrpc.ErrMethodIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			statusCode, _, rawResponse := s.serveHTTP(tt, buildRequest(t, http.MethodPost, "/api/v2/requests", tc.reqBody, nil))

			// then
			require.Equal(tt, http.StatusBadRequest, statusCode)
			rpcErr := intoJSONRPCError(tt, rawResponse, "123456789")
			assert.Equal(tt, "Invalid Request", rpcErr.Message)
			assert.Equal(tt, jsonrpc.ErrorCodeInvalidRequest, rpcErr.Code)
			assert.Equal(tt, tc.error.Error(), rpcErr.Data)
		})
	}
}

func testServiceV2_PostRequests_CallingAdminMethodFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "admin.create_wallet", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Method not found", rpcErr.Message)
	assert.Equal(t, jsonrpc.ErrorCodeMethodNotFound, rpcErr.Code)
	assert.Equal(t, v2.ErrAdminEndpointsNotExposed.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_CallingUnknownMethodFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.create_wallet", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Method not found", rpcErr.Message)
	assert.Equal(t, jsonrpc.ErrorCodeMethodNotFound, rpcErr.Code)
	assert.Equal(t, "method \"client.create_wallet\" is not supported", rpcErr.Data)
}

func testServiceV2_PostRequests_GetChainIDSucceeds(t *testing.T) {
	// given
	expectedChainID := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.get_chain_id", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().GetChainID(gomock.Any()).Times(1).Return(&api.ClientGetChainIDResult{
		ChainID: expectedChainID,
	}, nil)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	result := intoClientGetChainIDResult(t, rawResponse, "123456789")
	assert.Equal(t, expectedChainID, result.ChainID)
}

func testServiceV2_PostRequests_GetChainIDAsNotificationReturnsNothing(t *testing.T) {
	// given
	expectedChainID := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.get_chain_id"}`

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().GetChainID(gomock.Any()).Times(1).Return(&api.ClientGetChainIDResult{
		ChainID: expectedChainID,
	}, nil)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusNoContent, statusCode)
	result := intoJSONRPCResult(t, rawResponse, "")
	assert.Empty(t, result)
}

func testServiceV2_PostRequests_GetChainIDGettingErrorFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.get_chain_id", "id": "123456789"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: vgrand.RandomStr(10),
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().GetChainID(gomock.Any()).Times(1).Return(nil, expectedErrorDetails)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_GetChainIDGettingInternalErrorFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.get_chain_id", "id": "123456789"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: "Internal error",
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().GetChainID(gomock.Any()).Times(1).Return(nil, expectedErrorDetails)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusInternalServerError, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_ConnectWalletSucceeds(t *testing.T) {
	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, responseHeaders, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	vwt := responseHeaders.Get("Authorization")
	assert.NotEmpty(t, vwt)
	assert.True(t, strings.HasPrefix(vwt, "VWT "))
	assert.True(t, len(vwt) > len("VWT "))
	result := intoJSONRPCResult(t, rawResponse, "123456789")
	assert.Empty(t, result)
}

func testServiceV2_PostRequests_ConnectWalletWithoutOriginFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, responseHeaders, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	vwt := responseHeaders.Get("Authorization")
	assert.Empty(t, vwt)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeHostnameResolutionFailure, rpcErr.Code)
	assert.Equal(t, v2.ErrOriginHeaderIsRequired.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_ConnectWalletAsNotificationReturnsNothing(t *testing.T) {
	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.connect_wallet"}`
	w := newWallet(t)

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusNoContent, statusCode)
	result := intoJSONRPCResult(t, rawResponse, "")
	assert.Empty(t, result)
}

func testServiceV2_PostRequests_ConnectWalletGettingErrorFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	expectedHostname := vgrand.RandomStr(5)
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: vgrand.RandomStr(10),
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(nil, expectedErrorDetails)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_ConnectWalletGettingInternalErrorFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	expectedHostname := vgrand.RandomStr(5)
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: "Internal error",
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s := getTestServiceV2(t)
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(nil, expectedErrorDetails)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusInternalServerError, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_DisconnectWalletSucceeds(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	connectionStatusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, connectionStatusCode)

	// given
	reqBodyDisconnectWallet := `{"jsonrpc": "2.0", "method": "client.disconnect_wallet", "id": "123456789"}`

	// when
	disconnectionStatusCode, _, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyDisconnectWallet, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusOK, disconnectionStatusCode)

	// given
	reqBodyListKeys := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`

	// when
	listStatusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyListKeys, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, listStatusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, connections.ErrNoConnectionAssociatedThisAuthenticationToken.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_ListKeysSucceeds(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodyListKeys := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`
	expectedKeys := []api.ClientNamedPublicKey{
		{
			Name:      vgrand.RandomStr(5),
			PublicKey: vgrand.RandomStr(64),
		}, {
			Name:      vgrand.RandomStr(5),
			PublicKey: vgrand.RandomStr(64),
		},
	}

	// setup
	s.clientAPI.EXPECT().ListKeys(gomock.Any(), gomock.Any()).Times(1).Return(&api.ClientListKeysResult{
		Keys: expectedKeys,
	}, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyListKeys, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	result := intoClientListKeysResult(t, rawResponse, "123456789")
	assert.Equal(t, expectedKeys, result.Keys)
}

func testServiceV2_PostRequests_ListKeysWithoutOriginFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeHostnameResolutionFailure, rpcErr.Code)
	assert.Equal(t, v2.ErrOriginHeaderIsRequired.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_ListKeysWithoutTokenFails(t *testing.T) {
	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, v2.ErrAuthorizationHeaderIsRequired.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_ListKeysWithUnknownTokenFails(t *testing.T) {
	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": "VWT " + vgrand.RandomStr(64),
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, connections.ErrNoConnectionAssociatedThisAuthenticationToken.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_ListKeysWithMismatchingHostnameFails(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin":        vgrand.RandomStr(5),
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, connections.ErrHostnamesMismatchForThisToken.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_ListKeysAsNotificationReturnsNothing(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodyListKeys := `{"jsonrpc": "2.0", "method": "client.list_keys"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: vgrand.RandomStr(10),
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s.clientAPI.EXPECT().ListKeys(gomock.Any(), gomock.Any()).Times(1).Return(nil, expectedErrorDetails)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyListKeys, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusNoContent, statusCode)
	result := intoJSONRPCResult(t, rawResponse, "")
	assert.Empty(t, result)
}

func testServiceV2_PostRequests_ListKeysGettingErrorFails(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodyListKeys := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: vgrand.RandomStr(10),
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s.clientAPI.EXPECT().ListKeys(gomock.Any(), gomock.Any()).Times(1).Return(nil, expectedErrorDetails)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyListKeys, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_ListKeysGettingInternalErrorFails(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodyListKeys := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: "Internal error",
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s.clientAPI.EXPECT().ListKeys(gomock.Any(), gomock.Any()).Times(1).Return(nil, expectedErrorDetails)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyListKeys, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusInternalServerError, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_ListKeysWithExpiredLongLivingTokenFails(t *testing.T) {
	// given
	token := connections.GenerateToken()
	w := newWallet(t)
	expectedPassphrase := vgrand.RandomStr(5)
	s := getTestServiceV2(t, longLivingTokenSetupForTest{
		tokenDescription: connections.TokenDescription{
			CreationDate:   time.Now().Add(-2 * time.Hour),
			ExpirationDate: ptr.From(time.Now().Add(-1 * time.Hour)),
			Token:          token,
			Wallet: connections.WalletCredentials{
				Name:       w.Name(),
				Passphrase: expectedPassphrase,
			},
		},
		wallet: w,
	})
	expectedHostname := vgrand.RandomStr(5)

	// given
	reqBodyListKeys := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`

	// setup
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyListKeys, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": "VWT " + token.String(),
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, connections.ErrTokenHasExpired.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_ListKeysWithLongLivingTokenSucceeds(t *testing.T) {
	// given
	token := connections.GenerateToken()
	w := newWallet(t)
	expectedPassphrase := vgrand.RandomStr(5)
	s := getTestServiceV2(t, longLivingTokenSetupForTest{
		tokenDescription: connections.TokenDescription{
			CreationDate: time.Now().Add(-2 * time.Hour),
			Token:        token,
			Wallet: connections.WalletCredentials{
				Name:       w.Name(),
				Passphrase: expectedPassphrase,
			},
		},
		wallet: w,
	})
	expectedHostname := vgrand.RandomStr(5)
	reqBodyListKeys := `{"jsonrpc": "2.0", "method": "client.list_keys", "id": "123456789"}`
	expectedKeys := []api.ClientNamedPublicKey{
		{
			Name:      vgrand.RandomStr(5),
			PublicKey: vgrand.RandomStr(64),
		}, {
			Name:      vgrand.RandomStr(5),
			PublicKey: vgrand.RandomStr(64),
		},
	}

	// setup
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())
	s.clientAPI.EXPECT().ListKeys(gomock.Any(), gomock.Any()).Times(1).Return(&api.ClientListKeysResult{
		Keys: expectedKeys,
	}, nil)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyListKeys, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": "VWT " + token.String(),
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	result := intoClientListKeysResult(t, rawResponse, "123456789")
	assert.Equal(t, expectedKeys, result.Keys)
}

func testServiceV2_PostRequests_SendTransactionSucceeds(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)
	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not generate a key for wallet in test: %v", err)
	}

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodySendTransaction := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "client.send_transaction",
		"id": "123456789",
		"params": {
			"publicKey": %q,
			"sendingMode": "TYPE_SYNC",
			"transaction": {
		  		"voteSubmission": {
					"proposalId": "eb2d3902fdda9c3eb6e369f2235689b871c7322cf3ab284dde3e9dfc13863a17",
					"value": "VALUE_YES"
		  		}
			}
		}
	}`, kp.PublicKey())
	expectedResult := &api.ClientSendTransactionResult{}

	// setup
	s.clientAPI.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(expectedResult, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodySendTransaction, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	result := intoClientSendTransactionResult(t, rawResponse, "123456789")
	assert.Equal(t, expectedResult, result)
}

func testServiceV2_PostRequests_SendTransactionWithoutOriginFails(t *testing.T) {
	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.send_transaction", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, nil))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeHostnameResolutionFailure, rpcErr.Code)
	assert.Equal(t, v2.ErrOriginHeaderIsRequired.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_SendTransactionWithoutTokenFails(t *testing.T) {
	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.send_transaction", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, v2.ErrAuthorizationHeaderIsRequired.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_SendTransactionWithUnknownTokenFails(t *testing.T) {
	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBody := `{"jsonrpc": "2.0", "method": "client.send_transaction", "id": "123456789"}`

	// setup
	s := getTestServiceV2(t)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": "VWT " + vgrand.RandomStr(64),
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, connections.ErrNoConnectionAssociatedThisAuthenticationToken.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_SendTransactionWithMismatchingHostnameFails(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBody := `{"jsonrpc": "2.0", "method": "client.send_transaction", "id": "123456789"}`

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBody, map[string]string{
		"Origin":        vgrand.RandomStr(5),
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, connections.ErrHostnamesMismatchForThisToken.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_SendTransactionAsNotificationReturnsNothing(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodySendTransaction := `{"jsonrpc": "2.0", "method": "client.send_transaction"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: vgrand.RandomStr(10),
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s.clientAPI.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, expectedErrorDetails)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodySendTransaction, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusNoContent, statusCode)
	result := intoJSONRPCResult(t, rawResponse, "")
	assert.Empty(t, result)
}

func testServiceV2_PostRequests_SendTransactionGettingErrorFails(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodySendTransaction := `{"jsonrpc": "2.0", "method": "client.send_transaction", "id": "123456789"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: vgrand.RandomStr(10),
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s.clientAPI.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, expectedErrorDetails)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodySendTransaction, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusBadRequest, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_SendTransactionGettingInternalErrorFails(t *testing.T) {
	s := getTestServiceV2(t)

	// given
	expectedHostname := vgrand.RandomStr(5)
	reqBodyConnectWallet := `{"jsonrpc": "2.0", "method": "client.connect_wallet", "id": "123456789"}`
	w := newWallet(t)

	// setup
	s.clientAPI.EXPECT().ConnectWallet(gomock.Any(), expectedHostname).Times(1).Return(w, nil)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, connectionResponseHeaders, _ := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodyConnectWallet, map[string]string{
		"Origin": expectedHostname,
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)

	// given
	reqBodySendTransaction := `{"jsonrpc": "2.0", "method": "client.send_transaction", "id": "123456789"}`
	expectedErrorDetails := &jsonrpc.ErrorDetails{
		Code:    123,
		Message: "Internal error",
		Data:    vgrand.RandomStr(10),
	}

	// setup
	s.clientAPI.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(nil, expectedErrorDetails)
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodySendTransaction, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": connectionResponseHeaders.Get("Authorization"),
	}))

	// then
	require.Equal(t, http.StatusInternalServerError, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, expectedErrorDetails, rpcErr)
}

func testServiceV2_PostRequests_SendTransactionWithExpiredLongLivingTokenFails(t *testing.T) {
	// given
	token := connections.GenerateToken()
	w := newWallet(t)
	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedPassphrase := vgrand.RandomStr(5)
	s := getTestServiceV2(t, longLivingTokenSetupForTest{
		tokenDescription: connections.TokenDescription{
			CreationDate:   time.Now().Add(-2 * time.Hour),
			ExpirationDate: ptr.From(time.Now().Add(-1 * time.Hour)),
			Token:          token,
			Wallet: connections.WalletCredentials{
				Name:       w.Name(),
				Passphrase: expectedPassphrase,
			},
		},
		wallet: w,
	})
	expectedHostname := vgrand.RandomStr(5)
	reqBodySendTransaction := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "client.send_transaction",
		"id": "123456789",
		"params": {
			"publicKey": %q,
			"sendingMode": "TYPE_SYNC",
			"transaction": {
		  		"voteSubmission": {
					"proposalId": "eb2d3902fdda9c3eb6e369f2235689b871c7322cf3ab284dde3e9dfc13863a17",
					"value": "VALUE_YES"
		  		}
			}
		}
	}`, kp.PublicKey())

	// setup
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodySendTransaction, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": "VWT " + token.String(),
	}))

	// then
	require.Equal(t, http.StatusUnauthorized, statusCode)
	rpcErr := intoJSONRPCError(t, rawResponse, "123456789")
	assert.Equal(t, api.ErrorCodeAuthenticationFailure, rpcErr.Code)
	assert.Equal(t, "Server error", rpcErr.Message)
	assert.Equal(t, connections.ErrTokenHasExpired.Error(), rpcErr.Data)
}

func testServiceV2_PostRequests_SendTransactionWithLongLivingTokenSucceeds(t *testing.T) {
	// given
	token := connections.GenerateToken()
	w := newWallet(t)
	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatal(err)
	}
	expectedPassphrase := vgrand.RandomStr(5)
	s := getTestServiceV2(t, longLivingTokenSetupForTest{
		tokenDescription: connections.TokenDescription{
			CreationDate: time.Now().Add(-2 * time.Hour),
			Token:        token,
			Wallet: connections.WalletCredentials{
				Name:       w.Name(),
				Passphrase: expectedPassphrase,
			},
		},
		wallet: w,
	})
	expectedHostname := vgrand.RandomStr(5)
	reqBodySendTransaction := fmt.Sprintf(`{
		"jsonrpc": "2.0",
		"method": "client.send_transaction",
		"id": "123456789",
		"params": {
			"publicKey": %q,
			"sendingMode": "TYPE_SYNC",
			"transaction": {
		  		"voteSubmission": {
					"proposalId": "eb2d3902fdda9c3eb6e369f2235689b871c7322cf3ab284dde3e9dfc13863a17",
					"value": "VALUE_YES"
		  		}
			}
		}
	}`, kp.PublicKey())
	expectedResult := &api.ClientSendTransactionResult{}

	// setup
	s.timeService.EXPECT().Now().Times(1).Return(time.Now())
	s.clientAPI.EXPECT().SendTransaction(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).Return(expectedResult, nil)

	// when
	statusCode, _, rawResponse := s.serveHTTP(t, buildRequest(t, http.MethodPost, "/api/v2/requests", reqBodySendTransaction, map[string]string{
		"Origin":        expectedHostname,
		"Authorization": "VWT " + token.String(),
	}))

	// then
	require.Equal(t, http.StatusOK, statusCode)
	result := intoClientSendTransactionResult(t, rawResponse, "123456789")
	assert.Equal(t, expectedResult, result)
}

func newWallet(t *testing.T) *wallet.HDWallet {
	t.Helper()

	w, _, err := wallet.NewHDWallet(vgrand.RandomStr(5))
	if err != nil {
		t.Fatalf("could not create a wallet for test: %v", err)
	}
	return w
}

func intoClientSendTransactionResult(t *testing.T, rawResponse []byte, id string) *api.ClientSendTransactionResult {
	t.Helper()

	rpcRes := intoJSONRPCResult(t, rawResponse, id)

	result := &api.ClientSendTransactionResult{}
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Metadata:   nil,
		DecodeHook: mapstructure.ComposeDecodeHookFunc(toTimeHookFunc()),
		Result:     result,
	})
	if err != nil {
		t.Fatalf("could not create de mapstructure decoder for the client.send_transaction result: %v", err)
	}

	if err := decoder.Decode(rpcRes); err != nil {
		t.Fatalf("could not decode the client.send_transaction result: %v", err)
	}
	return result
}

func toTimeHookFunc() mapstructure.DecodeHookFunc {
	return func(f reflect.Type, t reflect.Type, data interface{}) (interface{}, error) {
		if t != reflect.TypeOf(time.Time{}) {
			return data, nil
		}

		switch f.Kind() {
		case reflect.String:
			return time.Parse(time.RFC3339, data.(string))
		default:
			return data, nil
		}
	}
}

func intoClientListKeysResult(t *testing.T, rawResponse []byte, id string) *api.ClientListKeysResult {
	t.Helper()

	rpcRes := intoJSONRPCResult(t, rawResponse, id)
	result := &api.ClientListKeysResult{}
	if err := mapstructure.Decode(rpcRes, result); err != nil {
		t.Fatalf("could not parse the client.list_keys result: %v", err)
	}
	return result
}

func intoClientGetChainIDResult(t *testing.T, rawResponse []byte, id string) *api.ClientGetChainIDResult {
	t.Helper()

	rpcRes := intoJSONRPCResult(t, rawResponse, id)
	result := &api.ClientGetChainIDResult{}
	if err := mapstructure.Decode(rpcRes, result); err != nil {
		t.Fatalf("could not parse the client.get_chain_id result: %v", err)
	}
	return result
}

func intoJSONRPCError(t *testing.T, rawResponse []byte, id string) *jsonrpc.ErrorDetails {
	t.Helper()

	resp := &jsonrpc.Response{}
	if err := json.Unmarshal(rawResponse, resp); err != nil {
		t.Fatalf("couldn't unmarshal response from /api/v2/request: %v", err)
	}
	assert.Equal(t, "2.0", resp.Version)
	assert.Equal(t, id, resp.ID)
	assert.Nil(t, resp.Result)
	require.NotNil(t, id, resp.Error)

	return resp.Error
}

func intoJSONRPCResult(t *testing.T, rawResponse []byte, id string) jsonrpc.Result {
	t.Helper()

	if id == "" {
		assert.Empty(t, rawResponse)
		return nil
	}

	resp := &jsonrpc.Response{}
	if err := json.Unmarshal(rawResponse, resp); err != nil {
		t.Fatalf("couldn't unmarshal response from /api/v2/requests: %v", err)
	}
	assert.Equal(t, "2.0", resp.Version)
	assert.Equal(t, id, resp.ID)
	assert.Nil(t, resp.Error)
	require.NotNil(t, id, resp.Result)

	return resp.Result
}
