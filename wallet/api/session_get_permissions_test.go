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

func TestGetPermissions(t *testing.T) {
	t.Run("Getting permissions with invalid params fails", testGettingPermissionsWithInvalidParamsFails)
	t.Run("Getting permissions with valid params succeeds", testGettingPermissionsWithValidParamsSucceeds)
	t.Run("Getting permissions with invalid token fails", testGettingPermissionsWithInvalidTokenFails)
}

func testGettingPermissionsWithInvalidParamsFails(t *testing.T) {
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
			params: api.GetPermissionsParams{
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
			handler := newGetPermissionsHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assert.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testGettingPermissionsWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	expectedPermissions := wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	}
	hostname := "vega.xyz"
	w, _ := walletWithPerms(t, hostname, expectedPermissions)

	// setup
	handler := newGetPermissionsHandler(t)
	token := connectWallet(t, handler.sessions, hostname, w)

	// when
	result, errorDetails := handler.handle(t, ctx, api.GetPermissionsParams{
		Token: token,
	})

	// then
	assert.Nil(t, errorDetails)
	require.NotEmpty(t, result)
	assert.Equal(t, wallet.PermissionsSummary{"public_keys": "read"}, result.Permissions)
}

func testGettingPermissionsWithInvalidTokenFails(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newGetPermissionsHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.GetPermissionsParams{
		Token: vgrand.RandomStr(5),
	})

	// then
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrNoWalletConnected)
}

type GetPermissionsHandler struct {
	*api.GetPermissions
	sessions *api.Sessions
}

func (h *GetPermissionsHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.GetPermissionsResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params)
	if rawResult != nil {
		result, ok := rawResult.(api.GetPermissionsResult)
		if !ok {
			t.Fatal("GetPermissions handler result is not a GetPermissionsResult")
		}
		return result, err
	}
	return api.GetPermissionsResult{}, err
}

func newGetPermissionsHandler(t *testing.T) *GetPermissionsHandler {
	t.Helper()

	sessions := api.NewSessions()

	return &GetPermissionsHandler{
		GetPermissions: api.NewGetPermissions(sessions),
		sessions:       sessions,
	}
}
