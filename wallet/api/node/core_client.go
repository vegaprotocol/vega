package node

import (
	"context"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"

	"google.golang.org/grpc"
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/node_mock.go -package mocks code.vegaprotocol.io/vega/wallet/api/node CoreClient
type CoreClient interface {
	Host() string
	SubmitTransaction(ctx context.Context, in *apipb.SubmitTransactionRequest, opts ...grpc.CallOption) (*apipb.SubmitTransactionResponse, error)
	LastBlockHeight(ctx context.Context, in *apipb.LastBlockHeightRequest, opts ...grpc.CallOption) (*apipb.LastBlockHeightResponse, error)
	GetVegaTime(ctx context.Context, in *apipb.GetVegaTimeRequest, opts ...grpc.CallOption) (*apipb.GetVegaTimeResponse, error)
	CheckTransaction(ctx context.Context, in *apipb.CheckTransactionRequest, opts ...grpc.CallOption) (*apipb.CheckTransactionResponse, error)
	Stop() error
}
