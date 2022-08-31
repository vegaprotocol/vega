package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type GetPermissionsParams struct {
	Token string `json:"token"`
}

type GetPermissionsResult struct {
	Permissions wallet.PermissionsSummary `json:"permissions"`
}

type GetPermissions struct {
	sessions *Sessions
}

// Handle returns the permissions set on the given hostname.
//
// If a third-party application does not have enough permissions, it has to
// request them using `request_permissions` handler.
//
// Using this handler does not require permissions.
func (h *GetPermissions) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateGetPermissionsParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	connectedWallet, err := h.sessions.GetConnectedWallet(params.Token)
	if err != nil {
		return nil, invalidParams(err)
	}

	return GetPermissionsResult{
		Permissions: connectedWallet.Permissions().Summary(),
	}, nil
}

func validateGetPermissionsParams(rawParams jsonrpc.Params) (GetPermissionsParams, error) {
	if rawParams == nil {
		return GetPermissionsParams{}, ErrParamsRequired
	}

	params := GetPermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return GetPermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return GetPermissionsParams{}, ErrConnectionTokenIsRequired
	}

	return params, nil
}

func NewGetPermissions(sessions *Sessions) *GetPermissions {
	return &GetPermissions{
		sessions: sessions,
	}
}
