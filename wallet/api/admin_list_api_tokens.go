package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api/session"
)

type AdminListAPITokensResult struct {
	Tokens []session.TokenSummary `json:"tokens"`
}

type AdminListAPITokens struct {
	tokenStore TokenStore
}

// Handle generates a long-living API token.
func (h *AdminListAPITokens) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	err := validateAdminListAPITokensParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	tokens, err := h.tokenStore.ListTokens()
	if err != nil {
		return nil, internalError(fmt.Errorf("could not list the tokens: %w", err))
	}

	return AdminListAPITokensResult{
		Tokens: tokens,
	}, nil
}

func validateAdminListAPITokensParams(rawParams jsonrpc.Params) error {
	if rawParams != nil {
		return ErrMethodWithoutParameters
	}

	return nil
}

func NewAdminListAPITokens(
	tokenStore TokenStore,
) *AdminListAPITokens {
	return &AdminListAPITokens{
		tokenStore: tokenStore,
	}
}
