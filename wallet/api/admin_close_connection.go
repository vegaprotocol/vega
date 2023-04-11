package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminCloseConnectionParams struct {
	Hostname string `json:"hostname"`
	Wallet   string `json:"wallet"`
}

type AdminCloseConnection struct {
	connectionsManager ConnectionsManager
}

// Handle closes the connection between a third-party application and a wallet
// opened in the service that run against the specified network.
// It does not fail if the service or the connection are already closed.
func (h *AdminCloseConnection) Handle(_ context.Context, rawParams jsonrpc.Params) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminCloseConnectionParams(rawParams)
	if err != nil {
		return nil, InvalidParams(err)
	}

	h.connectionsManager.EndSessionConnection(params.Hostname, params.Wallet)

	return nil, nil
}

func validateAdminCloseConnectionParams(rawParams jsonrpc.Params) (AdminCloseConnectionParams, error) {
	if rawParams == nil {
		return AdminCloseConnectionParams{}, ErrParamsRequired
	}

	params := AdminCloseConnectionParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminCloseConnectionParams{}, ErrParamsDoNotMatch
	}

	if params.Hostname == "" {
		return AdminCloseConnectionParams{}, ErrHostnameIsRequired
	}

	if params.Wallet == "" {
		return AdminCloseConnectionParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminCloseConnection(connectionsManager ConnectionsManager) *AdminCloseConnection {
	return &AdminCloseConnection{
		connectionsManager: connectionsManager,
	}
}
