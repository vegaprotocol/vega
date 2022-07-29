// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.VEGA file and at https://www.mariadb.com/bsl11.
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

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types/num"
	vgcontext "code.vegaprotocol.io/vega/libs/context"

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/core/types"
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
	ctx = vgcontext.WithTxHash(ctx, "testTxHash")
	ctx = vgcontext.WithTraceID(ctx, "testTraceID")
	ctx = vgcontext.WithChainID(ctx, "testChainID")

	accEvent := events.NewAccountEvent(ctx, account)
	busEvent := accEvent.StreamMessage()
	assert.Equal(t, uint32(eventspb.Version), busEvent.Version)

	acc := events.AccountEventFromStream(context.Background(), busEvent)
	assert.Equal(t, "testId", acc.Account().Id)
	assert.Equal(t, "TESTTXHASH", acc.TxHash())
	assert.Equal(t, "TESTTRACEID", acc.TraceID())
	assert.Equal(t, "testChainID", acc.ChainID())

	chainID, _ := vgcontext.ChainIDFromContext(acc.Context())
	assert.Equal(t, "testChainID", chainID)
	_, traceID := vgcontext.TraceIDFromContext(acc.Context())
	assert.Equal(t, "TESTTRACEID", traceID)
	txHash, _ := vgcontext.TxHashFromContext(acc.Context())
	assert.Equal(t, "TESTTXHASH", txHash)
}
