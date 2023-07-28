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
