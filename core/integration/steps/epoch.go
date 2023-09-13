package steps

import (
	"fmt"

	"code.vegaprotocol.io/vega/core/integration/stubs"
)

func TheCurrentEpochIs(broker *stubs.BrokerStub, epoch string) error {
	seq, err := U64(epoch)
	if err != nil {
		return err
	}
	last := broker.GetCurrentEpoch()

	// If we haven't had an epoch event yet
	// assume we are on epoch 0
	ce := uint64(0)
	if last != nil {
		ce = last.Epoch().GetSeq()
	}
	if ce != seq {
		return fmt.Errorf("expected current epoch to be %d, instead saw %d", seq, ce)
	}
	return nil
}
