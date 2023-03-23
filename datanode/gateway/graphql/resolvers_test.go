// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package gql_test

import (
	"context"
	"errors"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"

	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/gateway"
	gql "code.vegaprotocol.io/vega/datanode/gateway/graphql"
	"code.vegaprotocol.io/vega/datanode/gateway/graphql/mocks"
	"code.vegaprotocol.io/vega/logging"
	v2 "code.vegaprotocol.io/vega/protos/data-node/api/v2"
	protoTypes "code.vegaprotocol.io/vega/protos/vega"
	datav1 "code.vegaprotocol.io/vega/protos/vega/data/v1"
)

func TestNewResolverRoot_ConstructAndResolve(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	assert.NotNil(t, root)

	partyResolver := root.Party()
	assert.NotNil(t, partyResolver)

	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	depthResolver := root.MarketDepth()
	assert.NotNil(t, depthResolver)

	candleResolver := root.Candle()
	assert.NotNil(t, candleResolver)

	orderResolver := root.Order()
	assert.NotNil(t, orderResolver)

	tradeResolver := root.Trade()
	assert.NotNil(t, tradeResolver)

	priceLevelResolver := root.PriceLevel()
	assert.NotNil(t, priceLevelResolver)

	positionResolver := root.Position()
	assert.NotNil(t, positionResolver)

	queryResolver := root.Query()
	assert.NotNil(t, queryResolver)

	subsResolver := root.Subscription()
	assert.NotNil(t, subsResolver)
}

func TestNewResolverRoot_QueryResolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	assert.NotNil(t, root)

	queryResolver := root.Query()
	assert.NotNil(t, queryResolver)
}

func getTestMarket() *protoTypes.Market {
	pk := types.CreateSignerFromString("0xDEADBEEF", types.DataSignerTypePubKey)

	return &protoTypes.Market{
		Id: "BTC/DEC19",
		TradableInstrument: &protoTypes.TradableInstrument{
			Instrument: &protoTypes.Instrument{
				Id:   "Crypto/BTCUSD/Futures/Dec19",
				Code: "FX:BTCUSD/DEC19",
				Name: "December 2019 BTC vs USD future",
				Metadata: &protoTypes.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &protoTypes.Instrument_Future{
					Future: &protoTypes.Future{
						SettlementAsset: "Ethereum/Ether",
						DataSourceSpecForSettlementData: &protoTypes.DataSourceSpec{
							Data: protoTypes.NewDataSourceDefinition(
								protoTypes.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&protoTypes.DataSourceSpecConfiguration{
									Signers: []*datav1.Signer{pk.IntoProto()},
									Filters: []*datav1.Filter{
										{
											Key: &datav1.PropertyKey{
												Name: "prices.ETH.value",
												Type: datav1.PropertyKey_TYPE_INTEGER,
											},
											Conditions: []*datav1.Condition{},
										},
									},
								},
							),
						},
						DataSourceSpecForTradingTermination: &protoTypes.DataSourceSpec{
							Data: protoTypes.NewDataSourceDefinition(
								protoTypes.DataSourceDefinitionTypeExt,
							).SetOracleConfig(
								&protoTypes.DataSourceSpecConfiguration{
									Signers: []*datav1.Signer{pk.IntoProto()},
									Filters: []*datav1.Filter{
										{
											Key: &datav1.PropertyKey{
												Name: "trading.terminated",
												Type: datav1.PropertyKey_TYPE_BOOLEAN,
											},
											Conditions: []*datav1.Condition{},
										},
									},
								},
							),
						},
						DataSourceSpecBinding: &protoTypes.DataSourceSpecToFutureBinding{
							SettlementDataProperty:     "prices.ETH.value",
							TradingTerminationProperty: "trading.terminated",
						},
					},
				},
			},
			MarginCalculator: &protoTypes.MarginCalculator{
				ScalingFactors: &protoTypes.ScalingFactors{
					SearchLevel:       1.1,
					InitialMargin:     1.2,
					CollateralRelease: 1.4,
				},
			},
			RiskModel: &protoTypes.TradableInstrument_LogNormalRiskModel{
				LogNormalRiskModel: &protoTypes.LogNormalRiskModel{
					RiskAversionParameter: 0.01,
					Tau:                   1.0 / 365.25 / 24,
					Params: &protoTypes.LogNormalModelParams{
						Mu:    0,
						R:     0.016,
						Sigma: 0.09,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &protoTypes.LiquidityMonitoringParameters{
			TriggeringRatio: "0.3",
		},
	}
}

func TestNewResolverRoot_Resolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()

	marketNotExistsErr := errors.New("market does not exist")
	markets := map[string]*protoTypes.Market{
		"BTC/DEC19": getTestMarket(),
		"ETH/USD18": nil,
	}

	root.tradingDataClient.EXPECT().GetAsset(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes().Return(&v2.GetAssetResponse{Asset: &protoTypes.Asset{}}, nil)

	root.tradingDataClient.EXPECT().GetMarket(gomock.Any(), gomock.Any()).Times(len(markets)).DoAndReturn(func(_ context.Context, req *v2.GetMarketRequest, _ ...grpc.CallOption) (*v2.GetMarketResponse, error) {
		m, ok := markets[req.MarketId]
		assert.True(t, ok)
		if m == nil {
			return nil, marketNotExistsErr
		}
		return &v2.GetMarketResponse{Market: m}, nil
	})

	name := "BTC/DEC19"
	vMarkets, err := root.Query().MarketsConnection(ctx, &name, nil, nil)
	assert.Nil(t, err)
	assert.NotNil(t, vMarkets)
	assert.Len(t, vMarkets.Edges, 1)

	name = "ETH/USD18"
	vMarkets, err = root.Query().MarketsConnection(ctx, &name, nil, nil)
	assert.Error(t, err)
	assert.Nil(t, vMarkets)

	name = "barney"
	root.tradingDataClient.EXPECT().ListParties(gomock.Any(), gomock.Any()).Times(1).Return(&v2.ListPartiesResponse{
		Parties: &v2.PartyConnection{
			Edges: []*v2.PartyEdge{
				{
					Node:   &protoTypes.Party{Id: name},
					Cursor: name,
				},
			},
		},
	}, nil)
	vParties, err := root.Query().PartiesConnection(ctx, &name, nil)
	assert.Nil(t, err)
	assert.NotNil(t, vParties)
	assert.Len(t, vParties.Edges, 1)

	root.tradingDataClient.EXPECT().ListParties(gomock.Any(), gomock.Any()).Times(1).Return(&v2.ListPartiesResponse{Parties: &v2.PartyConnection{
		Edges:    nil,
		PageInfo: &v2.PageInfo{},
	}}, nil)
	vParties, err = root.Query().PartiesConnection(ctx, nil, nil)
	assert.NoError(t, err)
	assert.NotNil(t, vParties)
	assert.Equal(t, len(vParties.Edges), 0)
}

func TestNewResolverRoot_MarketResolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()

	marketID := "BTC/DEC19"
	market := &protoTypes.Market{
		Id: marketID,
	}

	root.tradingDataClient.EXPECT().ListOrders(gomock.Any(), gomock.Any()).Times(1).Return(&v2.ListOrdersResponse{Orders: &v2.OrderConnection{
		Edges: []*v2.OrderEdge{
			{
				Node: &protoTypes.Order{
					Id:        "order-id-1",
					Price:     "1000",
					CreatedAt: 1,
				},
				Cursor: "1",
			},
			{
				Node: &protoTypes.Order{
					Id:        "order-id-2",
					Price:     "2000",
					CreatedAt: 2,
				},
				Cursor: "2",
			},
		},
	}}, nil)

	marketResolver := root.Market()
	assert.NotNil(t, marketResolver)

	orders, err := marketResolver.OrdersConnection(ctx, market, nil, nil)
	assert.NotNil(t, orders)
	assert.Nil(t, err)
	assert.Len(t, orders.Edges, 2)
}

func TestRewardsRresolver(t *testing.T) {
	root := buildTestResolverRoot(t)
	defer root.Finish()
	ctx := context.Background()
	partyResolver := root.Party()
	root.tradingDataClient.EXPECT().ListRewardSummaries(gomock.Any(), gomock.Any()).Times(1).Return(nil, errors.New("some error"))
	assetID := "asset"
	r, e := partyResolver.RewardSummaries(ctx, &protoTypes.Party{Id: "some"}, &assetID)
	require.Nil(t, r)
	require.NotNil(t, e)
}

//nolint:interfacebloat
type resolverRoot interface {
	Query() gql.QueryResolver
	Candle() gql.CandleResolver
	MarketDepth() gql.MarketDepthResolver
	MarketDepthUpdate() gql.MarketDepthUpdateResolver
	PriceLevel() gql.PriceLevelResolver
	Market() gql.MarketResolver
	Order() gql.OrderResolver
	Trade() gql.TradeResolver
	Position() gql.PositionResolver
	Party() gql.PartyResolver
	Subscription() gql.SubscriptionResolver
}

type testResolver struct {
	resolverRoot
	log               *logging.Logger
	ctrl              *gomock.Controller
	coreProxyClient   *mocks.MockCoreProxyServiceClient
	tradingDataClient *mocks.MockTradingDataServiceClientV2
}

func buildTestResolverRoot(t *testing.T) *testResolver {
	t.Helper()
	ctrl := gomock.NewController(t)
	log := logging.NewTestLogger()
	conf := gateway.NewDefaultConfig()
	coreProxyClient := mocks.NewMockCoreProxyServiceClient(ctrl)
	tradingDataClientV2 := mocks.NewMockTradingDataServiceClientV2(ctrl)
	resolver := gql.NewResolverRoot(
		log,
		conf,
		coreProxyClient,
		tradingDataClientV2,
	)
	return &testResolver{
		resolverRoot:      resolver,
		log:               log,
		ctrl:              ctrl,
		coreProxyClient:   coreProxyClient,
		tradingDataClient: tradingDataClientV2,
	}
}

func (t *testResolver) Finish() {
	_ = t.log.Sync()
	t.ctrl.Finish()
}
