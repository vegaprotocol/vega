package api_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/wallet/api"
	"code.vegaprotocol.io/vega/wallet/api/mocks"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAdminDeleteAPIToken(t *testing.T) {
	t.Run("Deleting a token with invalid params fails", testDeletingTokenWithInvalidParamsFails)
	t.Run("Deleting a token with valid params succeeds", testDeletingTokenWithValidParamsSucceeds)
	t.Run("Deleting a token that does not exists fails", testDeletingTokenThatDoesNotExistsFails)
	t.Run("Getting internal error during verification does not delete the token", testGettingInternalErrorDuringVerificationDoesNotDeleteToken)
	t.Run("Getting internal error during deletion does not delete the token", testGettingInternalErrorDuringDeletionDoesNotDeleteToken)
}

func testDeletingTokenWithInvalidParamsFails(t *testing.T) {
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
			name: "with empty token",
			params: api.AdminDeleteAPITokenParams{
				Token: "",
			},
			expectedError: api.ErrTokenIsRequired,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newDeleteAPITokenHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testDeletingTokenWithValidParamsSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	expectedTokenConfig := session.Token{
		Description: vgrand.RandomStr(5),
		Token:       vgrand.RandomStr(5),
		Wallet: session.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// setup
	handler := newDeleteAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(true, nil)
	handler.tokenStore.EXPECT().DeleteToken(expectedTokenConfig.Token).Times(1).Return(nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDeleteAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Nil(t, result)
}

func testDeletingTokenThatDoesNotExistsFails(t *testing.T) {
	// given
	ctx := context.Background()
	expectedTokenConfig := session.Token{
		Description: vgrand.RandomStr(5),
		Token:       vgrand.RandomStr(5),
		Wallet: session.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// setup
	handler := newDeleteAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDeleteAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrTokenDoesNotExist)
}

func testGettingInternalErrorDuringVerificationDoesNotDeleteToken(t *testing.T) {
	// given
	ctx := context.Background()
	expectedTokenConfig := session.Token{
		Description: vgrand.RandomStr(5),
		Token:       vgrand.RandomStr(5),
		Wallet: session.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// setup
	handler := newDeleteAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDeleteAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the token existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringDeletionDoesNotDeleteToken(t *testing.T) {
	// given
	ctx := context.Background()
	expectedTokenConfig := session.Token{
		Description: vgrand.RandomStr(5),
		Token:       vgrand.RandomStr(5),
		Wallet: session.WalletCredentials{
			Name:       vgrand.RandomStr(5),
			Passphrase: vgrand.RandomStr(5),
		},
	}

	// setup
	handler := newDeleteAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(true, nil)
	handler.tokenStore.EXPECT().DeleteToken(expectedTokenConfig.Token).Times(1).Return(assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDeleteAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not delete the token: %w", assert.AnError))
}

type adminDeleteAPITokenHandler struct {
	*api.AdminDeleteAPIToken
	ctrl       *gomock.Controller
	tokenStore *mocks.MockTokenStore
}

func (h *adminDeleteAPITokenHandler) handle(t *testing.T, ctx context.Context, params interface{}) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	t.Helper()

	return h.Handle(ctx, params, jsonrpc.RequestMetadata{})
}

func newDeleteAPITokenHandler(t *testing.T) *adminDeleteAPITokenHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	return &adminDeleteAPITokenHandler{
		AdminDeleteAPIToken: api.NewAdminDeleteAPIToken(tokenStore),
		ctrl:                ctrl,
		tokenStore:          tokenStore,
	}
}
