package limits_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/limits"
	"code.vegaprotocol.io/vega/logging"

	"github.com/stretchr/testify/assert"
)

type limitsTest struct {
	*limits.Engine
	log *logging.Logger
}

func getLimitsTest() *limitsTest {
	log := logging.NewTestLogger()
	return &limitsTest{
		Engine: limits.New(log, limits.NewDefaultConfig()),
		log:    log,
	}
}

func (l *limitsTest) loadGenesisState(t *testing.T, lstate *limits.GenesisState) {
	state := struct {
		Limits *limits.GenesisState `json:"network_limits"`
	}{
		Limits: lstate,
	}

	buf, err := json.Marshal(state)
	assert.NoError(t, err)
	assert.NotNil(t, buf)

	assert.NoError(t, l.UponGenesis(context.Background(), buf))
}

func TestLimits(t *testing.T) {
	t.Run("test empty genesis", testEmptyGenesis)
	t.Run("test nil genesis", testNilGenesis)
	t.Run("test all disabled", testAllDisabled)
	t.Run("test all enabled", testAllEnabled)
	t.Run("test market enabled asset disabled", testMarketEnabledAssetDisabled)
	t.Run("test market disabled asset enbled", testMarketdisabledAssetenabled)
	t.Run("proposal enabled with time reach becomes enabled", testDisabledUntilTimeIsReach)
	t.Run("proposals disabled with time reach stay disabled", testStayDisabledIfTimeIsReachedButEnabledIsFalse)
	t.Run("bootstrap finished enabled proposals", testBootstrapFinishedEnabledProposals)
	t.Run("bootstrap in progress enabled proposals", testBootstrapInProgressEnabledProposals)
}

func testEmptyGenesis(t *testing.T) {
	lmts := getLimitsTest()

	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanTrade())
	assert.False(t, lmts.BootstrapFinished())
}

func testNilGenesis(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, nil)

	// need to call onTick
	lmts.OnTick(context.Background(), time.Unix(1000, 0))

	assert.True(t, lmts.CanProposeAsset())
	assert.True(t, lmts.CanProposeMarket())
	assert.True(t, lmts.CanTrade())
}

func testAllDisabled(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, &limits.GenesisState{})

	// need to call onTick
	lmts.OnTick(context.Background(), time.Unix(1000, 0))

	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanTrade())
}

func testAllEnabled(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  true,
		ProposeMarketEnabled: true,
	})

	// need to call onTick
	lmts.OnTick(context.Background(), time.Unix(1000, 0))

	assert.True(t, lmts.CanProposeAsset())
	assert.True(t, lmts.CanProposeMarket())
	assert.True(t, lmts.CanTrade())
}

func testMarketEnabledAssetDisabled(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  false,
		ProposeMarketEnabled: true,
	})

	// need to call onTick
	lmts.OnTick(context.Background(), time.Unix(1000, 0))

	assert.True(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())
}

func testMarketdisabledAssetenabled(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  true,
		ProposeMarketEnabled: false,
	})

	// need to call onTick
	lmts.OnTick(context.Background(), time.Unix(1000, 0))

	assert.False(t, lmts.CanProposeMarket())
	assert.True(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())

}

func testDisabledUntilTimeIsReach(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:      true,
		ProposeMarketEnabled:     true,
		ProposeAssetEnabledFrom:  timePtr(time.Unix(2000, 0)),
		ProposeMarketEnabledFrom: timePtr(time.Unix(2000, 0)),
	})

	// need to call onTick
	lmts.OnTick(context.Background(), time.Unix(1000, 0))

	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())

	// need to call onTick again, now it should be fine
	lmts.OnTick(context.Background(), time.Unix(3000, 0))

	assert.True(t, lmts.CanProposeMarket())
	assert.True(t, lmts.CanProposeAsset())
	assert.True(t, lmts.CanTrade())

}

func testStayDisabledIfTimeIsReachedButEnabledIsFalse(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:      false,
		ProposeMarketEnabled:     false,
		ProposeAssetEnabledFrom:  timePtr(time.Unix(2000, 0)),
		ProposeMarketEnabledFrom: timePtr(time.Unix(2000, 0)),
	})

	// need to call onTick
	lmts.OnTick(context.Background(), time.Unix(1000, 0))

	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())

	// need to call onTick again, now it should be fine
	lmts.OnTick(context.Background(), time.Unix(3000, 0))

	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())
}

func testBootstrapFinishedEnabledProposals(t *testing.T) {
	lmts := getLimitsTest()
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:      true,
		ProposeMarketEnabled:     true,
		ProposeAssetEnabledFrom:  timePtr(time.Unix(2000, 0)),
		ProposeMarketEnabledFrom: timePtr(time.Unix(2000, 0)),
		BootstrapBlockCount:      2,
	})

	// block count is 0 call on Tick once, it's should still
	// be impossible to do anything, both boolean are OK
	// and the time is OK
	lmts.OnTick(context.Background(), time.Unix(3000, 0))

	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())
	assert.False(t, lmts.BootstrapFinished())
}

func testBootstrapInProgressEnabledProposals(t *testing.T) {

}

func timePtr(t time.Time) *time.Time {
	return &t
}
