package events_test

import (
	"context"
	"testing"

	proto "code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/assert"
)

func TestTransferResponseDeepClone(t *testing.T) {
	ctx := context.Background()

	tr := []*types.TransferResponse{
		{
			Transfers: []*types.LedgerEntry{
				{
					FromAccount: "FromAccount",
					ToAccount:   "ToAccount",
					Amount:      num.NewUint(1000),
					Reference:   "Reference",
					Type:        "Type",
					Timestamp:   2000,
				},
			},
			Balances: []*types.TransferBalance{
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

	trEvent := events.NewTransferResponse(ctx, tr)
	tr2 := trEvent.TransferResponses()

	// Change the original values
	tr[0].Transfers[0].Amount = num.NewUint(999)
	tr[0].Transfers[0].FromAccount = "Changed"
	tr[0].Transfers[0].Reference = "Changed"
	tr[0].Transfers[0].Timestamp = 999
	tr[0].Transfers[0].ToAccount = "Changed"
	tr[0].Transfers[0].Type = "Changed"
	tr[0].Balances[0].Account.Asset = "Changed"
	tr[0].Balances[0].Account.Balance = num.NewUint(999)
	tr[0].Balances[0].Account.ID = "Changed"
	tr[0].Balances[0].Account.MarketID = "Changed"
	tr[0].Balances[0].Account.Owner = "Changed"
	tr[0].Balances[0].Account.Type = proto.AccountType_ACCOUNT_TYPE_UNSPECIFIED
	tr[0].Balances[0].Balance = num.NewUint(999)

	// Check things have changed
	assert.NotEqual(t, tr[0].Transfers[0].Amount, tr2[0].Transfers[0].Amount)
	assert.NotEqual(t, tr[0].Transfers[0].FromAccount, tr2[0].Transfers[0].FromAccount)
	assert.NotEqual(t, tr[0].Transfers[0].Reference, tr2[0].Transfers[0].Reference)
	assert.NotEqual(t, tr[0].Transfers[0].Timestamp, tr2[0].Transfers[0].Timestamp)
	assert.NotEqual(t, tr[0].Transfers[0].ToAccount, tr2[0].Transfers[0].ToAccount)
	assert.NotEqual(t, tr[0].Transfers[0].Type, tr2[0].Transfers[0].Type)
	assert.NotEqual(t, tr[0].Balances[0].Account.Asset, tr2[0].Balances[0].Account.Asset)
	assert.NotEqual(t, tr[0].Balances[0].Account.Balance, tr2[0].Balances[0].Account.Balance)
	assert.NotEqual(t, tr[0].Balances[0].Account.ID, tr2[0].Balances[0].Account.Id)
	assert.NotEqual(t, tr[0].Balances[0].Account.MarketID, tr2[0].Balances[0].Account.MarketId)
	assert.NotEqual(t, tr[0].Balances[0].Account.Owner, tr2[0].Balances[0].Account.Owner)
	assert.NotEqual(t, tr[0].Balances[0].Account.Type, tr2[0].Balances[0].Account.Type)
	assert.NotEqual(t, tr[0].Balances[0].Balance, tr2[0].Balances[0].Balance)
}
