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
	if ce := last.Epoch().GetSeq(); ce != seq {
		return fmt.Errorf("expected current epoch to be %d, instead saw %d", seq, ce)
	}
	return nil
}
