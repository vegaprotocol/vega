package query

import (
	"context"
	"time"

	api "code.vegaprotocol.io/protos/vega/api/v1"

	"google.golang.org/grpc"
)

func getClient(address string) (api.CoreStateServiceClient, error) {
	tdconn, err := grpc.Dial(address, grpc.WithInsecure())
	if err != nil {
		return nil, err
	}
	return api.NewCoreStateServiceClient(tdconn), nil
}

func timeoutContext() (context.Context, func()) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}
