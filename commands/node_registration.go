package commands

import commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

func CheckNodeRegistration(cmd *commandspb.NodeRegistration) error {
	return checkNodeRegistration(cmd).ErrorOrNil()
}

func checkNodeRegistration(cmd *commandspb.NodeRegistration) Errors {
	errs := NewErrors()

	if len(cmd.PubKey) == 0 {
		errs.AddForProperty("node_registration.pub_key", ErrIsRequired)
	}

	if len(cmd.ChainPubKey) == 0 {
		errs.AddForProperty("node_registration.chain_pub_key", ErrIsRequired)
	}

	return errs
}
