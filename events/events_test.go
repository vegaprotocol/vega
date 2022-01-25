package events_test

import (
	"context"
	"testing"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/types"
)

func TestEventCtxIsSet(t *testing.T) {
	account := types.Account{
		ID:       "testId",
		Owner:    "testOwner",
		Balance:  num.NewUint(10),
		Asset:    "testAsset",
		MarketID: "testMarket",
		Type:     0,
	}

	ctx := context.Background()
	ctx = vgcontext.WithTranxID(ctx, "testTranxID")
	ctx = vgcontext.WithTraceID(ctx, "testTraceID")
	ctx = vgcontext.WithChainID(ctx, "testChainID")

	accEvent := events.NewAccountEvent(ctx, account)
	busEvent := accEvent.StreamMessage()
	assert.Equal(t, uint32(eventspb.Version), busEvent.Version)

	acc := events.AccountEventFromStream(context.Background(), busEvent)
	assert.Equal(t, "testId", acc.Account().Id)
	assert.Equal(t, "testTranxID", acc.TranxID())
	assert.Equal(t, "testTraceID", acc.TraceID())
	assert.Equal(t, "testChainID", acc.ChainID())

	chainID, _ := vgcontext.ChainIDFromContext(acc.Context())
	assert.Equal(t, "testChainID", chainID)
	traceID, _ := vgcontext.TraceIDFromContext(acc.Context())
	assert.Equal(t, "testTraceID", traceID)
	tranxID, _ := vgcontext.TranxIDFromContext(acc.Context())
	assert.Equal(t, "testTranxID", tranxID)
}
