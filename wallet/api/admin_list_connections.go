package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"code.vegaprotocol.io/vega/wallet/api/session"
	"github.com/mitchellh/mapstructure"
)

type AdminListConnectionsParams struct {
	Network string `json:"network"`
}

type AdminListConnectionsResult struct {
	ActiveConnections []session.Connection `json:"activeConnections"`
}

type AdminListConnections struct {
	servicesManager *ServicesManager
}

// Handle closes all opened connections to a running service and stop the service.
func (h *AdminListConnections) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminListConnectionsParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	sessions, err := h.servicesManager.Sessions(params.Network)
	if err != nil {
		return nil, invalidParams(err)
	}

	return AdminListConnectionsResult{
		ActiveConnections: sessions.ListConnections(),
	}, nil
}

func validateAdminListConnectionsParams(rawParams jsonrpc.Params) (AdminListConnectionsParams, error) {
	if rawParams == nil {
		return AdminListConnectionsParams{}, ErrParamsRequired
	}

	params := AdminListConnectionsParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminListConnectionsParams{}, ErrParamsDoNotMatch
	}

	if params.Network == "" {
		return AdminListConnectionsParams{}, ErrNetworkIsRequired
	}

	return params, nil
}

func NewAdminListConnections(servicesManager *ServicesManager) *AdminListConnections {
	return &AdminListConnections{
		servicesManager: servicesManager,
	}
}
