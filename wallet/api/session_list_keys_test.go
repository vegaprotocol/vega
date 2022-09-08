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

func TestListKeys(t *testing.T) {
	t.Run("Listing keys with invalid params fails", testListingKeysWithInvalidParamsFails)
	t.Run("Listing keys with valid params succeeds", testListingKeysWithValidParamsSucceeds)
	t.Run("Listing keys with invalid token fails", testListingKeysWithInvalidTokenFails)
	t.Run("Listing keys with not enough permissions fails", testListingKeysWithNotEnoughPermissionsFails)
}

func testListingKeysWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with nil params",
			params:        nil,
			expectedError: api.ErrParamsRequired,
		}, {
			name:          "with wrong type of params",
			params:        "test",
			expectedError: api.ErrParamsDoNotMatch,
		}, {
			name: "with empty connection token",
			params: api.ListKeysParams{
				Token: "",
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx, _ := contextWithTraceID()

			// setup
			handler := newListKeysHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testListingKeysWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	hostname := "vega.xyz"
	w := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	})
	_, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatal("could not generate key for tests: %w", err)
	}
	expectedPubKeys := make([]string, 0, len(w.ListPublicKeys()))
	for _, key := range w.ListPublicKeys() {
		expectedPubKeys = append(expectedPubKeys, key.Key())
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, hostname, w)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ListKeysParams{
		Token: token,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, expectedPubKeys, result.Keys)
}

func testListingKeysWithInvalidTokenFails(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newListKeysHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ListKeysParams{
		Token: vgrand.RandomStr(5),
	})

	// then
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrNoWalletConnected)
}

func testListingKeysWithNotEnoughPermissionsFails(t *testing.T) {
	// given
	ctx := context.Background()
	expectedPermissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.NoAccess,
			RestrictedKeys: []string{},
		},
	}
	hostname := "vega.xyz"
	w := walletWithPerms(t, hostname, expectedPermissions)
	_, err := w.GenerateKeyPair(nil)
	if err != nil {
		t.Fatal("could not generate key for tests: %w", err)
	}

	// setup
	handler := newListKeysHandler(t)
	token := connectWallet(t, handler.sessions, hostname, w)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ListKeysParams{
		Token: token,
	})

	// then
	assert.Empty(t, result)
	assertRequestNotPermittedError(t, errorDetails, api.ErrReadAccessOnPublicKeysRequired)
}

type listKeysHandler struct {
	*api.ListKeys
	sessions *api.Sessions
}

func (h *listKeysHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.ListKeysResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.ListKeysResult)
		if !ok {
			t.Fatal("ListKeys handler result is not a ListKeysResult")
		}
		return result, err
	}
	return api.ListKeysResult{}, err
}

func newListKeysHandler(t *testing.T) *listKeysHandler {
	t.Helper()

	sessions := api.NewSessions()

	return &listKeysHandler{
		ListKeys: api.NewListKeys(sessions),
		sessions: sessions,
	}
}
