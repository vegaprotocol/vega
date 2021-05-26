package commands

import commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

func CheckChainEvent(cmd *commandspb.ChainEvent) error {
	return checkChainEvent(cmd).ErrorOrNil()
}

func checkChainEvent(cmd *commandspb.ChainEvent) Errors {
	errs := NewErrors()

	if len(cmd.TxId) == 0 {
		errs.AddForProperty("chain_event.tx_id", ErrIsRequired)
	}

	if cmd.Nonce == 0 {
		errs.AddForProperty("chain_event.nonce", ErrIsRequired)
	}

	return errs
}

