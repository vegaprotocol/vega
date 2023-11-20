// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

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
		errs.AddForProperty("validator_heartbeat.node_id", ErrIsRequired)
	} else {
		if !IsVegaPublicKey(cmd.NodeId) {
			errs.AddForProperty("validator_heartbeat.node_id", ErrShouldBeAValidVegaPublicKey)
		}
	}

	if cmd.EthereumSignature == nil || len(cmd.EthereumSignature.Value) == 0 {
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
