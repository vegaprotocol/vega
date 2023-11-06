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

package v1

import (
	"code.vegaprotocol.io/vega/wallet/network"

	"go.uber.org/zap"
)

type API struct {
	log *zap.Logger

	network *network.Network

	handler     WalletHandler
	auth        Auth
	nodeForward NodeForward
	policy      Policy
	spam        SpamHandler
}

func NewAPI(
	log *zap.Logger,
	handler WalletHandler,
	auth Auth,
	nodeForward NodeForward,
	policy Policy,
	net *network.Network,
	spam SpamHandler,
) *API {
	return &API{
		log:         log,
		network:     net,
		handler:     handler,
		auth:        auth,
		nodeForward: nodeForward,
		policy:      policy,
		spam:        spam,
	}
}
