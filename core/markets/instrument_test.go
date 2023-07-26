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
	"code.vegaprotocol.io/vega/logging"

	"code.vegaprotocol.io/vega/core/products"
	"code.vegaprotocol.io/vega/core/types"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstrument(t *testing.T) {
	t.Run("Create a valid new instrument", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t), mocks.NewMockBroker(gomock.NewController(t)))
		assert.NotNil(t, inst)
		assert.Nil(t, err)
	})

	t.Run("nil product", func(t *testing.T) {
		pinst := getValidInstrumentProto()
		pinst.Product = nil
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t), mocks.NewMockBroker(gomock.NewController(t)))
		assert.Nil(t, inst)
		assert.NotNil(t, err)
		assert.Equal(t, err.Error(), "unable to instantiate product from instrument configuration: nil product")
	})

	t.Run("nil oracle spec", func(t *testing.T) {
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
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t), mocks.NewMockBroker(gomock.NewController(t)))
		require.NotNil(t, err)
		assert.Nil(t, inst)
		assert.Equal(t, "unable to instantiate product from instrument configuration: a data source spec and spec binding are required", err.Error())
	})

	t.Run("nil oracle spec binding", func(t *testing.T) {
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
		inst, err := markets.NewInstrument(context.Background(), logging.NewTestLogger(), pinst, newOracleEngine(t), mocks.NewMockBroker(gomock.NewController(t)))
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
