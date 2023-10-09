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

func TestSubmittingNoKeyRotateSubmissionCommandFails(t *testing.T) {
	err := checkKeyRotateSubmission(nil)

	assert.Contains(t, err.Get("key_rotate_submission"), commands.ErrIsRequired)
}

func TestKeyRotateSubmissionSubmittingEmptyCommandFails(t *testing.T) {
	err := checkKeyRotateSubmission(&commandspb.KeyRotateSubmission{})

	assert.Contains(t, err.Get("key_rotate_submission.new_pub_key"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("key_rotate_submission.new_pub_key_index"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("key_rotate_submission.current_pub_key_hash"), commands.ErrIsRequired)
	assert.Contains(t, err.Get("key_rotate_submission.target_block"), commands.ErrIsRequired)
}

func TestKeyRotateSubmissionMissingNewPubKeyFails(t *testing.T) {
	err := checkKeyRotateSubmission(&commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    10,
		TargetBlock:       100,
		CurrentPubKeyHash: "w3werertdg",
	})

	assert.Contains(t, err.Get("key_rotate_submission.new_pub_key"), commands.ErrIsRequired)
}

func TestKeyRotateSubmissionMissingNewPubKeyIndexFails(t *testing.T) {
	err := checkKeyRotateSubmission(&commandspb.KeyRotateSubmission{
		NewPubKey:         "123456789abcdef",
		TargetBlock:       100,
		CurrentPubKeyHash: "w3werertdg",
	})

	assert.Contains(t, err.Get("key_rotate_submission.new_pub_key_index"), commands.ErrIsRequired)
}

func TestKeyRotateSubmissionMissingCurrentPubKeyHashFails(t *testing.T) {
	err := checkKeyRotateSubmission(&commandspb.KeyRotateSubmission{
		NewPubKey:      "123456789abcdef",
		NewPubKeyIndex: 10,
		TargetBlock:    100,
	})

	assert.Contains(t, err.Get("key_rotate_submission.current_pub_key_hash"), commands.ErrIsRequired)
}

func TestKeyRotateSubmissionMissingTargetBlockFails(t *testing.T) {
	err := checkKeyRotateSubmission(&commandspb.KeyRotateSubmission{
		NewPubKey:         "123456789abcdef",
		NewPubKeyIndex:    10,
		CurrentPubKeyHash: "w3werertdg",
	})

	assert.Contains(t, err.Get("key_rotate_submission.target_block"), commands.ErrIsRequired)
}

func TestSubmittingEmptyCommandSuccess(t *testing.T) {
	err := checkKeyRotateSubmission(&commandspb.KeyRotateSubmission{
		NewPubKeyIndex:    10,
		NewPubKey:         "123456789abcdef",
		TargetBlock:       100,
		CurrentPubKeyHash: "w3werertdg",
	})

	assert.True(t, err.Empty())
}

func checkKeyRotateSubmission(cmd *commandspb.KeyRotateSubmission) commands.Errors {
	err := commands.CheckKeyRotateSubmission(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}
	return e
}
