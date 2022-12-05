package api_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/stretchr/testify/assert"
)

func TestDisconnectWallet(t *testing.T) {
	t.Run("Disconnecting a wallet with invalid params fails", testDisconnectingWalletWithInvalidParamsFails)
	t.Run("Disconnecting a wallet with valid params succeeds", testDisconnectingWalletWithValidParamsSucceeds)
	t.Run("Disconnecting a wallet with invalid token succeeds", testDisconnectingWalletWithInvalidTokenSucceeds)
	t.Run("Disconnecting a wallet with long-living token succeeds", testDisconnectingWalletWithLongLivingTokenSucceeds)
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
			params: api.ClientDisconnectWalletParams{
				Token: "",
			},
			expectedError: api.ErrConnectionTokenIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

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
	w, _ := walletWithPerms(t, hostname, wallet.Permissions{
		PublicKeys: wallet.PublicKeysPermission{
			Access:         wallet.ReadAccess,
			RestrictedKeys: []string{},
		},
	})

	// setup
	handler := newDisconnectWalletHandler(t)
	token := connectWallet(t, handler.sessions, hostname, w)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientDisconnectWalletParams{
		Token: token,
	})

	// then
	assert.Nil(t, errorDetails)
	assert.Nil(t, result)
	connectedWallet, err := handler.sessions.GetConnectedWallet(token, time.Now())
	assert.Nil(t, connectedWallet)
	assert.Error(t, session.ErrNoWalletConnected, err)
}

func testDisconnectingWalletWithInvalidTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	w, _ := walletWithKey(t)
	token := vgrand.RandomStr(10)

	// setup
	handler := newDisconnectWalletHandler(t)
	if err := handler.sessions.ConnectWalletForLongLivingConnection(token, w, time.Now(), nil); err != nil {
		t.Fatalf("could not connect test wallet to a long-living sessions: %v", err)
	}

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientDisconnectWalletParams{
		Token: token,
	})

	// then
	assert.Nil(t, result)
	assertRequestNotPermittedError(t, errorDetails, session.ErrCannotEndLongLivingSessions)
}

func testDisconnectingWalletWithLongLivingTokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newDisconnectWalletHandler(t)

	// when
	result, errorDetails := handler.handle(t, ctx, api.ClientDisconnectWalletParams{
		Token: vgrand.RandomStr(5),
	})

	// then
	assert.Nil(t, result)
	assert.Nil(t, errorDetails)
}

type disconnectWalletHandler struct {
	*api.ClientDisconnectWallet
	sessions *session.Sessions
}

func (h *disconnectWalletHandler) handle(t *testing.T, ctx context.Context, params interface{}) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	t.Helper()

	return h.Handle(ctx, params, requestMetadataForTest())
}

func newDisconnectWalletHandler(t *testing.T) *disconnectWalletHandler {
	t.Helper()

	sessions := session.NewSessions()

	return &disconnectWalletHandler{
		ClientDisconnectWallet: api.NewDisconnectWallet(sessions),
		sessions:               sessions,
	}
}
