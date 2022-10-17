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

	"github.com/stretchr/testify/assert"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	proto "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

func TestTransferResponseDeepClone(t *testing.T) {
	ctx := context.Background()

	tr := []*types.LedgerMovement{
		{
			Entries: []*types.LedgerEntry{
				{
					FromAccount: &types.AccountDetails{Owner: "FromAccount"},
					ToAccount:   &types.AccountDetails{Owner: "ToAccount"},
					Amount:      num.NewUint(1000),
					Type:        types.TransferTypeBondLow,
					Timestamp:   2000,
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
	assert.NotEqual(t, tr[0].Balances[0].Account.Asset, tr2[0].Balances[0].Account.AssetId)
	assert.NotEqual(t, tr[0].Balances[0].Balance, tr2[0].Balances[0].Balance)
	assert.NotEqual(t, tr[0].Balances[0].Account.MarketID, tr2[0].Balances[0].Account.MarketId)
	assert.NotEqual(t, tr[0].Balances[0].Account.Owner, tr2[0].Balances[0].Account.Owner)
	assert.NotEqual(t, tr[0].Balances[0].Account.Type, tr2[0].Balances[0].Account.Type)
	assert.NotEqual(t, tr[0].Balances[0].Balance, tr2[0].Balances[0].Balance)
}

func TestLedgerMovements_IsParty(t1 *testing.T) {
	type fields struct {
		ledgerMovements []*proto.LedgerMovement
	}
	type args struct {
		id string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name: "is party",
			fields: fields{
				ledgerMovements: []*proto.LedgerMovement{
					{
						Entries: []*proto.LedgerEntry{
							{
								FromAccount: &proto.AccountDetails{Owner: toPtr("party")},
								ToAccount:   &proto.AccountDetails{Owner: toPtr("dest")},
								Timestamp:   2000,
								Amount:      "1000",
								Type:        types.TransferTypeBondLow,
							},
						},
					},
				},
			},
			args: args{
				id: "party",
			},
			want: true,
		}, {
			name: "is not party",
			fields: fields{
				ledgerMovements: []*proto.LedgerMovement{
					{
						Entries: []*proto.LedgerEntry{
							{
								FromAccount: &proto.AccountDetails{Owner: toPtr("not-party")},
								ToAccount:   &proto.AccountDetails{Owner: toPtr("dest")},
								Amount:      "1000",
								Type:        types.TransferTypeBondLow,
								Timestamp:   2000,
							},
						},
					},
				},
			},
			args: args{
				id: "party",
			},
			want: false,
		}, {
			name: "is not party: missing FromAccount owner",
			fields: fields{
				ledgerMovements: []*proto.LedgerMovement{
					{
						Entries: []*proto.LedgerEntry{
							{
								FromAccount: &proto.AccountDetails{Owner: nil},
								ToAccount:   &proto.AccountDetails{Owner: toPtr("dest")},
								Amount:      "1000",
								Type:        types.TransferTypeBondLow,
								Timestamp:   2000,
							},
						},
					},
				},
			},
			args: args{
				id: "party",
			},
			want: false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			be := &v1.BusEvent{
				Id: "id-1",
				Event: &v1.BusEvent_LedgerMovements{
					LedgerMovements: &v1.LedgerMovements{
						LedgerMovements: tt.fields.ledgerMovements,
					},
				},
			}
			t := events.TransferResponseEventFromStream(context.Background(), be)
			assert.Equalf(t1, tt.want, t.IsParty(tt.args.id), "IsParty(%v)", tt.args.id)
		})
	}
}

func toPtr(s string) *string {
	return &s
}
