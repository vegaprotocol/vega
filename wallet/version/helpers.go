package version

import (
	"context"
	"errors"
	"fmt"
	"time"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrNoHealthyNodeAvailableForVersionCheck = errors.New("no healthy node available for version check")

func GetNetworkVersionThroughGRPC(hosts []string) (string, error) {
	for _, host := range hosts {
		connection, err := grpc.Dial(host, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return "", fmt.Errorf("couldn't initialize gRPC client: %w", err)
		}

		client := apipb.NewCoreServiceClient(connection)
		timeout, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
		statistics, err := client.Statistics(timeout, &apipb.StatisticsRequest{})
		if err != nil {
			cancelFn()
			continue
		}
		cancelFn()
		return statistics.Statistics.AppVersion, nil
	}
	return "", ErrNoHealthyNodeAvailableForVersionCheck
}
