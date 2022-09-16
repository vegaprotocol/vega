package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
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
	assert.Equal(t, "Application error", errorDetails.Message)
	assert.Equal(t, expectedErr.Error(), errorDetails.Data)
}

func assertRequestInterruptionError(t *testing.T, errorDetails *jsonrpc.ErrorDetails) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, jsonrpc.ErrorCodeRequestHasBeenInterrupted, errorDetails.Code)
	assert.Equal(t, "Server error", errorDetails.Message)
	assert.Equal(t, api.ErrRequestInterrupted.Error(), errorDetails.Data)
}

func assertConnectionClosedError(t *testing.T, errorDetails *jsonrpc.ErrorDetails) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeConnectionHasBeenClosed, errorDetails.Code)
	assert.Equal(t, api.ErrUserCloseTheConnection.Error(), errorDetails.Data)
	assert.Equal(t, "User error", errorDetails.Message)
}

func assertInternalError(t *testing.T, errorDetails *jsonrpc.ErrorDetails, expectedErr error) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, jsonrpc.ErrorCodeInternalError, errorDetails.Code)
	assert.Equal(t, "Internal error", errorDetails.Message)
	assert.Equal(t, expectedErr.Error(), errorDetails.Data)
}

func assertUserRejectionError(t *testing.T, errorDetails *jsonrpc.ErrorDetails) {
	t.Helper()
	require.NotNil(t, errorDetails)
	assert.Equal(t, api.ErrorCodeRequestHasBeenRejected, errorDetails.Code)
	assert.Equal(t, api.ErrUserRejectedTheRequest.Error(), errorDetails.Data)
	assert.Equal(t, "User error", errorDetails.Message)
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

func generateKey(t *testing.T, w wallet.Wallet) wallet.KeyPair {
	t.Helper()

	kp, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatalf("could not generate key for test wallet: %v", err)
	}
	return kp
}

func contextWithTraceID() (context.Context, string) {
	traceID := vgrand.RandomStr(5)
	//revive:disable:context-keys-type
	//nolint:staticcheck
	return context.WithValue(context.Background(), "trace-id", traceID), traceID
}

func connectWallet(t *testing.T, sessions *api.Sessions, hostname string, w wallet.Wallet) string {
	t.Helper()
	token, err := sessions.ConnectWallet(hostname, w)
	if err != nil {
		t.Fatalf("could not connect to a wallet for test: %v", err)
	}
	return token
}
