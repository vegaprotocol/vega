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
	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/protos/vega"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestNilStateVarProposalFundsFails(t *testing.T) {
	err := checkStateVarProposal(nil)
	assert.Contains(t, err.Get("state_variable_proposal"), commands.ErrIsRequired)
}

func TestStateVarProposals(t *testing.T) {
	cases := []struct {
		stateVar  commandspb.StateVariableProposal
		errString string
	}{
		{
			stateVar: commandspb.StateVariableProposal{
				Proposal: &vega.StateValueProposal{
					StateVarId: vgcrypto.RandomHash(),
					EventId:    "",
					Kvb: []*vega.KeyValueBundle{
						{
							Key:       vgcrypto.RandomHash(),
							Tolerance: "11000",
							Value:     &vega.StateVarValue{},
						},
					},
				},
			},
			errString: "state_variable_proposal.event_id (is required)",
		},
		{
			stateVar: commandspb.StateVariableProposal{
				Proposal: &vega.StateValueProposal{
					StateVarId: "",
					EventId:    vgcrypto.RandomHash(),
					Kvb: []*vega.KeyValueBundle{
						{
							Key:       vgcrypto.RandomHash(),
							Tolerance: "11000",
							Value:     &vega.StateVarValue{},
						},
					},
				},
			},
			errString: "state_variable_proposal.state_var_id (is required)",
		},
		{
			stateVar: commandspb.StateVariableProposal{
				Proposal: &vega.StateValueProposal{
					StateVarId: "",
					EventId:    vgcrypto.RandomHash(),
					Kvb: []*vega.KeyValueBundle{
						{
							Key:       vgcrypto.RandomHash(),
							Tolerance: "11000",
							Value:     nil,
						},
					},
				},
			},
			errString: "state_variable_proposal.key_value_bundle.0.value (is required)",
		},
		{
			stateVar: commandspb.StateVariableProposal{
				Proposal: &vega.StateValueProposal{
					StateVarId: vgcrypto.RandomHash(),
					EventId:    vgcrypto.RandomHash(),
					Kvb: []*vega.KeyValueBundle{
						{
							Key:       vgcrypto.RandomHash(),
							Tolerance: "11000",
							Value:     &vega.StateVarValue{},
						},
					},
				},
			},
			errString: "",
		},
	}

	for _, c := range cases {
		err := commands.CheckStateVariableProposal(&c.stateVar)
		if len(c.errString) <= 0 {
			assert.NoError(t, err)
			continue
		}
		assert.Contains(t, err.Error(), c.errString)
	}
}

func checkStateVarProposal(cmd *commandspb.StateVariableProposal) commands.Errors {
	err := commands.CheckStateVariableProposal(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
