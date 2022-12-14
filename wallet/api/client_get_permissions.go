package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"code.vegaprotocol.io/vega/wallet/wallet"
	"github.com/mitchellh/mapstructure"
)

type ClientGetPermissionsParams struct {
	Token string `json:"token"`
}

type ClientGetPermissionsResult struct {
	Permissions wallet.PermissionsSummary `json:"permissions"`
}

type ClientGetPermissions struct {
	sessions *session.Sessions
}

// Handle returns the permissions set on the given hostname.
//
// If a third-party application does not have enough permissions, it has to
// request them using `request_permissions` handler.
//
// Using this handler does not require permissions.
func (h *ClientGetPermissions) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateGetPermissionsParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	connectedWallet, err := h.sessions.GetConnectedWallet(params.Token, time.Now())
	if err != nil {
		return nil, invalidParams(err)
	}

	return ClientGetPermissionsResult{
		Permissions: connectedWallet.Permissions().Summary(),
	}, nil
}

func validateGetPermissionsParams(rawParams jsonrpc.Params) (ClientGetPermissionsParams, error) {
	if rawParams == nil {
		return ClientGetPermissionsParams{}, ErrParamsRequired
	}

	params := ClientGetPermissionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return ClientGetPermissionsParams{}, ErrParamsDoNotMatch
	}

	if params.Token == "" {
		return ClientGetPermissionsParams{}, ErrConnectionTokenIsRequired
	}

	return params, nil
}

func NewGetPermissions(sessions *session.Sessions) *ClientGetPermissions {
	return &ClientGetPermissions{
		sessions: sessions,
	}
}
