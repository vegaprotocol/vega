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
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
)

func TestDeposits(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	server := NewTestServer(t, ctx, true)
	defer server.ctrl.Finish()

	PublishEvents(t, ctx, server.broker, func(be *eventspb.BusEvent) (events.Event, error) {
		deposit := be.GetDeposit()
		require.NotNil(t, deposit)
		amt, _ := num.UintFromString(deposit.Amount, 10)
		e := events.NewDepositEvent(ctx, types.Deposit{
			ID:           deposit.Id,
			Status:       deposit.Status,
			PartyID:      deposit.PartyId,
			Asset:        deposit.Asset,
			Amount:       amt,
			TxHash:       deposit.TxHash,
			CreditDate:   deposit.CreditedTimestamp,
			CreationDate: deposit.CreatedTimestamp,
		})
		return e, nil
	}, "deposit-events.golden")

	client := apipb.NewTradingDataServiceClient(server.clientConn)
	require.NotNil(t, client)

	depositID := "af6e66ee1e1a643338f55b8dfe00129b09b926a997edddf1f10e76b31c65cdad"
	depositPartyID := "c5fdc709b3464ca10292437ce493dc0e497b2c3ea22a5fde714c4e487b93011d"

	var resp *apipb.DepositsResponse
	var err error

loop:
	for {
		select {
		case <-ctx.Done():
			t.Fatalf("test timeout")
		case <-time.Tick(50 * time.Millisecond):
			resp, err = client.Deposits(ctx, &apipb.DepositsRequest{
				PartyId: depositPartyID,
			})
			require.NotNil(t, resp)
			require.NoError(t, err)
			if len(resp.Deposits) > 0 {
				break loop
			}
		}
	}

	assert.NoError(t, err)
	assert.Equal(t, depositID, resp.Deposits[0].Id)
	assert.Equal(t, depositPartyID, resp.Deposits[0].PartyId)
}
