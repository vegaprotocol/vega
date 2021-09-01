package staking

import (
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types/num"
)

type AllEthereumClient interface {
	EthereumClient
	EthereumClientConfirmations
}

func New(
	log *logging.Logger,
	cfg Config,
	broker Broker,
	tt TimeTicker,
	witness Witness,
	ethClient AllEthereumClient,
	netp *netparams.Store,
) (*Accounting, *StakeVerifier) {

	accs := NewAccounting(log, cfg, broker)
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

	return accs, sakeV
}

type AccountsW struct {
	*Accounting
}

func (a *AccountsW) GetBalanceNow(party string) *num.Uint {
	balance, _ := a.GetAvailableBalance(party)
	return balance
}

func (a *AccountsW) GetBalanceForEpoch(party string, from, to time.Time) *num.Uint {
	balance, _ := a.GetAvailableBalanceInRange(party, from, to)
	return balance
}
