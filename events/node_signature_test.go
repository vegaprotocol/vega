package events_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/events"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

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
