package collateral_test

import (
	"context"
	"testing"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/collateral/mocks"
	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	checkpoint "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkpointTestEngine struct {
	*collateral.Engine
	ctrl   *gomock.Controller
	broker *bmocks.MockBroker
}

func newCheckpointTestEngine(t *testing.T) *checkpointTestEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	timeSvc := mocks.NewMockTimeService(ctrl)
	timeSvc.EXPECT().GetTimeNow().AnyTimes()

	broker := bmocks.NewMockBroker(ctrl)
	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}

	broker.EXPECT().Send(gomock.Any()).Times(6)

	e := collateral.New(logging.NewTestLogger(), conf, timeSvc, broker)
	e.EnableAsset(context.Background(), types.Asset{
		ID: "VEGA",
		Details: &types.AssetDetails{
			Name:     "VEGA",
			Symbol:   "VEGA",
			Decimals: 5,
			Quantum:  num.DecimalZero(),
			Source: &types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.UintZero(),
				},
			},
		},
	})

	return &checkpointTestEngine{
		Engine: e,
		ctrl:   ctrl,
		broker: broker,
	}
}

func TestCheckPointLoadingWithAlias(t *testing.T) {
	e := newCheckpointTestEngine(t)

	e.broker.EXPECT().Send(gomock.Any()).Times(7).Do(func(e events.Event) {
		ledgerMovmenentsE, ok := e.(*events.LedgerMovements)
		if !ok {
			return
		}

		mvts := ledgerMovmenentsE.LedgerMovements()
		assert.Len(t, mvts, 4)
		assert.Len(t, mvts[0].Entries, 1)
		// no owner + from externa
		assert.Nil(t, mvts[0].Entries[0].FromAccount.Owner)
		assert.Equal(t, mvts[0].Entries[0].FromAccount.Type, types.AccountTypeExternal)
		assert.Equal(t, mvts[0].Entries[0].Amount, "1000")
		// to no owner + to reward
		assert.Nil(t, mvts[0].Entries[0].ToAccount.Owner)
		assert.Equal(t, mvts[0].Entries[0].ToAccount.Type, types.AccountTypeNetworkTreasury)

		// second transfer
		assert.Len(t, mvts[1].Entries, 1)
		// no owner + from external
		assert.Nil(t, mvts[1].Entries[0].FromAccount.Owner)
		assert.Equal(t, mvts[1].Entries[0].FromAccount.Type, types.AccountTypeExternal)
		assert.Equal(t, mvts[1].Entries[0].Amount, "2000")
		// to no owner + to reward
		assert.Nil(t, mvts[1].Entries[0].ToAccount.Owner)
		assert.Equal(t, mvts[1].Entries[0].ToAccount.Type, types.AccountTypeNetworkTreasury)

		// third transfer
		assert.Len(t, mvts[2].Entries, 1)
		// no owner + from external
		assert.Nil(t, mvts[2].Entries[0].FromAccount.Owner)
		assert.Equal(t, mvts[2].Entries[0].FromAccount.Type, types.AccountTypeExternal)
		assert.Equal(t, mvts[2].Entries[0].Amount, "9000")
		// to no owner + to global insurnace
		assert.Nil(t, mvts[2].Entries[0].ToAccount.Owner)
		assert.Equal(t, mvts[2].Entries[0].ToAccount.Type, types.AccountTypeGlobalInsurance)
	})

	ab := []*checkpoint.AssetBalance{
		{Party: "*", Asset: "VEGA", Balance: "1000"},
		{Party: "*ACCOUNT_TYPE_NETWORK_TREASURY", Asset: "VEGA", Balance: "2000"},
		{Party: "*ACCOUNT_TYPE_GLOBAL_INSURANCE", Asset: "VEGA", Balance: "9000"},
		// covers for vesting accounts
		{Party: "vesting6d449ee7716fc5c740b2fe7596ceb91d671ec6f7b9d771edf4a610829bb8a658", Asset: "VEGA", Balance: "4242424"},
	}

	msg := &checkpoint.Collateral{
		Balances: ab,
	}

	ret, err := proto.Marshal(msg)
	require.NoError(t, err)

	e.Load(context.Background(), ret)

	acc, err := e.GetNetworkTreasuryAccount("VEGA")
	require.NoError(t, err)
	require.Equal(t, "3000", acc.Balance.String())

	acc, err = e.GetGlobalInsuranceAccount("VEGA")
	require.NoError(t, err)
	require.Equal(t, "9000", acc.Balance.String())

	acc = e.GetOrCreatePartyVestingRewardAccount(
		context.Background(),
		"6d449ee7716fc5c740b2fe7596ceb91d671ec6f7b9d771edf4a610829bb8a658",
		"VEGA",
	)

	require.Equal(t, "4242424", acc.Balance.String())

	_, err = e.GetPartyGeneralAccount("*ACCOUNT_TYPE_GLOBAL_REWARD", "VEGA")
	require.Error(t, err)
}

type feesTransfer struct {
	totalFeesAmountsPerParty map[string]*num.Uint
	transfers                []*types.Transfer
}

func (f *feesTransfer) TotalFeesAmountPerParty() map[string]*num.Uint {
	ret := make(map[string]*num.Uint, len(f.totalFeesAmountsPerParty))
	for k, v := range f.totalFeesAmountsPerParty {
		ret[k] = v.Clone()
	}
	return ret
}
func (f *feesTransfer) Transfers() []*types.Transfer { return f.transfers }

// TestCheckPointWithUndistributedLPFees takes a checkpoint with undistributed balance in the lp fees account of a market and verifies that it goes
// back to the network treasury of the asset as takes a checkpoint.
func TestCheckPointWithUndistributedLPFees(t *testing.T) {
	e := newCheckpointTestEngine(t)

	e.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	asset1 := types.Asset{
		ID: "MYASSET1",
		Details: &types.AssetDetails{
			Symbol: "MYASSET1",
		},
	}
	err := e.EnableAsset(context.Background(), asset1)
	require.NoError(t, err)

	asset2 := types.Asset{
		ID: "MYASSET2",
		Details: &types.AssetDetails{
			Symbol: "MYASSET2",
		},
	}
	err = e.EnableAsset(context.Background(), asset2)
	e.EnableAsset(context.Background(), asset2)
	require.NoError(t, err)

	// create necessary accounts
	_, _, err = e.CreateMarketAccounts(context.Background(), "market1", "MYASSET1")
	require.NoError(t, err)

	_, _, err = e.CreateMarketAccounts(context.Background(), "market2", "MYASSET1")
	require.NoError(t, err)

	_, _, err = e.CreateMarketAccounts(context.Background(), "market3", "MYASSET2")
	require.NoError(t, err)

	_, err = e.CreatePartyGeneralAccount(context.Background(), "zohar", "MYASSET1")
	require.NoError(t, err)

	_, err = e.CreatePartyGeneralAccount(context.Background(), "zohar", "MYASSET2")
	require.NoError(t, err)

	marginAccount1, err := e.CreatePartyMarginAccount(context.Background(), "zohar", "market1", "MYASSET1")
	require.NoError(t, err)
	e.IncrementBalance(context.Background(), marginAccount1, num.NewUint(500000))

	marginAccount2, err := e.CreatePartyMarginAccount(context.Background(), "zohar", "market2", "MYASSET1")
	require.NoError(t, err)
	e.IncrementBalance(context.Background(), marginAccount2, num.NewUint(500000))

	marginAccount3, err := e.CreatePartyMarginAccount(context.Background(), "zohar", "market3", "MYASSET2")
	require.NoError(t, err)
	e.IncrementBalance(context.Background(), marginAccount3, num.NewUint(500000))

	_, err = e.GetOrCreateLiquidityFeesBonusDistributionAccount(context.Background(), "market1", "MYASSET1")
	require.NoError(t, err)

	partyLiquidityFeeAccountID, err := e.CreatePartyLiquidityFeeAccount(context.Background(), "zohar", "market1", "MYASSET1")
	require.NoError(t, err)

	e.IncrementBalance(context.Background(), partyLiquidityFeeAccountID, num.NewUint(1234))

	// setup some balance on the LP fee pay account for MYASSET1/market1
	lpTransfers := &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "MYASSET1",
			Amount: num.NewUint(2000),
		},
		Type: types.TransferTypeLiquidityFeePay,
	}
	_, err = e.TransferFees(context.Background(), "market1", "MYASSET1", &feesTransfer{transfers: []*types.Transfer{lpTransfers}})
	require.NoError(t, err)

	// artificially fill the LP fee account for spots to demonstrate that the unpaid collected goes to the network treasury and what's left
	// on the party LP fee account goes to the general party account
	lpSpotTransfers := &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "MYASSET1",
			Amount: num.NewUint(1230),
		},
		Type: types.TransferTypeLiquidityFeeUnpaidCollect,
	}
	_, err = e.TransferSpotFees(context.Background(), "market1", "MYASSET1", &feesTransfer{transfers: []*types.Transfer{lpSpotTransfers}})
	require.NoError(t, err)

	// setup some balance on the LP fee pay account for MYASSET1/market2
	lpTransfers = &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "MYASSET1",
			Amount: num.NewUint(3000),
		},
		Type: types.TransferTypeLiquidityFeePay,
	}
	_, err = e.TransferFees(context.Background(), "market2", "MYASSET1", &feesTransfer{transfers: []*types.Transfer{lpTransfers}})
	require.NoError(t, err)

	// setup some balance on the LP fee pay account for MYASSET1/market1
	lpTransfers = &types.Transfer{
		Owner: "zohar",
		Amount: &types.FinancialAmount{
			Asset:  "MYASSET2",
			Amount: num.NewUint(7000),
		},
		Type: types.TransferTypeLiquidityFeePay,
	}
	_, err = e.TransferFees(context.Background(), "market3", "MYASSET2", &feesTransfer{transfers: []*types.Transfer{lpTransfers}})
	require.NoError(t, err)

	// take a checkpoint, at this point we expect the funds to be dropped into the network treasury of the asset2.
	ret, err := e.Checkpoint()
	require.NoError(t, err)

	e.Load(context.Background(), ret)

	netTreasury1, err := e.GetNetworkTreasuryAccount("MYASSET1")
	require.NoError(t, err)
	require.Equal(t, "6230", netTreasury1.Balance.String())

	netTreasury2, err := e.GetNetworkTreasuryAccount("MYASSET2")
	require.NoError(t, err)
	require.Equal(t, "7000", netTreasury2.Balance.String())

	// 1000000 - 5000 + 4
	acc, err := e.GetPartyGeneralAccount("zohar", "MYASSET1")
	require.NoError(t, err)
	require.Equal(t, "995004", acc.Balance.String())
}
