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

func TestAdminListAPITokens(t *testing.T) {
	t.Run("Listing API tokens with invalid params fails", testListingTokensWithInvalidParamsFails)
	t.Run("Listing API tokens succeeds", testAdminListAPITokenSucceeds)
	t.Run("Getting internal error during API tokens listing fails", testGettingInternalErrorDuringAPITokenListingFails)
}

func testListingTokensWithInvalidParamsFails(t *testing.T) {
	tcs := []struct {
		name          string
		params        interface{}
		expectedError error
	}{
		{
			name:          "with non nil params",
			params:        struct{}{},
			expectedError: api.ErrMethodWithoutParameters,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(tt *testing.T) {
			// given
			ctx := context.Background()

			// setup
			handler := newAdminListAPITokenHandlers(tt)

			// when
			result, errorDetails := handler.handle(t, ctx, tc.params)

			// then
			require.Empty(tt, result)
			assertInvalidParams(tt, errorDetails, tc.expectedError)
		})
	}
}

func testAdminListAPITokenSucceeds(t *testing.T) {
	// given
	ctx := context.Background()
	token1 := session.TokenSummary{
		Description: vgrand.RandomStr(5),
		Token:       vgrand.RandomStr(5),
	}
	token2 := session.TokenSummary{
		Description: vgrand.RandomStr(5),
		Token:       vgrand.RandomStr(5),
	}

	// setup
	handler := newAdminListAPITokenHandlers(t)
	// -- expected calls
	tokens := []session.TokenSummary{token1, token2}
	handler.tokenStore.EXPECT().ListTokens().Times(1).Return(tokens, nil)

	// when
	result, errorDetails := handler.handle(t, ctx, nil)

	// then
	require.Nil(t, errorDetails)
	assert.Equal(t, tokens, result.Tokens)
}

func testGettingInternalErrorDuringAPITokenListingFails(t *testing.T) {
	// given
	ctx := context.Background()

	// setup
	handler := newAdminListAPITokenHandlers(t)
	// -- expected calls
	handler.tokenStore.EXPECT().ListTokens().Times(1).Return(nil, assert.AnError)

	// when
	result, errorDetails := handler.handle(t, ctx, nil)

	// then
	require.NotNil(t, errorDetails)
	assert.Empty(t, result)
	assertInternalError(t, errorDetails, fmt.Errorf("could not list the tokens: %w", assert.AnError))
}

type adminListAPITokenHandler struct {
	*api.AdminListAPITokens
	ctrl       *gomock.Controller
	tokenStore *mocks.MockTokenStore
}

func (h *adminListAPITokenHandler) handle(t *testing.T, ctx context.Context, params interface{}) (api.AdminListAPITokensResult, *jsonrpc.ErrorDetails) {
	t.Helper()

	rawResult, err := h.Handle(ctx, params, jsonrpc.RequestMetadata{})
	if rawResult != nil {
		result, ok := rawResult.(api.AdminListAPITokensResult)
		if !ok {
			t.Fatal("AdminListAPITokens handler result is not a AdminListAPITokensResult")
		}
		return result, err
	}
	return api.AdminListAPITokensResult{}, err
}

func newAdminListAPITokenHandlers(t *testing.T) *adminListAPITokenHandler {
	t.Helper()

	ctrl := gomock.NewController(t)
	tokenStore := mocks.NewMockTokenStore(ctrl)

	return &adminListAPITokenHandler{
		AdminListAPITokens: api.NewAdminListAPITokens(tokenStore),
		ctrl:               ctrl,
		tokenStore:         tokenStore,
	}
}
