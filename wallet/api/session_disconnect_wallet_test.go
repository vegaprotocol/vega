package api_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
)

func TestDisconnectWallet(t *testing.T) {
	t.Run("Disconnecting a wallet with invalid params fails", testDisconnectingWalletWithInvalidParamsFails)
	t.Run("Disconnecting a wallet with valid params succeeds", testDisconnectingWalletWithValidParamsSucceeds)
	t.Run("Disconnecting a wallet with invalid token succeeds", testDisconnectingWalletWithInvalidTokenSucceeds)
}

func testDisconnectingWalletWithInvalidParamsFails(t *testing.T) {
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
			params: api.DisconnectWalletParams{
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
			handler := newDisconnectWalletHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			assert.Nil(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testDisconnectingWalletWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	hostname := "vega.xyz"
	w := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	})

	// setup
	handler := newDisconnectWalletHandler(t)
	token := connectWallet(t, handler.sessions, hostname, w)

	// when
	result, errorDetails := handler.handle(t, ctx, api.DisconnectWalletParams{
		Token: token,
	})

	// then
	assert.Nil(t, errorDetails)
	assert.Nil(t, result)
	connectedWallet, err := handler.sessions.GetConnectedWallet(token)
	assert.Nil(t, connectedWallet)
	assert.Error(t, api.ErrNoWalletConnected, err)
}

func testDisconnectingWalletWithInvalidTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newDisconnectWalletHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.DisconnectWalletParams{
		Token: vgrand.RandomStr(5),
	})

	// then
	assert.Nil(t, result)
	assert.Nil(t, errorDetails)
}

type disconnectWalletHandler struct {
	*api.DisconnectWallet
	sessions *api.Sessions
}

func (h *disconnectWalletHandler) handle(t *testing.T, ctx context.Context, params interface{}) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	t.Helper()

	return h.Handle(ctx, params)
}

func newDisconnectWalletHandler(t *testing.T) *disconnectWalletHandler {
	t.Helper()

	sessions := api.NewSessions()

	return &disconnectWalletHandler{
		DisconnectWallet: api.NewDisconnectWallet(sessions),
		sessions:         sessions,
	}
}
