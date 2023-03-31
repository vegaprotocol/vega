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
