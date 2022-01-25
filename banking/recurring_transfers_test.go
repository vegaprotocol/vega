package banking_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/protos/vega"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRecurringTransfers(t *testing.T) {
	t.Run("recurring invalid transfers", testRecurringTransferInvalidTransfers)
	t.Run("valid recurring transfers", testValidRecurringTransfer)
	t.Run("valid forever transfers, cancelled not enough funds", testForeverTransferCancelledNotEnoughFunds)
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
				From:            "from",
				FromAccountType: types.AccountTypeGeneral,
				To:              "to",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(100),
				Reference:       "someref",
			},
			StartEpoch: 10,
			EndEpoch:   nil, // forever
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
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
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

	// assert the calculation of fees and transfer request are correct
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context,
			transfers []*types.Transfer,
			accountTypes []types.AccountType,
			references []string,
			feeTransfers []*types.Transfer,
			feeTransfersAccountTypes []types.AccountType,
		) ([]*types.TransferResponse, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Len(t, transfers, 2)
				assert.Equal(t, transfers[0].Owner, "from")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(100))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 1 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "from")
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
			assert.Equal(t, e.Proto().Status, types.TransferStatusStopped)
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
				From:            "from",
				FromAccountType: types.AccountTypeGeneral,
				To:              "to",
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

	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
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
		) ([]*types.TransferResponse, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Len(t, transfers, 2)
				assert.Equal(t, transfers[0].Owner, "from")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(100))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 1 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "from")
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
		) ([]*types.TransferResponse, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Len(t, transfers, 2)
				assert.Equal(t, transfers[0].Owner, "from")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(90))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 1 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "from")
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
		From:            "from",
		FromAccountType: types.AccountTypeGeneral,
		To:              "to",
		ToAccountType:   types.AccountTypeGeneral,
		Asset:           "eth",
		Amount:          num.NewUint(10),
		Reference:       "someref",
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)

	var baseCpy types.TransferBase

	t.Run("invalid from account", func(t *testing.T) {
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
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.Recurring.TransferBase = &baseCpy
		transfer.Recurring.Amount = num.Zero()
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrCannotTransferZeroFunds.Error(),
		)
	})

	var (
		endEpoch100 uint64 = 100
		endEpoch0   uint64 = 0
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

		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrStartEpochIsZero.Error(),
		)
	})

	t.Run("bad end time", func(t *testing.T) {
		transfer.Recurring.StartEpoch = 90
		transfer.Recurring.EndEpoch = &endEpoch0

		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrEndEpochIsZero.Error(),
		)
	})

	t.Run("negative factor", func(t *testing.T) {
		transfer.Recurring.EndEpoch = &endEpoch100
		transfer.Recurring.Factor = num.MustDecimalFromString("-1")

		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidFactor.Error(),
		)
	})

	t.Run("zero factor", func(t *testing.T) {
		transfer.Recurring.Factor = num.MustDecimalFromString("0")

		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidFactor.Error(),
		)
	})

	t.Run("start epoch after end epoch", func(t *testing.T) {
		transfer.Recurring.Factor = num.MustDecimalFromString("1")
		transfer.Recurring.EndEpoch = &endEpoch1

		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrStartEpochAfterEndEpoch.Error(),
		)
	})

	t.Run("end epoch nil", func(t *testing.T) {
		transfer.Recurring.EndEpoch = nil

		e.broker.EXPECT().Send(gomock.Any()).Times(1)

		assert.NoError(t,
			e.TransferFunds(ctx, &transfer),
		)
	})
}
