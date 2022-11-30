package api

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminDescribeAPITokenParams struct {
	Token string `json:"token"`
}

type AdminDescribeAPITokenResult struct {
	Token       string    `json:"token"`
	Description string    `json:"description"`
	Wallet      string    `json:"wallet"`
	CreatedAt   time.Time `json:"createdAt"`
}

type AdminDescribeAPIToken struct {
	tokenStore TokenStore
}

// Handle describes a long-living API token and its configuration.
func (h *AdminDescribeAPIToken) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminDescribeAPITokenParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	if exist, err := h.tokenStore.TokenExists(params.Token); err != nil {
		return nil, internalError(fmt.Errorf("could not verify the token existence: %w", err))
	} else if !exist {
		return nil, invalidParams(ErrTokenDoesNotExist)
	}

	token, err := h.tokenStore.GetToken(params.Token)
	if err != nil {
		return nil, internalError(fmt.Errorf("could not retrieve the token: %w", err))
	}

	return AdminDescribeAPITokenResult{
		Token:       token.Token,
		Description: token.Description,
		Wallet:      token.Wallet.Name,
	}, nil
}

func validateAdminDescribeAPITokenParams(rawParams jsonrpc.Params) (AdminDescribeAPITokenParams, error) {
	if rawParams == nil {
		return AdminDescribeAPITokenParams{}, ErrParamsRequired
	}

	params := AdminDescribeAPITokenParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminDescribeAPITokenParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return AdminDescribeAPITokenParams{}, ErrTokenIsRequired
	}

	return params, nil
}

func NewAdminDescribeAPIToken(
	tokenStore TokenStore,
) *AdminDescribeAPIToken {
	return &AdminDescribeAPIToken{
		tokenStore: tokenStore,
	}
}
