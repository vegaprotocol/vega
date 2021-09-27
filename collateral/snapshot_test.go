package collateral_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/collateral"
	"code.vegaprotocol.io/vega/config/encoding"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSnapshot(t *testing.T) {
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
	err = loadEng.Load(snapshot)
	require.NoError(t, err)
	loadedPartyAcc, err := loadEng.GetPartyGeneralAccount(party, testMarketAsset)
	require.NoError(t, err)
	require.Equal(t, bal, loadedPartyAcc.Balance)

	loadedGlobInsPool, err := loadEng.GetAssetInsurancePoolAccount(testMarketAsset)
	require.NoError(t, err)
	require.Equal(t, insBal, loadedGlobInsPool.Balance)
}
