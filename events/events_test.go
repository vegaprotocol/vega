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
	ctx = vgcontext.WithTxHash(ctx, "textTxHash")
	ctx = vgcontext.WithTraceID(ctx, "testTraceID")
	ctx = vgcontext.WithChainID(ctx, "testChainID")

	accEvent := events.NewAccountEvent(ctx, account)
	busEvent := accEvent.StreamMessage()
	assert.Equal(t, uint32(eventspb.Version), busEvent.Version)

	acc := events.AccountEventFromStream(context.Background(), busEvent)
	assert.Equal(t, "testId", acc.Account().Id)
	assert.Equal(t, "textTxHash", acc.TxHash())
	assert.Equal(t, "testTraceID", acc.TraceID())
	assert.Equal(t, "testChainID", acc.ChainID())

	chainID, _ := vgcontext.ChainIDFromContext(acc.Context())
	assert.Equal(t, "testChainID", chainID)
	_, traceID := vgcontext.TraceIDFromContext(acc.Context())
	assert.Equal(t, "testTraceID", traceID)
	txHash, _ := vgcontext.TxHashFromContext(acc.Context())
	assert.Equal(t, "textTxHash", txHash)
}
