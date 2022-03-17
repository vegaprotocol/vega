package banking_test

import (
	"bytes"
	"context"
	"strconv"
	"testing"
	"time"

	snapshot "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/assets"
	"code.vegaprotocol.io/vega/assets/builtin"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func deposit(eng *testEngine, asset, party string, amount *num.Uint) *types.BuiltinAssetDeposit {
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
	now := time.Now()
	eng.OnTick(context.Background(), now)
	return &types.BuiltinAssetDeposit{
		VegaAssetID: asset,
		PartyID:     party,
		Amount:      amount,
	}
}

func TestAssetActionsSnapshotRoundTrip(t *testing.T) {
	aaKey := (&types.PayloadBankingAssetActions{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	d1 := deposit(eng, "VGT1", "someparty1", num.NewUint(42))
	err := eng.DepositBuiltinAsset(context.Background(), d1, "depositid1", 42)
	assert.NoError(t, err)

	d2 := deposit(eng, "VGT1", "someparty2", num.NewUint(24))
	err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid2", 24)
	assert.NoError(t, err)

	// 	eng.OnTick(context.Background(), time.Now())
	hash, err := eng.GetHash(aaKey)
	require.Nil(t, err)
	state, _, err := eng.GetState(aaKey)
	require.Nil(t, err)

	// verify hash is consistent in the absence of change
	hashNoChange, err := eng.GetHash(aaKey)
	require.Nil(t, err)
	stateNoChange, _, err := eng.GetState(aaKey)
	require.Nil(t, err)

	require.True(t, bytes.Equal(hash, hashNoChange))
	require.True(t, bytes.Equal(state, stateNoChange))

	// reload the state
	var assetActions snapshot.Payload
	proto.Unmarshal(state, &assetActions)
	payload := types.PayloadFromProto(&assetActions)
	_, err = eng.LoadState(context.Background(), payload)
	require.Nil(t, err)
	hashPostReload, _ := eng.GetHash(aaKey)
	require.True(t, bytes.Equal(hash, hashPostReload))
	statePostReload, _, _ := eng.GetState(aaKey)
	require.True(t, bytes.Equal(state, statePostReload))
}

func TestSeenSnapshotRoundTrip(t *testing.T) {
	seenKey := (&types.PayloadBankingSeen{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()

	hash1, err := eng.GetHash(seenKey)
	require.Nil(t, err)
	eng.col.EXPECT().Deposit(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.TransferResponse{}, nil)

	d1 := deposit(eng, "VGT1", "someparty1", num.NewUint(42))
	err = eng.DepositBuiltinAsset(context.Background(), d1, "depositid1", 42)
	assert.NoError(t, err)
	eng.erc.f(eng.erc.r, true)

	d2 := deposit(eng, "VGT2", "someparty2", num.NewUint(24))
	err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid2", 24)
	assert.NoError(t, err)
	eng.erc.f(eng.erc.r, true)

	eng.OnTick(context.Background(), time.Now())
	hash2, err := eng.GetHash(seenKey)
	require.Nil(t, err)
	state2, _, err := eng.GetState(seenKey)
	require.Nil(t, err)

	require.NotEqual(t, hash1, hash2)

	// verify hash is consistent in the absence of change
	hashNoChange, err := eng.GetHash(seenKey)
	require.Nil(t, err)
	stateNoChange, _, err := eng.GetState(seenKey)
	require.Nil(t, err)

	require.True(t, bytes.Equal(hash2, hashNoChange))
	require.True(t, bytes.Equal(state2, stateNoChange))

	// reload the state
	var seen snapshot.Payload
	proto.Unmarshal(state2, &seen)

	payload := types.PayloadFromProto(&seen)

	_, err = eng.LoadState(context.Background(), payload)
	require.Nil(t, err)
	hashPostReload, _ := eng.GetHash(seenKey)
	require.True(t, bytes.Equal(hash2, hashPostReload))
	statePostReload, _, _ := eng.GetState(seenKey)
	require.True(t, bytes.Equal(state2, statePostReload))
}

func TestWithdrawlsSnapshotRoundTrip(t *testing.T) {
	testAsset := assets.NewAsset(builtin.New("VGT", &types.AssetDetails{
		Name:   "VEGA TOKEN",
		Symbol: "VGT",
	}))

	withdrawalsKey := (&types.PayloadBankingWithdrawals{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	for i := 0; i < 10; i++ {
		d1 := deposit(eng, "VGT"+strconv.Itoa(i*2), "someparty"+strconv.Itoa(i*2), num.NewUint(42))
		err := eng.DepositBuiltinAsset(context.Background(), d1, "depositid"+strconv.Itoa(i*2), 42)
		assert.NoError(t, err)

		eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
		eng.assets.EXPECT().Get(gomock.Any()).AnyTimes().Return(testAsset, nil)
		eng.col.EXPECT().Withdraw(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&types.TransferResponse{}, nil)
		err = eng.WithdrawBuiltinAsset(context.Background(), "VGT"+strconv.Itoa(i*2), "someparty"+strconv.Itoa(i*2), "VGT"+strconv.Itoa(i*2), num.NewUint(2))
		require.Nil(t, err)
		err = eng.WithdrawBuiltinAsset(context.Background(), "VGT"+strconv.Itoa(i*2+1), "someparty"+strconv.Itoa(i*2), "VGT"+strconv.Itoa(i*2), num.NewUint(10))
		require.Nil(t, err)

		hash, err := eng.GetHash(withdrawalsKey)
		require.Nil(t, err)
		state, _, err := eng.GetState(withdrawalsKey)
		require.Nil(t, err)

		// verify hash is consistent in the absence of change
		hashNoChange, err := eng.GetHash(withdrawalsKey)
		require.Nil(t, err)
		stateNoChange, _, err := eng.GetState(withdrawalsKey)
		require.Nil(t, err)

		require.True(t, bytes.Equal(hash, hashNoChange))
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var withdrawals snapshot.Payload
		proto.Unmarshal(state, &withdrawals)

		payload := types.PayloadFromProto(&withdrawals)

		_, err = eng.LoadState(context.Background(), payload)
		require.Nil(t, err)
		hashPostReload, _ := eng.GetHash(withdrawalsKey)
		require.True(t, bytes.Equal(hash, hashPostReload))
		statePostReload, _, _ := eng.GetState(withdrawalsKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}

func TestDepositSnapshotRoundTrip(t *testing.T) {
	depositsKey := (&types.PayloadBankingDeposits{}).Key()
	eng := getTestEngine(t)
	defer eng.ctrl.Finish()
	for i := 0; i < 10; i++ {
		d1 := deposit(eng, "VGT"+strconv.Itoa(i*2), "someparty"+strconv.Itoa(i*2), num.NewUint(42))
		err := eng.DepositBuiltinAsset(context.Background(), d1, "depositid"+strconv.Itoa(i*2), 42)
		assert.NoError(t, err)

		d2 := deposit(eng, "VGT"+strconv.Itoa(i*2+1), "someparty"+strconv.Itoa(i*2+1), num.NewUint(24))
		err = eng.DepositBuiltinAsset(context.Background(), d2, "depositid"+strconv.Itoa(i*2+1), 24)
		assert.NoError(t, err)

		hash, err := eng.GetHash(depositsKey)
		require.Nil(t, err)
		state, _, err := eng.GetState(depositsKey)
		require.Nil(t, err)

		// verify hash is consistent in the absence of change
		hashNoChange, err := eng.GetHash(depositsKey)
		require.Nil(t, err)
		stateNoChange, _, err := eng.GetState(depositsKey)
		require.Nil(t, err)

		require.True(t, bytes.Equal(hash, hashNoChange))
		require.True(t, bytes.Equal(state, stateNoChange))

		// reload the state
		var deposits snapshot.Payload
		proto.Unmarshal(state, &deposits)
		payload := types.PayloadFromProto(&deposits)
		_, err = eng.LoadState(context.Background(), payload)
		require.Nil(t, err)
		hashPostReload, _ := eng.GetHash(depositsKey)
		require.True(t, bytes.Equal(hash, hashPostReload))
		statePostReload, _, _ := eng.GetState(depositsKey)
		require.True(t, bytes.Equal(state, statePostReload))
	}
}
