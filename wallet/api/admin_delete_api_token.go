package api

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminDeleteAPITokenParams struct {
	Token string `json:"token"`
}

type AdminDeleteAPIToken struct {
	tokenStore TokenStore
}

// Handle generates a long-living API token.
func (h *AdminDeleteAPIToken) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminDeleteAPITokenParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.tokenStore.TokenExists(params.Token); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the token existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrTokenDoesNotExist)
	}

	if err := h.tokenStore.DeleteToken(params.Token); err != nil {
		return nil, internalError(fmt.Errorf("could not delete the token: %w", err))
	}

	return nil, nil
}

func validateAdminDeleteAPITokenParams(rawParams jsonrpc.Params) (AdminDeleteAPITokenParams, error) {
	if rawParams == nil {
		return AdminDeleteAPITokenParams{}, ErrParamsRequired
	}

	params := AdminDeleteAPITokenParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDeleteAPITokenParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return AdminDeleteAPITokenParams{}, ErrTokenIsRequired
	}

	return params, nil
}

func NewAdminDeleteAPIToken(
	tokenStore TokenStore,
) *AdminDeleteAPIToken {
	return &AdminDeleteAPIToken{
		tokenStore: tokenStore,
	}
}
