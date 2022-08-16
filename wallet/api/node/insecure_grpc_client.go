package node

import (
	"context"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"google.golang.org/grpc/credentials/insecure"

	"google.golang.org/grpc"
)

type InsecureGRPCClient struct {
	client     apipb.CoreServiceClient
	connection *grpc.ClientConn

	host string
}

func (c *InsecureGRPCClient) Host() string {
	return c.host
}

func (c *InsecureGRPCClient) SubmitTransaction(ctx context.Context, req *apipb.SubmitTransactionRequest, opts ...grpc.CallOption) (*apipb.SubmitTransactionResponse, error) {
	return c.client.SubmitTransaction(ctx, req, opts...)
}

func (c *InsecureGRPCClient) LastBlockHeight(ctx context.Context, req *apipb.LastBlockHeightRequest, opts ...grpc.CallOption) (*apipb.LastBlockHeightResponse, error) {
	return c.client.LastBlockHeight(ctx, req, opts...)
}

func (c *InsecureGRPCClient) GetVegaTime(ctx context.Context, req *apipb.GetVegaTimeRequest, opts ...grpc.CallOption) (*apipb.GetVegaTimeResponse, error) {
	return c.client.GetVegaTime(ctx, req, opts...)
}

func (c *InsecureGRPCClient) CheckTransaction(ctx context.Context, req *apipb.CheckTransactionRequest, opts ...grpc.CallOption) (*apipb.CheckTransactionResponse, error) {
	return c.client.CheckTransaction(ctx, req, opts...)
}

func (c *InsecureGRPCClient) Stop() error {
	return c.connection.Close()
}

func NewInsecureGRPCClient(host string) (*InsecureGRPCClient, error) {
	connection, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	return &InsecureGRPCClient{
		client:     apipb.NewCoreServiceClient(connection),
		connection: connection,
		host:       host,
	}, nil
}
