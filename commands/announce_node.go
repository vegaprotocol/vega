package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckAnnounceNode(cmd *commandspb.AnnounceNode) error {
	return checkAnnounceNode(cmd).ErrorOrNil()
}

func checkAnnounceNode(cmd *commandspb.AnnounceNode) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("announce_node", ErrIsRequired)
	}

	if len(cmd.VegaPubKey) == 0 {
		errs.AddForProperty("announce_node.vega_pub_key", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.VegaPubKey) {
		errs.AddForProperty("announce_node.vega_pub_key", ErrShouldBeAValidVegaPubkey)
	}

	if len(cmd.Id) == 0 {
		errs.AddForProperty("announce_node.id", ErrIsRequired)
	} else if !IsVegaPubkey(cmd.Id) {
		errs.AddForProperty("announce_node.id", ErrShouldBeAValidVegaPubkey)
	}

	if len(cmd.EthereumAddress) == 0 {
		errs.AddForProperty("announce_node.ethereum_address", ErrIsRequired)
	}

	if len(cmd.ChainPubKey) == 0 {
		errs.AddForProperty("announce_node.chain_pub_key", ErrIsRequired)
	}

	return errs
}
