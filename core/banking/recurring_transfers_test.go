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

package banking_test

import (
	"context"
	"fmt"
	"testing"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/protos/vega"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecurringTransfers(t *testing.T) {
	t.Run("recurring invalid transfers", testRecurringTransferInvalidTransfers)
	t.Run("valid recurring transfers", testValidRecurringTransfer)
	t.Run("valid forever transfers, cancelled not enough funds", testForeverTransferCancelledNotEnoughFunds)
	t.Run("invalid recurring transfers, duplicates", testInvalidRecurringTransfersDuplicates)
	t.Run("invalid recurring transfers, bad amount", testInvalidRecurringTransfersBadAmount)
	t.Run("invalid recurring transfers, in the past", testInvalidRecurringTransfersInThePast)
}

func TestMaturation(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()
	ctx := context.Background()

	e.OnMinTransferQuantumMultiple(context.Background(), num.DecimalFromFloat(1))
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(10)}), nil)
	e.tsvc.EXPECT().GetTimeNow().AnyTimes()
	e.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	fromAcc := types.Account{
		Balance: num.NewUint(100000), // enough for the all
	}
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)

	endEpoch := uint64(12)
	transfers := []*types.TransferFunds{}
	for i := 0; i < 10; i++ {
		transfers = append(transfers, &types.TransferFunds{
			Kind: types.TransferCommandKindRecurring,
			Recurring: &types.RecurringTransfer{
				TransferBase: &types.TransferBase{
					ID:              fmt.Sprintf("TRANSFERID-%d", i),
					From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					FromAccountType: types.AccountTypeGeneral,
					To:              crypto.RandomHash(),
					ToAccountType:   types.AccountTypeGeneral,
					Asset:           "eth",
					Amount:          num.NewUint(10),
					Reference:       "someref",
				},
				StartEpoch: 10,
				EndEpoch:   &endEpoch,
				Factor:     num.MustDecimalFromString("1"),
			},
		})
		require.NoError(t, e.TransferFunds(ctx, transfers[i]))
	}
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, nil).AnyTimes()

	seenEvts := []events.Event{}
	e.broker.EXPECT().SendBatch(gomock.Any()).DoAndReturn(func(evts []events.Event) {
		seenEvts = append(seenEvts, evts...)
	}).AnyTimes()
	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_END})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_END})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 12, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 12, Action: vega.EpochAction_EPOCH_ACTION_END})

	require.Equal(t, 10, len(seenEvts))
	stoppedIDs := map[string]struct{}{}
	for _, e2 := range seenEvts {
		if e2.StreamMessage().GetTransfer().Status == types.TransferStatusDone {
			stoppedIDs[e2.StreamMessage().GetTransfer().Id] = struct{}{}
		}
	}
	require.Equal(t, 10, len(stoppedIDs))
}

func testInvalidRecurringTransfersBadAmount(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			StartEpoch: 10,
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	e.OnMinTransferQuantumMultiple(context.Background(), num.DecimalFromFloat(1))
	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	e.tsvc.EXPECT().GetTimeNow().Times(1)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)

	assert.EqualError(t,
		e.TransferFunds(ctx, transfer),
		"could not transfer funds, less than minimal amount requested to transfer",
	)
}

func testInvalidRecurringTransfersInThePast(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_START})

	var endEpoch13 uint64 = 11
	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(100),
				Reference:       "someref",
			},
			StartEpoch: 6,
			EndEpoch:   &endEpoch13,
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{quantum: num.DecimalFromFloat(10)}), nil)
	e.tsvc.EXPECT().GetTimeNow().Times(2)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.EqualError(t,
		e.TransferFunds(ctx, transfer),
		"start epoch in the past",
	)

	// now all should be fine, let's try to start another same transfer use the current epoch

	transfer2 := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID2",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(50),
				Reference:       "someotherref",
			},
			StartEpoch: 7,
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.NoError(t,
		e.TransferFunds(ctx, transfer2),
	)
}

func testInvalidRecurringTransfersDuplicates(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_END})

	var endEpoch13 uint64 = 11
	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
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

	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{quantum: num.DecimalFromFloat(10)}), nil)
	e.tsvc.EXPECT().GetTimeNow().Times(3)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.NoError(t, e.TransferFunds(ctx, transfer))

	// now all should be fine, let's try to start another same transfer

	transfer2 := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID2",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(50),
				Reference:       "someotherref",
			},
			StartEpoch: 15,
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.EqualError(t,
		e.TransferFunds(ctx, transfer2),
		banking.ErrCannotSubmitDuplicateRecurringTransferWithSameFromAndTo.Error(),
	)

	// same from/to different asset - should pass
	transfer3 := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID3",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "VEGA",
				Amount:          num.NewUint(50),
				Reference:       "someotherref",
			},
			StartEpoch: 15,
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.NoError(t, e.TransferFunds(ctx, transfer3))
}

func testForeverTransferCancelledNotEnoughFunds(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_END})

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "0000000000000000000000000000000000000000000000000000000000000000",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(100),
				Reference:       "someref",
			},
			DispatchStrategy: &vega.DispatchStrategy{Metric: vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED},
			StartEpoch:       10,
			EndEpoch:         nil, // forever
			Factor:           num.MustDecimalFromString("0.9"),
		},
	}

	e.marketActivityTracker.EXPECT().GetMarketScores(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return([]*types.MarketContributionScore{
		{Asset: "asset1", Market: "market1", Metric: vega.DispatchMetric_DISPATCH_METRIC_MAKER_FEES_RECEIVED, Score: num.DecimalFromFloat(1)},
	})
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{quantum: num.DecimalFromFloat(10)}), nil)
	e.tsvc.EXPECT().GetTimeNow().Times(2)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.NoError(t, e.TransferFunds(ctx, transfer))

	// now let's move epochs to see the others transfers
	// first 2 epochs nothing happen
	e.OnEpoch(context.Background(), types.Epoch{Seq: 8, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 8, Action: vega.EpochAction_EPOCH_ACTION_END})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 9, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 9, Action: vega.EpochAction_EPOCH_ACTION_END})
	// now we are in business

	fromAcc := types.Account{
		Balance: num.NewUint(160), // enough for the first transfer
	}

	// asset exists
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(2).Return(&fromAcc, nil)

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
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 1 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(50))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			return nil, nil
		})

	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_END})

	fromAcc = types.Account{
		Balance: num.NewUint(10), // not enough for the second transfer
	}

	// asset exists
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

	e.broker.EXPECT().SendBatch(gomock.Any()).DoAndReturn(func(evts []events.Event) {
		t.Run("ensure transfer is stopped", func(t *testing.T) {
			assert.Len(t, evts, 1)
			e, ok := evts[0].(*events.TransferFunds)
			assert.True(t, ok, "unexpected event from the bus")
			assert.Equal(t, types.TransferStatusStopped, e.Proto().Status)
			assert.Equal(t, "could not pay the fee for transfer: not enough funds to transfer", *e.Proto().Reason)
		})
	})

	// ensure it's not called
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_END})

	// then nothing happen, we are done
	e.OnEpoch(context.Background(), types.Epoch{Seq: 12, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 12, Action: vega.EpochAction_EPOCH_ACTION_END})
}

func testValidRecurringTransfer(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 7, Action: vega.EpochAction_EPOCH_ACTION_END})

	var endEpoch13 uint64 = 11
	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				ID:              "TRANSFERID",
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
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

	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{quantum: num.DecimalFromFloat(10)}), nil)
	e.tsvc.EXPECT().GetTimeNow().Times(3)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.NoError(t, e.TransferFunds(ctx, transfer))

	// now let's move epochs to see the others transfers
	// first 2 epochs nothing happen
	e.OnEpoch(context.Background(), types.Epoch{Seq: 8, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 8, Action: vega.EpochAction_EPOCH_ACTION_END})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 9, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 9, Action: vega.EpochAction_EPOCH_ACTION_END})
	// now we are in business

	fromAcc := types.Account{
		Balance: num.NewUint(1000),
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
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 1 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(50))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			return nil, nil
		})

	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 10, Action: vega.EpochAction_EPOCH_ACTION_END})

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
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(90))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 1 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(45))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			return nil, nil
		})

	e.broker.EXPECT().SendBatch(gomock.Any()).DoAndReturn(func(evts []events.Event) {
		t.Run("ensure transfer is done", func(t *testing.T) {
			assert.Len(t, evts, 1)
			e, ok := evts[0].(*events.TransferFunds)
			assert.True(t, ok, "unexpected event from the bus")
			assert.Equal(t, e.Proto().Status, types.TransferStatusDone)
		})
	})

	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 11, Action: vega.EpochAction_EPOCH_ACTION_END})

	// then nothing happen, we are done
	e.OnEpoch(context.Background(), types.Epoch{Seq: 12, Action: vega.EpochAction_EPOCH_ACTION_START})
	e.OnEpoch(context.Background(), types.Epoch{Seq: 12, Action: vega.EpochAction_EPOCH_ACTION_END})
}

func testRecurringTransferInvalidTransfers(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transfer := types.TransferFunds{
		Kind:      types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{},
	}

	transferBase := types.TransferBase{
		From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
		FromAccountType: types.AccountTypeGeneral,
		To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
		ToAccountType:   types.AccountTypeGeneral,
		Asset:           "eth",
		Amount:          num.NewUint(10),
		Reference:       "someref",
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(1)}), nil)

	var baseCpy types.TransferBase

	t.Run("invalid from account", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy := transferBase
		transfer.Recurring.TransferBase = &baseCpy
		transfer.Recurring.From = ""
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidFromAccount.Error(),
		)
	})

	t.Run("invalid to account", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.Recurring.TransferBase = &baseCpy
		transfer.Recurring.To = ""
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidToAccount.Error(),
		)
	})

	t.Run("unsupported from account type", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.Recurring.TransferBase = &baseCpy
		transfer.Recurring.FromAccountType = types.AccountTypeBond
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrUnsupportedFromAccountType.Error(),
		)
	})

	t.Run("unsuported to account type", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.Recurring.TransferBase = &baseCpy
		transfer.Recurring.ToAccountType = types.AccountTypeBond
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrUnsupportedToAccountType.Error(),
		)
	})

	t.Run("zero funds transfer", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.Recurring.TransferBase = &baseCpy
		transfer.Recurring.Amount = num.UintZero()
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrCannotTransferZeroFunds.Error(),
		)
	})

	var (
		endEpoch100 uint64 = 100
		endEpoch0   uint64
		endEpoch1   uint64 = 1
	)
	// now testing the recurring specific stuff
	baseCpy = transferBase
	transfer.Recurring.TransferBase = &baseCpy
	transfer.Recurring.EndEpoch = &endEpoch100
	transfer.Recurring.StartEpoch = 90
	transfer.Recurring.Factor = num.MustDecimalFromString("0.1")

	t.Run("bad start time", func(t *testing.T) {
		transfer.Recurring.StartEpoch = 0

		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrStartEpochIsZero.Error(),
		)
	})

	t.Run("bad end time", func(t *testing.T) {
		transfer.Recurring.StartEpoch = 90
		transfer.Recurring.EndEpoch = &endEpoch0

		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrEndEpochIsZero.Error(),
		)
	})

	t.Run("negative factor", func(t *testing.T) {
		transfer.Recurring.EndEpoch = &endEpoch100
		transfer.Recurring.Factor = num.MustDecimalFromString("-1")

		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidFactor.Error(),
		)
	})

	t.Run("zero factor", func(t *testing.T) {
		transfer.Recurring.Factor = num.MustDecimalFromString("0")

		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidFactor.Error(),
		)
	})

	t.Run("start epoch after end epoch", func(t *testing.T) {
		transfer.Recurring.Factor = num.MustDecimalFromString("1")
		transfer.Recurring.EndEpoch = &endEpoch1

		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrStartEpochAfterEndEpoch.Error(),
		)
	})

	t.Run("end epoch nil", func(t *testing.T) {
		transfer.Recurring.EndEpoch = nil

		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.NoError(t,
			e.TransferFunds(ctx, &transfer),
		)
	})
}
