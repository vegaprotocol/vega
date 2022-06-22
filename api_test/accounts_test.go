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

package api_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	pb "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func TestGetPartyAccounts(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		acc := be.GetAccount()
		require.NotNil(t, acc)
		balance, _ := num.UintFromString(acc.Balance, 10)
		e := events.NewAccountEvent(ctx, types.Account{
			ID:       acc.Id,
			Owner:    acc.Owner,
			Balance:  balance,
			Asset:    acc.Asset,
			MarketID: acc.MarketId,
			Type:     acc.Type,
		})

		return e, nil
	}, "account-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	partyID := "6fb72005cde8e239f8d3b08c5fbcec06f93bfb45e9013208f662954923343fba"

	var resp *apipb.PartyAccountsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.PartyAccounts(ctx, &apipb.PartyAccountsRequest{
				PartyId: partyID,
				Type:    pb.AccountType_ACCOUNT_TYPE_GENERAL,
			})
			if err == nil && len(resp.Accounts) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Len(t, resp.Accounts, 1)
	assert.Equal(t, partyID, resp.Accounts[0].Owner)
}
