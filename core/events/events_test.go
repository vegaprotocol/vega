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
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

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
