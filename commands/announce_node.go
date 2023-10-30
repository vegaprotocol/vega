// Copyright (C) 2023  Gobalsky Labs Limited
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

	"code.vegaprotocol.io/vega/libs/crypto"
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
	} else if !IsVegaPublicKey(cmd.VegaPubKey) {
		errs.AddForProperty("announce_node.vega_pub_key", ErrShouldBeAValidVegaPublicKey)
	}

	if len(cmd.Id) == 0 {
		errs.AddForProperty("announce_node.id", ErrIsRequired)
	} else if !IsVegaPublicKey(cmd.Id) {
		errs.AddForProperty("announce_node.id", ErrShouldBeAValidVegaPublicKey)
	}

	if len(cmd.EthereumAddress) == 0 {
		errs.AddForProperty("announce_node.ethereum_address", ErrIsRequired)
	} else if !crypto.EthereumIsValidAddress(cmd.EthereumAddress) {
		errs.AddForProperty("announce_node.ethereum_address", ErrIsNotValidEthereumAddress)
	}

	if len(cmd.ChainPubKey) == 0 {
		errs.AddForProperty("announce_node.chain_pub_key", ErrIsRequired)
	}

	if cmd.EthereumSignature == nil || len(cmd.EthereumSignature.Value) == 0 {
		errs.AddForProperty("announce_node.ethereum_signature", ErrIsRequired)
	} else {
		_, err := hex.DecodeString(cmd.EthereumSignature.Value)
		if err != nil {
			errs.AddForProperty("announce_node.ethereum_signature.value", ErrShouldBeHexEncoded)
		}
	}

	if cmd.VegaSignature == nil || len(cmd.VegaSignature.Value) == 0 {
		errs.AddForProperty("announce_node.vega_signature", ErrIsRequired)
	} else {
		_, err := hex.DecodeString(cmd.VegaSignature.Value)
		if err != nil {
			errs.AddForProperty("announce_node.vega_signature.value", ErrShouldBeHexEncoded)
		}
	}

	if len(cmd.SubmitterAddress) != 0 && !crypto.EthereumIsValidAddress(cmd.SubmitterAddress) {
		errs.AddForProperty("announce_node.submitter_address", ErrIsNotValidEthereumAddress)
	}

	return errs
}
