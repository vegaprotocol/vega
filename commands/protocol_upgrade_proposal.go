package commands

import (
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckProtocolUpgradeProposal(cmd *commandspb.ProtocolUpgradeProposal) error {
	return checkProtocolUpgradeProposal(cmd).ErrorOrNil()
}

func checkProtocolUpgradeProposal(cmd *commandspb.ProtocolUpgradeProposal) Errors {
	errs := NewErrors()
	if cmd == nil {
		return errs.FinalAddForProperty("protocol_upgrade_proposal", ErrIsRequired)
	}

	if len(cmd.VegaReleaseTag) == 0 {
		errs.AddForProperty("protocol_upgrade_proposal.vega_release_tag", ErrIsRequired)
	}

	return errs
}
