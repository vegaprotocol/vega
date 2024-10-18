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

func TestTransfers(t *testing.T) {
	t.Run("invalid transfer kind", testInvalidTransferKind)
	t.Run("onefoff not enough funds to transfer", testOneOffTransferNotEnoughFundsToTransfer)
	t.Run("onefoff invalid transfers", testOneOffTransferInvalidTransfers)
	t.Run("valid oneoff transfer", testValidOneOffTransfer)
	t.Run("valid staking transfers", testStakingTransfers)
	t.Run("valid oneoff with deliverOn", testValidOneOffTransferWithDeliverOn)
	t.Run("valid oneoff with deliverOn in the past is done straight away", testValidOneOffTransferWithDeliverOnInThePastStraightAway)
	t.Run("rejected if doesn't reach minimal amount", testRejectedIfDoesntReachMinimalAmount)
	t.Run("valid oneoff transfer from derived key", testValidOneOffTransferWithFromDerivedKey)
	t.Run("onefoff invalid transfers from derived key", testOneOffTransferInvalidTransfersWithFromDerivedKey)
	t.Run("onefoff invalid owner transfers from derived key", testOneOffTransferInvalidOwnerTransfersWithFromDerivedKey)
}

func testRejectedIfDoesntReachMinimalAmount(t *testing.T) {
	e := getTestEngine(t)

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           assetNameETH,
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
		},
	}

	e.OnMinTransferQuantumMultiple(context.Background(), num.DecimalFromFloat(1))
	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)

	assert.EqualError(t,
		e.TransferFunds(ctx, transfer),
		"could not transfer funds, less than minimal amount requested to transfer",
	)
}

func testInvalidTransferKind(t *testing.T) {
	e := getTestEngine(t)

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

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           assetNameETH,
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
		},
	}

	fromAcc := types.Account{
		Balance: num.NewUint(1),
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
	e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)
	e.broker.EXPECT().Send(gomock.Any()).Times(1)

	assert.EqualError(t,
		e.TransferFunds(ctx, transfer),
		fmt.Errorf("could not pay the fee for transfer: %w", banking.ErrNotEnoughFundsToTransfer).Error(),
	)
}

func testOneOffTransferInvalidTransfers(t *testing.T) {
	e := getTestEngine(t)

	ctx := context.Background()
	transfer := types.TransferFunds{
		Kind:   types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{},
	}

	transferBase := types.TransferBase{
		From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
		FromAccountType: types.AccountTypeGeneral,
		To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
		ToAccountType:   types.AccountTypeGeneral,
		Asset:           assetNameETH,
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
		transfer.OneOff.Amount = num.UintZero()
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrCannotTransferZeroFunds.Error(),
		)
	})
}

func testValidOneOffTransfer(t *testing.T) {
	e := getTestEngine(t)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))

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
		},
	}

	fromAcc := types.Account{
		Balance: num.NewUint(100),
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
		assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
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
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, assetNameETH)
				assert.Equal(t, transfers[1].Owner, "0000000000000000000000000000000000000000000000000000000000000000")
				assert.Equal(t, transfers[1].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[1].Amount.Asset, assetNameETH)

				// 2 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
				assert.Equal(t, accountTypes[1], types.AccountTypeGlobalReward)
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
}

func testStakingTransfers(t *testing.T) {
	e := getTestEngine(t)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnStakingAsset(context.Background(), "ETH")

	ctx := context.Background()

	t.Run("cannot transfer to another pubkey lock_for_staking", func(t *testing.T) {
		transfer := &types.TransferFunds{
			Kind: types.TransferCommandKindOneOff,
			OneOff: &types.OneOffTransfer{
				TransferBase: &types.TransferBase{
					From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					FromAccountType: types.AccountTypeGeneral,
					To:              "10ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					ToAccountType:   types.AccountTypeLockedForStaking,
					Asset:           assetNameETH,
					Amount:          num.NewUint(10),
					Reference:       "someref",
				},
			},
		}

		// asset exists
		e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
			assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		assert.EqualError(t, e.TransferFunds(ctx, transfer), "transfers to locked for staking allowed only from own general account")
	})

	t.Run("cannot transfer from lock_for_staking to another general account", func(t *testing.T) {
		transfer := &types.TransferFunds{
			Kind: types.TransferCommandKindOneOff,
			OneOff: &types.OneOffTransfer{
				TransferBase: &types.TransferBase{
					From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					FromAccountType: types.AccountTypeLockedForStaking,
					To:              "10ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					ToAccountType:   types.AccountTypeGeneral,
					Asset:           assetNameETH,
					Amount:          num.NewUint(10),
					Reference:       "someref",
				},
			},
		}

		// asset exists
		e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
			assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		assert.EqualError(t, e.TransferFunds(ctx, transfer), "transfers from locked for staking allowed only to own general account")
	})

	t.Run("can only transfer from lock_for_staking to own general account", func(t *testing.T) {
		transfer := &types.TransferFunds{
			Kind: types.TransferCommandKindOneOff,
			OneOff: &types.OneOffTransfer{
				TransferBase: &types.TransferBase{
					From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					FromAccountType: types.AccountTypeLockedForStaking,
					To:              "0000000000000000000000000000000000000000000000000000000000000000",
					ToAccountType:   types.AccountTypeGlobalReward,
					Asset:           assetNameETH,
					Amount:          num.NewUint(10),
					Reference:       "someref",
				},
			},
		}

		// asset exists
		e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
			assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		assert.EqualError(t, e.TransferFunds(ctx, transfer), "can only transfer from locked for staking to general account")
	})

	t.Run("can transfer from general to locked_for_staking and emit stake deposited", func(t *testing.T) {
		transfer := &types.TransferFunds{
			Kind: types.TransferCommandKindOneOff,
			OneOff: &types.OneOffTransfer{
				TransferBase: &types.TransferBase{
					From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					FromAccountType: types.AccountTypeGeneral,
					To:              "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					ToAccountType:   types.AccountTypeLockedForStaking,
					Asset:           assetNameETH,
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
			assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
		e.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

		// assert the calculation of fees and transfer request are correct
		e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		e.broker.EXPECT().Send(gomock.Any()).Times(4)

		// expect a call to the stake accounting
		e.stakeAccounting.EXPECT().AddEvent(gomock.Any(), gomock.Any()).Times(1).Do(
			func(_ context.Context, evt *types.StakeLinking) {
				assert.Equal(t, evt.Type, types.StakeLinkingTypeDeposited)
			})
		assert.NoError(t, e.TransferFunds(ctx, transfer))
	})

	t.Run("can transfer from locked_for_staking to general and emit stake removed", func(t *testing.T) {
		transfer := &types.TransferFunds{
			Kind: types.TransferCommandKindOneOff,
			OneOff: &types.OneOffTransfer{
				TransferBase: &types.TransferBase{
					From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					FromAccountType: types.AccountTypeLockedForStaking,
					To:              "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					ToAccountType:   types.AccountTypeGeneral,
					Asset:           assetNameETH,
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
			assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
		e.col.EXPECT().GetPartyLockedForStaking(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

		// assert the calculation of fees and transfer request are correct
		e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		e.broker.EXPECT().Send(gomock.Any()).Times(4)

		// expect a call to the stake accounting
		e.stakeAccounting.EXPECT().AddEvent(gomock.Any(), gomock.Any()).Times(1).Do(
			func(_ context.Context, evt *types.StakeLinking) {
				assert.Equal(t, evt.Type, types.StakeLinkingTypeRemoved)
			})
		assert.NoError(t, e.TransferFunds(ctx, transfer))
	})

	t.Run("can transfer from vested to general and emit stake removed", func(t *testing.T) {
		transfer := &types.TransferFunds{
			Kind: types.TransferCommandKindOneOff,
			OneOff: &types.OneOffTransfer{
				TransferBase: &types.TransferBase{
					From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					FromAccountType: types.AccountTypeVestedRewards,
					To:              "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
					ToAccountType:   types.AccountTypeGeneral,
					Asset:           assetNameETH,
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
			assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
		e.col.EXPECT().GetPartyVestedRewardAccount(gomock.Any(), gomock.Any()).Times(1).Return(&fromAcc, nil)

		// assert the calculation of fees and transfer request are correct
		e.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(1)

		e.broker.EXPECT().Send(gomock.Any()).Times(4)

		// expect a call to the stake accounting
		e.stakeAccounting.EXPECT().AddEvent(gomock.Any(), gomock.Any()).Times(1).Do(
			func(_ context.Context, evt *types.StakeLinking) {
				assert.Equal(t, evt.Type, types.StakeLinkingTypeRemoved)
			})
		assert.NoError(t, e.TransferFunds(ctx, transfer))
	})
}

func testValidOneOffTransferWithDeliverOnInThePastStraightAway(t *testing.T) {
	e := getTestEngine(t)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(9, 0)
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
				assert.Len(t, transfers, 2)
				assert.Equal(t, transfers[0].Owner, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301")
				assert.Equal(t, transfers[0].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[0].Amount.Asset, assetNameETH)
				assert.Equal(t, transfers[1].Owner, "0000000000000000000000000000000000000000000000000000000000000000")
				assert.Equal(t, transfers[1].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[1].Amount.Asset, assetNameETH)

				// 2 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeGeneral)
				assert.Equal(t, accountTypes[1], types.AccountTypeGlobalReward)
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
}

func testValidOneOffTransferWithDeliverOn(t *testing.T) {
	e := getTestEngine(t)

	// let's do a massive fee, easy to test
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

	e.OnTick(context.Background(), time.Unix(11, 0))

	// assert the calculation of fees and transfer request are correct
	e.broker.EXPECT().Send(gomock.Any()).AnyTimes()
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

	e.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	e.OnTick(context.Background(), time.Unix(12, 0))
}

func testValidOneOffTransferWithFromDerivedKey(t *testing.T) {
	e := getTestEngine(t)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))

	partyKey := "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301"
	derivedKey := "c84fbf3442a2a9f9ca87c9cefe686aed241ff49981dd8ce819dd532cd42a8427"
	amount := num.NewUint(10)

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            partyKey,
				FromDerivedKey:  &derivedKey,
				FromAccountType: types.AccountTypeVestedRewards,
				To:              partyKey,
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           assetNameETH,
				Amount:          amount,
				Reference:       "someref",
			},
		},
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
		assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)

	vestedAccount := types.Account{
		Owner: derivedKey,
		// The amount is the same as the transfer amount to ensure that no fee is charged for this type of transaction.
		Balance: amount,
		Asset:   assetNameETH,
	}

	e.col.EXPECT().GetPartyVestedRewardAccount(derivedKey, assetNameETH).Return(&vestedAccount, nil).Times(1)
	e.parties.EXPECT().CheckDerivedKeyOwnership(types.PartyID(partyKey), derivedKey).Return(true).Times(1)

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
				assert.Equal(t, derivedKey, transfers[0].Owner)
				assert.Equal(t, num.NewUint(10), transfers[0].Amount.Amount)
				assert.Equal(t, assetNameETH, transfers[0].Amount.Asset)
				assert.Equal(t, partyKey, transfers[1].Owner)
				assert.Equal(t, transfers[1].Amount.Amount, num.NewUint(10))
				assert.Equal(t, transfers[1].Amount.Asset, assetNameETH)

				// 2 account types too
				assert.Len(t, accountTypes, 2)
				assert.Equal(t, accountTypes[0], types.AccountTypeVestedRewards)
				assert.Equal(t, accountTypes[1], types.AccountTypeGeneral)
			})

			t.Run("ensure fee transfers are correct", func(t *testing.T) {
				assert.Len(t, feeTransfers, 1)
				assert.Equal(t, partyKey, feeTransfers[0].Owner)
				assert.Equal(t, num.UintZero(), feeTransfers[0].Amount.Amount)
				assert.Equal(t, assetNameETH, feeTransfers[0].Amount.Asset)

				// then the fees account types
				assert.Len(t, feeTransfersAccountTypes, 1)
				assert.Equal(t, accountTypes[0], types.AccountTypeVestedRewards)
			})
			return nil, nil
		})

	e.broker.EXPECT().Send(gomock.Any()).Times(3)
	assert.NoError(t, e.TransferFunds(ctx, transfer))
}

func testOneOffTransferInvalidTransfersWithFromDerivedKey(t *testing.T) {
	e := getTestEngine(t)

	ctx := context.Background()
	transfer := types.TransferFunds{
		Kind:   types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{},
	}

	partyKey := "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301"
	derivedKey := "c84fbf3442a2a9f9ca87c9cefe686aed241ff49981dd8ce819dd532cd42a8427"

	transferBase := types.TransferBase{
		From:            partyKey,
		FromDerivedKey:  &derivedKey,
		FromAccountType: types.AccountTypeVestedRewards,
		To:              partyKey,
		ToAccountType:   types.AccountTypeGeneral,
		Asset:           assetNameETH,
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
		transfer.OneOff.FromAccountType = types.AccountTypeGeneral
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrUnsupportedFromAccountType.Error(),
		)
	})

	t.Run("unsuported to account type", func(t *testing.T) {
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.OneOff.TransferBase = &baseCpy
		transfer.OneOff.ToAccountType = types.AccountTypeVestedRewards
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrUnsupportedToAccountType.Error(),
		)
	})

	t.Run("zero funds transfer", func(t *testing.T) {
		e.broker.EXPECT().Send(gomock.Any()).Times(1)
		baseCpy = transferBase
		transfer.OneOff.TransferBase = &baseCpy
		transfer.OneOff.Amount = num.UintZero()
		assert.EqualError(t,
			e.TransferFunds(ctx, &transfer),
			types.ErrCannotTransferZeroFunds.Error(),
		)
	})
}

func testOneOffTransferInvalidOwnerTransfersWithFromDerivedKey(t *testing.T) {
	e := getTestEngine(t)

	// let's do a massive fee, easy to test
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))

	partyKey := "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301"
	derivedKey := "c84fbf3442a2a9f9ca87c9cefe686aed241ff49981dd8ce819dd532cd42a8427"
	amount := num.NewUint(10)

	ctx := context.Background()
	transfer := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            partyKey,
				FromDerivedKey:  &derivedKey,
				FromAccountType: types.AccountTypeVestedRewards,
				To:              partyKey,
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           assetNameETH,
				Amount:          amount,
				Reference:       "someref",
			},
		},
	}

	// asset exists
	e.assets.EXPECT().Get(gomock.Any()).Times(1).Return(
		assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)

	e.parties.EXPECT().CheckDerivedKeyOwnership(types.PartyID(partyKey), derivedKey).Return(false).Times(1)

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	assert.ErrorContains(t, e.TransferFunds(ctx, transfer), "does not own derived key")
}
