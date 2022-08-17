package client

import (
	"context"

	"code.vegaprotocol.io/vega/core/admin"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/logging"
)

type AdminClient interface {
	UpgradeStatus(ctx context.Context) (*types.UpgradeStatus, error)
}

type ClientFactory interface {
	GetClient(socketPath, httpPath string) AdminClient
}

type clientFactory struct {
	log *logging.Logger
}

func NewClientFactory(log *logging.Logger) ClientFactory {
	return &clientFactory{
		log: log,
	}
}

func (cf *clientFactory) GetClient(socketPath, httpPath string) AdminClient {
	return admin.NewClient(cf.log, admin.Config{
		Server: admin.ServerConfig{
			SocketPath: socketPath,
			HttpPath:   httpPath,
		},
	})
}
