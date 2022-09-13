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
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/banking"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestTransferInstructions(t *testing.T) {
	t.Run("invalid transfer instruction kind", testInvalidTransferKind)
	t.Run("onefoff not enough funds to transfer", testOneOffTransferNotEnoughFundsToTransfer)
	t.Run("onefoff invalid transfer instructions", testOneOffTransferInvalidTransfers)
	t.Run("valid oneoff transfer instruction", testValidOneOffTransfer)
	t.Run("valid oneoff with deliverOn", testValidOneOffTransferWithDeliverOn)
	t.Run("valid oneoff with deliverOn in the past is done straight away", testValidOneOffTransferWithDeliverOnInThePastStraightAway)
	t.Run("rejected if doesn't reach minimal amount", testRejectedIfDoesntReachMinimalAmount)
}

func testRejectedIfDoesntReachMinimalAmount(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transferInstruction := &types.TransferInstructionFunds{
		Kind: types.TransferInstructionCommandKindOneOff,
		OneOff: &types.OneOffTransferInstruction{
			TransferInstructionBase: &types.TransferInstructionBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
		},
	}

	e.OnMinTransferQuantumMultiple(context.Background(), num.DecimalFromFloat(1))
	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	e.tsvc.EXPECT().GetTimeNow().Times(1)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)

	assert.EqualError(t,
		e.TransferFunds(ctx, transferInstruction),
		"could not transfer funds, less than minimal amount requested to transfer",
	)
}

func testInvalidTransferKind(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transferInstruction := &types.TransferInstructionFunds{
		Kind: types.TransferInstructionCommandKind(-1),
	}
	e.tsvc.EXPECT().GetTimeNow().Times(1)
	assert.EqualError(t,
		e.TransferFunds(ctx, transferInstruction),
		banking.ErrUnsupportedTransferKind.Error(),
	)
}

func testOneOffTransferNotEnoughFundsToTransfer(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()
	e.tsvc.EXPECT().GetTimeNow().Times(1)

	ctx := context.Background()
	transferInstruction := &types.TransferInstructionFunds{
		Kind: types.TransferInstructionCommandKindOneOff,
		OneOff: &types.OneOffTransferInstruction{
			TransferInstructionBase: &types.TransferInstructionBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
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
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(1)}), nil)
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)

	assert.EqualError(t,
		e.TransferFunds(ctx, transferInstruction),
		fmt.Errorf("could not pay the fee for transfer: %w", banking.ErrNotEnoughFundsToTransfer).Error(),
	)
}

func testOneOffTransferInvalidTransfers(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	ctx := context.Background()
	transferInstruction := types.TransferInstructionFunds{
		Kind:   types.TransferInstructionCommandKindOneOff,
		OneOff: &types.OneOffTransferInstruction{},
	}

	transferInstructionBase := types.TransferInstructionBase{
		From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
		FromAccountType: types.AccountTypeGeneral,
		To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
		ToAccountType:   types.AccountTypeGeneral,
		Asset:           "eth",
		Amount:          num.NewUint(10),
		Reference:       "someref",
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(nil, nil)
	var baseCpy types.TransferInstructionBase

	t.Run("invalid from account", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy := transferInstructionBase
		transferInstruction.OneOff.TransferInstructionBase = &baseCpy
		transferInstruction.OneOff.From = ""
		assert.EqualError(t,
			e.TransferFunds(ctx, &transferInstruction),
			types.ErrInvalidFromAccount.Error(),
		)
	})

	t.Run("invalid to account", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferInstructionBase
		transferInstruction.OneOff.TransferInstructionBase = &baseCpy
		transferInstruction.OneOff.To = ""
		assert.EqualError(t,
			e.TransferFunds(ctx, &transferInstruction),
			types.ErrInvalidToAccount.Error(),
		)
	})

	t.Run("unsupported from account type", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferInstructionBase
		transferInstruction.OneOff.TransferInstructionBase = &baseCpy
		transferInstruction.OneOff.FromAccountType = types.AccountTypeBond
		assert.EqualError(t,
			e.TransferFunds(ctx, &transferInstruction),
			types.ErrUnsupportedFromAccountType.Error(),
		)
	})

	t.Run("unsuported to account type", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferInstructionBase
		transferInstruction.OneOff.TransferInstructionBase = &baseCpy
		transferInstruction.OneOff.ToAccountType = types.AccountTypeBond
		assert.EqualError(t,
			e.TransferFunds(ctx, &transferInstruction),
			types.ErrUnsupportedToAccountType.Error(),
		)
	})

	t.Run("zero funds transfer", func(t *testing.T) {
		e.tsvc.EXPECT().GetTimeNow().Times(1)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferInstructionBase
		transferInstruction.OneOff.TransferInstructionBase = &baseCpy
		transferInstruction.OneOff.Amount = num.UintZero()
		assert.EqualError(t,
			e.TransferFunds(ctx, &transferInstruction),
			types.ErrCannotTransferInstructionZeroFunds.Error(),
		)
	})
}

func testValidOneOffTransfer(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))

	ctx := context.Background()
	transferInstruction := &types.TransferInstructionFunds{
		Kind: types.TransferInstructionCommandKindOneOff,
		OneOff: &types.OneOffTransferInstruction{
			TransferInstructionBase: &types.TransferInstructionBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
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
	e.tsvc.EXPECT().GetTimeNow().Times(1)
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
		assets.NewAsset(&mockAsset{num.DecimalFromFloat(1)}), nil)
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

	// assert the calculation of fees and transfer request are correct
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context,
			transferInstructions []*types.TransferInstruction,
			accountTypes []types.AccountType,
			references []string,
			feeTransfers []*types.TransferInstruction,
			feeTransfersAccountTypes []types.AccountType,
		) ([]*types.TransferInstructionResponse, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Len(t, transferInstructions, 2)
				assert.Equal(t, transferInstructions[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, transferInstructions[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transferInstructions[0].Amount.Asset, "eth")
				assert.Equal(t, transferInstructions[1].Owner, "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0")
				assert.Equal(t, transferInstructions[1].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transferInstructions[1].Amount.Asset, "eth")

				// 2 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
				assert.Equal(t, accountTypes[1], types.AccountTypeGlobalReward)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(2)
	e.tsvc.EXPECT().GetTimeNow().AnyTimes()
	assert.NoError(t, e.TransferFunds(ctx, transferInstruction))
}

func testValidOneOffTransferWithDeliverOnInThePastStraightAway(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(10, 0)
		}).Times(3)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(9, 0)
	ctx := context.Background()
	transferInstruction := &types.TransferInstructionFunds{
		Kind: types.TransferInstructionCommandKindOneOff,
		OneOff: &types.OneOffTransferInstruction{
			TransferInstructionBase: &types.TransferInstructionBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
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
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(1)}), nil)
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

	// assert the calculation of fees and transfer request are correct
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context,
			transferInstructions []*types.TransferInstruction,
			accountTypes []types.AccountType,
			references []string,
			feeTransfers []*types.TransferInstruction,
			feeTransfersAccountTypes []types.AccountType,
		) ([]*types.TransferInstructionResponse, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Len(t, transferInstructions, 2)
				assert.Equal(t, transferInstructions[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, transferInstructions[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transferInstructions[0].Amount.Asset, "eth")
				assert.Equal(t, transferInstructions[1].Owner, "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0")
				assert.Equal(t, transferInstructions[1].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transferInstructions[1].Amount.Asset, "eth")

				// 2 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
				assert.Equal(t, accountTypes[1], types.AccountTypeGlobalReward)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(2)
	assert.NoError(t, e.TransferFunds(ctx, transferInstruction))
}

func testValidOneOffTransferWithDeliverOn(t *testing.T) {
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	// Time given to OnTick call - base time Unix(10, 0)
	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(10, 0)
		}).Times(1)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(12, 0)
	ctx := context.Background()
	transferInstruction := &types.TransferInstructionFunds{
		Kind: types.TransferInstructionCommandKindOneOff,
		OneOff: &types.OneOffTransferInstruction{
			TransferInstructionBase: &types.TransferInstructionBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
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

	// Time given to e.Transferfunds - base time Unix(10,0)
	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(10, 0)
		}).Times(2)

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(1)}), nil)
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

	// assert the calculation of fees and transfer request are correct
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context,
			transferInstructions []*types.TransferInstruction,
			accountTypes []types.AccountType,
			references []string,
			feeTransfers []*types.TransferInstruction,
			feeTransfersAccountTypes []types.AccountType,
		) ([]*types.TransferInstructionResponse, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Len(t, transferInstructions, 1)
				assert.Equal(t, transferInstructions[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, transferInstructions[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transferInstructions[0].Amount.Asset, "eth")

				// 2 account types too
				assert.Len(t, accountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, feeTransfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, feeTransfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, feeTransfers[0].Amount.Asset, "eth")

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(2)
	assert.NoError(t, e.TransferFunds(ctx, transferInstruction))

	// Run OnTick with time.Unix(11, 0) and expect nothing.
	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(11, 0)
		}).Times(1)

	e.OnTick(context.Background(), time.Unix(11, 0))

	// Give time to trigger transfers.
	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(12, 0)
		}).Times(1)

	// assert the calculation of fees and transfer request are correct
	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1).DoAndReturn(
		func(ctx context.Context,
			transferInstructions []*types.TransferInstruction,
			accountTypes []types.AccountType,
			references []string,
			feeTransfers []*types.TransferInstruction,
			feeTransfersAccountTypes []types.AccountType,
		) ([]*types.TransferInstructionResponse, error,
		) {
			t.Run("ensure transfers are correct", func(t *testing.T) {
				// transfer is done fully instantly, we should have 2 transfer
				assert.Equal(t, transferInstructions[0].Owner, "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0")
				assert.Equal(t, transferInstructions[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transferInstructions[0].Amount.Asset, "eth")

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
