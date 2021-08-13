package api

import (
	"context"

	"code.vegaprotocol.io/data-node/metrics"
	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	pb "code.vegaprotocol.io/protos/vega"
	"google.golang.org/grpc/codes"
)

// ValidatorService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/validator_service_mock.go -package mocks code.vegaprotocol.io/data-node/api  ValidatorService
type ValidatorService interface {
	GetNodeData(ctx context.Context) (*pb.NodeData, error)
	GetNodes(ctx context.Context) ([]*pb.Node, error)
	GetNodeByID(ctx context.Context, id string) (*pb.Node, error)
	GetEpochByID(ctx context.Context, id uint64) (*pb.Epoch, error)
	GetEpoch(ctx context.Context) (*pb.Epoch, error)
}

func (t *tradingDataService) GetNodeData(ctx context.Context, req *apipb.GetNodeDataRequest) (*apipb.GetNodeDataResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeData")()

	data, err := t.validatorService.GetNodeData(ctx)
	if err != nil {
		return nil, apiError(codes.Internal, ErrValidatorServiceGetNodeData, err)
	}

	return &apipb.GetNodeDataResponse{
		NodeData: data,
	}, nil
}

func (t *tradingDataService) GetNodes(ctx context.Context, req *apipb.GetNodesRequest) (*apipb.GetNodesResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodes")()
	return nil, nil
}

func (t *tradingDataService) GetNodeByID(ctx context.Context, req *apipb.GetNodeByIDRequest) (*apipb.GetNodeByIDResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetNodeByID")()
	return nil, nil
}

func (t *tradingDataService) GetEpoch(ctx context.Context, req *apipb.GetEpochRequest) (*apipb.GetEpochResponse, error) {
	defer metrics.StartAPIRequestAndTimeGRPC("GetEpoch")()
	return nil, nil
}
