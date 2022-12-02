package api_test

import (
	"encoding/json"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/node"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/network"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func assertInvalidParams(t *testing.T, errorDetails *jsonrpc.ErrorDetails, expectedErr error) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, jsonrpc.ErrorCodeInvalidParams, errorDetails.Code)
	assert.Equal(t, "Invalid params", errorDetails.Message)
	assert.Equal(t, expectedErr.Error(), errorDetails.Data)
}

func assertRequestNotPermittedError(t *testing.T, errorDetails *jsonrpc.ErrorDetails, expectedErr error) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeRequestNotPermitted, errorDetails.Code)
	assert.Equal(t, string(api.ApplicationError), errorDetails.Message)
	assert.Equal(t, expectedErr.Error(), errorDetails.Data)
}

func assertRequestInterruptionError(t *testing.T, errorDetails *jsonrpc.ErrorDetails) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, jsonrpc.ErrorCodeRequestHasBeenInterrupted, errorDetails.Code)
	assert.Equal(t, string(api.ServerError), errorDetails.Message)
	assert.Equal(t, api.ErrRequestInterrupted.Error(), errorDetails.Data)
}

func assertConnectionClosedError(t *testing.T, errorDetails *jsonrpc.ErrorDetails) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeConnectionHasBeenClosed, errorDetails.Code)
	assert.Equal(t, string(api.UserError), errorDetails.Message)
	assert.Equal(t, api.ErrUserCloseTheConnection.Error(), errorDetails.Data)
}

func assertInternalError(t *testing.T, errorDetails *jsonrpc.ErrorDetails, expectedErr error) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, jsonrpc.ErrorCodeInternalError, errorDetails.Code)
	assert.Equal(t, string(api.InternalError), errorDetails.Message)
	assert.Equal(t, expectedErr.Error(), errorDetails.Data)
}

func assertNetworkError(t *testing.T, errorDetails *jsonrpc.ErrorDetails, expectedErr error) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeNodeCommunicationFailed, errorDetails.Code)
	assert.Equal(t, string(api.NetworkError), errorDetails.Message)
	assert.Equal(t, expectedErr.Error(), errorDetails.Data)
}

func assertUserRejectionError(t *testing.T, errorDetails *jsonrpc.ErrorDetails) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeRequestHasBeenRejected, errorDetails.Code)
	assert.Equal(t, string(api.UserError), errorDetails.Message)
	assert.Equal(t, api.ErrUserRejectedTheRequest.Error(), errorDetails.Data)
}

func assertApplicationCancellationError(t *testing.T, errorDetails *jsonrpc.ErrorDetails) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeRequestHasBeenCanceledByApplication, errorDetails.Code)
	assert.Equal(t, string(api.ApplicationError), errorDetails.Message)
	assert.Equal(t, api.ErrApplicationCanceledTheRequest.Error(), errorDetails.Data)
}

func walletWithPerms(t *testing.T, hostname string, perms wallet.Permissions) (wallet.Wallet, wallet.KeyPair) {
	t.Helper()

	walletName := vgrand.RandomStr(5)

	w, _, err := wallet.NewHDWallet(walletName)
	if err != nil {
		t.Fatalf("could not create wallet for test: %v", err)
	}

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not generate a key on the wallet for test: %v", err)
	}

	if err := w.UpdatePermissions(hostname, perms); err != nil {
		t.Fatalf("could not update permissions on wallet for test: %v", err)
	}

	return w, kp
}

func walletWithKey(t *testing.T) (wallet.Wallet, wallet.KeyPair) {
	t.Helper()

	walletName := vgrand.RandomStr(5)

	w, _, err := wallet.NewHDWallet(walletName)
	if err != nil {
		t.Fatalf("could not create wallet for test: %v", err)
	}

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not update permissions on wallet for test: %v", err)
	}

	return w, kp
}

func newNetwork(t *testing.T) network.Network {
	t.Helper()

	return network.Network{
		Name: vgrand.RandomStr(5),
		API: network.APIConfig{
			GRPC: network.GRPCConfig{
				Hosts: []string{
					"n01.localtest.vega.xyz:3007",
				},
				Retries: 5,
			},
		},
	}
}

func generateKey(t *testing.T, w wallet.Wallet) wallet.KeyPair {
	t.Helper()

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not generate key for test wallet: %v", err)
	}
	return kp
}

func requestMetadataForTest() jsonrpc.RequestMetadata {
	return jsonrpc.RequestMetadata{
		TraceID:  vgrand.RandomStr(5),
		Hostname: vgrand.RandomStr(5) + ".xyz",
	}
}

func connectWallet(t *testing.T, sessions *session.Sessions, hostname string, w wallet.Wallet) string {
	t.Helper()
	token, err := sessions.ConnectWallet(hostname, w)
	if err != nil {
		t.Fatalf("could not connect to a wallet for test: %v", err)
	}
	return token
}

func unexpectedNodeSelectorCall(t *testing.T) api.NodeSelectorBuilder {
	t.Helper()

	return func(hosts []string, retries uint64) (node.Selector, error) {
		t.Fatalf("node selector shouldn't be called")
		return nil, nil
	}
}

func dummyServiceShutdownSwitch() *api.ServiceShutdownSwitch {
	return api.NewServiceShutdownSwitch(func(err error) {})
}

var (
	testTransactionJSON          = `{"voteSubmission":{"proposalId":"eb2d3902fdda9c3eb6e369f2235689b871c7322cf3ab284dde3e9dfc13863a17","value":"VALUE_YES"}}`
	testMalformedTransactionJSON = `{"voteSubmission":{"proposalId":"not real id","value":"VALUE_YES"}}`
)

func transactionFromJSON(t *testing.T, JSON string) map[string]any {
	t.Helper()
	testTransaction := make(map[string]any)
	assert.NoError(t, json.Unmarshal([]byte(JSON), &testTransaction))
	return testTransaction
}

func testTransaction(t *testing.T) map[string]any {
	t.Helper()
	return transactionFromJSON(t, testTransactionJSON)
}

func testMalformedTransaction(t *testing.T) map[string]any {
	t.Helper()
	return transactionFromJSON(t, testMalformedTransactionJSON)
}

var testEncodedTransaction = "ewogICAgInZvdGVTdWJtaXNzaW9uIjogewogICAgICAgICJwcm9wb3NhbElkIjogImViMmQzOTAyZmRkYTljM2ViNmUzNjlmMjIzNTY4OWI4NzFjNzMyMmNmM2FiMjg0ZGRlM2U5ZGZjMTM4NjNhMTciLAogICAgICAgICJ2YWx1ZSI6ICJWQUxVRV9ZRVMiCiAgICB9Cn0K"
