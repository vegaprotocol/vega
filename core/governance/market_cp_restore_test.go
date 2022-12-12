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

package governance_test

import (
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	amocks "code.vegaprotocol.io/vega/core/assets/mocks"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/checkpoint"
	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/execution"
	emocks "code.vegaprotocol.io/vega/core/execution/mocks"
	"code.vegaprotocol.io/vega/core/governance"
	"code.vegaprotocol.io/vega/core/governance/mocks"
	"code.vegaprotocol.io/vega/core/netparams"
	"code.vegaprotocol.io/vega/core/nodewallets"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/proto"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	vgtesting "code.vegaprotocol.io/vega/libs/testing"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	checkpointpb "code.vegaprotocol.io/vega/protos/vega/checkpoint/v1"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

//go:embed testcp/checkpoint.cp
var cpFile []byte

// Disable 'TestMarketRestoreFromCheckpoint' for now. 'testcp/checkpoint.cp' needs to be regenerated for the new data sourcing types.
// Rest of functions disabled because linter complains.

func TestMarketRestoreFromCheckpoint(t *testing.T) {
	t.Skipf("Skipping test as need to regenerate testcp/checkpoint.cp with appropriate values for LP - Zohar to fix")
	now := time.Now()
	ex, gov, cpEng := createExecutionEngine(t, now)
	genesis := &checkpoint.GenesisState{
		CheckpointHash:  "b60aa26b5b167ecf72620778b481a65d029367c1a56bd280b55614b5586f3e8c",
		CheckpointState: base64.StdEncoding.EncodeToString(cpFile),
	}
	gd := &struct {
		Checkpoint *checkpoint.GenesisState `json:"checkpoint"`
	}{}

	gd.Checkpoint = genesis
	gdBytes, _ := json.Marshal(gd)

	require.NoError(t, cpEng.UponGenesis(context.Background(), gdBytes))

	expectedMarkets := []string{
		"86948f946a64e14bb2e284f825cd46879d1cb581ce405cc62e4f74fcded190d3",
		"3fed7242cce2cbe7df8cc3a2808969fc6e108f2047838c4af323c10430dfe041",
		"5ed56476934d952229c0d796e143be4d1a96871d1607f9188dfa3727bdd6f7a0",
		"2bf3cab7a239f34f40145a0700f8f12bc504bd2ec3a65d5915c4e58881dfcb52",
		"eda61c9948ae97182c344b6a900e960a6c85271a4db1926eb26c82d847d9ba78",
		"954410d873a6d1a419b8a11e7e3a5178f65b976f3140bb78fc97d21daf08877f",
		"14719259af09239e479c107c6a69dbdae05dbde619ad06632af27a2fc2c9a9c7",
		"3201812426fed4cc6d5cfbacdaa54e738791deb9f72743f8b18d3e9f6a3e222c",
	}
	govProposalsCP, _ := gov.Checkpoint()
	proposals := &checkpointpb.Proposals{}
	err := proto.Unmarshal(govProposalsCP, proposals)
	require.NoError(t, err)
	require.Equal(t, len(expectedMarkets), len(proposals.Proposals))

	for i, expectedMarket := range expectedMarkets {
		m, exists := ex.GetMarket(expectedMarket)
		require.True(t, exists)
		require.Equal(t, types.MarketTradingModeOpeningAuction, m.TradingMode)
		require.Equal(t, types.MarketStatePending, m.State)
		require.Equal(t, expectedMarket, proposals.Proposals[i].Id)
	}
}

func getNodeWallet() *nodewallets.NodeWallets {
	vegaPaths, cleanupFn := vgtesting.NewVegaPaths()
	defer cleanupFn()
	registryPass := vgrand.RandomStr(10)
	walletsPass := vgrand.RandomStr(10)
	config := nodewallets.NewDefaultConfig()
	createTestNodeWallets(vegaPaths, registryPass, walletsPass)
	nw, _ := nodewallets.GetNodeWallets(config, vegaPaths, registryPass)
	return nw
}

func createTestNodeWallets(vegaPaths paths.Paths, registryPass, walletPass string) {
	if _, err := nodewallets.GenerateEthereumWallet(vegaPaths, registryPass, walletPass, "", false); err != nil {
		panic("couldn't generate Ethereum node wallet for tests")
	}

	if _, err := nodewallets.GenerateVegaWallet(vegaPaths, registryPass, walletPass, false); err != nil {
		panic("couldn't generate Vega node wallet for tests")
	}
}

func createExecutionEngine(t *testing.T, tm time.Time) (*execution.Engine, *governance.Engine, *checkpoint.Engine) {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	executionConfig := execution.NewDefaultConfig()
	broker := bmocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()
	broker.EXPECT().SendBatch(gomock.Any()).AnyTimes()
	timeService := mocks.NewMockTimeService(ctrl)
	timeService.EXPECT().GetTimeNow().Return(tm).AnyTimes()

	collateralService := collateral.New(log, collateral.NewDefaultConfig(), timeService, broker)
	oracleService := emocks.NewMockOracleEngine(ctrl)
	oracleService.EXPECT().Subscribe(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	statevar := emocks.NewMockStateVarEngine(ctrl)
	statevar.EXPECT().RegisterStateVariable(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
	statevar.EXPECT().NewEvent(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

	epochEngine := emocks.NewMockEpochEngine(ctrl)
	epochEngine.EXPECT().NotifyOnEpoch(gomock.Any(), gomock.Any()).Times(1)

	bridgeView := amocks.NewMockERC20BridgeView(ctrl)
	notary := amocks.NewMockNotary(ctrl)

	asset := assets.New(log, assets.NewDefaultConfig(), getNodeWallet(), nil, broker, bridgeView, notary, false)
	marketTracker := execution.NewMarketActivityTracker(log, epochEngine)
	exec := execution.NewEngine(log, executionConfig, timeService, collateralService, oracleService, broker, statevar, marketTracker, asset)
	accounts := mocks.NewMockStakingAccounts(ctrl)

	witness := mocks.NewMockWitness(ctrl)
	netp := netparams.New(log, netparams.NewDefaultConfig(), broker)

	gov := governance.NewEngine(log, governance.NewDefaultConfig(), accounts, timeService, broker, asset, witness, exec, netp)
	cpEngine, _ := checkpoint.New(log, checkpoint.NewDefaultConfig(), gov, netp, asset, collateralService, marketTracker)

	return exec, gov, cpEngine
}
