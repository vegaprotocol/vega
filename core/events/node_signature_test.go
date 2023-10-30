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

package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/events"
	commandspb "code.vegaprotocol.io/vega/protos/vega/commands/v1"

	"github.com/stretchr/testify/assert"
)

func TestNodeSignatureDeepClone(t *testing.T) {
	ctx := context.Background()

	ns := &commandspb.NodeSignature{
		Id:   "Id",
		Kind: commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_ASSET_NEW,
		Sig:  []byte{'A', 'B', 'C'},
	}

	nsEvent := events.NewNodeSignatureEvent(ctx, *ns)
	ns2 := nsEvent.NodeSignature()

	// Change the original values
	ns.Id = "Changed"
	ns.Kind = commandspb.NodeSignatureKind_NODE_SIGNATURE_KIND_UNSPECIFIED
	ns.Sig[0] = 'X'
	ns.Sig[1] = 'Y'
	ns.Sig[2] = 'Z'

	// Check things have changed
	assert.NotEqual(t, ns.Id, ns2.Id)
	assert.NotEqual(t, ns.Kind, ns2.Kind)
	for i := 0; i < len(ns.Sig); i++ {
		assert.NotEqual(t, ns.Sig[i], ns2.Sig[i])
	}
}
