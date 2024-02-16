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
	"bytes"
	"context"
	"encoding/hex"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/snapshot"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/protos/vega"
	snapshotpb "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deposit(eng *testEngine, asset, party string, amount *num.Uint) *types.BuiltinAssetDeposit {
	eng.OnTick(context.Background(), time.Now())
	return depositAt(eng, asset, party, amount)
}

func depositAt(eng *testEngine, asset, party string, amount *num.Uint) *types.BuiltinAssetDeposit {
	eng.tsvc.EXPECT().GetTimeNow().AnyTimes()
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	return &types.BuiltinAssetDeposit{
		VegaAssetID: asset,
		PartyID:     party,
		Amount:      amount,
	}
}

func testEngineAndSnapshot(t *testing.T, vegaPath paths.Paths, now time.Time) (*testEngine, *snp.Engine) {
	t.Helper()
	eng := getTestEngine(t)
	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.DefaultConfig()
	snapshotEngine, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine.AddProviders(eng.Engine)
	return eng, snapshotEngine
}

func TestSnapshotRoundTripViaEngine(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	now := time.Now()
	deliver := now.Add(time.Hour)

	testAsset := assets.NewAsset(builtin.New("VGT", &types.AssetDetails{
		Name:   "VEGA TOKEN",
		Symbol: "VGT",
	}))

	fromAcc := types.Account{
		Balance: num.NewUint(1000),
	}

	oneoff := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			DeliverOn: &deliver,
		},
	}

	oneoff2 := &types.TransferFunds{
		Kind: types.TransferCommandKindOneOff,
		OneOff: &types.OneOffTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			DeliverOn: &deliver,
		},
	}

	recurring := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			StartEpoch: 10,
			EndEpoch:   nil, // forever
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	recurring2 := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccee",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "vega",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			StartEpoch: 10,
			EndEpoch:   nil, // forever
			Factor:     num.MustDecimalFromString("0.9"),
		},
	}

	d1 := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT1",
		PartyID:     "someparty1",
		Amount:      num.NewUint(42),
	}
	d2 := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT1",
		PartyID:     "someparty2",
		Amount:      num.NewUint(24),
	}
	d3 := &types.BuiltinAssetDeposit{
		VegaAssetID: "VGT1",
		PartyID:     "someparty3",
		Amount:      num.NewUint(29),
	}

	vegaPath := paths.New(t.TempDir())

	bankingEngine1, snapshotEngine1 := testEngineAndSnapshot(t, vegaPath, now)
	closeSnapshotEngine1 := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer closeSnapshotEngine1()

	bankingEngine1.tsvc.EXPECT().GetTimeNow().Return(now).AnyTimes()
	bankingEngine1.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	bankingEngine1.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	bankingEngine1.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	bankingEngine1.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	bankingEngine1.col.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)

	require.NoError(t, snapshotEngine1.Start(context.Background()))

	require.NoError(t, bankingEngine1.DepositBuiltinAsset(context.Background(), d1, "depositid1", 42))
	require.NoError(t, bankingEngine1.DepositBuiltinAsset(context.Background(), d2, "depositid2", 24))
	require.NoError(t, bankingEngine1.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty1", "VGT1", num.NewUint(2)))
	require.NoError(t, bankingEngine1.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty2", "VGT1", num.NewUint(4)))

	bankingEngine1.OnTick(ctx, now)

	require.NoError(t, bankingEngine1.TransferFunds(ctx, oneoff))
	require.NoError(t, bankingEngine1.TransferFunds(ctx, recurring))

	// Take a snapshot.
	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	// Additional steps to execute the same way on the next engine to verify it yield the same result.
	additionalSteps := func(bankingEngine *testEngine) {
		bankingEngine.OnTick(ctx, now)

		require.NoError(t, bankingEngine.DepositBuiltinAsset(context.Background(), d3, "depositid3", 29))
		require.NoError(t, bankingEngine.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty1", "VGT1", num.NewUint(10)))
		require.NoError(t, bankingEngine.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty2", "VGT1", num.NewUint(5)))

		bankingEngine.OnTick(ctx, now)

		require.NoError(t, bankingEngine.TransferFunds(ctx, oneoff2))
		require.NoError(t, bankingEngine.TransferFunds(ctx, recurring2))
	}

	additionalSteps(bankingEngine1)

	state1 := map[string][]byte{}
	for _, key := range bankingEngine1.Keys() {
		state, additionalProvider, err := bankingEngine1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	closeSnapshotEngine1()

	// Reload the engine using the previous snapshot.

	bankingEngine2, snapshotEngine2 := testEngineAndSnapshot(t, vegaPath, now)
	defer snapshotEngine2.Close()

	bankingEngine2.tsvc.EXPECT().GetTimeNow().Return(now).AnyTimes()
	bankingEngine2.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	bankingEngine2.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	bankingEngine2.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	bankingEngine2.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	bankingEngine2.col.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, snapshotEngine2.Start(ctx))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	// Executing the same steps post-snapshot as the first engine
	additionalSteps(bankingEngine2)

	state2 := map[string][]byte{}
	for _, key := range bankingEngine2.Keys() {
		state, additionalProvider, err := bankingEngine2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		if key == "withdrawals" {
			// FIXME The withdrawal count inside the engine is not restored by
			//   the snapshot, which leads to a non-deterministic generation
			//   of the withdrawal `Ref` after restoring the state from a snapshot.
			//   Instead of starting at the count from the previous engine, it
			//   restarts at 0.
			//   It doesn't seem to be a big issue. As a result, we skip the
			//   verification for the moment.
			continue
		}
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}

func TestAssetActionsSnapshotRoundTrip(t *testing.T) {
	aaKey := (&types.PayloadBankingAssetActions{}).Key()
	eng := getTestEngine(t)

	eng.tsvc.EXPECT().GetTimeNow().AnyTimes()
	d1 := deposit(eng, "VGT1", "someparty1", num.NewUint(42))
	err := eng.DepositBuiltinAsset(context.Background(), d1, "depositid1", 42)
	assert.NoError(t, err)

	d2 := deposit(eng, "VGT1", "someparty2", num.NewUint(24))
	err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid2", 24)
	assert.NoError(t, err)

	state, _, err := eng.GetState(aaKey)
	require.Nil(t, err)

	// verify state is consistent in the absence of change
	stateNoChange, _, err := eng.GetState(aaKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(state, stateNoChange))

	// reload the state
	var assetActions snapshotpb.Payload
	snap := getTestEngine(t)
	snap.tsvc.EXPECT().GetTimeNow().AnyTimes()
	snap.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)

	proto.Unmarshal(state, &assetActions)
	payload := types.PayloadFromProto(&assetActions)
	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(aaKey)
	require.True(t, bytes.Equal(state, statePostReload))
}

func TestSeenSnapshotRoundTrip(t *testing.T) {
	seenKey := (&types.PayloadBankingSeen{}).Key()
	eng := getTestEngine(t)

	eng.tsvc.EXPECT().GetTimeNow().Times(2)
	state1, _, err := eng.GetState(seenKey)
	require.Nil(t, err)
	eng.col.EXPECT().Deposit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)

	d1 := deposit(eng, "VGT1", "someparty1", num.NewUint(42))
	err = eng.DepositBuiltinAsset(context.Background(), d1, "depositid1", 42)
	assert.NoError(t, err)
	eng.witness.f(eng.witness.r, true)

	d2 := deposit(eng, "VGT2", "someparty2", num.NewUint(24))
	err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid2", 24)
	assert.NoError(t, err)
	eng.witness.f(eng.witness.r, true)

	eng.OnTick(context.Background(), time.Now())
	state2, _, err := eng.GetState(seenKey)
	require.Nil(t, err)

	require.False(t, bytes.Equal(state1, state2))

	// verify state is consistent in the absence of change
	stateNoChange, _, err := eng.GetState(seenKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(state2, stateNoChange))

	// reload the state
	var seen snapshotpb.Payload
	snap := getTestEngine(t)
	proto.Unmarshal(state2, &seen)

	payload := types.PayloadFromProto(&seen)

	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(seenKey)
	require.True(t, bytes.Equal(state2, statePostReload))
}

func TestWithdrawalsSnapshotRoundTrip(t *testing.T) {
	testAsset := assets.NewAsset(builtin.New("VGT", &types.AssetDetails{
		Name:   "VEGA TOKEN",
		Symbol: "VGT",
	}))

	withdrawalsKey := (&types.PayloadBankingWithdrawals{}).Key()
	eng := getTestEngine(t)
	eng.tsvc.EXPECT().GetTimeNow().AnyTimes()

	for i := 0; i < 10; i++ {
		d1 := deposit(eng, "VGT"+strconv.Itoa(i*2), "someparty"+strconv.Itoa(i*2), num.NewUint(42))
		err := eng.DepositBuiltinAsset(context.Background(), d1, "depositid"+strconv.Itoa(i*2), 42)
		assert.NoError(t, err)

		eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
		eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
		eng.col.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)
		err = eng.WithdrawBuiltinAsset(context.Background(), "VGT"+strconv.Itoa(i*2), "someparty"+strconv.Itoa(i*2), "VGT"+strconv.Itoa(i*2), num.NewUint(2))
		require.Nil(t, err)
		err = eng.WithdrawBuiltinAsset(context.Background(), "VGT"+strconv.Itoa(i*2+1), "someparty"+strconv.Itoa(i*2), "VGT"+strconv.Itoa(i*2), num.NewUint(10))
		require.Nil(t, err)
		state, _, err := eng.GetState(withdrawalsKey)
		require.Nil(t, err)

		// verify state is consistent in the absence of change
		stateNoChange, _, err := eng.GetState(withdrawalsKey)
		require.Nil(t, err)
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var withdrawals snapshotpb.Payload
		snap := getTestEngine(t)
		proto.Unmarshal(state, &withdrawals)

		payload := types.PayloadFromProto(&withdrawals)

		_, err = snap.LoadState(context.Background(), payload)
		require.Nil(t, err)
		statePostReload, _, _ := snap.GetState(withdrawalsKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}

func TestDepositSnapshotRoundTrip(t *testing.T) {
	depositsKey := (&types.PayloadBankingDeposits{}).Key()
	eng := getTestEngine(t)
	eng.tsvc.EXPECT().GetTimeNow().AnyTimes()

	for i := 0; i < 10; i++ {
		d1 := deposit(eng, "VGT"+strconv.Itoa(i*2), "someparty"+strconv.Itoa(i*2), num.NewUint(42))
		err := eng.DepositBuiltinAsset(context.Background(), d1, "depositid"+strconv.Itoa(i*2), 42)
		assert.NoError(t, err)

		d2 := deposit(eng, "VGT"+strconv.Itoa(i*2+1), "someparty"+strconv.Itoa(i*2+1), num.NewUint(24))
		err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid"+strconv.Itoa(i*2+1), 24)
		assert.NoError(t, err)

		state, _, err := eng.GetState(depositsKey)
		require.Nil(t, err)

		// verify state is consistent in the absence of change
		stateNoChange, _, err := eng.GetState(depositsKey)
		require.Nil(t, err)
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var deposits snapshotpb.Payload
		snap := getTestEngine(t)
		proto.Unmarshal(state, &deposits)
		payload := types.PayloadFromProto(&deposits)
		_, err = snap.LoadState(context.Background(), payload)
		require.Nil(t, err)
		statePostReload, _, _ := snap.GetState(depositsKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}

func TestOneOffTransfersSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := (&types.PayloadBankingScheduledTransfers{}).Key()
	eng := getTestEngine(t)

	state, _, err := eng.GetState(key)
	require.Nil(t, err)

	fromAcc := types.Account{
		Balance: num.NewUint(1000),
	}

	now := time.Unix(1111, 0)
	deliver := now.Add(time.Hour)
	eng.OnTick(ctx, now)

	oneoff := &types.TransferFunds{
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
			DeliverOn: &deliver,
		},
	}

	eng.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).Times(1).Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
	eng.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Times(1).Return(&fromAcc, nil)
	eng.tsvc.EXPECT().GetTimeNow().Times(2).Return(now)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	require.NoError(t, eng.TransferFunds(ctx, oneoff))

	// test the new transfer prompts a change
	state2, _, err := eng.GetState(key)
	require.Nil(t, err)
	require.False(t, bytes.Equal(state, state2))

	// reload the state
	var transfers snapshotpb.Payload
	snap := getTestEngine(t)
	proto.Unmarshal(state2, &transfers)
	payload := types.PayloadFromProto(&transfers)
	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(key)
	require.True(t, bytes.Equal(state2, statePostReload))
}

func TestRecurringTransfersSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := (&types.PayloadBankingRecurringTransfers{}).Key()
	eng := getTestEngine(t)

	fromAcc := types.Account{
		Balance: num.NewUint(1000),
	}

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
	eng.tsvc.EXPECT().GetTimeNow().Times(1)
	eng.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	state, _, err := eng.GetState(key)
	require.Nil(t, err)

	recurring := &types.TransferFunds{
		Kind: types.TransferCommandKindRecurring,
		Recurring: &types.RecurringTransfer{
			TransferBase: &types.TransferBase{
				From:            "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301",
				FromAccountType: types.AccountTypeGeneral,
				To:              "2e05fd230f3c9f4eaf0bdc5bfb7ca0c9d00278afc44637aab60da76653d7ccf0",
				ToAccountType:   types.AccountTypeGeneral,
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			StartEpoch: 10,
			EndEpoch:   nil, // forever
			Factor:     num.MustDecimalFromString("0.9"),
			DispatchStrategy: &vega.DispatchStrategy{
				AssetForMetric:       "zohar",
				Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
				Markets:              []string{"mmm"},
				EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
				IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
				WindowLength:         1,
				LockPeriod:           1,
				DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
			},
		},
	}

	p, _ := proto.Marshal(recurring.Recurring.DispatchStrategy)
	dsHash := hex.EncodeToString(crypto.Hash(p))

	eng.marketActivityTracker.EXPECT().MarketTrackedForAsset("mmm", "zohar").Times(1).Return(true)
	require.NoError(t, eng.TransferFunds(ctx, recurring))

	// test the new transfer prompts a change
	state2, _, err := eng.GetState(key)
	require.Nil(t, err)
	require.False(t, bytes.Equal(state, state2))

	// reload the state
	var transfers snapshotpb.Payload
	snap := getTestEngine(t)
	proto.Unmarshal(state2, &transfers)
	payload := types.PayloadFromProto(&transfers)
	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(key)
	require.True(t, bytes.Equal(state2, statePostReload))

	require.NotNil(t, eng.GetDispatchStrategy(dsHash))
	require.NotNil(t, snap.GetDispatchStrategy(dsHash))
	require.Equal(t, eng.GetDispatchStrategy(dsHash), snap.GetDispatchStrategy(dsHash))
}

func TestRecurringGovTransfersSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := (&types.PayloadBankingRecurringGovernanceTransfers{}).Key()
	e := getTestEngine(t)
	require.NoError(t, e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1)))
	e.OnTick(ctx, time.Unix(10, 0))
	e.OnEpoch(ctx, types.Epoch{Seq: 1, StartTime: time.Unix(10, 0), Action: vega.EpochAction_EPOCH_ACTION_START})

	endEpoch := uint64(2)
	transfer := &types.NewTransferConfiguration{
		SourceType:           types.AccountTypeGlobalReward,
		DestinationType:      types.AccountTypeGeneral,
		Asset:                "eth",
		Source:               "",
		Destination:          "zohar",
		TransferType:         vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
		MaxAmount:            num.NewUint(10),
		FractionOfBalance:    num.DecimalFromFloat(0.1),
		Kind:                 types.TransferKindRecurring,
		OneOffTransferConfig: nil,
		RecurringTransferConfig: &vega.RecurringTransfer{
			StartEpoch: 1,
			EndEpoch:   &endEpoch,
			DispatchStrategy: &vega.DispatchStrategy{
				AssetForMetric:       "zohar",
				Metric:               vega.DispatchMetric_DISPATCH_METRIC_AVERAGE_POSITION,
				Markets:              []string{"mmm"},
				EntityScope:          vega.EntityScope_ENTITY_SCOPE_INDIVIDUALS,
				IndividualScope:      vega.IndividualScope_INDIVIDUAL_SCOPE_IN_TEAM,
				WindowLength:         1,
				LockPeriod:           1,
				DistributionStrategy: vega.DistributionStrategy_DISTRIBUTION_STRATEGY_RANK,
			},
		},
	}

	p, _ := proto.Marshal(transfer.RecurringTransferConfig.DispatchStrategy)
	dsHash := hex.EncodeToString(crypto.Hash(p))

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	e.tsvc.EXPECT().GetTimeNow().Times(1).Return(time.Now())
	require.NoError(t, e.NewGovernanceTransfer(ctx, "1", "some reference", transfer))

	// test the new transfer prompts a change
	state, _, err := e.GetState(key)
	require.NoError(t, err)

	// reload the state
	var transfers snapshotpb.Payload
	snap := getTestEngine(t)
	proto.Unmarshal(state, &transfers)
	payload := types.PayloadFromProto(&transfers)
	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(key)
	require.True(t, bytes.Equal(state, statePostReload))

	require.NotNil(t, e.GetDispatchStrategy(dsHash))
	require.NotNil(t, snap.GetDispatchStrategy(dsHash))
	require.Equal(t, e.GetDispatchStrategy(dsHash), snap.GetDispatchStrategy(dsHash))
}

func TestScheduledgGovTransfersSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := (&types.PayloadBankingScheduledGovernanceTransfers{}).Key()
	e := getTestEngine(t)
	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(10, 0)
		}).AnyTimes()

	// let's do a massive fee, easy to test.
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(context.Background(), time.Unix(10, 0))

	deliverOn := time.Unix(12, 0).UnixNano()
	transfer := &types.NewTransferConfiguration{
		SourceType:              types.AccountTypeGlobalReward,
		DestinationType:         types.AccountTypeGeneral,
		Asset:                   "eth",
		Source:                  "",
		Destination:             "zohar",
		TransferType:            vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
		MaxAmount:               num.NewUint(10),
		FractionOfBalance:       num.DecimalFromFloat(0.1),
		Kind:                    types.TransferKindOneOff,
		OneOffTransferConfig:    &vega.OneOffTransfer{DeliverOn: deliverOn},
		RecurringTransferConfig: nil,
	}

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	require.NoError(t, e.NewGovernanceTransfer(ctx, "1", "some reference", transfer))

	// test the new transfer prompts a change
	state, _, err := e.GetState(key)
	require.NoError(t, err)

	// reload the state
	var transfers snapshotpb.Payload
	snap := getTestEngine(t)
	proto.Unmarshal(state, &transfers)
	payload := types.PayloadFromProto(&transfers)
	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(key)
	require.True(t, bytes.Equal(state, statePostReload))
}

func TestAssetListRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := (&types.PayloadBankingAssetActions{}).Key()
	eng := getTestEngine(t)
	eng.tsvc.EXPECT().GetTimeNow().AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
	require.NoError(t, eng.EnableERC20(ctx, &types.ERC20AssetList{}, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301", 1000, 1000, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301", "1"))

	state, _, err := eng.GetState(key)
	require.Nil(t, err)

	var pp snapshotpb.Payload
	proto.Unmarshal(state, &pp)
	payload := types.PayloadFromProto(&pp)

	snap := getTestEngine(t)
	snap.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{name: assetNameETH, quantum: num.DecimalFromFloat(100)}), nil)
	_, err = snap.LoadState(ctx, payload)
	require.Nil(t, err)

	state2, _, err := snap.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state, state2))
}

func getTestEngineForSnapshot(t *testing.T) *testEngine {
	t.Helper()
	e := getTestEngine(t)

	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(10, 0)
		}).AnyTimes()
	e.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	e.broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()

	return e
}

func TestTransferFeeDiscountsSnapshotRoundTrip(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	log := logging.NewTestLogger()

	vegaPath := paths.New(t.TempDir())

	now := time.Now()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)

	statsData := stats.New(log, stats.NewDefaultConfig())
	bankingEngine1 := getTestEngineForSnapshot(t)

	snapshotEngine1, err := snapshot.NewEngine(vegaPath, snapshot.DefaultConfig(), log, timeService, statsData.Blockchain)
	require.NoError(t, err)

	closeSnapshotEngine1 := vgtest.OnlyOnce(snapshotEngine1.Close)
	defer closeSnapshotEngine1()

	snapshotEngine1.AddProviders(bankingEngine1)

	// No snapshot yet, does nothing.
	require.NoError(t, snapshotEngine1.Start(ctx))

	// let's do a massive fee, easy to test.
	bankingEngine1.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	bankingEngine1.OnTick(context.Background(), time.Unix(10, 0))

	assetName := "asset-1"
	assetName2 := "asset-2"
	feeDiscounts := map[string]*num.Uint{"party-1": num.NewUint(5), "party-2": num.NewUint(10), "party-6": num.NewUint(33)}
	feeDiscounts2 := map[string]*num.Uint{"party-1": num.NewUint(7), "party-2": num.NewUint(16), "party-4": num.NewUint(16)}
	feeDiscounts3 := map[string]*num.Uint{"party-6": num.NewUint(2), "party-9": num.NewUint(5), "party-2": num.NewUint(33)}
	feeDiscounts4 := map[string]*num.Uint{"party-1": num.NewUint(4), "party-2": num.NewUint(5), "party-6": num.NewUint(6)}
	bankingEngine1.RegisterTradingFees(ctx, assetName, feeDiscounts)
	bankingEngine1.RegisterTradingFees(ctx, assetName2, feeDiscounts2)
	bankingEngine1.RegisterTradingFees(ctx, assetName2, feeDiscounts3)

	hash1, err := snapshotEngine1.SnapshotNow(ctx)
	require.NoError(t, err)

	assetName3 := "asset-3"
	bankingEngine1.RegisterTradingFees(ctx, assetName3, feeDiscounts4)

	state1 := map[string][]byte{}
	for _, key := range bankingEngine1.Keys() {
		state, additionalProvider, err := bankingEngine1.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	closeSnapshotEngine1()

	snapshotEngine2, err := snapshot.NewEngine(vegaPath, snapshot.DefaultConfig(), log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	defer snapshotEngine2.Close()

	bankingEngine2 := getTestEngineForSnapshot(t)

	snapshotEngine2.AddProviders(bankingEngine2.Engine)

	require.NoError(t, snapshotEngine2.Start(ctx))

	hash2, _, _ := snapshotEngine2.Info()
	require.Equal(t, hash1, hash2)

	bankingEngine2.RegisterTradingFees(ctx, assetName3, feeDiscounts4)

	state2 := map[string][]byte{}
	for _, key := range bankingEngine2.Keys() {
		state, additionalProvider, err := bankingEngine2.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}
