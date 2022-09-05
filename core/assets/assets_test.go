package assets_test

import (
	"testing"

	"code.vegaprotocol.io/vega/core/assets"
	erc20mocks "code.vegaprotocol.io/vega/core/assets/erc20/mocks"
	"code.vegaprotocol.io/vega/core/assets/mocks"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/nodewallets"
	nweth "code.vegaprotocol.io/vega/core/nodewallets/eth"
	nwvega "code.vegaprotocol.io/vega/core/nodewallets/vega"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testService struct {
	*assets.Service
	broker     *bmocks.MockInterface
	bridgeView *mocks.MockERC20BridgeView
	ctrl       *gomock.Controller
}

func TestAssets(t *testing.T) {
	t.Run("Staging asset update for unknown asset fails", testStagingAssetUpdateForUnknownAssetFails)
}

func testStagingAssetUpdateForUnknownAssetFails(t *testing.T) {
	service := getTestService(t)

	// given
	asset := &types.Asset{
		ID: vgrand.RandomStr(5),
		Details: &types.AssetDetails{
			Name:     vgrand.RandomStr(5),
			Symbol:   vgrand.RandomStr(3),
			Decimals: 10,
			Quantum:  num.DecimalFromInt64(42),
			Source: &types.AssetDetailsErc20{
				ERC20: &types.ERC20{
					ContractAddress:   vgrand.RandomStr(5),
					LifetimeLimit:     num.NewUint(42),
					WithdrawThreshold: num.NewUint(84),
				},
			},
		},
	}

	// when
	err := service.StageAssetUpdate(asset)

	// then
	require.ErrorIs(t, err, assets.ErrAssetDoesNotExist)
}

func getTestService(t *testing.T) *testService {
	t.Helper()
	conf := assets.NewDefaultConfig()
	logger := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	ethClient := erc20mocks.NewMockETHClient(ctrl)
	broker := bmocks.NewMockInterface(ctrl)
	bridgeView := mocks.NewMockERC20BridgeView(ctrl)
	nodeWallets := &nodewallets.NodeWallets{
		Vega:     &nwvega.Wallet{},
		Ethereum: &nweth.Wallet{},
	}
	service := assets.New(logger, conf, nodeWallets, ethClient, broker, bridgeView, true)
	return &testService{
		Service:    service,
		broker:     broker,
		ctrl:       ctrl,
		bridgeView: bridgeView,
	}
}
