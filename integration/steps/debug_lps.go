package steps

import (
	"code.vegaprotocol.io/vega/integration/stubs"
	"code.vegaprotocol.io/vega/logging"
)

func DebugLPs(broker *stubs.BrokerStub, log *logging.Logger) {
	log.Info("DUMPING LIQUIDITY PROVISION EVENTS")
	data := broker.GetLPEvents()
	for _, lp := range data {
		p := lp.Proto()
		log.Infof("LP %s, %#v\n", p.String(), p)
	}
}
