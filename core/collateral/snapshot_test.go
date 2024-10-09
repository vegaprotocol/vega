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

package collateral_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpoint(t *testing.T) {
	eng := getTestEngine(t)
	ctx := context.Background()

	party := "foo"
	bal := num.NewUint(500)
	insBal := num.NewUint(42)
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	acc, err := eng.Engine.CreatePartyGeneralAccount(ctx, party, testMarketAsset)
	assert.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, acc, bal)
	assert.Nil(t, err)

	// create a market then top insurance pool,
	// this should get restored in the global pool
	mktInsAcc, err := eng.GetMarketInsurancePoolAccount(testMarketID, testMarketAsset)
	assert.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, mktInsAcc.ID, insBal)
	assert.Nil(t, err)

	pendingTransfersAcc := eng.GetPendingTransfersAccount(testMarketAsset)
	assert.NoError(t, eng.UpdateBalance(ctx, pendingTransfersAcc.ID, num.NewUint(1789)))

	pendingTransfersAcc = eng.GetPendingTransfersAccount(testMarketAsset)
	assert.NoError(t, eng.UpdateBalance(ctx, pendingTransfersAcc.ID, num.NewUint(1789)))

	// topup the global reward account
	rewardAccount, err := eng.GetGlobalRewardAccount("VOTE")
	assert.Nil(t, err)
	err = eng.Engine.UpdateBalance(ctx, rewardAccount.ID, num.NewUint(10000))
	assert.Nil(t, err)

	// topup the infra fee account for the test asset
	for _, feeAccount := range eng.GetInfraFeeAccountIDs() {
		// restricting the topup and the check later to the test asset because that gets enabled back in the test
		if strings.Contains(feeAccount, testMarketAsset) {
			err = eng.Engine.UpdateBalance(ctx, feeAccount, num.NewUint(12345))
			require.NoError(t, err)
		}
	}

	// topup some reward accounts for markets
	makerReceivedFeeReward1, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market1", types.AccountTypeMakerReceivedFeeReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, makerReceivedFeeReward1.ID, num.NewUint(11111))
	require.NoError(t, err)

	makerReceivedFeeReward2, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market2", types.AccountTypeMakerReceivedFeeReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, makerReceivedFeeReward2.ID, num.NewUint(22222))
	require.NoError(t, err)

	makerPaidFeeReward1, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market3", types.AccountTypeMakerPaidFeeReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, makerPaidFeeReward1.ID, num.NewUint(33333))
	require.NoError(t, err)

	makerPaidFeeReward2, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market4", types.AccountTypeMakerPaidFeeReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, makerPaidFeeReward2.ID, num.NewUint(44444))
	require.NoError(t, err)

	lpFeeReward1, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market5", types.AccountTypeLPFeeReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, lpFeeReward1.ID, num.NewUint(55555))
	require.NoError(t, err)

	lpFeeReward2, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market6", types.AccountTypeLPFeeReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, lpFeeReward2.ID, num.NewUint(66666))
	require.NoError(t, err)

	marketBonusReward1, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market7", types.AccountTypeMarketProposerReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, marketBonusReward1.ID, num.NewUint(77777))
	require.NoError(t, err)

	marketBonusReward2, err := eng.Engine.GetOrCreateRewardAccount(ctx, "VOTE", "market8", types.AccountTypeMarketProposerReward)
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, marketBonusReward2.ID, num.NewUint(88888))
	require.NoError(t, err)

	treasury, err := eng.Engine.GetNetworkTreasuryAccount("VOTE")
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, treasury.ID, num.NewUint(99999))
	require.NoError(t, err)

	ins, err := eng.Engine.GetGlobalInsuranceAccount("VOTE")
	require.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, ins.ID, num.NewUint(100900))
	require.NoError(t, err)

	rewardAccounts := []*types.Account{makerReceivedFeeReward1, makerReceivedFeeReward2, makerPaidFeeReward1, makerPaidFeeReward2, lpFeeReward1, lpFeeReward2, marketBonusReward1, marketBonusReward2}

	checkpoint, err := eng.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, checkpoint)

	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	// system accounts created
	loadEng := collateral.New(logging.NewTestLogger(), conf, eng.timeSvc, eng.broker)
	enableGovernanceAsset(t, loadEng)

	asset := types.Asset{
		ID: testMarketAsset,
		Details: &types.AssetDetails{
			Symbol: testMarketAsset,
		},
	}
	// we need to enable the assets before being able to load the balances
	loadEng.EnableAsset(ctx, asset)
	require.NoError(t, err)

	err = loadEng.Load(ctx, checkpoint)
	require.NoError(t, err)
	loadedPartyAcc, err := loadEng.GetPartyGeneralAccount(party, testMarketAsset)
	require.NoError(t, err)
	require.Equal(t, bal, loadedPartyAcc.Balance)

	loadedIns, err := loadEng.GetGlobalInsuranceAccount(testMarketAsset)
	require.NoError(t, err)
	require.Equal(t, insBal, loadedIns.Balance)

	loadedTreasury, err := loadEng.GetNetworkTreasuryAccount("VOTE")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(99999), loadedTreasury.Balance)

	loadedReward, err := loadEng.GetGlobalRewardAccount("VOTE")
	require.NoError(t, err)
	require.Equal(t, num.NewUint(10000), loadedReward.Balance)

	loadedPendingTransfers := loadEng.GetPendingTransfersAccount(testMarketAsset)
	require.Equal(t, num.NewUint(1789), loadedPendingTransfers.Balance)

	for _, feeAcc := range loadEng.GetInfraFeeAccountIDs() {
		if strings.Contains(feeAcc, testMarketAsset) {
			acc, err := loadEng.GetAccountByID(feeAcc)
			require.NoError(t, err)
			require.Equal(t, num.NewUint(12345), acc.Balance)
		}
	}

	for i, a := range rewardAccounts {
		acc, err := loadEng.GetAccountByID(a.ID)
		require.NoError(t, err)
		require.Equal(t, num.NewUint(uint64((i+1)*11111)), acc.Balance)
	}
}

func TestSnapshots(t *testing.T) {
	t.Run("Creating a snapshot produces the same hash every single time", testSnapshotConsistentHash)
	t.Run("Loading a snapshot should produce the same state - check snapshot after restore", testSnapshotRestore)
}

func testSnapshotConsistentHash(t *testing.T) {
	mkt := "market1"
	ctx := context.Background()
	asset := types.Asset{
		ID: "foo",
		Details: &types.AssetDetails{
			Name:     "foo",
			Symbol:   "FOO",
			Decimals: 5,
			Quantum:  num.DecimalFromFloat(1),
			Source: types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.NewUint(100000000),
				},
			},
		},
	}
	eng := getTestEngine(t)
	// create assets, accounts, and update balances
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	require.NoError(t, eng.EnableAsset(ctx, asset))
	parties := []string{
		"party1",
		"party2",
		"party3",
	}
	balances := map[string]map[types.AccountType]*num.Uint{
		parties[0]: {
			types.AccountTypeGeneral: num.NewUint(500),
			types.AccountTypeMargin:  num.NewUint(500),
		},
		parties[1]: {
			types.AccountTypeGeneral: num.NewUint(1000),
		},
		parties[2]: {
			types.AccountTypeGeneral: num.NewUint(100000),
			types.AccountTypeBond:    num.NewUint(100),
			types.AccountTypeMargin:  num.NewUint(500),
		},
	}
	inc := num.NewUint(50)
	var last string
	for _, p := range parties {
		// always create general account first
		if gb, ok := balances[p][types.AccountTypeGeneral]; ok {
			id, err := eng.CreatePartyGeneralAccount(ctx, p, asset.ID)
			require.NoError(t, err)
			require.NoError(t, eng.IncrementBalance(ctx, id, gb))
			last = id
		}
		for tp, b := range balances[p] {
			switch tp {
			case types.AccountTypeGeneral:
				continue
			case types.AccountTypeMargin:
				id, err := eng.CreatePartyMarginAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, eng.IncrementBalance(ctx, id, b))
				last = id
			case types.AccountTypeBond:
				id, err := eng.CreatePartyBondAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, eng.IncrementBalance(ctx, id, b))
				last = id
			}
		}
	}
	keys := eng.Keys()
	data := make(map[string][]byte, len(keys))
	for _, k := range keys {
		state, _, err := eng.GetState(k)
		require.NoError(t, err)
		data[k] = state
	}
	// now no changes, check hashes again:
	for k, d := range data {
		state, _, err := eng.GetState(k)
		require.NoError(t, err)
		require.EqualValues(t, d, state)
	}
	// now change one account:
	require.NoError(t, eng.IncrementBalance(ctx, last, inc))
	changes := 0
	for k, d := range data {
		got, _, err := eng.GetState(k)
		require.NoError(t, err)
		if !bytes.Equal(d, got) {
			changes++
		}
	}
	require.Equal(t, 1, changes)
}

func testSnapshotRestore(t *testing.T) {
	mkt := "market1"
	ctx := context.Background()
	erc20 := types.AssetDetailsErc20{
		ERC20: &types.ERC20{
			ContractAddress: "0x6d53C489bbda35B8096C8b4Cb362e2889F82E19B",
			ChainID:         "1",
		},
	}
	asset := types.Asset{
		ID: "foo",
		Details: &types.AssetDetails{
			Name:     "foo",
			Symbol:   "FOO",
			Decimals: 5,
			Quantum:  num.DecimalFromFloat(1),
			Source:   erc20,
		},
	}
	eng := getTestEngine(t)
	// create assets, accounts, and update balances
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	require.NoError(t, eng.EnableAsset(ctx, asset))
	parties := []string{
		"party1",
		"party2",
		"party3",
		"*",
	}
	balances := map[string]map[types.AccountType]*num.Uint{
		parties[0]: {
			types.AccountTypeGeneral: num.NewUint(500),
			types.AccountTypeMargin:  num.NewUint(500),
		},
		parties[1]: {
			types.AccountTypeGeneral: num.NewUint(1000),
		},
		parties[2]: {
			types.AccountTypeGeneral: num.NewUint(100000),
			types.AccountTypeBond:    num.NewUint(100),
			types.AccountTypeMargin:  num.NewUint(500),
		},
		"*": {
			types.AccountTypeBuyBackFees: num.NewUint(1000),
		},
	}
	inc := num.NewUint(50)
	var last string
	for _, p := range parties {
		// always create general account first
		if gb, ok := balances[p][types.AccountTypeGeneral]; ok && p != "!" {
			id, err := eng.CreatePartyGeneralAccount(ctx, p, asset.ID)
			require.NoError(t, err)
			require.NoError(t, eng.IncrementBalance(ctx, id, gb))
			last = id
		}
		for tp, b := range balances[p] {
			switch tp {
			case types.AccountTypeGeneral:
				continue
			case types.AccountTypeMargin:
				id, err := eng.CreatePartyMarginAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, eng.IncrementBalance(ctx, id, b))
				last = id
			case types.AccountTypeBond:
				id, err := eng.CreatePartyBondAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, eng.IncrementBalance(ctx, id, b))
				last = id
			case types.AccountTypeBuyBackFees:
				id := eng.GetOrCreateBuyBackFeesAccountID(ctx, asset.ID)
				require.NoError(t, eng.IncrementBalance(ctx, id, b))
				last = id
			}
		}
	}
	// earmark 500 out of the 1000 in the buy back account
	_, err := eng.EarmarkForAutomatedPurchase(asset.ID, types.AccountTypeBuyBackFees, num.UintZero(), num.NewUint(500))
	require.NoError(t, err)

	keys := eng.Keys()
	payloads := make(map[string]*types.Payload, len(keys))
	data := make(map[string][]byte, len(keys))
	for _, k := range keys {
		payloads[k] = &types.Payload{}
		s, _, err := eng.GetState(k)
		require.NoError(t, err)
		data[k] = s
	}
	newEng := getTestEngine(t)
	// we expect 2 batches of events to be sent

	newEng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	newEng.broker.EXPECT().SendBatch(gomock.Any()).Times(2)
	for k, pl := range payloads {
		state := data[k]
		ptype := pl.IntoProto()
		require.NoError(t, proto.Unmarshal(state, ptype))
		payloads[k] = types.PayloadFromProto(ptype)
		_, err := newEng.LoadState(ctx, payloads[k])
		require.NoError(t, err)
	}
	for k, d := range data {
		got, _, err := newEng.GetState(k)
		require.NoError(t, err)
		require.EqualValues(t, d, got)
	}
	require.NoError(t, eng.IncrementBalance(ctx, last, inc))
	// unearmark 200 on eng
	require.NoError(t, eng.UnearmarkForAutomatedPurchase(asset.ID, types.AccountTypeBuyBackFees, num.NewUint(200)))
	// now we expect 1 different hash
	diff := 0
	for k, h := range data {
		old, _, err := eng.GetState(k)
		require.NoError(t, err)
		reload, _, err := newEng.GetState(k)
		require.NoError(t, err)
		if !bytes.Equal(h, old) {
			diff++
			require.NotEqualValues(t, reload, old)
		}
	}
	require.Equal(t, 1, diff)
	require.NoError(t, newEng.IncrementBalance(ctx, last, inc))
	require.NoError(t, newEng.UnearmarkForAutomatedPurchase(asset.ID, types.AccountTypeBuyBackFees, num.NewUint(200)))
	// now the state should match up once again
	for k := range data {
		old, _, err := eng.GetState(k)
		require.NoError(t, err)
		restore, _, err := newEng.GetState(k)
		require.NoError(t, err)
		require.EqualValues(t, old, restore)
	}
}

func TestSnapshotRoundTripViaEngine(t *testing.T) {
	mkt := "market1"
	ctx := vgtest.VegaContext("chainid", 100)

	erc20 := types.AssetDetailsErc20{
		ERC20: &types.ERC20{
			ContractAddress: "0x6d53C489bbda35B8096C8b4Cb362e2889F82E19B",
			ChainID:         "1",
		},
	}
	asset := types.Asset{
		ID: "foo",
		Details: &types.AssetDetails{
			Name:     "foo",
			Symbol:   "FOO",
			Decimals: 5,
			Quantum:  num.DecimalFromFloat(1),
			Source:   erc20,
		},
	}
	collateralEngine1 := getTestEngine(t)
	// create assets, accounts, and update balances
	collateralEngine1.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	require.NoError(t, collateralEngine1.EnableAsset(ctx, asset))
	parties := []string{
		"party1",
		"party2",
		"party3",
	}
	balances := map[string]map[types.AccountType]*num.Uint{
		parties[0]: {
			types.AccountTypeGeneral: num.NewUint(500),
			types.AccountTypeMargin:  num.NewUint(500),
		},
		parties[1]: {
			types.AccountTypeGeneral: num.NewUint(1000),
		},
		parties[2]: {
			types.AccountTypeGeneral: num.NewUint(100000),
			types.AccountTypeBond:    num.NewUint(100),
			types.AccountTypeMargin:  num.NewUint(500),
		},
	}
	for _, p := range parties {
		// always create general account first
		if gb, ok := balances[p][types.AccountTypeGeneral]; ok {
			id, err := collateralEngine1.CreatePartyGeneralAccount(ctx, p, asset.ID)
			require.NoError(t, err)
			require.NoError(t, collateralEngine1.IncrementBalance(ctx, id, gb))
		}
		for tp, b := range balances[p] {
			switch tp {
			case types.AccountTypeGeneral:
				continue
			case types.AccountTypeMargin:
				id, err := collateralEngine1.CreatePartyMarginAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, collateralEngine1.IncrementBalance(ctx, id, b))
			case types.AccountTypeBond:
				id, err := collateralEngine1.CreatePartyBondAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, collateralEngine1.IncrementBalance(ctx, id, b))
			}
		}
	}

	newAsset := types.Asset{
		ID: "foo2",
		Details: &types.AssetDetails{
			Name:     "foo2",
			Symbol:   "FOO2",
			Decimals: 5,
			Quantum:  num.DecimalFromFloat(2),
			Source:   erc20,
		},
	}

	// setup snapshot engine
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	vegaPath := paths.New(t.TempDir())
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snapshot.DefaultConfig()

	snapshotEngine1, err := snapshot.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine1.AddProviders(collateralEngine1.Engine)
	snapshotEngine1CloseFn := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer snapshotEngine1CloseFn()

	require.NoError(t, snapshotEngine1.Start(ctx))

	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	require.NoError(t, collateralEngine1.EnableAsset(ctx, newAsset))

	id, err := collateralEngine1.CreatePartyGeneralAccount(ctx, "party4", newAsset.ID)
	require.NoError(t, err)
	require.NoError(t, collateralEngine1.IncrementBalance(ctx, id, num.NewUint(100)))

	state1 := map[string][]byte{}
	for _, key := range collateralEngine1.Keys() {
		state, additionalProvider, err := collateralEngine1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	snapshotEngine1CloseFn()

	collateralEngine2 := getTestEngine(t)
	collateralEngine2.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	collateralEngine2.broker.EXPECT().Send(gomock.Any()).AnyTimes()

	snapshotEngine2, err := snapshot.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	defer snapshotEngine2.Close()

	snapshotEngine2.AddProviders(collateralEngine2.Engine)

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	require.NoError(t, collateralEngine2.EnableAsset(ctx, newAsset))

	id2, err := collateralEngine2.CreatePartyGeneralAccount(ctx, "party4", newAsset.ID)
	require.NoError(t, err)
	require.NoError(t, collateralEngine2.IncrementBalance(ctx, id2, num.NewUint(100)))

	state2 := map[string][]byte{}
	for _, key := range collateralEngine2.Keys() {
		state, additionalProvider, err := collateralEngine2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
