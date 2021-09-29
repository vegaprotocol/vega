package collateral_test

import (
	"bytes"
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCheckpoint(t *testing.T) {
	eng := getTestEngine(t, "market1")
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

	snapshot, err := eng.Checkpoint()
	require.NoError(t, err)
	require.NotEmpty(t, snapshot)

	conf := collateral.NewDefaultConfig()
	conf.Level = encoding.LogLevel{Level: logging.DebugLevel}
	// system accounts created
	loadEng := collateral.New(logging.NewTestLogger(), conf, eng.broker, time.Now())

	asset := types.Asset{
		ID: testMarketAsset,
		Details: &types.AssetDetails{
			Symbol: testMarketAsset,
		},
	}
	// we need to enable the assets before being able to load the balances
	loadEng.EnableAsset(ctx, asset)
	err = loadEng.Load(ctx, snapshot)
	require.NoError(t, err)
	loadedPartyAcc, err := loadEng.GetPartyGeneralAccount(party, testMarketAsset)
	require.NoError(t, err)
	require.Equal(t, bal, loadedPartyAcc.Balance)

	loadedGlobInsPool, err := loadEng.GetAssetInsurancePoolAccount(testMarketAsset)
	require.NoError(t, err)
	require.Equal(t, insBal, loadedGlobInsPool.Balance)
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
			Name:        "foo",
			Symbol:      "FOO",
			TotalSupply: num.NewUint(100000000),
			Decimals:    5,
			MinLpStake:  num.NewUint(1),
			Source: types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.NewUint(100000000),
				},
			},
		},
	}
	eng := getTestEngine(t, mkt)
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
	hashes := make(map[string][]byte, len(keys))
	data := make(map[string][]byte, len(keys))
	for _, k := range keys {
		hash, err := eng.GetHash(k)
		require.NoError(t, err)
		hashes[k] = hash
		state, err := eng.GetState(k)
		require.NoError(t, err)
		data[k] = state
	}
	// now no changes, check hashes again:
	for k, exp := range hashes {
		got, err := eng.GetHash(k)
		require.NoError(t, err)
		require.EqualValues(t, exp, got)
		state, err := eng.GetState(k)
		require.NoError(t, err)
		require.EqualValues(t, data[k], state)
	}
	// now change one account:
	require.NoError(t, eng.IncrementBalance(ctx, last, inc))
	changes := 0
	for k, hash := range hashes {
		got, err := eng.GetHash(k)
		require.NoError(t, err)
		if !bytes.Equal(hash, got) {
			// compare data
			state, err := eng.GetState(k)
			require.NoError(t, err)
			require.NotEqualValues(t, data[k], state)
			changes++
		}
	}
	require.Equal(t, 1, changes)
}

func testSnapshotRestore(t *testing.T) {
	mkt := "market1"
	ctx := context.Background()
	asset := types.Asset{
		ID: "foo",
		Details: &types.AssetDetails{
			Name:        "foo",
			Symbol:      "FOO",
			TotalSupply: num.NewUint(100000000),
			Decimals:    5,
			MinLpStake:  num.NewUint(1),
			Source: types.AssetDetailsBuiltinAsset{
				BuiltinAsset: &types.BuiltinAsset{
					MaxFaucetAmountMint: num.NewUint(100000000),
				},
			},
		},
	}
	eng := getTestEngine(t, mkt)
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
	tmp := types.PayloadCollateralAssets{}
	assetKey := tmp.Key()
	keys := eng.Keys()
	payloads := make(map[string]*types.Payload, len(keys))
	data := make(map[string][]byte, len(keys))
	hashes := make(map[string][]byte, len(keys))
	for _, k := range keys {
		payloads[k] = &types.Payload{}
		h, err := eng.GetHash(k)
		require.NoError(t, err)
		hashes[k] = h
		s, err := eng.GetState(k)
		require.NoError(t, err)
		data[k] = s
	}
	newEng := getTestEngine(t, mkt)
	defer newEng.ctrl.Finish()
	// we expect 2 batches of events to be sent
	newEng.broker.EXPECT().SendBatch(gomock.Any()).Times(2)
	for k, pl := range payloads {
		state := data[k]
		ptype := pl.IntoProto()
		require.NoError(t, proto.Unmarshal(state, ptype))
		payloads[k] = types.PayloadFromProto(ptype)
		require.NoError(t, newEng.LoadState(ctx, payloads[k]))
	}
	eng.PrintSnapstate("old")
	newEng.PrintSnapstate("new")
	for k, exp := range hashes {
		got, err := newEng.GetHash(k)
		require.NoError(t, err)
		if k != assetKey {
			require.EqualValues(t, exp, got)
		}
	}
	require.NoError(t, eng.IncrementBalance(ctx, last, inc))
	// now we expect 1 different hash
	diff := 0
	for k, h := range hashes {
		old, err := eng.GetHash(k)
		require.NoError(t, err)
		reload, err := newEng.GetHash(k)
		require.NoError(t, err)
		if k != assetKey {
			if !bytes.Equal(h, old) {
				diff++
				require.NotEqualValues(t, reload, old)
			}
		}
	}
	require.Equal(t, 1, diff)
	newEng.broker.EXPECT().Send(gomock.Any()).Times(1)
	require.NoError(t, newEng.IncrementBalance(ctx, last, inc))
	// now the state should match up once again
	for k := range hashes {
		old, err := eng.GetHash(k)
		require.NoError(t, err)
		restore, err := newEng.GetHash(k)
		require.NoError(t, err)
		if k != assetKey {
			require.EqualValues(t, old, restore)
		}
	}
}
