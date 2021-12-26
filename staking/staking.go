package staking

import (
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
)

type AllEthereumClient interface {
	EthereumClient
	EthereumClientConfirmations
	EthereumClientCaller
}

func New(
	log *logging.Logger,
	cfg Config,
	broker Broker,
	tt TimeTicker,
	witness Witness,
	ethClient AllEthereumClient,
	netp *netparams.Store,
	evtFwd EvtForwarder,
) (*Accounting, *StakeVerifier) {
	accs := NewAccounting(log, cfg, broker, ethClient, evtFwd, witness)
	ethCfns := NewEthereumConfirmations(ethClient, nil)
	ocv := NewOnChainVerifier(cfg, log, ethClient, ethCfns)
	sakeV := NewStakeVerifier(log, cfg, accs, tt, witness, broker, ocv)

	netp.Watch(netparams.WatchParam{
		Param:   netparams.BlockchainsEthereumConfig,
		Watcher: ethCfns.OnEthereumConfigUpdate,
	})
	netp.Watch(netparams.WatchParam{
		Param:   netparams.BlockchainsEthereumConfig,
		Watcher: ocv.OnEthereumConfigUpdate,
	})
	netp.Watch(netparams.WatchParam{
		Param:   netparams.BlockchainsEthereumConfig,
		Watcher: accs.OnEthereumConfigUpdate,
	})

	return accs, sakeV
}
