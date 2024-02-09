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

package banking_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/common"
	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCancelTransfer(t *testing.T) {
	e := getTestEngine(t)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_END})

	var endEpoch13 uint64 = 11
	transferID := "TRANSFERID"
	partyID := "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301"
	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:   transferID,
				From: partyID,

				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(100),
				Reference:       "someref",
			},
			StartEpoch: 10,
			EndEpoch:   &endEpoch13,
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	asset := "ETH"

	e.assets.EXPECT().Get(gomock.Any()).Times(2).Return(
		assets.NewAsset(&mockAsset{quantum: num.DecimalFromFloat(100), name: asset}), nil)
	e.tsvc.EXPECT().GetTimeNow().Times(2)
	e.broker.EXPECT().Send(gomock.Any()).Times(2)
	assert.NoError(t, e.TransferFunds(ctx, transfer))

	// now we try to cancel an non-existing transfer
	assert.EqualError(t,
		e.CancelTransferFunds(ctx, &types.CancelTransferFunds{TransferID: "NOPE"}),
		banking.ErrRecurringTransferDoesNotExists.Error(),
	)

	// now we try to cancel the right transfer, but with the wrong party
	assert.EqualError(t,
		e.CancelTransferFunds(ctx, &types.CancelTransferFunds{
			TransferID: transferID,
			Party:      "NOPE",
		}),
		banking.ErrCannotCancelOtherPartiesRecurringTransfers.Error(),
	)

	// now we move in time just a bit so we get some transfer processed, but then cancel before
	// then end of the transfer
	fromAcc := types.Account{
		Balance: num.NewUint(1000),
		Asset:   asset,
	}

	// asset exists
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

	// assert the calculation of fees and transfer request are correct
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context,
			transfers []*types.Transfer,
			accountTypes []types.AccountType,
			references []string,
			feeTransfers []*types.Transfer,
			feeTransfersAccountTypes []types.AccountType,
		) ([]*types.LedgerMovement, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Len(t, transfers, 2)
				assert.Equal(t, transfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(100))
				assert.Equal(t, transfers[0].Amount.Asset, asset)

				// 1 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(50))
				assert.Equal(t, feeTransfers[0].Amount.Asset, asset)

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			return nil, nil
		})

	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_END})

	// now we cancel it, we should get no error and and event
	e.broker.EXPECT().Send(gomock.Any()).DoAndReturn(func(evt events.Event) {
		t.Run("ensure transfer is done", func(t *testing.T) {
			e, ok := evt.(*events.TransferFunds)
			assert.True(t, ok, "unexpected event from the bus")
			assert.Equal(t, e.Proto().Status, types.TransferStatusCancelled)
			assert.Equal(t, "transfer cancelled", *e.Proto().Reason)
		})
	})

	key := (&types.PayloadBankingRecurringTransfers{}).Key()
	_, _, err := e.GetState(key)
	require.NoError(t, err)

	assert.NoError(t,
		e.CancelTransferFunds(ctx, &types.CancelTransferFunds{
			TransferID: transferID,
			Party:      partyID,
		}),
	)
	// now we move in time, the recurring transfer was suppose to go
	// 'til epoch 11, but it's not cancelled, and nothing should happen
	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_END})
}

type mockAsset struct {
	quantum num.Decimal
	name    string
}

func (m *mockAsset) Type() *types.Asset {
	return &types.Asset{
		ID: m.name,
		Details: &types.AssetDetails{
			Name:    m.name,
			Symbol:  m.name,
			Quantum: m.quantum,
		},
	}
}

func (m *mockAsset) SetPendingListing() {}
func (m *mockAsset) SetRejected()       {}
func (m *mockAsset) SetEnabled()        {}

func (m *mockAsset) GetAssetClass() common.AssetClass { return common.ERC20 }
func (m *mockAsset) IsValid() bool                    { return true }
func (m *mockAsset) SetValid()                        {}
func (m *mockAsset) String() string                   { return "" }
