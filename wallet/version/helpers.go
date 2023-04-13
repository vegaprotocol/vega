package version

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var ErrNoHealthyNodeAvailableForVersionCheck = errors.New("no healthy node available for version check")

func GetNetworkVersionThroughGRPC(hosts []string) (string, error) {
	for _, host := range hosts {
		version, err := queryNetworkVersion(host)
		if err != nil {
			continue
		}
		return version, nil
	}
	return "", ErrNoHealthyNodeAvailableForVersionCheck
}

func queryNetworkVersion(host string) (string, error) {
	useTLS := strings.HasPrefix(host, "tls://")

	var creds credentials.TransportCredentials
	if useTLS {
		host = host[6:]
		creds = credentials.NewClientTLSFromCert(nil, "")
	} else {
		creds = insecure.NewCredentials()
	}

	connection, err := grpc.Dial(host, grpc.WithTransportCredentials(creds))
	if err != nil {
		return "", fmt.Errorf("couldn't initialize gRPC client: %w", err)
	}
	defer func() {
		_ = connection.Close()
	}()

	client := apipb.NewCoreServiceClient(connection)

	timeout, cancelFn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFn()

	statistics, err := client.Statistics(timeout, &apipb.StatisticsRequest{})
	if err != nil {
		return "", err
	}

	return statistics.Statistics.AppVersion, nil
}
