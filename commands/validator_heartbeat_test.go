package commands_test

import (
	"testing"

	"code.vegaprotocol.io/vega/commands"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"
	"github.com/stretchr/testify/assert"
)

func TestNilValidatorHeartbeatFails(t *testing.T) {
	err := checkValidatorHeartbeat(nil)

	assert.Contains(t, err.Get("validator_heartbeat"), commands.ErrIsRequired)
}

func TestValidatorHeartbeat(t *testing.T) {
	cases := []struct {
		vh        commandspb.ValidatorHeartbeat
		errString string
	}{
		{
			vh: commandspb.ValidatorHeartbeat{
				NodeId: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				EthereumSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				},
				VegaSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
					Algo:  "some/algo",
				},
			},
		},
		{
			vh: commandspb.ValidatorHeartbeat{
				NodeId: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				EthereumSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				},
				VegaSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				},
			},
			errString: "validator_heartbeat.vega_signature.algo (is required)",
		},
		{
			vh: commandspb.ValidatorHeartbeat{
				NodeId: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				EthereumSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				},
				VegaSignature: &commandspb.Signature{
					Algo: "some/algo",
				},
			},
			errString: "validator_heartbeat.vega_signature.value (is required)",
		},
		{
			vh: commandspb.ValidatorHeartbeat{
				NodeId:            "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				EthereumSignature: &commandspb.Signature{},
				VegaSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
					Algo:  "some/algo",
				},
			},
			errString: "validator_heartbeat.ethereum_signature.value (is required)",
		},
		{
			vh: commandspb.ValidatorHeartbeat{
				EthereumSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				},
				VegaSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
					Algo:  "some/algo",
				},
			},
			errString: "validator_heartbeat.vega_pub_key (is required)",
		},
	}

	for _, c := range cases {
		err := commands.CheckValidatorHeartbeat(&c.vh)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.EqualError(t, err, c.errString)
	}
}

func checkValidatorHeartbeat(cmd *commandspb.ValidatorHeartbeat) commands.Errors {
	err := commands.CheckValidatorHeartbeat(cmd)

	e, ok := err.(commands.Errors)
	if !ok {
		return commands.NewErrors()
	}

	return e
}
