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
