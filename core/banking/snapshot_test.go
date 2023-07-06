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
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	"code.vegaprotocol.io/vega/core/assets/builtin"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/protos/vega"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
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

func testEngineAndSnapshot(t *testing.T) (*testEngine, *snp.Engine) {
	t.Helper()
	eng := getTestEngine(t)
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
	return eng, snapshotEngine
}

func TestSnapshotRoundtripViaEngine(t *testing.T) {
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")

	testAsset := assets.NewAsset(builtin.New("VGT", &types.AssetDetails{
		Name:   "VEGA TOKEN",
		Symbol: "VGT",
	}))

	eng, snap := testEngineAndSnapshot(t)
	defer eng.ctrl.Finish()
	defer snap.Close()

	now := time.Now()

	// setup some deposits
	d1 := depositAt(eng, "VGT1", "someparty1", num.NewUint(42))
	err := eng.DepositBuiltinAsset(context.Background(), d1, "depositid1", 42)
	assert.NoError(t, err)

	d2 := depositAt(eng, "VGT1", "someparty2", num.NewUint(24))
	err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid2", 24)
	assert.NoError(t, err)

	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	eng.col.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)
	// setup some withdrawals
	err = eng.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty1", "VGT1", num.NewUint(2))
	require.Nil(t, err)

	err = eng.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty2", "VGT1", num.NewUint(4))
	require.Nil(t, err)

	fromAcc := types.Account{
		Balance: num.NewUint(1000),
	}

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	eng.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	deliver := now.Add(time.Hour)
	eng.OnTick(ctx, now)

	// setup one time transfer
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

	require.NoError(t, eng.TransferFunds(ctx, oneoff))

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	eng.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// setup recurring transfer
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

	require.NoError(t, eng.TransferFunds(ctx, recurring))

	_, err = snap.Snapshot(ctx)
	require.NoError(t, err)
	snaps, err := snap.List()
	require.NoError(t, err)
	snap1 := snaps[0]

	engineLoad, snapLoad := testEngineAndSnapshot(t)
	engineLoad.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	engineLoad.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	engineLoad.tsvc.EXPECT().GetTimeNow().AnyTimes()
	snapLoad.ReceiveSnapshot(snap1)
	snapLoad.ApplySnapshot(ctx)
	snapLoad.CheckLoaded()
	defer snapLoad.Close()

	// verify equal right after snapshot load
	b, err := snap.Snapshot(ctx)
	require.NoError(t, err)
	bLoad, err := snapLoad.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(b, bLoad))

	eng.OnTick(ctx, now)
	engineLoad.OnTick(ctx, now)

	// setup some more deposits
	d3 := depositAt(eng, "VGT1", "someparty3", num.NewUint(29))
	err = eng.DepositBuiltinAsset(context.Background(), d3, "depositid3", 29)
	require.NoError(t, err)
	err = engineLoad.DepositBuiltinAsset(context.Background(), d3, "depositid3", 29)
	require.NoError(t, err)

	// setup some withdrawals
	engineLoad.col.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)
	err = eng.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty1", "VGT1", num.NewUint(10))
	require.Nil(t, err)
	err = engineLoad.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty1", "VGT1", num.NewUint(10))
	require.Nil(t, err)
	err = eng.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty2", "VGT1", num.NewUint(5))
	require.Nil(t, err)
	err = engineLoad.WithdrawBuiltinAsset(context.Background(), "VGT1", "someparty2", "VGT1", num.NewUint(5))
	require.Nil(t, err)

	// setup some transfers
	engineLoad.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	engineLoad.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	engineLoad.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	engineLoad.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	deliver = now.Add(time.Hour)
	eng.OnTick(ctx, now)

	// setup one time transfer
	oneoff = &types.TransferFunds{
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

	require.NoError(t, eng.TransferFunds(ctx, oneoff))
	require.NoError(t, engineLoad.TransferFunds(ctx, oneoff))

	engineLoad.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	engineLoad.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	engineLoad.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	engineLoad.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	// setup recurring transfer
	recurring = &types.TransferFunds{
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

	require.NoError(t, eng.TransferFunds(ctx, recurring))
	require.NoError(t, engineLoad.TransferFunds(ctx, recurring))
}

func TestAssetActionsSnapshotRoundTrip(t *testing.T) {
	aaKey := (&types.PayloadBankingAssetActions{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

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
	var assetActions snapshot.Payload
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
	defer eng.ctrl.Finish()

	eng.tsvc.EXPECT().GetTimeNow().Times(2)
	state1, _, err := eng.GetState(seenKey)
	require.Nil(t, err)
	eng.col.EXPECT().Deposit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.LedgerMovement{}, nil)

	d1 := deposit(eng, "VGT1", "someparty1", num.NewUint(42))
	err = eng.DepositBuiltinAsset(context.Background(), d1, "depositid1", 42)
	assert.NoError(t, err)
	eng.erc.f(eng.erc.r, true)

	d2 := deposit(eng, "VGT2", "someparty2", num.NewUint(24))
	err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid2", 24)
	assert.NoError(t, err)
	eng.erc.f(eng.erc.r, true)

	eng.OnTick(context.Background(), time.Now())
	state2, _, err := eng.GetState(seenKey)
	require.Nil(t, err)

	require.False(t, bytes.Equal(state1, state2))

	// verify state is consistent in the absence of change
	stateNoChange, _, err := eng.GetState(seenKey)
	require.Nil(t, err)
	require.True(t, bytes.Equal(state2, stateNoChange))

	// reload the state
	var seen snapshot.Payload
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

	defer eng.ctrl.Finish()
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
		var withdrawals snapshot.Payload
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

	defer eng.ctrl.Finish()
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
		var deposits snapshot.Payload
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
	defer eng.ctrl.Finish()

	fromAcc := types.Account{
		Balance: num.NewUint(1000),
	}

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	eng.tsvc.EXPECT().GetTimeNow().Times(4)
	eng.col.EXPECT().GetPartyGeneralAccount(gomock.Any(), gomock.Any()).AnyTimes().Return(&fromAcc, nil)
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.col.EXPECT().TransferFunds(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	state, _, err := eng.GetState(key)
	require.Nil(t, err)

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
				Asset:           "eth",
				Amount:          num.NewUint(10),
				Reference:       "someref",
			},
			DeliverOn: &deliver,
		},
	}

	require.NoError(t, eng.TransferFunds(ctx, oneoff))

	// test the new transfer prompts a change
	state2, _, err := eng.GetState(key)
	require.Nil(t, err)
	require.False(t, bytes.Equal(state, state2))

	// reload the state
	var transfers snapshot.Payload
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
	defer eng.ctrl.Finish()

	fromAcc := types.Account{
		Balance: num.NewUint(1000),
	}

	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
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
		},
	}

	require.NoError(t, eng.TransferFunds(ctx, recurring))

	// test the new transfer prompts a change
	state2, _, err := eng.GetState(key)
	require.Nil(t, err)
	require.False(t, bytes.Equal(state, state2))

	// reload the state
	var transfers snapshot.Payload
	snap := getTestEngine(t)
	proto.Unmarshal(state2, &transfers)
	payload := types.PayloadFromProto(&transfers)
	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(key)
	require.True(t, bytes.Equal(state2, statePostReload))
}

func TestRecurringGovTransfersSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := (&types.PayloadBankingRecurringGovernanceTransfers{}).Key()
	e := getTestEngine(t)
	defer e.ctrl.Finish()

	e.tsvc.EXPECT().GetTimeNow().DoAndReturn(
		func() time.Time {
			return time.Unix(10, 0)
		}).Times(3)
	e.OnTransferFeeFactorUpdate(context.Background(), num.NewDecimalFromFloat(1))
	e.OnTick(ctx, time.Unix(10, 0))
	e.OnEpoch(ctx, types.Epoch{Seq: 1, StartTime: time.Unix(10, 0), Action: vega.EpochAction_EPOCH_ACTION_START})

	endEpoch := uint64(2)
	transfer := &types.NewTransferConfiguration{
		SourceType:              types.AccountTypeGlobalReward,
		DestinationType:         types.AccountTypeGeneral,
		Asset:                   "eth",
		Source:                  "",
		Destination:             "zohar",
		TransferType:            vega.GovernanceTransferType_GOVERNANCE_TRANSFER_TYPE_ALL_OR_NOTHING,
		MaxAmount:               num.NewUint(10),
		FractionOfBalance:       num.DecimalFromFloat(0.1),
		Kind:                    types.TransferKindRecurring,
		OneOffTransferConfig:    nil,
		RecurringTransferConfig: &vega.RecurringTransfer{StartEpoch: 1, EndEpoch: &endEpoch},
	}

	e.broker.EXPECT().Send(gomock.Any()).Times(1)
	require.NoError(t, e.NewGovernanceTransfer(ctx, "1", "some reference", transfer))

	// test the new transfer prompts a change
	state, _, err := e.GetState(key)
	require.NoError(t, err)

	// reload the state
	var transfers snapshot.Payload
	snap := getTestEngine(t)
	proto.Unmarshal(state, &transfers)
	payload := types.PayloadFromProto(&transfers)
	_, err = snap.LoadState(context.Background(), payload)
	require.Nil(t, err)
	statePostReload, _, _ := snap.GetState(key)
	require.True(t, bytes.Equal(state, statePostReload))
}

func TestScheduledgGovTransfersSnapshotRoundTrip(t *testing.T) {
	ctx := context.Background()
	key := (&types.PayloadBankingScheduledGovernanceTransfers{}).Key()
	e := getTestEngine(t)
	defer e.ctrl.Finish()
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
	var transfers snapshot.Payload
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
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	require.NoError(t, eng.EnableERC20(ctx, &types.ERC20AssetList{}, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301", 1000, 1000, "03ae90688632c649c4beab6040ff5bd04dbde8efbf737d8673bbda792a110301"))

	state, _, err := eng.GetState(key)
	require.Nil(t, err)

	var pp snapshot.Payload
	proto.Unmarshal(state, &pp)
	payload := types.PayloadFromProto(&pp)

	snap := getTestEngine(t)
	snap.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(assets.NewAsset(&mockAsset{num.DecimalFromFloat(100)}), nil)
	_, err = snap.LoadState(ctx, payload)
	require.Nil(t, err)

	state2, _, err := snap.GetState(key)
	require.NoError(t, err)
	require.True(t, bytes.Equal(state, state2))
}
