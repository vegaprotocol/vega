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

package commands_test

import (
	"errors"
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
			errString: "validator_heartbeat.node_id (is required)",
		},
		{
			vh: commandspb.ValidatorHeartbeat{
				NodeId: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
				VegaSignature: &commandspb.Signature{
					Value: "84e2b15102a8d6c1c6b4bdf40af8a0dc21b040eaaa1c94cd10d17604b75fdc35",
					Algo:  "some/algo",
				},
			},
			errString: "validator_heartbeat.ethereum_signature.value (is required)",
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

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
