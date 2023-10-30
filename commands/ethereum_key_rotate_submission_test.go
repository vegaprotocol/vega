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

package commands_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestSubmittingNonEthereumKeyRotateSubmissionCommandFails(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(nil)

	assert.Contains(t, err.Get("ethereum_key_rotate_submission"), commands.ErrIsRequired)
}

func TestEthereumKeyRotateSubmissionSubmittingEmptyCommandFails(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(&commandspb.EthereumKeyRotateSubmission{})

	assert.Contains(t, err.Get("ethereum_key_rotate_submission.new_address"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("ethereum_key_rotate_submission.current_address"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("ethereum_key_rotate_submission.target_block"), commands.ErrIsRequired)
}

func TestEthereumKeyRotateSubmissionMissingNewAddressFails(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(&commandspb.EthereumKeyRotateSubmission{
		TargetBlock:    100,
		CurrentAddress: "0xED816fd7a6e39bce5d2df6A756F3812da06960fC",
		EthereumSignature: &commandspb.Signature{
			Value: "deadbeef",
			Algo:  "vega/ed25519",
		},
	})

	assert.Contains(t, err.Get("ethereum_key_rotate_submission.new_address"), commands.ErrIsRequired)
}

func TestEthereumKeyRotateSubmissionMissingCurrentAddressFails(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(&commandspb.EthereumKeyRotateSubmission{
		NewAddress:  "0xE7d65d1A6CD6eCcfbE78A5Aea2f096Dc60C4C127",
		TargetBlock: 100,
		EthereumSignature: &commandspb.Signature{
			Value: "deadbeef",
			Algo:  "vega/ed25519",
		},
	})

	assert.Contains(t, err.Get("ethereum_key_rotate_submission.current_address"), commands.ErrIsRequired)
}

func TestEthereumKeyRotateSubmissionMissingTargetBlockFails(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(&commandspb.EthereumKeyRotateSubmission{
		NewAddress:     "0xE7d65d1A6CD6eCcfbE78A5Aea2f096Dc60C4C127",
		CurrentAddress: "0xED816fd7a6e39bce5d2df6A756F3812da06960fC",
		EthereumSignature: &commandspb.Signature{
			Value: "deadbeef",
			Algo:  "vega/ed25519",
		},
	})

	assert.Contains(t, err.Get("ethereum_key_rotate_submission.target_block"), commands.ErrIsRequired)
}

func TestSubmittingNonEmptyEthereumKeyRotateSubmissionCommandSuccess(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(&commandspb.EthereumKeyRotateSubmission{
		TargetBlock:    100,
		NewAddress:     "0xE7d65d1A6CD6eCcfbE78A5Aea2f096Dc60C4C127",
		CurrentAddress: "0xED816fd7a6e39bce5d2df6A756F3812da06960fC",
		EthereumSignature: &commandspb.Signature{
			Value: "deadbeef",
			Algo:  "vega/ed25519",
		},
	})

	assert.True(t, err.Empty())
}

func TestEthereumKeyRotateSubmissionInvalidEthereumAddresses(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(&commandspb.EthereumKeyRotateSubmission{
		NewAddress:       "0xE7d65d1A6CD6eCc",
		CurrentAddress:   "0xED816fd7a6e",
		SubmitterAddress: "0xED816fd7a6e",
		EthereumSignature: &commandspb.Signature{
			Value: "deadbeef",
			Algo:  "vega/ed25519",
		},
	})

	assert.Contains(t, err.Get("ethereum_key_rotate_submission.new_address"), commands.ErrIsNotValidEthereumAddress)
	assert.Contains(t, err.Get("ethereum_key_rotate_submission.current_address"), commands.ErrIsNotValidEthereumAddress)
	assert.Contains(t, err.Get("ethereum_key_rotate_submission.submitter_address"), commands.ErrIsNotValidEthereumAddress)
}

func TestSubmittingNonEmptyEthereumKeyRotateSubmissionWithoutSigFails(t *testing.T) {
	err := checkEthereumKeyRotateSubmission(&commandspb.EthereumKeyRotateSubmission{
		TargetBlock:    100,
		NewAddress:     "0xE7d65d1A6CD6eCcfbE78A5Aea2f096Dc60C4C127",
		CurrentAddress: "0xED816fd7a6e39bce5d2df6A756F3812da06960fC",
	})

	assert.Contains(t, err.Get("ethereum_key_rotate_submission.signature"), commands.ErrIsRequired)
}

func checkEthereumKeyRotateSubmission(cmd *commandspb.EthereumKeyRotateSubmission) commands.Errors {
	err := commands.CheckEthereumKeyRotateSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}
	return e
}
