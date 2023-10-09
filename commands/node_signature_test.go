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

func TestCheckNodeSignature(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilNodeSignatureFails)
	t.Run("Submitting a node signature without id fails", testNodeSignatureWithoutIDFails)
	t.Run("Submitting a node signature with id succeeds", testNodeSignatureWithIDSucceeds)
	t.Run("Submitting a node signature without sig fails", testNodeSignatureWithoutSigFails)
	t.Run("Submitting a node signature with sig succeeds", testNodeSignatureWithSigSucceeds)
	t.Run("Submitting a node signature without kind fails", testNodeSignatureWithoutKindFails)
	t.Run("Submitting a node signature with invalid kind fails", testNodeSignatureWithInvalidKindFails)
	t.Run("Submitting a node signature with kind succeeds", testNodeSignatureWithKindSucceeds)
}

func testNilNodeSignatureFails(t *testing.T) {
	err := checkNodeSignature(nil)

	assert.Error(t, err)
}

func testNodeSignatureWithoutIDFails(t *testing.T) {
	err := checkNodeSignature(&commandspb.NodeSignature{})
	assert.Contains(t, err.Get("node_signature.id"), commands.ErrIsRequired)
}

func testNodeSignatureWithIDSucceeds(t *testing.T) {
	err := checkNodeSignature(&commandspb.NodeSignature{
		Id: "My ID",
	})
	assert.NotContains(t, err.Get("node_signature.id"), commands.ErrIsRequired)
}

func testNodeSignatureWithoutSigFails(t *testing.T) {
	err := checkNodeSignature(&commandspb.NodeSignature{})
	assert.Contains(t, err.Get("node_signature.sig"), commands.ErrIsRequired)
}

func testNodeSignatureWithSigSucceeds(t *testing.T) {
	err := checkNodeSignature(&commandspb.NodeSignature{
		Sig: []byte("0xDEADBEEF"),
	})
	assert.NotContains(t, err.Get("node_signature.sig"), commands.ErrIsRequired)
}

func testNodeSignatureWithoutKindFails(t *testing.T) {
	err := checkNodeSignature(&commandspb.NodeSignature{})
	assert.Contains(t, err.Get("node_signature.kind"), commands.ErrIsRequired)
}

func testNodeSignatureWithInvalidKindFails(t *testing.T) {
	err := checkNodeSignature(&commandspb.NodeSignature{
		Kind: commandspb.NodeSignatureKind(-42),
	})
	assert.Contains(t, err.Get("node_signature.kind"), commands.ErrIsNotValid)
}

func testNodeSignatureWithKindSucceeds(t *testing.T) {
	testCases := []struct {
		msg   string
		value commandspb.NodeSignatureKind
	}{
		{
			msg:   "with new kind",
			value: commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW,
		}, {
			msg:   "with withdrawal kind",
			value: commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_WITHDRAWAL,
		},
	}
	for _, tc := range testCases {
		t.Run(tc.msg, func(t *testing.T) {
			err := checkNodeSignature(&commandspb.NodeSignature{
				Kind: tc.value,
			})
			assert.NotContains(t, err.Get("node_signature.kind"), commands.ErrIsRequired)
			assert.NotContains(t, err.Get("node_signature.kind"), commands.ErrIsNotValid)
		})
	}
}

func checkNodeSignature(cmd *commandspb.NodeSignature) commands.Errors {
	err := commands.CheckNodeSignature(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
