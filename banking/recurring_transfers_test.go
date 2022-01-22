package banking_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestRecurringTransfers(t *testing.T) {
	t.Run("recurring invalid transfers", testRecurringTransferInvalidTransfers)
	// t.Run("valid recurring transfers", testValidRecurringTransfer)
}

// func testValidRecurringTransfer(t *testing.T) {
// 	e := getTestEngine(t)
// 	defer e.ctrl.Finish()

// 	// let's do a massive fee, easy to test
// 	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(0.5))
// 	e.OnEpoch(context.Background(), types.Epoch{Seq: 7})

// 	ctx := context.Background()
// 	transfer := &types.TransferFunds{
// 		Kind: types.TransferCommandKindRecurring,
// 		Recurring: &types.RecurringTransfer{
// 			TransferBase: &types.TransferBase{
// 				From:            "from",
// 				FromAccountType: types.AccountTypeGeneral,
// 				To:              "to",
// 				ToAccountType:   types.AccountTypeGlobalReward,
// 				Asset:           "eth",
// 				Amount:          num.NewUint(100),
// 				Reference:       "someref",
// 			},
// 			StartEpoch: 10,
// 			EndEpoch:   13,
// 			Factor:     num.MustDecimalFromString("0.9"),
// 		},
// 	}

// 	fromAcc := types.Account{
// 		Balance: num.NewUint(1000),
// 	}

// 	// asset exists
// 	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(nil, nil)
// 	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

// 	// assert the calculation of fees and transfer request are correct
// 	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
// 		func(ctx context.Context,
// 			transfers []*types.Transfer,
// 			accountTypes []types.AccountType,
// 			references []string,
// 			feeTransfers []*types.Transfer,
// 			feeTransfersAccountTypes []types.AccountType,
// 		) ([]*types.TransferResponse, error,
// 		) {
// 			t.Run("ensure transfers are correct", func(t *testing.T) {
// 				// transfer is done fully instantly, we should have 2 transfer
// 				assert.Len(t, transfers, 1)
// 				assert.Equal(t, transfers[0].Owner, "from")
// 				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(271))
// 				assert.Equal(t, transfers[0].Amount.Asset, "eth")

// 				// 1 account types too
// 				assert.Len(t, accountTypes, 1)
// 				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
// 			})

// 			return nil, nil
// 		})

// 	e.broker.EXPECT().Send(gomock.Any()).Times(2)
// 	assert.NoError(t, e.TransferFunds(ctx, transfer))

// 	// now let's move epochs to see the others transfers
// 	// first 2 epochs nothing happen
// 	e.OnEpoch(context.Background(), types.Epoch{Seq: 8})
// 	e.OnEpoch(context.Background(), types.Epoch{Seq: 9})

// 	// now we are in business
// }

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
