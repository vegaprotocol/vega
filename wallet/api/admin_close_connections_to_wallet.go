package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminCloseConnectionsToWalletParams struct {
	Network string `json:"network"`
	Wallet  string `json:"wallet"`
}

type AdminCloseConnectionsToWallet struct {
	servicesManager *ServicesManager
}

// Handle closes all the connections from any hostname to the specified wallet
// opened in the service that run against the specified network.
// It does not fail if the service or the connections are already closed.
func (h *AdminCloseConnectionsToWallet) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminCloseConnectionsToWalletParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	sessions, err := h.servicesManager.Sessions(params.Network)
	if err != nil {
		return nil, nil //nolint:nilerr
	}

	connections := sessions.ListConnections()

	for _, connection := range connections {
		if connection.Wallet == params.Wallet {
			sessions.DisconnectWallet(connection.Hostname, params.Wallet)
		}
	}

	return nil, nil
}

func validateAdminCloseConnectionsToWalletParams(rawParams jsonrpc.Params) (AdminCloseConnectionsToWalletParams, error) {
	if rawParams == nil {
		return AdminCloseConnectionsToWalletParams{}, ErrParamsRequired
	}

	params := AdminCloseConnectionsToWalletParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminCloseConnectionsToWalletParams{}, ErrParamsDoNotMatch
	}

	if params.Network == "" {
		return AdminCloseConnectionsToWalletParams{}, ErrNetworkIsRequired
	}

	if params.Wallet == "" {
		return AdminCloseConnectionsToWalletParams{}, ErrWalletIsRequired
	}

	return params, nil
}

func NewAdminCloseConnectionsToWallet(servicesManager *ServicesManager) *AdminCloseConnectionsToWallet {
	return &AdminCloseConnectionsToWallet{
		servicesManager: servicesManager,
	}
}
