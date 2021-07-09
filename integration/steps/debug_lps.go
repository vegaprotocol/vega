package steps

import (
	"code.vegaprotocol.io/data-node/integration/stubs"
	"code.vegaprotocol.io/data-node/logging"
)

func DebugLPs(broker *stubs.BrokerStub, log *logging.Logger) error {
	log.Info("DUMPING LIQUIDITY PROVISION EVENTS")
	data := broker.GetLPEvents()
	for _, lp := range data {
		p := lp.Proto()
		log.Infof("LP %s, %#v\n", p.String(), p)
	}
	return nil
}
