package banking_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/banking"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTransfers(t *testing.T) {
	t.Run("invalid transfer kind", testInvalidTransferKind)
	t.Run("onefoff not enough funds to transfer", testOneOffTransferNotEnoughFundsToTransfer)
	t.Run("onefoff invalid transfers", testOneOffTransferInvalidTransfers)
	t.Run("valid oneoff transfer", testValidOneOffTransfer)
	t.Run("valid oneoff with deliverOn", testValidOneOffTransferWithDeliverOn)
	t.Run("valid oneoff with deliverOn in the past is done straight away", testValidOneOffTransferWithDeliverOnInThePastStraightAway)
	t.Run("rejected if doesn't reach minimal amount", testRejectedIfDoesntReachMinimalAmount)
}

func testRejectedIfDoesntReachMinimalAmount(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "from",
				FromAccountType: types.AccountTypeGeneral,
				To:              "to",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
		},
	}

	e.OnMinTransferQuantumMultiple(context.Background(), num.DecimalFromFloat(1))
	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.NewUint(100)}), nil)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)

	assert.EqualError(t,
		e.TransferFunds(ctx, transfer),
		"could not transfer funds, less than minimal amount requested to transfer",
	)
}

func testInvalidTransferKind(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKind(-1),
	}
	assert.EqualError(t,
		e.TransferFunds(ctx, transfer),
		banking.ErrUnsupportedTransferKind.Error(),
	)
}

func testOneOffTransferNotEnoughFundsToTransfer(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "from",
				FromAccountType: types.AccountTypeGeneral,
				To:              "to",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
		},
	}

	fromAcc := types.Account{
		Balance: num.NewUint(1),
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.NewUint(1)}), nil)
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)

	assert.EqualError(t,
		e.TransferFunds(ctx, transfer),
		fmt.Errorf("could not pay the fee for transfer: %w", banking.ErrNotEnoughFundsToTransfer).Error(),
	)
}

func testOneOffTransferInvalidTransfers(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transfer := types.TransferFunds{
		Kind:   types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{},
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
		transfer.OneOff.TransferBase = &baseCpy
		transfer.OneOff.From = ""
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidFromAccount.Error(),
		)
	})

	t.Run("invalid to account", func(t *testing.T) {
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.OneOff.TransferBase = &baseCpy
		transfer.OneOff.To = ""
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrInvalidToAccount.Error(),
		)
	})

	t.Run("unsupported from account type", func(t *testing.T) {
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.OneOff.TransferBase = &baseCpy
		transfer.OneOff.FromAccountType = types.AccountTypeBond
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrUnsupportedFromAccountType.Error(),
		)
	})

	t.Run("unsuported to account type", func(t *testing.T) {
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.OneOff.TransferBase = &baseCpy
		transfer.OneOff.ToAccountType = types.AccountTypeBond
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrUnsupportedToAccountType.Error(),
		)
	})

	t.Run("zero funds transfer", func(t *testing.T) {
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.OneOff.TransferBase = &baseCpy
		transfer.OneOff.Amount = num.Zero()
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrCannotTransferZeroFunds.Error(),
		)
	})
}

func testValidOneOffTransfer(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "from",
				FromAccountType: types.AccountTypeGeneral,
				To:              "to",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
		},
	}

	fromAcc := types.Account{
		Balance: num.NewUint(100),
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
		assets.NewAsset(&mockAsset{num.NewUint(1)}), nil)
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
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")
				assert.Equal(t, transfers[1].Owner, "to")
				assert.Equal(t, transfers[1].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[1].Amount.Asset, "eth")

				// 2 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
				assert.Equal(t, accountTypes[1], types.AccountTypeGlobalReward)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "from")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(2)

	assert.NoError(t, e.TransferFunds(ctx, transfer))
}

func testValidOneOffTransferWithDeliverOnInThePastStraightAway(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(9, 0)
	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "from",
				FromAccountType: types.AccountTypeGeneral,
				To:              "to",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
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
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.NewUint(1)}), nil)
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
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")
				assert.Equal(t, transfers[1].Owner, "to")
				assert.Equal(t, transfers[1].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[1].Amount.Asset, "eth")

				// 2 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
				assert.Equal(t, accountTypes[1], types.AccountTypeGlobalReward)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "from")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(2)

	assert.NoError(t, e.TransferFunds(ctx, transfer))
}

func testValidOneOffTransferWithDeliverOn(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(12, 0)
	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "from",
				FromAccountType: types.AccountTypeGeneral,
				To:              "to",
				ToAccountType:   types.AccountTypeGlobalReward,
				Asset:           "eth",
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
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.NewUint(1)}), nil)
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
				assert.Len(t, transfers, 1)
				assert.Equal(t, transfers[0].Owner, "from")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 2 account types too
				assert.Len(t, accountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "from")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(2)

	assert.NoError(t, e.TransferFunds(ctx, transfer))

	// nothing expected
	e.OnTick(context.Background(), time.Unix(11, 0))

	// now the funds are being paid

	// assert the calculation of fees and transfer request are correct
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
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
				assert.Equal(t, transfers[0].Owner, "to")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, "eth")

				// 1 account types too
				assert.Len(t, accountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGlobalReward)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 0)
			})
			return nil, nil
		})

	e.broker.EXPECT().SendBatch(gomock.Any()).Times(1)
	e.OnTick(context.Background(), time.Unix(12, 0))
}
