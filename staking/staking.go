package staking

import (
	"time"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/netparams"
	"code.vegaprotocol.io/vega/types/num"
)

const StakingAssetTotalSupply = "64999723000000000000000000"

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

	// @TODO instead of using hardcoded value:
	// 1. Use the staking abi code to call ethereum and get token ethereum address.
	// 2. Use the address to call the erc20 abi code an get the total supply of the token.
	sats, _ := num.UintFromString(StakingAssetTotalSupply, 10)

	accs := NewAccounting(log, cfg, broker, sats)
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
