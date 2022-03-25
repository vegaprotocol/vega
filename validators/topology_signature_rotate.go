package validators

import (
	"context"
	"fmt"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
)

func (t *Topology) RotateSignature(ctx context.Context, nodeID string, kr *commandspb.KeyRotateSubmission) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	node, ok := t.validators[nodeID]
	if !ok {
		return fmt.Errorf("failed to rotate signature for non existing validator %q", nodeID)
	}

	if node.status != ValidatorStatusTendermint {
		return fmt.Errorf("failed to rotate signature for non %q validator %q", ValidatorStatusTendermint, nodeID)
	}

	// node.data.EthereumAddress =

	return nil
}
