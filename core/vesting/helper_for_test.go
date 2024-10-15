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

package vesting_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets/common"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/core/vesting"
	"code.vegaprotocol.io/vega/core/vesting/mocks"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	vegapb "code.vegaprotocol.io/vega/protos/vega"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testEngine struct {
	*vesting.Engine

	ctrl    *gomock.Controller
	col     *collateralMock
	asvm    *mocks.MockActivityStreakVestingMultiplier
	broker  *bmocks.MockBroker
	assets  *mocks.MockAssets
	parties *mocks.MockParties
	t       *mocks.MockTime
}

func getTestEngine(t *testing.T) *testEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	logger := logging.NewTestLogger()
	col := newCollateralMock(t)
	broker := bmocks.NewMockBroker(ctrl)
	asvm := mocks.NewMockActivityStreakVestingMultiplier(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	parties := mocks.NewMockParties(ctrl)
	tim := mocks.NewMockTime(ctrl)

	return &testEngine{
		Engine: vesting.New(
			logger, col, asvm, broker, assets, parties, tim,
		),
		ctrl:    ctrl,
		broker:  broker,
		col:     col,
		asvm:    asvm,
		assets:  assets,
		parties: parties,
		t:       tim,
	}
}

type testSnapshotEngine struct {
	engine *vesting.SnapshotEngine

	ctrl    *gomock.Controller
	col     *collateralMock
	asvm    *mocks.MockActivityStreakVestingMultiplier
	broker  *bmocks.MockBroker
	assets  *mocks.MockAssets
	parties *mocks.MockParties
	t       *mocks.MockTime

	currentEpoch uint64
}

func newEngine(t *testing.T) *testSnapshotEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	col := newCollateralMock(t)
	asvm := mocks.NewMockActivityStreakVestingMultiplier(ctrl)
	broker := bmocks.NewMockBroker(ctrl)
	assets := mocks.NewMockAssets(ctrl)
	parties := mocks.NewMockParties(ctrl)
	tim := mocks.NewMockTime(ctrl)

	return &testSnapshotEngine{
		engine: vesting.NewSnapshotEngine(
			logging.NewTestLogger(), col, asvm, broker, assets, parties, tim,
		),
		ctrl:         ctrl,
		col:          col,
		asvm:         asvm,
		broker:       broker,
		assets:       assets,
		parties:      parties,
		currentEpoch: 10,
		t:            tim,
	}
}

type collateralMock struct {
	vestedAccountAmount            map[string]map[string]*num.Uint
	vestingQuantumBalanceCallCount int
}

func (c *collateralMock) InitVestedBalance(party, asset string, balance *num.Uint) {
	c.vestedAccountAmount[party] = map[string]*num.Uint{
		asset: balance,
	}
}

func (c *collateralMock) TransferVestedRewards(_ context.Context, transfers []*types.Transfer) ([]*types.LedgerMovement, error) {
	for _, transfer := range transfers {
		vestedAccount, ok := c.vestedAccountAmount[transfer.Owner]
		if !ok {
			vestedAccount = map[string]*num.Uint{}
			c.vestedAccountAmount[transfer.Owner] = map[string]*num.Uint{}
		}

		amount, ok := vestedAccount[transfer.Amount.Asset]
		if !ok {
			amount = num.UintZero()
			vestedAccount[transfer.Amount.Asset] = amount
		}

		amount.AddSum(transfer.Amount.Amount)
	}
	return []*types.LedgerMovement{}, nil
}

func (c *collateralMock) GetVestingRecovery() map[string]map[string]*num.Uint {
	// Only used for checkpoint.
	return nil
}

// GetAllVestingQuantumBalance is a custom implementation used to ensure
// the vesting engine account for benefit tiers during computation.
// Using this implementation saves us from mocking this at every call to
// `OnEpochEvent()` with consistent results.
func (c *collateralMock) GetAllVestingQuantumBalance(party string) num.Decimal {
	vestedAccount, ok := c.vestedAccountAmount[party]
	if !ok {
		return num.DecimalZero()
	}

	balance := num.DecimalZero()
	for _, n := range vestedAccount {
		balance = balance.Add(num.DecimalFromUint(n))
	}

	c.vestingQuantumBalanceCallCount += 1

	return balance
}

func (c *collateralMock) ResetVestingQuantumBalanceCallCount() {
	c.vestingQuantumBalanceCallCount = 0
}

func (c *collateralMock) GetVestingQuantumBalanceCallCount() int {
	return c.vestingQuantumBalanceCallCount
}

func (c *collateralMock) GetAllVestingAndVestedAccountForAsset(asset string) []*types.Account {
	return nil
}

func newCollateralMock(t *testing.T) *collateralMock {
	t.Helper()

	return &collateralMock{
		vestedAccountAmount: make(map[string]map[string]*num.Uint),
	}
}

type dummyAsset struct {
	quantum uint64
}

func (d dummyAsset) Type() *types.Asset {
	return &types.Asset{
		Details: &types.AssetDetails{
			Quantum: num.DecimalFromInt64(int64(d.quantum)),
		},
	}
}

func (dummyAsset) GetAssetClass() common.AssetClass { return common.ERC20 }
func (dummyAsset) IsValid() bool                    { return true }
func (dummyAsset) SetPendingListing()               {}
func (dummyAsset) SetRejected()                     {}
func (dummyAsset) SetEnabled()                      {}
func (dummyAsset) SetValid()                        {}
func (dummyAsset) String() string                   { return "" }

func newSnapshotEngine(t *testing.T, vegaPath paths.Paths, now time.Time, engine *vesting.SnapshotEngine) *snapshot.Engine {
	t.Helper()

	log := logging.NewTestLogger()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snapshot.DefaultConfig()

	snapshotEngine, err := snapshot.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)

	snapshotEngine.AddProviders(engine)

	return snapshotEngine
}

func nextEpoch(ctx context.Context, t *testing.T, te *testSnapshotEngine, startEpochTime time.Time) {
	t.Helper()

	te.engine.OnEpochEvent(ctx, types.Epoch{
		Seq:     te.currentEpoch,
		Action:  vegapb.EpochAction_EPOCH_ACTION_END,
		EndTime: startEpochTime.Add(-1 * time.Second),
	})

	te.currentEpoch += 1
	te.engine.OnEpochEvent(ctx, types.Epoch{
		Seq:       te.currentEpoch,
		Action:    vegapb.EpochAction_EPOCH_ACTION_START,
		StartTime: startEpochTime,
	})
}
