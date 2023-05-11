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

package collateral_test

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/config/encoding"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"

	snp "code.vegaprotocol.io/vega/core/snapshot"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/proto"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpoint(t *testing.T) {
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
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

	loadedGlobRewardPool, err := loadEng.GetGlobalRewardAccount(testMarketAsset)
	require.NoError(t, err)
	require.Equal(t, insBal, loadedGlobRewardPool.Balance)
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
	defer eng.ctrl.Finish()
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
	defer eng.ctrl.Finish()
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
	payloads := make(map[string]*types.Payload, len(keys))
	data := make(map[string][]byte, len(keys))
	for _, k := range keys {
		payloads[k] = &types.Payload{}
		s, _, err := eng.GetState(k)
		require.NoError(t, err)
		data[k] = s
	}
	newEng := getTestEngine(t)
	defer newEng.ctrl.Finish()
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
	// now the state should match up once again
	for k := range data {
		old, _, err := eng.GetState(k)
		require.NoError(t, err)
		restore, _, err := newEng.GetState(k)
		require.NoError(t, err)
		require.EqualValues(t, old, restore)
	}
}

func TestSnapshotRoundtripViaEngine(t *testing.T) {
	mkt := "market1"
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")

	erc20 := types.AssetDetailsErc20{
		ERC20: &types.ERC20{
			ContractAddress: "0x6d53C489bbda35B8096C8b4Cb362e2889F82E19B",
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
	defer eng.ctrl.Finish()
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
	for _, p := range parties {
		// always create general account first
		if gb, ok := balances[p][types.AccountTypeGeneral]; ok {
			id, err := eng.CreatePartyGeneralAccount(ctx, p, asset.ID)
			require.NoError(t, err)
			require.NoError(t, eng.IncrementBalance(ctx, id, gb))
		}
		for tp, b := range balances[p] {
			switch tp {
			case types.AccountTypeGeneral:
				continue
			case types.AccountTypeMargin:
				id, err := eng.CreatePartyMarginAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, eng.IncrementBalance(ctx, id, b))
			case types.AccountTypeBond:
				id, err := eng.CreatePartyBondAccount(ctx, p, mkt, asset.ID)
				require.NoError(t, err)
				require.NoError(t, eng.IncrementBalance(ctx, id, b))
			}
		}
	}

	// setup snapshot engine
	now := time.Now()
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.NewDefaultConfig()
	config.Storage = "memory"
	snapshotEngine, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngine.AddProviders(eng.Engine)
	snapshotEngine.ClearAndInitialise()
	defer snapshotEngine.Close()

	_, err := snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	snaps, err := snapshotEngine.List()
	require.NoError(t, err)
	snap1 := snaps[0]

	engLoad := getTestEngine(t)
	engLoad.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	engLoad.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	snapshotEngineLoad, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngineLoad.AddProviders(engLoad.Engine)
	snapshotEngineLoad.ClearAndInitialise()
	snapshotEngineLoad.ReceiveSnapshot(snap1)
	snapshotEngineLoad.ApplySnapshot(ctx)
	snapshotEngineLoad.CheckLoaded()
	defer snapshotEngineLoad.Close()

	// verify snapshot is equal right after loading
	b, err := snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	bLoad, err := snapshotEngineLoad.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))

	// now make some changes and recheck
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

	require.NoError(t, eng.EnableAsset(ctx, newAsset))
	require.NoError(t, engLoad.EnableAsset(ctx, newAsset))

	id, err := eng.CreatePartyGeneralAccount(ctx, "party4", newAsset.ID)
	require.NoError(t, err)
	require.NoError(t, eng.IncrementBalance(ctx, id, num.NewUint(100)))

	id2, err := engLoad.CreatePartyGeneralAccount(ctx, "party4", newAsset.ID)
	require.NoError(t, err)
	require.NoError(t, engLoad.IncrementBalance(ctx, id2, num.NewUint(100)))

	// verify snapshot is equal right after changes made
	b, err = snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	bLoad, err = snapshotEngineLoad.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))
}
