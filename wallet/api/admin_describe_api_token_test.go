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

func TestAdminDescribeAPIToken(t *testing.T) {
	t.Run("Describing a token with invalid params fails", testDescribingTokenWithInvalidParamsFails)
	t.Run("Describing a token with valid params succeeds", testDescribingTokenWithValidParamsSucceeds)
	t.Run("Describing a token that does not exists fails", testDescribingTokenThatDoesNotExistsFails)
	t.Run("Getting internal error during token verification fails", testGettingInternalErrorDuringTokenVerificationFails)
	t.Run("Getting internal error during token retrieval fails", testGettingInternalErrorDuringTokenRetrievalFails)
}

func testDescribingTokenWithInvalidParamsFails(t *testing.T) {
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
			params: api.AdminDescribeAPITokenParams{
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
			handler := newDescribeAPITokenHandler(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testDescribingTokenWithValidParamsSucceeds(t *testing.T) {
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
	handler := newDescribeAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(true, nil)
	handler.tokenStore.EXPECT().GetToken(expectedTokenConfig.Token).Times(1).Return(expectedTokenConfig, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, api.AdminDescribeAPITokenResult{
		Token:       expectedTokenConfig.Token,
		Description: expectedTokenConfig.Description,
		Wallet:      expectedTokenConfig.Wallet.Name,
	}, result)
}

func testDescribingTokenThatDoesNotExistsFails(t *testing.T) {
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
	handler := newDescribeAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(false, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInvalidParams(t, errorDetails, api.ErrTokenDoesNotExist)
}

func testGettingInternalErrorDuringTokenVerificationFails(t *testing.T) {
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
	handler := newDescribeAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(false, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not verify the token existence: %w", assert.AnError))
}

func testGettingInternalErrorDuringTokenRetrievalFails(t *testing.T) {
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
	handler := newDescribeAPITokenHandler(t)
	// -- expected calls
	handler.tokenStore.EXPECT().TokenExists(expectedTokenConfig.Token).Times(1).Return(true, nil)
	handler.tokenStore.EXPECT().GetToken(expectedTokenConfig.Token).Times(1).Return(session.Token{}, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, api.AdminDescribeAPITokenParams{
		Token: expectedTokenConfig.Token,
	})

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not retrieve the token: %w", assert.AnError))
}

type adminDescribeAPITokenHandler struct {
	*api.AdminDescribeAPIToken
	ctrl       *gomock.Controller
	tokenStore *mocks.MockTokenStore
}

func (h *adminDescribeAPITokenHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminDescribeAPITokenResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	if rawResult != nil {
		result, ok := rawResult.(api.AdminDescribeAPITokenResult)
		if !ok {
			t.Fatal("AdminDescribeAPIToken handler result is not a AdminDescribeAPITokenResult")
		}
		return result, err
	}
	return api.AdminDescribeAPITokenResult{}, err
}

func newDescribeAPITokenHandler(t *testing.T) *adminDescribeAPITokenHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	return &adminDescribeAPITokenHandler{
		AdminDescribeAPIToken: api.NewAdminDescribeAPIToken(tokenStore),
		ctrl:                  ctrl,
		tokenStore:            tokenStore,
	}
}
