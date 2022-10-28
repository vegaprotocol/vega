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

func TestMarketRestoreFromCheckpoint(t *testing.T) {
	now := time.Now()
	ex, gov, cpEng := createExecutionEngine(t, now)
	genesis := &checkpoint.GenesisState{
		CheckpointHash:  "36fe3e8a6dea4a89c983ae9467ad05e656fa7754857f6a917d588d157b9b064f",
		CheckpointState: base64.StdEncoding.EncodeToString(cpFile),
	}
	gd := &struct {
		Checkpoint *checkpoint.GenesisState `json:"checkpoint"`
	}{}

	gd.Checkpoint = genesis
	gdBytes, _ := json.Marshal(gd)

	require.NoError(t, cpEng.UponGenesis(context.Background(), gdBytes))

	expectedMarkets := []string{
		"82644318987f9c2a63c3cab6d210c2b034fb4caba0e22327f1f2ed47f4dfb97d",
		"18ab3360c634cb0f8f195b6336d1fefe9ec1f7e35fda2472529c76e82c2a3597",
		"5e66c6db9dc321c5351eef38b7d1c780e06577306ed29f773e929de0ca50183e",
		"22d3553ee217c8d8db5a0b975fd35d5f429cb529587d51ee1b2abfa8399c5e52",
		"ac5f1fdfe21bff8daa8a9008faa9760cae7df908c9cdd1d74c007240453b77db",
		"471973be39e0e242173117adcb0ddc1fa1fcef8ec8c108b86ce593924d0797db",
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
