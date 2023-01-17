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

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransferResponseDeepClone(t *testing.T) {
	ctx := context.Background()

	tr := []*types.LedgerMovement{
		{
			Entries: []*types.LedgerEntry{
				{
					FromAccount:        &types.AccountDetails{Owner: "FromAccount"},
					ToAccount:          &types.AccountDetails{Owner: "ToAccount"},
					Amount:             num.NewUint(1000),
					Type:               types.TransferTypeBondLow,
					Timestamp:          2000,
					FromAccountBalance: num.NewUint(3000),
					ToAccountBalance:   num.NewUint(4000),
				},
			},
			Balances: []*types.PostTransferBalance{
				{
					Account: &types.Account{
						ID:       "Id",
						Owner:    "Owner",
						Balance:  num.NewUint(3000),
						Asset:    "Asset",
						MarketID: "MarketId",
						Type:     types.AccountTypeBond,
					},
					Balance: num.NewUint(4000),
				},
			},
		},
	}

	trEvent := events.NewLedgerMovements(ctx, tr)
	tr2 := trEvent.LedgerMovements()

	// Change the original values
	tr[0].Entries[0].Amount = num.NewUint(999)
	tr[0].Entries[0].FromAccount = &types.AccountDetails{Owner: "Changed"}
	tr[0].Entries[0].Timestamp = 999
	tr[0].Entries[0].ToAccount = &types.AccountDetails{Owner: "Changed"}
	tr[0].Entries[0].Type = types.TransferTypeBondHigh
	tr[0].Entries[0].FromAccountBalance = num.NewUint(1000)
	tr[0].Entries[0].ToAccountBalance = num.NewUint(1700)

	tr[0].Balances[0].Account.Asset = "Changed"
	tr[0].Balances[0].Account.Balance = num.NewUint(999)
	tr[0].Balances[0].Account.ID = "Changed"
	tr[0].Balances[0].Account.MarketID = "Changed"
	tr[0].Balances[0].Account.Owner = "Changed"
	tr[0].Balances[0].Account.Type = proto.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	tr[0].Balances[0].Balance = num.NewUint(999)

	// Check things have changed
	assert.NotEqual(t, tr[0].Entries[0].Amount, tr2[0].Entries[0].Amount)
	assert.NotEqual(t, tr[0].Entries[0].FromAccount, tr2[0].Entries[0].FromAccount)
	assert.NotEqual(t, tr[0].Entries[0].Timestamp, tr2[0].Entries[0].Timestamp)
	assert.NotEqual(t, tr[0].Entries[0].ToAccount, tr2[0].Entries[0].ToAccount)
	assert.NotEqual(t, tr[0].Entries[0].Type, tr2[0].Entries[0].Type)
	assert.NotEqual(t, tr[0].Entries[0].FromAccountBalance, tr2[0].Entries[0].FromAccountBalance)
	assert.NotEqual(t, tr[0].Entries[0].ToAccountBalance, tr2[0].Entries[0].ToAccountBalance)

	assert.NotEqual(t, tr[0].Balances[0].Account.Asset, tr2[0].Balances[0].Account.AssetId)
	assert.NotEqual(t, tr[0].Balances[0].Balance, tr2[0].Balances[0].Balance)
	assert.NotEqual(t, tr[0].Balances[0].Account.MarketID, tr2[0].Balances[0].Account.MarketId)
	assert.NotEqual(t, tr[0].Balances[0].Account.Owner, tr2[0].Balances[0].Account.Owner)
	assert.NotEqual(t, tr[0].Balances[0].Account.Type, tr2[0].Balances[0].Account.Type)
	assert.NotEqual(t, tr[0].Balances[0].Balance, tr2[0].Balances[0].Balance)
}

func TestNilOwner(t *testing.T) {
	ctx := context.Background()
	systemOwner := "*"
	tr1 := []*types.LedgerMovement{
		{
			Entries: []*types.LedgerEntry{
				{
					FromAccount:        &types.AccountDetails{Owner: "FromZohar"},
					ToAccount:          &types.AccountDetails{Owner: "ToZohar"},
					Amount:             num.NewUint(1000),
					Type:               types.TransferTypeBondLow,
					Timestamp:          2000,
					FromAccountBalance: num.NewUint(2000),
					ToAccountBalance:   num.NewUint(1400),
				},
			},
		},
	}
	trNilFrom := []*types.LedgerMovement{
		{
			Entries: []*types.LedgerEntry{
				{
					FromAccount:        &types.AccountDetails{Owner: systemOwner},
					ToAccount:          &types.AccountDetails{Owner: "ToZohar"},
					Amount:             num.NewUint(1000),
					Type:               types.TransferTypeBondLow,
					Timestamp:          2000,
					FromAccountBalance: num.NewUint(100),
					ToAccountBalance:   num.NewUint(3800),
				},
			},
		},
	}
	trNilTo := []*types.LedgerMovement{
		{
			Entries: []*types.LedgerEntry{
				{
					FromAccount:        &types.AccountDetails{Owner: "FromZohar"},
					ToAccount:          &types.AccountDetails{Owner: systemOwner},
					Amount:             num.NewUint(1000),
					Type:               types.TransferTypeBondLow,
					Timestamp:          2000,
					FromAccountBalance: num.NewUint(500),
					ToAccountBalance:   num.NewUint(2300),
				},
			},
		},
	}

	trNilFromTo := []*types.LedgerMovement{
		{
			Entries: []*types.LedgerEntry{
				{
					FromAccount:        &types.AccountDetails{Owner: systemOwner},
					ToAccount:          &types.AccountDetails{Owner: systemOwner},
					Amount:             num.NewUint(1000),
					Type:               types.TransferTypeBondLow,
					Timestamp:          2000,
					FromAccountBalance: num.NewUint(1000),
					ToAccountBalance:   num.NewUint(2900),
				},
			},
		},
	}

	require.True(t, events.NewLedgerMovements(ctx, tr1).IsParty("FromZohar"))
	require.True(t, events.NewLedgerMovements(ctx, tr1).IsParty("ToZohar"))
	require.True(t, events.NewLedgerMovements(ctx, trNilTo).IsParty("FromZohar"))
	require.True(t, events.NewLedgerMovements(ctx, trNilFrom).IsParty("ToZohar"))
	require.False(t, events.NewLedgerMovements(ctx, trNilFromTo).IsParty("FromZohar"))
	require.False(t, events.NewLedgerMovements(ctx, trNilFromTo).IsParty("ToZohar"))
	require.True(t, events.TransferResponseEventFromStream(ctx, events.NewLedgerMovements(ctx, tr1).StreamMessage()).IsParty("FromZohar"))
	require.True(t, events.TransferResponseEventFromStream(ctx, events.NewLedgerMovements(ctx, tr1).StreamMessage()).IsParty("ToZohar"))
	require.True(t, events.TransferResponseEventFromStream(ctx, events.NewLedgerMovements(ctx, trNilTo).StreamMessage()).IsParty("FromZohar"))
	require.True(t, events.TransferResponseEventFromStream(ctx, events.NewLedgerMovements(ctx, trNilFrom).StreamMessage()).IsParty("ToZohar"))
	require.False(t, events.TransferResponseEventFromStream(ctx, events.NewLedgerMovements(ctx, trNilFromTo).StreamMessage()).IsParty("FromZohar"))
	require.False(t, events.TransferResponseEventFromStream(ctx, events.NewLedgerMovements(ctx, trNilFromTo).StreamMessage()).IsParty("ToZohar"))
}
