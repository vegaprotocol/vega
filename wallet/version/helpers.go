package version

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	apipb "code.vegaprotocol.io/vega/protos/vega/api/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
)

var (
	ErrConnectionError      = errors.New("verify you are connected to the internet")
	ErrCouldNotConnectNodes = errors.New("could not connect to any nodes: verify your network configuration is up to date")
)

func GetNetworkVersionThroughGRPC(hosts []string) (string, error) {
	for _, host := range hosts {
		version, err := queryNetworkVersion(host)
		if err != nil {
			continue
		}
		return version, nil
	}

	// Before advising the users to check their network configuration, the software
	// pings vega.xyz to verify whether or not they are connected to the internet.
	// This should help the user with troubleshooting their setup.
	ctx, cancelFn := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelFn()
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, "https://vega.xyz", nil)
	if err != nil {
		return "", fmt.Errorf("could not build the request that verifies the network connection: %w", err)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", ErrConnectionError
	}
	if resp.Body != nil {
		_ = resp.Body.Close()
	}

	// If it reaches that point, it's been possible to get a response to the ping to
	// `vega.xyz`, which suggest the issue comes from the network configuration
	// (outdated, or misconfigured), or from the nodes themselves (not responding,
	// or giving unexpected responses).
	return "", ErrCouldNotConnectNodes
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
