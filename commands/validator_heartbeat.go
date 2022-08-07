package commands

import (
	"encoding/hex"

	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
)

func CheckValidatorHeartbeat(cmd *commandspb.ValidatorHeartbeat) error {
	return checkValidatorHeartbeat(cmd).ErrorOrNil()
}

func checkValidatorHeartbeat(cmd *commandspb.ValidatorHeartbeat) Errors {
	errs := NewErrors()

	if cmd == nil {
		return errs.FinalAddForProperty("validator_heartbeat", ErrIsRequired)
	}

	if len(cmd.NodeId) == 0 {
		errs.AddForProperty("validator_heartbeat.vega_pub_key", ErrIsRequired)
	} else {
		_, err := hex.DecodeString(cmd.NodeId)
		if err != nil {
			errs.AddForProperty("validator_heartbeat.vega_pub_key", ErrShouldBeHexEncoded)
		}
	}

	if cmd.VegaSignature == nil || len(cmd.EthereumSignature.Value) == 0 {
		errs.AddForProperty("validator_heartbeat.ethereum_signature.value", ErrIsRequired)
	} else {
		_, err := hex.DecodeString(cmd.EthereumSignature.Value)
		if err != nil {
			errs.AddForProperty("validator_heartbeat.ethereum_signature.value", ErrShouldBeHexEncoded)
		}
	}

	if cmd.VegaSignature == nil || len(cmd.VegaSignature.Value) == 0 {
		errs.AddForProperty("validator_heartbeat.vega_signature.value", ErrIsRequired)
	} else {
		_, err := hex.DecodeString(cmd.VegaSignature.Value)
		if err != nil {
			errs.AddForProperty("validator_heartbeat.vega_signature.value", ErrShouldBeHexEncoded)
		}
	}

	if len(cmd.VegaSignature.Algo) == 0 {
		errs.AddForProperty("validator_heartbeat.vega_signature.algo", ErrIsRequired)
	}

	return errs
}
