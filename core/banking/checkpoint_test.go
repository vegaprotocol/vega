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
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const assetNameETH = "ETH"

func TestCheckpoint(t *testing.T) {
	t.Run("test simple scheduled transfer", testSimpledScheduledTransfer)
}

func TestDepositFinalisedAfterCheckpoint(t *testing.T) {
	eng := getTestEngine(t)

	eng.tsvc.EXPECT().GetTimeNow().AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	eng.OnTick(context.Background(), time.Now())
	bad := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT",
		PartyID:     "someparty",
		Amount:      num.NewUint(42),
	}

	// call the deposit function
	err := eng.DepositBuiltinAsset(context.Background(), bad, "depositid", 42)
	assert.NoError(t, err)

	// then we call the callback from the fake witness
	eng.witness.r.Check(context.Background())
	eng.witness.f(eng.witness.r, true)

	// now we take a checkpoint
	cp, err := eng.Checkpoint()
	require.NoError(t, err)

	loadEng := getTestEngine(t)

	loadEng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	loadEng.tsvc.EXPECT().GetTimeNow().AnyTimes()
	loadEng.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	// load from checkpoint
	require.NoError(t, loadEng.Load(context.Background(), cp))

	// now finalise the asset action
	// then we call time update, which should call the collateral to
	// to do the deposit
	loadEng.col.EXPECT().Deposit(gomock.Any(), bad.PartyID, bad.VegaAssetID, bad.Amount).Times(1).Return(&types.LedgerMovement{}, nil)
	loadEng.OnTick(context.Background(), time.Now())
}

func testSimpledScheduledTransfer(t *testing.T) {
	e := getTestEngine(t)

	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(10, 0)
		}).AnyTimes()

	// let's do a massive fee, easy to test.
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(12, 0)
	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           assetNameETH,
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			DeliverOn: &deliverOn,
		},
	}

	fromAcc := types.Account{
		Balance: num.NewUint(100),
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
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
				assert.Len(t, transfers, 1)
				assert.Equal(t, transfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, assetNameETH)

				// 2 account types too
				assert.Len(t, accountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, feeTransfers[0].Amount.Asset, assetNameETH)

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(3)
	assert.NoError(t, e.TransferFunds(ctx, transfer))

	checkp, err := e.Checkpoint()
	assert.NoError(t, err)

	// now second step, we start a new engine, and load the checkpoint
	e2 := getTestEngine(t)
	defer e2.ctrl.Finish()

	// load the checkpoint
	e2.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	assert.NoError(t, e2.Load(ctx, checkp))

	// then trigger the time update, and see the transfer
	// assert the calculation of fees and transfer request are correct
	e2.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(12, 0)
		}).AnyTimes()

	e2.broker.EXPECT().Send(gomock.Any()).Times(1)
	e2.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
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
				assert.Equal(t, transfers[0].Owner, "0000000000000000000000000000000000000000000000000000000000000000")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, assetNameETH)

				// 1 account types too
				assert.Len(t, accountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGlobalReward)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 0)
			})
			return nil, nil
		})

	e2.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	e2.OnTick(context.Background(), time.Unix(12, 0))
}

func TestGovernanceScheduledTransfer(t *testing.T) {
	e := getTestEngine(t)
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)

	// let's do a massive fee, easy to test.
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(12, 0).UnixNano()

	ctx := context.Background()
	transfer := &types.NewTransferConfiguration{
		SourceType:              types.AccountTypeGlobalReward,
		DestinationType:         types.AccountTypeGeneral,
		Asset:                   assetNameETH,
		Source:                  "",
		Destination:             "zohar",
		TransferType:            vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
		MaxAmount:               num.NewUint(10),
		FractionOfBalance:       num.DecimalFromFloat(0.1),
		Kind:                    types.TransferKindOneOff,
		OneOffTransferConfig:    &vega.OneOffTransfer{DeliverOn: deliverOn},
		RecurringTransferConfig: nil,
	}

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	e.tsvc.EXPECT().GetTimeNow().Times(1).Return(time.Unix(10, 0))
	require.NoError(t, e.NewGovernanceTransfer(ctx, "1", "some reference", transfer))

	checkp, err := e.Checkpoint()
	assert.NoError(t, err)

	// now second step, we start a new engine, and load the checkpoint
	e2 := getTestEngine(t)
	defer e2.ctrl.Finish()
	e2.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)

	// load the checkpoint
	e2.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	require.NoError(t, e2.Load(ctx, checkp))

	chp2, err := e2.Checkpoint()
	require.NoError(t, err)
	require.True(t, bytes.Equal(chp2, checkp))

	// progress time to when the scheduled gov transfer should be delivered on
	// then trigger the time update, and see the transfer going
	e2.broker.EXPECT().Send(gomock.Any()).Times(2)
	e2.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	e2.col.EXPECT().GetSystemAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).Return(num.NewUint(1000), nil).AnyTimes()
	e2.OnMaxAmountChanged(context.Background(), num.DecimalFromInt64(100000))
	e2.OnMaxFractionChanged(context.Background(), num.DecimalFromFloat(0.5))
	e2.col.EXPECT().GovernanceTransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	e2.OnTick(context.Background(), time.Unix(12, 0))

	// expect the transfer to have been removed from the engine so the checkpoint has changed
	chp3, err := e2.Checkpoint()
	require.NoError(t, err)
	require.False(t, bytes.Equal(chp2, chp3))
}

func TestGovernanceRecurringTransfer(t *testing.T) {
	e := getTestEngine(t)
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)

	ctx := context.Background()
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(ctx, time.Unix(10, 0))
	e.OnEpoch(ctx, types.Epoch{Seq: 0, StartTime: time.Unix(10, 0), Action: vega.EpochAction_EPOCH_ACTION_START})

	endEpoch := uint64(2)

	transfer := &types.NewTransferConfiguration{
		SourceType:              types.AccountTypeGlobalReward,
		DestinationType:         types.AccountTypeGeneral,
		Asset:                   assetNameETH,
		Source:                  "",
		Destination:             "zohar",
		TransferType:            vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
		MaxAmount:               num.NewUint(10),
		FractionOfBalance:       num.DecimalFromFloat(0.1),
		Kind:                    types.TransferKindRecurring,
		OneOffTransferConfig:    nil,
		RecurringTransferConfig: &vega.RecurringTransfer{StartEpoch: 1, EndEpoch: &endEpoch},
	}

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	e.tsvc.EXPECT().GetTimeNow().Times(1).Return(time.Unix(10, 0))
	require.NoError(t, e.NewGovernanceTransfer(ctx, "1", "some reference", transfer))

	checkp, err := e.Checkpoint()
	require.NoError(t, err)

	// now second step, we start a new engine, and load the checkpoint
	e2 := getTestEngine(t)
	defer e2.ctrl.Finish()
	e2.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil).AnyTimes()

	// load the checkpoint
	e2.broker.EXPECT().SendBatch(gomock.Any()).Times(2)
	require.NoError(t, e2.Load(ctx, checkp))

	chp2, err := e2.Checkpoint()
	require.NoError(t, err)
	require.True(t, bytes.Equal(chp2, checkp))

	// now lets end epoch 0 and 1 so that we can get the transfer out
	e2.col.EXPECT().GetSystemAccountBalance(gomock.Any(), gomock.Any(), gomock.Any()).Return(num.NewUint(1000), nil).AnyTimes()
	e2.OnMaxAmountChanged(context.Background(), num.DecimalFromInt64(10000))
	e2.OnMaxFractionChanged(context.Background(), num.DecimalFromFloat(0.5))
	e2.col.EXPECT().GovernanceTransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	e2.OnEpoch(ctx, types.Epoch{Seq: 0, StartTime: time.Unix(10, 0), Action: vega.EpochAction_EPOCH_ACTION_END})
	e2.OnEpoch(ctx, types.Epoch{Seq: 1, StartTime: time.Unix(20, 0), Action: vega.EpochAction_EPOCH_ACTION_START})
	e2.broker.EXPECT().Send(gomock.Any()).Times(3)
	e2.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	e2.OnEpoch(ctx, types.Epoch{Seq: 1, StartTime: time.Unix(20, 0), Action: vega.EpochAction_EPOCH_ACTION_END})

	// now end epoch 2 and expect the second transfer to be delivered and the transfer to be terminated
	e2.col.EXPECT().GovernanceTransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)
	e2.OnEpoch(ctx, types.Epoch{Seq: 2, StartTime: time.Unix(30, 0), Action: vega.EpochAction_EPOCH_ACTION_START})
	e2.OnEpoch(ctx, types.Epoch{Seq: 2, StartTime: time.Unix(30, 0), Action: vega.EpochAction_EPOCH_ACTION_END})

	chp3, err := e2.Checkpoint()
	require.NoError(t, err)
	require.False(t, bytes.Equal(chp2, chp3))
}
