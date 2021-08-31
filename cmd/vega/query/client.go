package query

import (
	"context"
	"time"

	coreapi "code.vegaprotocol.io/protos/vega/coreapi/v1"

	"google.golang.org/grpc"
)

func getClient(address string) (coreapi.CoreApiServiceClient, error) {
	tdconn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return coreapi.NewCoreApiServiceClient(tdconn), nil
}

func timeoutContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
