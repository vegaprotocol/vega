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

package markets_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/core/broker/mocks"
	"code.vegaprotocol.io/vega/core/datasource"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	emock "code.vegaprotocol.io/vega/core/execution/common/mocks"
	"code.vegaprotocol.io/vega/core/markets"
	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/types"
	tmocks "code.vegaprotocol.io/vega/core/vegatime/mocks"
	"code.vegaprotocol.io/vega/logging"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstrument(t *testing.T) {
	t.Run("Create a valid new instrument", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		pinst := getValidInstrumentProto()
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, "", tmocks.NewMockTimeService(ctrl), newOracleEngine(t), mocks.NewMockBroker(ctrl), 1)
		assert.NotNil(t, inst)
		assert.Nil(t, err)
	})

	t.Run("nil product", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		pinst := getValidInstrumentProto()
		pinst.Product = nil
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, "", tmocks.NewMockTimeService(ctrl), newOracleEngine(t), mocks.NewMockBroker(ctrl), 1)
		assert.Nil(t, inst)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "unable to instantiate product from instrument configuration: nil product")
	})

	t.Run("nil oracle spec", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		pinst := getValidInstrumentProto()
		pinst.Product = &types.InstrumentFuture{
			Future: &types.Future{
				SettlementAsset:                     "Ethereum/Ether",
				DataSourceSpecForSettlementData:     nil,
				DataSourceSpecForTradingTermination: nil,
				DataSourceSpecBinding: &datasource.SpecBindingForFuture{
					SettlementDataProperty:     "prices.ETH.value",
					TradingTerminationProperty: "trading.terminated",
				},
			},
		}
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, "", tmocks.NewMockTimeService(ctrl), newOracleEngine(t), mocks.NewMockBroker(ctrl), 1)
		require.NotNil(t, err)
		assert.Nil(t, inst)
		assert.Equal(t, "unable to instantiate product from instrument configuration: a data source spec and spec binding are required", err.Error())
	})

	t.Run("nil oracle spec binding", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		pinst := getValidInstrumentProto()
		pinst.Product = &types.InstrumentFuture{
			Future: &types.Future{
				SettlementAsset: "Ethereum/Ether",
				DataSourceSpecForSettlementData: &datasource.Spec{
					Data: datasource.NewDefinition(
						datasource.ContentTypeOracle,
					).SetOracleConfig(
						&signedoracle.SpecConfiguration{
							Signers: []*dstypes.Signer{dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)},
							Filters: []*dstypes.SpecFilter{
								{
									Key: &dstypes.SpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*dstypes.SpecCondition{},
								},
							},
						},
					),
				},
				DataSourceSpecForTradingTermination: &datasource.Spec{
					Data: datasource.NewDefinition(
						datasource.ContentTypeOracle,
					).SetOracleConfig(
						&signedoracle.SpecConfiguration{
							Signers: []*dstypes.Signer{dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)},
							Filters: []*dstypes.SpecFilter{
								{
									Key: &dstypes.SpecPropertyKey{
										Name: "trading.terminated",
										Type: datapb.PropertyKey_TYPE_BOOLEAN,
									},
									Conditions: []*dstypes.SpecCondition{},
								},
							},
						},
					),
				},
				DataSourceSpecBinding: nil,
			},
		}
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, "", tmocks.NewMockTimeService(ctrl), newOracleEngine(t), mocks.NewMockBroker(ctrl), 1)
		require.NotNil(t, err)
		assert.Nil(t, inst)
		assert.Equal(t, "unable to instantiate product from instrument configuration: a data source spec and spec binding are required", err.Error())
	})
}

func newOracleEngine(t *testing.T) products.OracleEngine {
	t.Helper()
	ctrl := gomock.NewController(t)
	broker := mocks.NewMockBroker(ctrl)
	broker.EXPECT().Send(gomock.Any()).AnyTimes()

	ts := emock.NewMockTimeService(ctrl)
	ts.EXPECT().GetTimeNow().AnyTimes()

	return spec.NewEngine(
		logging.NewTestLogger(),
		spec.NewDefaultConfig(),
		ts,
		broker,
	)
}

func getValidInstrumentProto() *types.Instrument {
	return &types.Instrument{
		ID:   "Crypto/BTCUSD/Futures/Dec19",
		Code: "FX:BTCUSD/DEC19",
		Name: "December 2019 BTC vs USD future",
		Metadata: &types.InstrumentMetadata{
			Tags: []string{
				"asset_class:fx/crypto",
				"product:futures",
			},
		},
		Product: &types.InstrumentFuture{
			Future: &types.Future{
				QuoteName:       "USD",
				SettlementAsset: "Ethereum/Ether",
				DataSourceSpecForSettlementData: &datasource.Spec{
					Data: datasource.NewDefinition(
						datasource.ContentTypeOracle,
					).SetOracleConfig(
						&signedoracle.SpecConfiguration{
							Signers: []*dstypes.Signer{dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)},
							Filters: []*dstypes.SpecFilter{
								{
									Key: &dstypes.SpecPropertyKey{
										Name: "prices.ETH.value",
										Type: datapb.PropertyKey_TYPE_INTEGER,
									},
									Conditions: []*dstypes.SpecCondition{},
								},
							},
						},
					),
				},
				DataSourceSpecForTradingTermination: &datasource.Spec{
					Data: datasource.NewDefinition(
						datasource.ContentTypeOracle,
					).SetOracleConfig(
						&signedoracle.SpecConfiguration{
							Signers: []*dstypes.Signer{dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey)},
							Filters: []*dstypes.SpecFilter{
								{
									Key: &dstypes.SpecPropertyKey{
										Name: "trading.terminated",
										Type: datapb.PropertyKey_TYPE_BOOLEAN,
									},
									Conditions: []*dstypes.SpecCondition{},
								},
							},
						},
					),
				},
				DataSourceSpecBinding: &datasource.SpecBindingForFuture{
					SettlementDataProperty:     "prices.ETH.value",
					TradingTerminationProperty: "trading.terminated",
				},
			},
		},
	}
}
