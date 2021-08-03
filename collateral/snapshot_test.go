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
	ctx := context.Background()
	defer eng.ctrl.Finish()

	party := "foo"
	bal := num.NewUint(500)
	// create party
	eng.broker.EXPECT().Send(gomock.Any()).AnyTimes()
	acc, err := eng.Engine.CreatePartyGeneralAccount(ctx, party, testMarketAsset)
	assert.NoError(t, err)
	err = eng.Engine.UpdateBalance(ctx, acc, bal)
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
	require.True(t, loadEng.HasBalance(party))
}
