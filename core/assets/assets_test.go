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

package assets_test

import (
	"context"
	"encoding/hex"
	"math/big"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/assets"
	erc20mocks "code.vegaprotocol.io/vega/core/assets/erc20/mocks"
	"code.vegaprotocol.io/vega/core/assets/mocks"
	bmocks "code.vegaprotocol.io/vega/core/broker/mocks"
	nwethmocks "code.vegaprotocol.io/vega/core/nodewallets/eth/mocks"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	vgrand "code.vegaprotocol.io/vega/libs/rand"
	"code.vegaprotocol.io/vega/logging"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
)

type testService struct {
	*assets.Service
	broker              *bmocks.MockInterface
	primaryBridgeView   *mocks.MockERC20BridgeView
	secondaryBridgeView *mocks.MockERC20BridgeView
	notary              *mocks.MockNotary
	ctrl                *gomock.Controller
	primaryEthClient    *erc20mocks.MockETHClient
	ethWallet           *nwethmocks.MockEthereumWallet
	secondaryEthClient  *erc20mocks.MockETHClient
}

func TestAssets(t *testing.T) {
	t.Run("Staging asset update for unknown asset fails", testStagingAssetUpdateForUnknownAssetFails)
	t.Run("Offers signature on tick success", testOffersSignaturesOnTickSuccess)
	t.Run("Checking an assets address when chain id is unknown", testValidateUnknownChainID)
}

func testOffersSignaturesOnTickSuccess(t *testing.T) {
	service := getTestService(t)

	assetID := hex.EncodeToString([]byte("asset_id"))

	assetDetails := &types.AssetDetails{
		Name:     vgrand.RandomStr(5),
		Symbol:   vgrand.RandomStr(3),
		Decimals: 10,
		Quantum:  num.DecimalFromInt64(42),
		Source: &types.AssetDetailsErc20{
			ERC20: &types.ERC20{
				ContractAddress:   vgrand.RandomStr(5),
				LifetimeLimit:     num.NewUint(42),
				WithdrawThreshold: num.NewUint(84),
				ChainID:           "1",
			},
		},
	}

	ctx := context.Background()

	service.broker.EXPECT().Send(gomock.Any()).Times(1)
	_, err := service.NewAsset(ctx, assetID, assetDetails)
	require.NoError(t, err)

	service.broker.EXPECT().Send(gomock.Any()).Times(1)
	require.NoError(t, service.Enable(ctx, assetID))

	nodeSignature := []byte("node_signature")
	service.notary.EXPECT().
		OfferSignatures(gomock.Any(), gomock.Any()).DoAndReturn(
		func(kind types.NodeSignatureKind, f func(id string) []byte) {
			require.Equal(t, kind, types.NodeSignatureKindAssetNew)
			require.Equal(t, nodeSignature, f(assetID))
		},
	)
	service.primaryEthClient.EXPECT().CollateralBridgeAddress().Times(1)
	service.ethWallet.EXPECT().Algo().Times(1)
	service.ethWallet.EXPECT().Sign(gomock.Any()).Return(nodeSignature, nil).Times(1)
	service.OnTick(ctx, time.Now())
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
					ChainID:           "1",
				},
			},
		},
	}

	// when
	err := service.StageAssetUpdate(asset)

	// then
	require.ErrorIs(t, err, assets.ErrAssetDoesNotExist)
}

func testValidateUnknownChainID(t *testing.T) {
	service := getTestService(t)

	require.NoError(t, service.ValidateEthereumAddress(vgrand.RandomStr(5), "1"))
	require.NoError(t, service.ValidateEthereumAddress(vgrand.RandomStr(5), "2"))
	require.ErrorIs(t, service.ValidateEthereumAddress(vgrand.RandomStr(5), "666"), assets.ErrUnknownChainID)
}

func getTestService(t *testing.T) *testService {
	t.Helper()
	conf := assets.NewDefaultConfig()
	logger := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	primaryEthClient := erc20mocks.NewMockETHClient(ctrl)
	primaryEthClient.EXPECT().ChainID(gomock.Any()).AnyTimes().Return(big.NewInt(1), nil)
	secondaryEthClient := erc20mocks.NewMockETHClient(ctrl)
	secondaryEthClient.EXPECT().ChainID(gomock.Any()).AnyTimes().Return(big.NewInt(2), nil)
	broker := bmocks.NewMockInterface(ctrl)
	primaryBridgeView := mocks.NewMockERC20BridgeView(ctrl)
	secondaryBridgeView := mocks.NewMockERC20BridgeView(ctrl)
	notary := mocks.NewMockNotary(ctrl)
	ethWallet := nwethmocks.NewMockEthereumWallet(ctrl)

	service, _ := assets.New(context.Background(), logger, conf, ethWallet, primaryEthClient, secondaryEthClient, broker, primaryBridgeView, secondaryBridgeView, notary, true)
	return &testService{
		Service:             service,
		broker:              broker,
		ctrl:                ctrl,
		primaryBridgeView:   primaryBridgeView,
		secondaryBridgeView: secondaryBridgeView,
		notary:              notary,
		primaryEthClient:    primaryEthClient,
		secondaryEthClient:  secondaryEthClient,
		ethWallet:           ethWallet,
	}
}
