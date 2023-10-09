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

func TestCheckNodeVote(t *testing.T) {
	t.Run("Submitting a nil command fails", testNilNodeVoteFails)
	t.Run("Submitting a node vote without reference fails", testNodeVoteWithoutReferenceFails)
	t.Run("Submitting a node vote with reference succeeds", testNodeVoteWithReferenceSucceeds)
}

func testNilNodeVoteFails(t *testing.T) {
	err := checkNodeVote(nil)

	assert.Error(t, err)
}

func testNodeVoteWithoutReferenceFails(t *testing.T) {
	err := checkNodeVote(&commandspb.NodeVote{})
	assert.Contains(t, err.Get("node_vote.reference"), commands.ErrIsRequired)
}

func testNodeVoteWithReferenceSucceeds(t *testing.T) {
	err := checkNodeVote(&commandspb.NodeVote{
		Reference: "my ref",
	})
	assert.NotContains(t, err.Get("node_vote.reference"), commands.ErrIsRequired)
}

func checkNodeVote(cmd *commandspb.NodeVote) commands.Errors {
	err := commands.CheckNodeVote(cmd)

	var e commands.Errors
	if ok := errors.As(err, &e); !ok {
		return commands.NewErrors()
	}

	return e
}
