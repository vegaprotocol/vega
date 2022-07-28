// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package events_test

import (
	"context"
	"testing"

	commandspb "code.vegaprotocol.io/protos/vega/commands/v1"
	"code.vegaprotocol.io/vega/core/events"

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
