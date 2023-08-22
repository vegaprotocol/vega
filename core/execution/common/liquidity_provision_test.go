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

package common_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/fee"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"

	vgcrypto "code.vegaprotocol.io/vega/libs/crypto"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	cmocks "code.vegaprotocol.io/vega/core/collateral/mocks"
	"code.vegaprotocol.io/vega/core/liquidity/v2"
	lmocks "code.vegaprotocol.io/vega/core/liquidity/v2/mocks"
)

type marketLiquidityTest struct {
	ctrl     *gomock.Controller
	ctx      context.Context
	marketID string
	asset    string

	marketLiquidity *common.MarketLiquidity

	liquidityEngine  *mocks.MockLiquidityEngine
	collateralEngine common.Collateral
	epochEngine      *mocks.MockEpochEngine
	equityShares     *mocks.MockEquityLikeShares
	broker           *bmocks.MockBroker
	orderBook        *lmocks.MockOrderBook
	timeService      *cmocks.MockTimeService
}

func newMarketLiquidity(t *testing.T) *marketLiquidityTest {
	t.Helper()

	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()

	liquidityEngine := mocks.NewMockLiquidityEngine(ctrl)
	epochEngine := mocks.NewMockEpochEngine(ctrl)
	equityShares := mocks.NewMockEquityLikeShares(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	orderBook := lmocks.NewMockOrderBook(ctrl)
	timeService := cmocks.NewMockTimeService(ctrl)

	collateralEngine := collateral.New(log, collateral.NewDefaultConfig(), timeService, broker)

	marketID := "market-1"
	settlementAsset := "USDT"

	fees, _ := fee.New(log, fee.NewDefaultConfig(), types.Fees{Factors: &types.FeeFactors{}}, settlementAsset, num.DecimalOne())

	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).AnyTimes()

	marketTracker := common.NewMarketActivityTracker(log, epochEngine)

	marketLiquidity := common.NewMarketLiquidity(
		log,
		liquidityEngine,
		collateralEngine,
		broker,
		orderBook,
		equityShares,
		marketTracker,
		fees,
		common.SpotMarketType,
		marketID,
		settlementAsset,
		num.NewUint(1),
		num.NewDecimalFromFloat(0.5),
		time.Second*3,
	)

	marketLiquidity.OnMinLPStakeQuantumMultiple(num.DecimalOne())
	marketLiquidity.OnEarlyExitPenalty(num.DecimalOne())

	return &marketLiquidityTest{
		ctrl:             ctrl,
		marketID:         marketID,
		asset:            settlementAsset,
		marketLiquidity:  marketLiquidity,
		liquidityEngine:  liquidityEngine,
		collateralEngine: collateralEngine,
		equityShares:     equityShares,
		epochEngine:      epochEngine,
		broker:           broker,
		orderBook:        orderBook,
		timeService:      timeService,
		ctx:              context.Background(),
	}
}

func createPartyAndPayLiquidityFee(t *testing.T, amount *num.Uint, testLiquidity *marketLiquidityTest) {
	t.Helper()

	tradingParty := "party-1"
	_, err := testLiquidity.collateralEngine.CreatePartyGeneralAccount(testLiquidity.ctx, tradingParty, testLiquidity.asset)
	assert.NoError(t, err)

	_, err = testLiquidity.collateralEngine.Deposit(testLiquidity.ctx, tradingParty, testLiquidity.asset, amount)
	assert.NoError(t, err)

	_, err = testLiquidity.collateralEngine.GetPartyGeneralAccount(tradingParty, testLiquidity.asset)
	assert.NoError(t, err)

	transfer := testLiquidity.marketLiquidity.NewTransfer(tradingParty, types.TransferTypeLiquidityFeePay, amount.Clone())

	_, err = testLiquidity.collateralEngine.TransferSpotFees(
		testLiquidity.ctx,
		testLiquidity.marketID,
		testLiquidity.asset,
		common.NewFeeTransfer([]*types.Transfer{transfer}, nil),
	)
	assert.NoError(t, err)
}

func TestLiquidityProvisionsFeeDistribution(t *testing.T) {
	testLiquidity := newMarketLiquidity(t)

	weightsPerLP := map[string]num.Decimal{
		"lp-1": num.NewDecimalFromFloat(0.008764241896),
		"lp-2": num.NewDecimalFromFloat(0.0008764241895),
		"lp-3": num.NewDecimalFromFloat(0.0175284838),
		"lp-4": num.NewDecimalFromFloat(0.03505689996),
		"lp-5": num.NewDecimalFromFloat(0.061349693),
		"lp-6": num.NewDecimalFromFloat(0.876424189),
	}

	expectedAllocatedFess := map[string]num.Uint{
		"lp-1": *num.NewUint(1000),
		"lp-2": *num.NewUint(100),
		"lp-3": *num.NewUint(2000),
		"lp-4": *num.NewUint(4000),
		"lp-5": *num.NewUint(7000),
		"lp-6": *num.NewUint(100000),
	}

	expectedDistributedFess := map[string]num.Uint{
		"lp-1": *num.NewUint(13923),
		"lp-2": *num.NewUint(1322),
		"lp-3": *num.NewUint(25061),
		"lp-4": *num.NewUint(44553),
		// expected fee is 29238 but the party will be selected to receive reaming distribution account funds (3).
		"lp-5": *num.NewUint(29241),
		"lp-6": *num.NewUint(0),
	}

	keys := []string{"lp-1", "lp-2", "lp-3", "lp-4", "lp-5", "lp-6"}

	ctx := context.Background()

	testLiquidity.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	testLiquidity.liquidityEngine.EXPECT().UpdatePartyCommitment(gomock.Any(), gomock.Any()).DoAndReturn(
		func(partyID string, amount *num.Uint) (*types.LiquidityProvision, error) {
			return &types.LiquidityProvision{
				Party:            partyID,
				CommitmentAmount: amount.Clone(),
			}, nil
		}).AnyTimes()

	// enable asset first.
	err := testLiquidity.collateralEngine.EnableAsset(ctx, types.Asset{
		ID: testLiquidity.asset,
		Details: &types.AssetDetails{
			Name:     testLiquidity.asset,
			Symbol:   testLiquidity.asset,
			Decimals: 0,
			Source: types.AssetDetailsErc20{
				ERC20: &types.ERC20{
					ContractAddress: "addrs",
				},
			},
		},
	})
	assert.NoError(t, err)

	// create all required accounts for spot market.
	err = testLiquidity.collateralEngine.CreateSpotMarketAccounts(ctx, testLiquidity.marketID, testLiquidity.asset)
	assert.NoError(t, err)

	testLiquidity.liquidityEngine.EXPECT().
		SubmitLiquidityProvision(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
		AnyTimes()

	testLiquidity.liquidityEngine.EXPECT().PendingProvision().Return(nil).AnyTimes()
	one := num.UintOne()
	testLiquidity.liquidityEngine.EXPECT().CalculateSuppliedStakeWithoutPending().Return(one).AnyTimes()
	testLiquidity.liquidityEngine.EXPECT().ApplyPendingProvisions(gomock.Any(), gomock.Any()).Return(nil).AnyTimes()

	testLiquidity.timeService.EXPECT().GetTimeNow().DoAndReturn(func() time.Time {
		return time.Now()
	}).AnyTimes()

	decimalOne := num.DecimalOne()
	uintOne := num.UintOne()
	commitmentAmount := num.NewUint(10)
	scoresPerLP := map[string]num.Decimal{}
	provisionsPerParty := map[string]*types.LiquidityProvision{}

	// create liquidity providers accounts and submit provision.
	for provider := range weightsPerLP {
		// set score to one.
		scoresPerLP[provider] = decimalOne

		// create providers general account and deposit funds into it.
		_, err := testLiquidity.collateralEngine.CreatePartyGeneralAccount(ctx, provider, testLiquidity.asset)
		assert.NoError(t, err)

		_, err = testLiquidity.collateralEngine.Deposit(ctx, provider, testLiquidity.asset, commitmentAmount)
		assert.NoError(t, err)

		// submit the provision.
		provision := &types.LiquidityProvisionSubmission{
			MarketID:         testLiquidity.marketID,
			CommitmentAmount: commitmentAmount,
			Reference:        provider,
		}

		deterministicID := hex.EncodeToString(vgcrypto.Hash([]byte(provider)))
		err = testLiquidity.marketLiquidity.SubmitLiquidityProvision(ctx, provision, provider,
			deterministicID, types.MarketStateActive)
		assert.NoError(t, err)

		// setup provision per party.
		provisionsPerParty[provider] = &types.LiquidityProvision{
			Party:            provider,
			CommitmentAmount: provision.CommitmentAmount.Clone(),
		}
	}

	// create party and make it pay liquidity fee.
	createPartyAndPayLiquidityFee(t, num.NewUint(114101), testLiquidity)

	testLiquidity.liquidityEngine.EXPECT().ProvisionsPerParty().DoAndReturn(func() liquidity.ProvisionsPerParty {
		return provisionsPerParty
	}).AnyTimes()

	testLiquidity.liquidityEngine.EXPECT().ResetSLAEpoch(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// start epoch.
	lastDistributionStep := time.Now()
	now := lastDistributionStep.Add(time.Second * 5)

	testLiquidity.marketLiquidity.OnEpochStart(testLiquidity.ctx, now, uintOne, uintOne, uintOne, decimalOne)

	testLiquidity.liquidityEngine.EXPECT().ResetAverageLiquidityScores()

	testLiquidity.equityShares.EXPECT().AllShares().DoAndReturn(func() map[string]num.Decimal {
		return weightsPerLP
	})

	testLiquidity.liquidityEngine.EXPECT().GetLastFeeDistributionTime().DoAndReturn(func() time.Time {
		return lastDistributionStep
	})

	testLiquidity.liquidityEngine.EXPECT().SetLastFeeDistributionTime(gomock.Any())

	testLiquidity.liquidityEngine.EXPECT().GetAverageLiquidityScores().DoAndReturn(func() map[string]num.Decimal {
		return scoresPerLP
	})

	// trigger a time tick - this should start allocation fees to LP fee accounts.
	testLiquidity.marketLiquidity.OnTick(ctx, now)

	for _, provider := range keys {
		acc, err := testLiquidity.collateralEngine.GetPartyLiquidityFeeAccount(
			testLiquidity.marketID,
			provider,
			testLiquidity.asset,
		)
		assert.NoError(t, err)

		expected := expectedAllocatedFess[provider]
		assert.True(t, expected.EQ(acc.Balance))
	}

	zeroPointFive := num.NewDecimalFromFloat(0.5)
	expectedSLAPenalties := map[string]*liquidity.SlaPenalty{
		"lp-1": {
			Fee:  num.NewDecimalFromFloat(0),
			Bond: zeroPointFive,
		},
		"lp-2": {
			Fee:  num.NewDecimalFromFloat(0.05),
			Bond: zeroPointFive,
		},
		"lp-3": {
			Fee:  num.NewDecimalFromFloat(0.1),
			Bond: zeroPointFive,
		},
		"lp-4": {
			Fee:  num.NewDecimalFromFloat(0.2),
			Bond: zeroPointFive,
		},
		"lp-5": {
			Fee:  num.NewDecimalFromFloat(0.7),
			Bond: zeroPointFive,
		},
		"lp-6": {
			Fee:  num.NewDecimalFromFloat(1),
			Bond: zeroPointFive,
		},
	}

	testLiquidity.liquidityEngine.EXPECT().CalculateSLAPenalties(gomock.Any()).DoAndReturn(
		func(_ time.Time) liquidity.SlaPenalties {
			return liquidity.SlaPenalties{
				PenaltiesPerParty: expectedSLAPenalties,
			}
		},
	)

	testLiquidity.liquidityEngine.EXPECT().
		LiquidityProvisionByPartyID(gomock.Any()).
		DoAndReturn(func(party string) *types.LiquidityProvision {
			return &types.LiquidityProvision{
				ID:               party,
				Party:            party,
				CommitmentAmount: commitmentAmount,
			}
		}).AnyTimes()

	// end epoch - this should trigger the SLA fees distribution.
	testLiquidity.marketLiquidity.OnEpochEnd(testLiquidity.ctx, now)

	for _, provider := range keys {
		generalAcc, err := testLiquidity.collateralEngine.GetPartyGeneralAccount(
			provider,
			testLiquidity.asset,
		)
		assert.NoError(t, err)

		expectedFee := expectedDistributedFess[provider]
		assert.Truef(t, expectedFee.EQ(generalAcc.Balance),
			"party %s general account balance is %s, expected: %s", provider, generalAcc.Balance, expectedFee)

		bondAcc, err := testLiquidity.collateralEngine.GetPartyBondAccount(testLiquidity.marketID, provider, testLiquidity.asset)
		assert.NoError(t, err)

		penalty := expectedSLAPenalties[provider]

		num.UintFromDecimal(penalty.Bond.Mul(commitmentAmount.ToDecimal()))
		expectedBondAccount, _ := num.UintFromDecimal(penalty.Bond.Mul(commitmentAmount.ToDecimal()))
		assert.True(t, bondAcc.Balance.EQ(expectedBondAccount))
	}

	acc, err := testLiquidity.collateralEngine.GetOrCreateLiquidityFeesBonusDistributionAccount(
		ctx,
		testLiquidity.marketID,
		testLiquidity.asset,
	)
	assert.NoError(t, err)
	assert.True(t, acc.Balance.EQ(num.UintZero()))

	testLiquidity.equityShares.EXPECT().SetPartyStake(gomock.Any(), gomock.Any()).AnyTimes()
	testLiquidity.equityShares.EXPECT().AllShares().AnyTimes()
	testLiquidity.marketLiquidity.OnEpochStart(testLiquidity.ctx, now, uintOne, uintOne, uintOne, decimalOne)
}
