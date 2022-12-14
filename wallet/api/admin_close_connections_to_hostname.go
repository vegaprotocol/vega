package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminCloseConnectionsToHostnameParams struct {
	Network  string `json:"network"`
	Hostname string `json:"hostname"`
}

type AdminCloseConnectionsToHostname struct {
	servicesManager *ServicesManager
}

// Handle closes all the connections from the specified hostname to any wallet
// opened in the service that run against the specified network.
// It does not fail if the service or the connections are already closed.
func (h *AdminCloseConnectionsToHostname) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminCloseConnectionsToHostnameParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	sessions, err := h.servicesManager.Sessions(params.Network)
	if err != nil {
		return nil, nil //nolint:nilerr
	}

	connections := sessions.ListConnections()

	for _, connection := range connections {
		if connection.Hostname == params.Hostname {
			sessions.DisconnectWallet(params.Hostname, connection.Wallet)
		}
	}

	return nil, nil
}

func validateAdminCloseConnectionsToHostnameParams(rawParams jsonrpc.Params) (AdminCloseConnectionsToHostnameParams, error) {
	if rawParams == nil {
		return AdminCloseConnectionsToHostnameParams{}, ErrParamsRequired
	}

	params := AdminCloseConnectionsToHostnameParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminCloseConnectionsToHostnameParams{}, ErrParamsDoNotMatch
	}

	if params.Network == "" {
		return AdminCloseConnectionsToHostnameParams{}, ErrNetworkIsRequired
	}

	if params.Hostname == "" {
		return AdminCloseConnectionsToHostnameParams{}, ErrHostnameIsRequired
	}

	return params, nil
}

func NewAdminCloseConnectionsToHostname(servicesManager *ServicesManager) *AdminCloseConnectionsToHostname {
	return &AdminCloseConnectionsToHostname{
		servicesManager: servicesManager,
	}
}
