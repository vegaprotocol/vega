// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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

type Factory interface {
	GetClient(socketPath, httpPath string) AdminClient
}

type clientFactory struct {
	log *logging.Logger
}

func NewClientFactory(log *logging.Logger) Factory {
	return &clientFactory{
		log: log,
	}
}

func (cf *clientFactory) GetClient(socketPath, httpPath string) AdminClient {
	return admin.NewClient(cf.log, admin.Config{
		Server: admin.ServerConfig{
			SocketPath: socketPath,
			HTTPPath:   httpPath,
		},
	})
}
