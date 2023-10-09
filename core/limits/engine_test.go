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

package limits_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/limits"
	"code.vegaprotocol.io/vega/core/limits/mocks"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

type limitsTest struct {
	*limits.Engine
	log *logging.Logger
}

func getLimitsTest(t *testing.T) *limitsTest {
	t.Helper()
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	timeSvc := mocks.NewMockTimeService(ctrl)
	timeSvc.EXPECT().GetTimeNow().AnyTimes()

	return &limitsTest{
		Engine: limits.New(log, limits.NewDefaultConfig(), timeSvc, broker),
		log:    log,
	}
}

func (l *limitsTest) loadGenesisState(t *testing.T, lstate *limits.GenesisState) {
	t.Helper()
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
}

func testEmptyGenesis(t *testing.T) {
	lmts := getLimitsTest(t)

	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanTrade())
}

func testNilGenesis(t *testing.T) {
	lmts := getLimitsTest(t)
	lmts.loadGenesisState(t, nil)

	assert.True(t, lmts.CanProposeAsset())
	assert.True(t, lmts.CanProposeMarket())
	assert.True(t, lmts.CanTrade())
}

func testAllDisabled(t *testing.T) {
	lmts := getLimitsTest(t)
	lmts.loadGenesisState(t, &limits.GenesisState{})

	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanTrade())
}

func testAllEnabled(t *testing.T) {
	lmts := getLimitsTest(t)
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  true,
		ProposeMarketEnabled: true,
	})

	assert.True(t, lmts.CanProposeAsset())
	assert.True(t, lmts.CanProposeMarket())
	assert.True(t, lmts.CanTrade())
}

func testMarketEnabledAssetDisabled(t *testing.T) {
	lmts := getLimitsTest(t)
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  false,
		ProposeMarketEnabled: true,
	})

	assert.True(t, lmts.CanProposeMarket())
	assert.False(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())
}

func testMarketdisabledAssetenabled(t *testing.T) {
	lmts := getLimitsTest(t)
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  true,
		ProposeMarketEnabled: false,
	})

	assert.False(t, lmts.CanProposeMarket())
	assert.True(t, lmts.CanProposeAsset())
	assert.False(t, lmts.CanTrade())
}

func testDisabledUntilTimeIsReach(t *testing.T) {
	lmts := getLimitsTest(t)
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  true,
		ProposeMarketEnabled: true,
	})

	lmts.OnLimitsProposeAssetEnabledFromUpdate(context.Background(), time.Unix(2000, 0).Format(time.RFC3339))
	lmts.OnLimitsProposeMarketEnabledFromUpdate(context.Background(), time.Unix(2000, 0).Format(time.RFC3339))

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
	lmts := getLimitsTest(t)
	lmts.loadGenesisState(t, &limits.GenesisState{
		ProposeAssetEnabled:  false,
		ProposeMarketEnabled: false,
	})

	lmts.OnLimitsProposeAssetEnabledFromUpdate(context.Background(), time.Unix(2000, 0).Format(time.RFC3339))
	lmts.OnLimitsProposeMarketEnabledFromUpdate(context.Background(), time.Unix(2000, 0).Format(time.RFC3339))

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
