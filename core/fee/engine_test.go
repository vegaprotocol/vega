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

package fee_test

import (
	"errors"
	"testing"

	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/fee/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testAsset = "ETH"
)

var (
	testFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(0.05),
			MakerFee:          num.DecimalFromFloat(0.02),
		},
	}
	extendedTestFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(0.05),
			MakerFee:          num.DecimalFromFloat(0.02),
			BuyBackFee:        num.DecimalFromFloat(0.002),
			TreasuryFee:       num.DecimalFromFloat(0.003),
		},
	}
)

type testFee struct {
	*fee.Engine
}

func getTestFee(t *testing.T) *testFee {
	t.Helper()
	eng, err := fee.New(
		logging.NewTestLogger(),
		fee.NewDefaultConfig(),
		testFees,
		testAsset,
		num.DecimalFromInt64(1),
	)
	assert.NoError(t, err)
	return &testFee{eng}
}

func getExtendedTestFee(t *testing.T) *testFee {
	t.Helper()
	eng, err := fee.New(
		logging.NewTestLogger(),
		fee.NewDefaultConfig(),
		extendedTestFees,
		testAsset,
		num.DecimalFromInt64(1),
	)
	assert.NoError(t, err)
	return &testFee{eng}
}

func TestFeeEngine(t *testing.T) {
	t.Run("update fee factors with invalid input", testUpdateFeeFactorsError)
	t.Run("update fee factors with valid input", testUpdateFeeFactors)
	t.Run("calculate continuous trading fee empty trade", testCalcContinuousTradingErrorEmptyTrade)
	t.Run("calculate continuous trading fee", testCalcContinuousTrading)
	t.Run("calculate continuous trading fee + check amounts", testCalcContinuousTradingAndCheckAmounts)
	t.Run("calculate continuous trading fee + check amounts with discounts and rewards", testCalcContinuousTradingAndCheckAmountsWithDiscount)
	t.Run("calculate auction trading fee empty trade", testCalcAuctionTradingErrorEmptyTrade)
	t.Run("calculate auction trading fee", testCalcAuctionTrading)
	t.Run("calculate batch auction trading fee empty trade", testCalcBatchAuctionTradingErrorEmptyTrade)
	t.Run("calculate batch auction trading fee same batch", testCalcBatchAuctionTradingSameBatch)
	t.Run("calculate batch auction trading fee different batches", testCalcBatchAuctionTradingDifferentBatches)
	t.Run("Build liquidity fee transfers with remainder", testBuildLiquidityFeesRemainder)
	t.Run("calculate closeout fees", testCloseoutFees)
}

func TestFeeEngineWithBuyBackAndTreasury(t *testing.T) {
	t.Run("update fee factors with invalid input", testUpdateExtendedFeeFactorsError)
	t.Run("update fee factors with valid input", testUpdateExtendedFeeFactors)
	t.Run("calculate continuous trading fee empty trade", testCalcContinuousTradingErrorEmptyTrade)
	t.Run("calculate continuous trading fee", testCalcContinuousTradingExtended)
	t.Run("calculate continuous trading fee + check amounts", testCalcContinuousTradingAndCheckAmountsExtended)
	t.Run("calculate continuous trading fee + check amounts with discounts and rewards", testCalcContinuousTradingAndCheckAmountsWithDiscountExtended)
	t.Run("calculate auction trading fee empty trade", testCalcAuctionTradingErrorEmptyTrade)
	t.Run("calculate auction trading fee", testCalcAuctionTradingExtended)
	t.Run("calculate batch auction trading fee empty trade", testCalcBatchAuctionTradingErrorEmptyTrade)
}

func testUpdateFeeFactors(t *testing.T) {
	eng := getTestFee(t)
	okFees := types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(0.5),
			MakerFee:          num.DecimalFromFloat(0.25),
		},
	}
	err := eng.UpdateFeeFactors(okFees)
	assert.NoError(t, err)
}

func testUpdateExtendedFeeFactors(t *testing.T) {
	eng := getExtendedTestFee(t)
	okFees := types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(0.5),
			MakerFee:          num.DecimalFromFloat(0.25),
			BuyBackFee:        num.DecimalFromFloat(0.3),
			TreasuryFee:       num.DecimalFromFloat(0.4),
		},
	}
	err := eng.UpdateFeeFactors(okFees)
	assert.NoError(t, err)
}

func testUpdateFeeFactorsError(t *testing.T) {
	eng := getTestFee(t)
	koFees := types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(-.1),
			InfrastructureFee: num.DecimalFromFloat(0.5),
			MakerFee:          num.DecimalFromFloat(0.25),
		},
	}
	err := eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)

	koFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(-.1),
			MakerFee:          num.DecimalFromFloat(0.25),
		},
	}
	err = eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)
	koFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(0.5),
			MakerFee:          num.DecimalFromFloat(-.1),
		},
	}
	err = eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)
}

func testUpdateExtendedFeeFactorsError(t *testing.T) {
	eng := getExtendedTestFee(t)
	koFees := types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(0.5),
			MakerFee:          num.DecimalFromFloat(0.25),
			BuyBackFee:        num.DecimalFromFloat(-1),
			TreasuryFee:       num.DecimalFromFloat(0.4),
		},
	}
	err := eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)

	koFees = types.Fees{
		Factors: &types.FeeFactors{
			LiquidityFee:      num.DecimalFromFloat(0.1),
			InfrastructureFee: num.DecimalFromFloat(0.11),
			MakerFee:          num.DecimalFromFloat(0.25),
			BuyBackFee:        num.DecimalFromFloat(0.41),
			TreasuryFee:       num.DecimalFromFloat(-1),
		},
	}
	err = eng.UpdateFeeFactors(koFees)
	assert.Error(t, err)
}

func testCalcContinuousTradingErrorEmptyTrade(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()

	_, err := eng.CalculateForContinuousMode([]*types.Trade{}, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())

	eng = getExtendedTestFee(t)
	_, err = eng.CalculateForContinuousMode([]*types.Trade{}, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())
}

func testCalcContinuousTradingAndCheckAmounts(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	require.NoError(t, eng.UpdateFeeFactors(types.Fees{
		Factors: &types.FeeFactors{
			MakerFee:          num.DecimalFromFloat(.000250),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			LiquidityFee:      num.DecimalFromFloat(0.001),
		},
	}))
	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      5,
			Price:     num.NewUint(100000),
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
			assert.Equal(t, num.NewUint(500), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
			assert.Equal(t, num.NewUint(250), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
			assert.Equal(t, num.NewUint(125), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
			assert.Equal(t, num.NewUint(125), v.Amount.Amount)
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
	assert.Equal(t, &eventspb.FeesStats{
		Market:                   "",
		Asset:                    testAsset,
		EpochSeq:                 0,
		TotalRewardsReceived:     []*eventspb.PartyAmount{},
		ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{},
		RefereesDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "0",
				QuantumAmount: "0",
			},
		},
		VolumeDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "0",
				QuantumAmount: "0",
			},
		},
		TotalMakerFeesReceived: []*eventspb.PartyAmount{
			{
				Party:         "party2",
				Amount:        "125",
				QuantumAmount: "125",
			},
		},
		MakerFeesGenerated: []*eventspb.MakerFeesGenerated{
			{
				Taker: "party1",
				MakerFeesPaid: []*eventspb.PartyAmount{
					{
						Party:         "party2",
						Amount:        "125",
						QuantumAmount: "125",
					},
				},
			},
		},
		TotalFeesPaidAndReceived: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "875",
				QuantumAmount: "875",
			},
			{
				Party:         "party2",
				Amount:        "125",
				QuantumAmount: "125",
			},
		},
	}, eng.GetFeesStatsOnEpochEnd(num.DecimalFromInt64(1)))
}

func testCalcContinuousTradingAndCheckAmountsExtended(t *testing.T) {
	eng := getExtendedTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(types.PartyID("party2")).Return(num.DecimalFromFloat(0.0025)).AnyTimes()
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	require.NoError(t, eng.UpdateFeeFactors(types.Fees{
		Factors: &types.FeeFactors{
			MakerFee:          num.DecimalFromFloat(.000250),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			LiquidityFee:      num.DecimalFromFloat(0.001),
			BuyBackFee:        num.DecimalFromFloat(0.002),
			TreasuryFee:       num.DecimalFromFloat(0.003),
		},
	}))
	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      5,
			Price:     num.NewUint(100000),
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity, bb, treasury, hmp, hmr int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
			assert.Equal(t, num.NewUint(500), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
			assert.Equal(t, num.NewUint(250), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
			assert.Equal(t, num.NewUint(125), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
			assert.Equal(t, num.NewUint(125), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeBuyBackFeePay {
			bb++
			assert.Equal(t, num.NewUint(500), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeTreasuryPay {
			treasury++
			assert.Equal(t, num.NewUint(750), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeHighMakerRebatePay {
			hmp++
			assert.Equal(t, num.NewUint(1250), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeHighMakerRebateReceive {
			hmr++
			assert.Equal(t, num.NewUint(1250), v.Amount.Amount)
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, bb, 1)
	assert.Equal(t, treasury, 1)
	assert.Equal(t, hmp, 1)
	assert.Equal(t, hmr, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
	assert.Equal(t, &eventspb.FeesStats{
		Market:                   "",
		Asset:                    testAsset,
		EpochSeq:                 0,
		TotalRewardsReceived:     []*eventspb.PartyAmount{},
		ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{},
		RefereesDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "0",
				QuantumAmount: "0",
			},
		},
		VolumeDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "0",
				QuantumAmount: "0",
			},
		},
		TotalMakerFeesReceived: []*eventspb.PartyAmount{
			{
				Party:         "party2",
				Amount:        "125",
				QuantumAmount: "125",
			},
		},
		MakerFeesGenerated: []*eventspb.MakerFeesGenerated{
			{
				Taker: "party1",
				MakerFeesPaid: []*eventspb.PartyAmount{
					{
						Party:         "party2",
						Amount:        "125",
						QuantumAmount: "125",
					},
				},
			},
		},
		TotalFeesPaidAndReceived: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "875",
				QuantumAmount: "875",
			},
			{
				Party:         "party2",
				Amount:        "125",
				QuantumAmount: "125",
			},
		},
	}, eng.GetFeesStatsOnEpochEnd(num.DecimalFromInt64(1)))
}

func testCalcContinuousTradingAndCheckAmountsWithDiscount(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.3),
		Maker:     num.NewDecimalFromFloat(0.3),
		Liquidity: num.NewDecimalFromFloat(0.3),
	}).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.2),
		Maker:     num.NewDecimalFromFloat(0.2),
		Liquidity: num.NewDecimalFromFloat(0.2),
	}).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(
		types.Factors{
			Infra:     num.NewDecimalFromFloat(0.1),
			Maker:     num.NewDecimalFromFloat(0.1),
			Liquidity: num.NewDecimalFromFloat(0.1),
		})
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID("party3"), nil).AnyTimes()
	require.NoError(t, eng.UpdateFeeFactors(types.Fees{
		Factors: &types.FeeFactors{
			MakerFee:          num.DecimalFromFloat(.000250),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			LiquidityFee:      num.DecimalFromFloat(0.001),
		},
	}))
	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      5,
			Price:     num.NewUint(100000),
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
			assert.Equal(t, num.NewUint(252), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
			assert.Equal(t, num.NewUint(127), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
			assert.Equal(t, num.NewUint(64), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
			assert.Equal(t, num.NewUint(64), v.Amount.Amount)
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
	assert.Equal(t, &eventspb.FeesStats{
		Asset: testAsset,
		TotalRewardsReceived: []*eventspb.PartyAmount{
			{
				Party:         "party3",
				Amount:        "110",
				QuantumAmount: "110",
			},
		},
		ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
			{
				Referrer: "party3",
				GeneratedReward: []*eventspb.PartyAmount{
					{
						Party:         "party1",
						Amount:        "110",
						QuantumAmount: "110",
					},
				},
			},
		},
		RefereesDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "262",
				QuantumAmount: "262",
			},
		},
		VolumeDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "60",
				QuantumAmount: "60",
			},
		},
		TotalMakerFeesReceived: []*eventspb.PartyAmount{
			{
				Party:         "party2",
				Amount:        "64",
				QuantumAmount: "64",
			},
		},
		MakerFeesGenerated: []*eventspb.MakerFeesGenerated{
			{
				Taker: "party1",
				MakerFeesPaid: []*eventspb.PartyAmount{
					{
						Party:         "party2",
						Amount:        "64",
						QuantumAmount: "64",
					},
				},
			},
		},
		TotalFeesPaidAndReceived: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "443",
				QuantumAmount: "443",
			},
			{
				Party:         "party2",
				Amount:        "64",
				QuantumAmount: "64",
			},
		},
	}, eng.GetFeesStatsOnEpochEnd(num.DecimalFromInt64(1)))
}

func testCalcContinuousTradingAndCheckAmountsWithDiscountExtended(t *testing.T) {
	eng := getExtendedTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalFromFloat(0.0025)).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.3),
		Maker:     num.NewDecimalFromFloat(0.3),
		Liquidity: num.NewDecimalFromFloat(0.3),
	}).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.2),
		Maker:     num.NewDecimalFromFloat(0.2),
		Liquidity: num.NewDecimalFromFloat(0.2),
	}).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(
		types.Factors{
			Infra:     num.NewDecimalFromFloat(0.1),
			Maker:     num.NewDecimalFromFloat(0.1),
			Liquidity: num.NewDecimalFromFloat(0.1),
		})
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID("party3"), nil).AnyTimes()
	require.NoError(t, eng.UpdateFeeFactors(types.Fees{
		Factors: &types.FeeFactors{
			MakerFee:          num.DecimalFromFloat(.000250),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			LiquidityFee:      num.DecimalFromFloat(0.001),
			BuyBackFee:        num.DecimalFromFloat(0.002),
			TreasuryFee:       num.DecimalFromFloat(0.003),
		},
	}))
	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      5,
			Price:     num.NewUint(100000),
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity, bb, treasury, hmp, hmr int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
			assert.Equal(t, num.NewUint(252), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
			assert.Equal(t, num.NewUint(127), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
			assert.Equal(t, num.NewUint(64), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
			assert.Equal(t, num.NewUint(64), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeBuyBackFeePay {
			bb++
			assert.Equal(t, num.NewUint(500), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeTreasuryPay {
			treasury++
			assert.Equal(t, num.NewUint(750), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeHighMakerRebatePay {
			hmp++
			assert.Equal(t, num.NewUint(1250), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeHighMakerRebateReceive {
			hmr++
			assert.Equal(t, num.NewUint(1250), v.Amount.Amount)
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, bb, 1)
	assert.Equal(t, treasury, 1)
	assert.Equal(t, hmp, 1)
	assert.Equal(t, hmr, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
	assert.Equal(t, &eventspb.FeesStats{
		Asset: testAsset,
		TotalRewardsReceived: []*eventspb.PartyAmount{
			{
				Party:         "party3",
				Amount:        "110",
				QuantumAmount: "110",
			},
		},
		ReferrerRewardsGenerated: []*eventspb.ReferrerRewardsGenerated{
			{
				Referrer: "party3",
				GeneratedReward: []*eventspb.PartyAmount{
					{
						Party:         "party1",
						Amount:        "110",
						QuantumAmount: "110",
					},
				},
			},
		},
		RefereesDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "262",
				QuantumAmount: "262",
			},
		},
		VolumeDiscountApplied: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "60",
				QuantumAmount: "60",
			},
		},
		TotalMakerFeesReceived: []*eventspb.PartyAmount{
			{
				Party:         "party2",
				Amount:        "64",
				QuantumAmount: "64",
			},
		},
		MakerFeesGenerated: []*eventspb.MakerFeesGenerated{
			{
				Taker: "party1",
				MakerFeesPaid: []*eventspb.PartyAmount{
					{
						Party:         "party2",
						Amount:        "64",
						QuantumAmount: "64",
					},
				},
			},
		},
		TotalFeesPaidAndReceived: []*eventspb.PartyAmount{
			{
				Party:         "party1",
				Amount:        "443",
				QuantumAmount: "443",
			},
			{
				Party:         "party2",
				Amount:        "64",
				QuantumAmount: "64",
			},
		},
	}, eng.GetFeesStatsOnEpochEnd(num.DecimalFromInt64(1)))
}

func testCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewardsBySide(t *testing.T, aggressorSide types.Side) {
	t.Helper()
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)

	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()

	eng.UpdateFeeFactors(types.Fees{
		Factors: &types.FeeFactors{
			MakerFee:          num.DecimalFromFloat(.000250),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			LiquidityFee:      num.DecimalFromFloat(0.001),
		},
	})

	trades := []*types.Trade{
		{
			Aggressor: aggressorSide,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      5,
			Price:     num.NewUint(100000),
		},
	}

	aggressor := "party1"
	if aggressorSide == types.SideBuy {
		aggressor = "party2"
	}

	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.5),
		Maker:     num.NewDecimalFromFloat(0.5),
		Liquidity: num.NewDecimalFromFloat(0.5),
	}).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.25),
		Maker:     num.NewDecimalFromFloat(0.25),
		Liquidity: num.NewDecimalFromFloat(0.25),
	})
	discountRewardService.EXPECT().GetReferrer(types.PartyID(aggressor)).Return(types.PartyID("referrer"), nil).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(types.PartyID("party1")).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.3),
		Maker:     num.NewDecimalFromFloat(0.3),
		Liquidity: num.NewDecimalFromFloat(0.3),
	}).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(types.PartyID("party2")).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.3),
		Maker:     num.NewDecimalFromFloat(0.3),
		Liquidity: num.NewDecimalFromFloat(0.3),
	}).AnyTimes()

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity, reward int

	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
			// lf = 500 before discounts and rewards
			// lf = 500 - 0.5 * 500 = 250 after applying referral discount
			// lf = 250 - 0.25 * 250 = 250 - 62 = 188
			// applying rewards
			// lf = 188 - 188 * 0.3 = 188 - 56 = 132
			assert.Equal(t, num.NewUint(132), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeFeeReferrerRewardPay {
			reward++
			// 14 + 56 + 31 = 96
			require.Equal(t, num.NewUint(98), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeFeeReferrerRewardDistribute {
			reward++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
			// inf = 250 before discounts and rewards
			// inf = 250 - 0.5*250 = 125 after applying referral discount
			// inf = 125 - 0.25*125 = 125-31 = 94
			// applying rewards
			// inf = 94 - 94 *0.3 = 66
			assert.Equal(t, num.NewUint(66), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
			// mf = 125 before discounts and rewards
			// inf = 125 - 0.5*125 = 63 after applying referral discount
			// inf = 63 - 0.25*63 = 63-15 = 48
			// applying rewards
			// inf = 48 - 48 *0.3 = 48 - 14 = 34
			assert.Equal(t, num.NewUint(34), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
			assert.Equal(t, num.NewUint(34), v.Amount.Amount)
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
	assert.Equal(t, reward, 2)
}

func testCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewardsBySideMultipeMakers(t *testing.T, aggressorSide types.Side) {
	t.Helper()
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)

	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()

	eng.UpdateFeeFactors(types.Fees{
		Factors: &types.FeeFactors{
			MakerFee:          num.DecimalFromFloat(.000250),
			InfrastructureFee: num.DecimalFromFloat(0.0005),
			LiquidityFee:      num.DecimalFromFloat(0.001),
		},
	})

	trades := []*types.Trade{
		{
			Aggressor: aggressorSide,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      1,
			Price:     num.NewUint(100000),
		},
		{
			Aggressor: aggressorSide,
			Seller:    "party1",
			Buyer:     "party3",
			Size:      2,
			Price:     num.NewUint(100000),
		},
		{
			Aggressor: aggressorSide,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      2,
			Price:     num.NewUint(100000),
		},
	}

	aggressor := "party1"
	if aggressorSide == types.SideBuy {
		aggressor = "party2"
	}

	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.5),
		Maker:     num.NewDecimalFromFloat(0.5),
		Liquidity: num.NewDecimalFromFloat(0.5),
	}).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.25),
		Maker:     num.NewDecimalFromFloat(0.25),
		Liquidity: num.NewDecimalFromFloat(0.25),
	}).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID("referrer"), nil).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(types.PartyID(aggressor)).Return(types.PartyID("referrer"), nil).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(aggressor).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.3),
		Maker:     num.NewDecimalFromFloat(0.3),
		Liquidity: num.NewDecimalFromFloat(0.3),
	}).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.3),
		Maker:     num.NewDecimalFromFloat(0.3),
		Liquidity: num.NewDecimalFromFloat(0.3),
	}).AnyTimes()

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity, reward int
	totalPaidMakerFee := num.UintZero()
	totalReceivedMakerFee := num.UintZero()

	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
			// lf1 = 100 - 0.5 * 100 = 50
			// lf1 = 50 - 0.25 * 50 = 50-12 = 38
			// lf1 = 38 - 38 * 0.3 = 38 - 11 = 27

			// lf2 = 200 - 0.5 * 200 = 100
			// lf2 = 100 - 0.25 * 100 = 75
			// lf2 = 75 - 75 * 0.3 = 75 - 22 = 53

			// lf3 = 200 - 0.5 * 200 = 100
			// lf3 = 100 - 0.25 * 100 = 75
			// lf3 = 75 - 75 * 0.3 = 75 - 22 = 53
			assert.Equal(t, num.NewUint(133), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
			// inf1 = 50 - 0.5 * 50 = 25
			// inf1 = 25 - 0.25 * 25 = 25-6 = 19
			// inf1 = 19 - 19 * 0.3 = 19 - 5 = 14

			// inf2 = 100 - 0.5 * 100 = 50
			// inf2 = 50 - 0.25 * 50 = 50-12 = 38
			// inf2 = 38 - 38 * 0.3 = 38 - 11 = 27

			// inf3 = 100 - 0.5 * 100 = 50
			// inf3 = 50 - 0.25 * 50 = 50-12 = 38
			// inf3 = 38 - 38 * 0.3 = 38 - 11 = 27
			assert.Equal(t, num.NewUint(68), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
			totalPaidMakerFee.AddSum(v.Amount.Amount)
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
			totalReceivedMakerFee.AddSum(v.Amount.Amount)
		}
		if v.Type == types.TransferTypeFeeReferrerRewardPay {
			reward++
			// 55 + 27 + 13
			assert.Equal(t, num.NewUint(95), v.Amount.Amount)
		}
		if v.Type == types.TransferTypeFeeReferrerRewardDistribute {
			reward++
			// 55 + 27 + 13
			assert.Equal(t, num.NewUint(95), v.Amount.Amount)
		}
	}

	// mf1 = 25 - 0.5 * 25 = 13
	// mf1 = 13 - 0.25 * 13 = 13-3=10
	// mf1 = 10 - 10 * 0.3 = 10 - 3 = 7
	// mf2 = 50 - 0.5 * 50 = 25
	// mf2 = 25 - 0.25 * 25 = 25-6 = 19
	// mf2 = 19 - 19 * 0.3 = 19 - 5 = 14
	// mf3 = 50 - 0.5 * 50 = 25
	// mf3 = 25 - 0.25 * 25 = 25-6 = 19
	// mf3 = 19 - 19 * 0.3 = 19 - 5 = 14
	assert.Equal(t, num.NewUint(35), totalPaidMakerFee)
	assert.Equal(t, num.NewUint(35), totalReceivedMakerFee)

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
	assert.Equal(t, reward, 2)
}

func TestCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewards(t *testing.T) {
	testCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewardsBySide(t, types.SideSell)
	testCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewardsBySide(t, types.SideBuy)
}

func TestCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewardsMultiMakers(t *testing.T) {
	testCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewardsBySideMultipeMakers(t, types.SideSell)
	testCalcContinuousTradingAndCheckAmountsWithDiscountsAndRewardsBySideMultipeMakers(t, types.SideBuy)
}

func testBuildLiquidityFeesRemainder(t *testing.T) {
	eng := getTestFee(t)
	shares := map[string]num.Decimal{
		"lp1": num.DecimalFromFloat(0.8),
		"lp2": num.DecimalFromFloat(0.15),
		"lp3": num.DecimalFromFloat(0.05),
	}
	// amount to distribute
	acc := &types.Account{
		Balance: num.NewUint(1002),
	}
	// 1002 * .8 = 801.6 == 801
	// 1002 * .15 = 150.3 = 150
	// 1002 * 0.05 = 50.1 = 50
	// 801 + 150 + 50 = 1001 -> remainder is 1
	expRemainder := num.NewUint(1)
	expFees := map[string]*num.Uint{
		"lp1": num.NewUint(801),
		"lp2": num.NewUint(150),
		"lp3": num.NewUint(50),
	}
	ft := eng.BuildLiquidityFeeDistributionTransfer(shares, acc)
	got := ft.TotalFeesAmountPerParty()
	for p, amt := range got {
		require.True(t, amt.EQ(expFees[p]))
	}
	// get the total transfer amount from the transfers
	total := num.UintZero()
	for _, t := range ft.Transfers() {
		total.AddSum(t.Amount.Amount)
	}
	rem := num.UintZero().Sub(acc.Balance, total)
	require.True(t, rem.EQ(expRemainder))
}

func testCalcContinuousTrading(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID("party1"), errors.New("not a referrer")).AnyTimes()

	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      10,
			Price:     num.NewUint(10000),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party3",
			Size:      1,
			Price:     num.NewUint(10300),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party4",
			Size:      7,
			Price:     num.NewUint(10300),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      2,
			Price:     num.NewUint(10500),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party5",
			Size:      5,
			Price:     num.NewUint(11000),
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(43928), party1Amount)

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
}

func testCalcContinuousTradingExtended(t *testing.T) {
	eng := getExtendedTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(types.PartyID("party2")).Return(num.DecimalFromFloat(0.00005)).AnyTimes()
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(types.PartyID("party3")).Return(num.DecimalZero()).AnyTimes()
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(types.PartyID("party4")).Return(num.DecimalZero()).AnyTimes()
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(types.PartyID("party5")).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID("party1"), errors.New("not a referrer")).AnyTimes()

	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      10,
			Price:     num.NewUint(10000),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party3",
			Size:      1,
			Price:     num.NewUint(10300),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party4",
			Size:      7,
			Price:     num.NewUint(10300),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      2,
			Price:     num.NewUint(10500),
		},
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party5",
			Size:      5,
			Price:     num.NewUint(11000),
		},
	}

	ft, err := eng.CalculateForContinuousMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(43928), party1Amount)

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity, highMakerPay, highMakerReceive, bb, treasury int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
		}
		if v.Type == types.TransferTypeHighMakerRebatePay {
			highMakerPay++
		}
		if v.Type == types.TransferTypeHighMakerRebateReceive {
			highMakerReceive++
		}
		if v.Type == types.TransferTypeBuyBackFeePay {
			bb++
		}
		if v.Type == types.TransferTypeTreasuryPay {
			treasury++
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, highMakerPay, 2)
	assert.Equal(t, highMakerReceive, 2)
	assert.Equal(t, recv, len(trades))
	assert.Equal(t, pay, len(trades))
	assert.Equal(t, bb, len(trades))
	assert.Equal(t, treasury, len(trades))
}

func testCalcAuctionTradingErrorEmptyTrade(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	_, err := eng.CalculateForAuctionMode([]*types.Trade{}, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())

	eng = getExtendedTestFee(t)
	_, err = eng.CalculateForAuctionMode([]*types.Trade{}, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())
}

func testCalcAuctionTrading(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      1,
			Price:     num.NewUint(100),
		},
	}

	ft, err := eng.CalculateForAuctionMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	// fees are (100 * 0.1 + 100 * 0.05) = 15
	// 15 / 2 = 7.5
	// internally the engine Ceil all fees.
	// so here we will expect 8 for each
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(8), party1Amount)
	party2Amount, ok := feeAmounts["party2"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(8), party2Amount)

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
		}
	}

	assert.Equal(t, liquidity, 2)
	assert.Equal(t, infra, 2)
	assert.Equal(t, recv, 0)
	assert.Equal(t, pay, 0)
}

func testCalcAuctionTradingExtended(t *testing.T) {
	eng := getExtendedTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalFromFloat(0.0025)).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      1,
			Price:     num.NewUint(100),
		},
	}

	ft, err := eng.CalculateForAuctionMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	// fees are (100 * 0.1 + 100 * 0.05) = 15
	// 15 / 2 = 7.5
	// internally the engine Ceil all fees.
	// so here we will expect 8 for each
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(8), party1Amount)
	party2Amount, ok := feeAmounts["party2"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(8), party2Amount)

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity, hmp, hmr, bb, treasury int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
		}
		if v.Type == types.TransferTypeHighMakerRebatePay {
			hmp++
		}
		if v.Type == types.TransferTypeHighMakerRebateReceive {
			hmr++
		}
		if v.Type == types.TransferTypeBuyBackFeePay {
			bb++
		}
		if v.Type == types.TransferTypeTreasuryPay {
			treasury++
		}
	}

	assert.Equal(t, 2, liquidity)
	assert.Equal(t, 2, infra)
	assert.Equal(t, 0, recv)
	assert.Equal(t, 0, pay)
	assert.Equal(t, 0, hmp)
	assert.Equal(t, 0, hmr)
	assert.Equal(t, 2, bb)
	assert.Equal(t, 2, treasury)
}

func TestCalcAuctionTradingWithDiscountsAndRewards(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).DoAndReturn(func(p types.PartyID) types.Factors {
		if p == types.PartyID("party1") {
			return types.EmptyFactors
		} else {
			return types.Factors{
				Infra:     num.NewDecimalFromFloat(0.5),
				Maker:     num.NewDecimalFromFloat(0.5),
				Liquidity: num.NewDecimalFromFloat(0.5),
			}
		}
	}).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).DoAndReturn(func(p types.PartyID) types.Factors {
		if p == types.PartyID("party1") {
			return types.Factors{
				Infra:     num.NewDecimalFromFloat(0.2),
				Maker:     num.NewDecimalFromFloat(0.2),
				Liquidity: num.NewDecimalFromFloat(0.2),
			}
		} else {
			return types.Factors{
				Infra:     num.NewDecimalFromFloat(0.3),
				Maker:     num.NewDecimalFromFloat(0.3),
				Liquidity: num.NewDecimalFromFloat(0.3),
			}
		}
	}).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).DoAndReturn(func(p types.PartyID) (types.PartyID, error) {
		if p == types.PartyID("party1") {
			return types.PartyID("referrer"), nil
		} else {
			return types.PartyID(""), errors.New("No referrer")
		}
	}).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(types.PartyID("party1")).Return(types.Factors{
		Infra:     num.NewDecimalFromFloat(0.5),
		Maker:     num.NewDecimalFromFloat(0.5),
		Liquidity: num.NewDecimalFromFloat(0.5),
	}).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(types.PartyID("party2")).Return(types.EmptyFactors).AnyTimes()

	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    "party1",
			Buyer:     "party2",
			Size:      1,
			Price:     num.NewUint(100),
		},
	}

	ft, err := eng.CalculateForAuctionMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()

	// liquidity fee before discounts = 0.5 * 0.1 * 100 = 5
	// party1
	// lfAfterRefDiscount = 5 - 0
	// lfAfterVolDiscount = 5 - 0.2*5 = 5-1 = 4
	// lfAfterReward = 4 - 0.5*4 = 2

	// party2
	// lfAfterRefDiscount = 5 - 0.5*5 = 3
	// lfAfterVolDiscount = 3 - 0.3*3 = 3-0 = 3
	// lfAfterReward = 3 (no referrer)

	// infra fee before discounts = 0.5 * 0.05 * 100 = 3
	// party1
	// infAfterRefDiscount = 3 - 0
	// infAfterVolDiscount = 3 - 0.2*3 = 3
	// infAfterReward = 3 - 0.5*3 = 2

	// party2
	// infAfterRefDiscount = 3 - 0.5*3 = 2
	// infAfterVolDiscount = 2 - 0.3*2 = 2
	// infAfterReward = 2 (no referrer)

	party1Amount, ok := feeAmounts["party1"]
	require.True(t, ok)
	require.Equal(t, num.NewUint(4), party1Amount)
	party2Amount, ok := feeAmounts["party2"]
	require.True(t, ok)
	require.Equal(t, num.NewUint(5), party2Amount)

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity, reward int
	totalReward := num.UintZero()
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
		}
		if v.Type == types.TransferTypeFeeReferrerRewardPay {
			reward++
			totalReward.AddSum(v.Amount.Amount)
		}
		if v.Type == types.TransferTypeFeeReferrerRewardDistribute {
			reward++
		}
	}
	require.Equal(t, num.NewUint(3), totalReward)
	require.Equal(t, 2, liquidity)
	require.Equal(t, 2, infra)
	require.Equal(t, 0, recv)
	require.Equal(t, 0, pay)
	require.Equal(t, 2, reward)
}

func testCalcBatchAuctionTradingErrorEmptyTrade(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	_, err := eng.CalculateForFrequentBatchesAuctionMode([]*types.Trade{}, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())

	eng = getExtendedTestFee(t)
	_, err = eng.CalculateForFrequentBatchesAuctionMode([]*types.Trade{}, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.EqualError(t, err, fee.ErrEmptyTrades.Error())
}

func testCalcBatchAuctionTradingSameBatch(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	trades := []*types.Trade{
		{
			Aggressor:          types.SideSell,
			Seller:             "party1",
			Buyer:              "party2",
			Size:               1,
			Price:              num.NewUint(100),
			SellerAuctionBatch: 10,
			BuyerAuctionBatch:  10,
		},
	}

	ft, err := eng.CalculateForFrequentBatchesAuctionMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	// fees are (100 * 0.1 + 100 * 0.05) = 15
	// 15 / 2 = 7.5
	// internally the engine Ceil all fees.
	// so here we will expect 8 for each
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(8), party1Amount)
	party2Amount, ok := feeAmounts["party2"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(8), party2Amount)

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
		}
	}

	assert.Equal(t, liquidity, 2)
	assert.Equal(t, infra, 2)
	assert.Equal(t, recv, 0)
	assert.Equal(t, pay, 0)
}

func testCalcBatchAuctionTradingDifferentBatches(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	volumeRebateService := mocks.NewMockVolumeRebateService(ctrl)
	volumeRebateService.EXPECT().VolumeRebateFactorForParty(gomock.Any()).Return(num.DecimalZero()).AnyTimes()
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	trades := []*types.Trade{
		{
			Aggressor:          types.SideSell,
			Seller:             "party1",
			Buyer:              "party2",
			Size:               1,
			Price:              num.NewUint(100),
			SellerAuctionBatch: 11,
			BuyerAuctionBatch:  10,
		},
	}

	ft, err := eng.CalculateForFrequentBatchesAuctionMode(trades, discountRewardService, volumeDiscountService, volumeRebateService)
	assert.NotNil(t, ft)
	assert.Nil(t, err)

	// get the amounts map
	feeAmounts := ft.TotalFeesAmountPerParty()
	// fees are (100 * 0.1 + 100 * 0.05 + 100 *0.02) = 17
	party1Amount, ok := feeAmounts["party1"]
	assert.True(t, ok)
	assert.Equal(t, num.NewUint(17), party1Amount)
	party2Amount, ok := feeAmounts["party2"]
	assert.True(t, ok)
	assert.True(t, party2Amount.IsZero())

	// get the transfer and check we have enough of each types
	transfers := ft.Transfers()
	var pay, recv, infra, liquidity int
	for _, v := range transfers {
		if v.Type == types.TransferTypeLiquidityFeePay {
			liquidity++
		}
		if v.Type == types.TransferTypeInfrastructureFeePay {
			infra++
		}
		if v.Type == types.TransferTypeMakerFeeReceive {
			recv++
		}
		if v.Type == types.TransferTypeMakerFeePay {
			pay++
		}
	}

	assert.Equal(t, liquidity, 1)
	assert.Equal(t, infra, 1)
	assert.Equal(t, recv, 1)
	assert.Equal(t, pay, 1)
}

func testCloseoutFees(t *testing.T) {
	eng := getTestFee(t)
	ctrl := gomock.NewController(t)
	discountRewardService := mocks.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscountService := mocks.NewMockVolumeDiscountService(ctrl)
	discountRewardService.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscountService.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	discountRewardService.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	trades := []*types.Trade{
		{
			Aggressor: types.SideSell,
			Seller:    types.NetworkParty,
			Buyer:     "party1",
			Size:      1,
			Price:     num.NewUint(100),
		},
		{
			Aggressor: types.SideSell,
			Seller:    types.NetworkParty,
			Buyer:     "party2",
			Size:      1,
			Price:     num.NewUint(100),
		},
		{
			Aggressor: types.SideSell,
			Seller:    types.NetworkParty,
			Buyer:     "party3",
			Size:      1,
			Price:     num.NewUint(100),
		},
		{
			Aggressor: types.SideSell,
			Seller:    types.NetworkParty,
			Buyer:     "party4",
			Size:      1,
			Price:     num.NewUint(100),
		},
	}

	ft, fee := eng.GetFeeForPositionResolution(trades)
	assert.NotNil(t, fee)
	allTransfers := ft.Transfers()
	// first we have the network -> pay transfers, then 1 transfer per good party
	// two additional transfers for the buy back and treasury fees
	assert.Equal(t, len(trades), len(allTransfers)-5)
	goodPartyTransfers := allTransfers[5:]
	networkTransfers := allTransfers[:5]

	numTrades := num.NewUint(uint64(len(trades)))
	// maker fee is 100 * 0.02 == 2
	// total network fee is 2 * len(trades)
	feeAmt := num.NewUint(2)
	total := num.UintZero().Mul(feeAmt, numTrades)
	// this must be the total fee for the network
	require.True(t, total.EQ(fee.MakerFee))

	// now check the transfers for the fee payouts
	for _, trans := range goodPartyTransfers {
		require.True(t, trans.Amount.Amount.EQ(feeAmt))
		require.True(t, trans.MinAmount.EQ(num.UintZero()))
	}
	// other fees below, making these transfers may be a bit silly at times...
	// liquidity fees are 100 * 0.1 -> 10 * 4 = 40
	liqFee := num.UintZero().Mul(numTrades, num.NewUint(10))
	require.True(t, liqFee.EQ(fee.LiquidityFee))
	// infraFee is 100 * 0.05 -> 5 * 4 == 20
	infraFee := num.UintZero().Mul(numTrades, num.NewUint(5))
	require.True(t, infraFee.EQ(fee.InfrastructureFee))
	for _, tf := range networkTransfers {
		require.True(t, num.UintZero().EQ(tf.MinAmount))
		switch tf.Type {
		case types.TransferTypeMakerFeePay:
			require.True(t, tf.Amount.Amount.EQ(fee.MakerFee))
		case types.TransferTypeLiquidityFeePay:
			require.True(t, tf.Amount.Amount.EQ(fee.LiquidityFee))
		case types.TransferTypeInfrastructureFeePay:
			require.True(t, tf.Amount.Amount.EQ(fee.InfrastructureFee))
		}
	}
}
