package staking

import "code.vegaprotocol.io/vega/logging"

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
	ethClient AllEthereumClient) (*Accounting, *StakeVerifier) {

	accs := NewAccounting(log, cfg, broker)
	ethCfns := NewEthereumConfirmations(ethClient, nil)
	ocv := NewOnChainVerifier(cfg, log, ethClient, ethCfns)
	sakeV := NewStakeVerifier(log, cfg, accs, tt, witness, broker, ocv)

	return accs, sakeV
}
