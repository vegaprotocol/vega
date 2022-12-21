package api

import (
	"context"

	"code.vegaprotocol.io/vega/libs/jsonrpc"
	"github.com/mitchellh/mapstructure"
)

type AdminStopServiceParams struct {
	Network string `json:"network"`
}

type AdminStopService struct {
	servicesManager *ServicesManager
}

// Handle closes all opened connections to a running service and stop the service.
func (h *AdminStopService) Handle(_ context.Context, rawParams jsonrpc.Params, _ jsonrpc.RequestMetadata) (jsonrpc.Result, *jsonrpc.ErrorDetails) {
	params, err := validateAdminStopServiceParams(rawParams)
	if err != nil {
		return nil, invalidParams(err)
	}

	h.servicesManager.StopService(params.Network)

	return nil, nil
}

func validateAdminStopServiceParams(rawParams jsonrpc.Params) (AdminStopServiceParams, error) {
	if rawParams == nil {
		return AdminStopServiceParams{}, ErrParamsRequired
	}

	params := AdminStopServiceParams{}
	if err := mapstructure.Decode(rawParams, &params); err != nil {
		return AdminStopServiceParams{}, ErrParamsDoNotMatch
	}

	if params.Network == "" {
		return AdminStopServiceParams{}, ErrNetworkIsRequired
	}

	return params, nil
}

func NewAdminStopService(servicesManager *ServicesManager) *AdminStopService {
	return &AdminStopService{
		servicesManager: servicesManager,
	}
}
