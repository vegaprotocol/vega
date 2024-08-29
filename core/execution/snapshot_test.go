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

package execution_test

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/collateral"
	"code.vegaprotocol.io/vega/core/datasource"
	dstypes "code.vegaprotocol.io/vega/core/datasource/common"
	"code.vegaprotocol.io/vega/core/datasource/external/signedoracle"
	"code.vegaprotocol.io/vega/core/datasource/spec"
	"code.vegaprotocol.io/vega/core/epochtime"
	"code.vegaprotocol.io/vega/core/execution"
	"code.vegaprotocol.io/vega/core/execution/common"
	"code.vegaprotocol.io/vega/core/execution/common/mocks"
	fmock "code.vegaprotocol.io/vega/core/fee/mocks"
	"code.vegaprotocol.io/vega/core/integration/stubs"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vgtest "code.vegaprotocol.io/vega/libs/test"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	"code.vegaprotocol.io/vega/protos/vega"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type snapshotTestData struct {
	engine           *execution.Engine
	oracleEngine     *spec.Engine
	snapshotEngine   *snp.Engine
	timeService      *stubs.TimeStub
	collateralEngine *collateral.Engine
}

type stubIDGen struct {
	calls int
}

// TestSnapshotOraclesTerminatingMarketFromSnapshot tests that market loaded from snapshot can be terminated with its oracle.
func TestSnapshotOraclesTerminatingMarketFromSnapshot(t *testing.T) {
	now := time.Now()
	exec := getEngine(t, paths.New(t.TempDir()), now)
	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}
	mkt := newMarket("MarketID", pubKey)
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	marketState, _, _ := exec.engine.GetState("")

	exec2 := getEngine(t, paths.New(t.TempDir()), now)
	marketSnap := &snapshot.Payload{}
	proto.Unmarshal(marketState, marketSnap)

	_, _ = exec2.engine.LoadState(context.Background(), types.PayloadFromProto(marketSnap))

	// restore collateral
	accountsState, _, _ := exec.collateralEngine.GetState("accounts")
	accountsSnap := &snapshot.Payload{}
	proto.Unmarshal(accountsState, accountsSnap)

	_, _ = exec2.collateralEngine.LoadState(context.Background(), types.PayloadFromProto(accountsSnap))

	state2, _, _ := exec2.engine.GetState("")

	err = exec.engine.StartOpeningAuction(context.Background(), mkt.ID)
	require.NoError(t, err)
	mktState, err := exec.engine.GetMarketState("MarketID")
	require.NoError(t, err)
	require.Equal(t, types.MarketStateActive, mktState)

	err = exec2.engine.StartOpeningAuction(context.Background(), mkt.ID)
	require.NoError(t, err)
	mktState, err = exec2.engine.GetMarketState("MarketID")
	require.NoError(t, err)
	require.Equal(t, types.MarketStateActive, mktState)

	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString(pubKey.PubKey.Key, dstypes.SignerTypePubKey),
	}

	exec.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"trading.terminated": "true"},
	})

	exec2.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"trading.terminated": "true"},
	})

	marketState1, _ := exec.engine.GetMarketState("MarketID")
	marketState2, _ := exec2.engine.GetMarketState("MarketID")

	require.Equal(t, marketState1, marketState2)
	require.Equal(t, types.MarketStateTradingTerminated, marketState1)
	require.Equal(t, types.MarketStateTradingTerminated, marketState2)

	exec.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"prices.ETH.value": "100"},
	})

	exec2.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"prices.ETH.value": "100"},
	})

	marketState1, _ = exec.engine.GetMarketState("MarketID")
	marketState2, _ = exec2.engine.GetMarketState("MarketID")
	require.Equal(t, marketState1, marketState2)
	require.Equal(t, types.MarketStateSettled, marketState1)
	require.Equal(t, types.MarketStateSettled, marketState2)

	require.True(t, bytes.Equal(marketState, state2))
}

// TestSnapshotOraclesTerminatingMarketSettleAfterSnapshot tests that market loaded from snapshot can be terminated with its oracle.
// the settlement data will be sent before the snapshot is taken, to ensure settlement data is restored correctly.
func TestSnapshotOraclesTerminatingMarketSettleAfterSnapshot(t *testing.T) {
	now := time.Now()
	exec := getEngineWithParties(t, now, num.NewUint(1000000000), "lp", "p1", "p2", "p3", "p4")
	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}

	mkt := newMarketWithAuctionDuration("MarketID", pubKey, &types.AuctionDuration{Duration: 1})
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	err = exec.engine.StartOpeningAuction(context.Background(), mkt.ID)
	require.NoError(t, err)
	mktState, err := exec.engine.GetMarketState("MarketID")
	require.NoError(t, err)
	require.Equal(t, types.MarketStatePending, mktState)

	md, err := exec.engine.GetMarketData(mkt.ID)
	require.NoError(t, err)
	require.Equal(t, types.MarketTradingModeOpeningAuction, md.MarketTradingMode)

	idgen := &stubIDGen{}
	// now let's submit some orders and get market to trade continuously
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         mkt.ID,
		CommitmentAmount: num.NewUint(1000000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "lp1",
	}
	// submit LP
	vgctx := vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("0deadbeef")))
	_ = exec.engine.SubmitLiquidityProvision(vgctx, lpSubmission, "lp", idgen.NextID())
	// uncrossing orders
	os1 := &types.OrderSubmission{
		MarketID:    mkt.ID,
		Price:       num.NewUint(99),
		Size:        1,
		Side:        types.SideBuy,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		Reference:   "o1",
	}
	os2 := &types.OrderSubmission{
		MarketID:    mkt.ID,
		Price:       num.NewUint(99),
		Size:        1,
		Side:        types.SideSell,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		Reference:   "o2",
	}
	_, _ = exec.engine.SubmitOrder(vgctx, os1, "p1", idgen, "o1p1")
	_, _ = exec.engine.SubmitOrder(vgctx, os2, "p2", idgen, "o2p2")
	// have some volume on the book
	os1 = &types.OrderSubmission{
		MarketID:    mkt.ID,
		Price:       num.NewUint(85),
		Size:        1,
		Side:        types.SideBuy,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		Reference:   "o3",
	}
	os2 = &types.OrderSubmission{
		MarketID:    mkt.ID,
		Price:       num.NewUint(110),
		Size:        1,
		Side:        types.SideSell,
		TimeInForce: types.OrderTimeInForceGTC,
		Type:        types.OrderTypeLimit,
		Reference:   "o4",
	}
	_, _ = exec.engine.SubmitOrder(vgctx, os1, "p3", idgen, "o3p3")
	_, _ = exec.engine.SubmitOrder(vgctx, os2, "p4", idgen, "o4p4")

	// OK, we now have stuff on the book, so we should be able to leave opening auction
	now = now.Add(60 * time.Second) // move ahead 1 minute
	// We probably need to add a hash to this context
	vgctx = vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("1deadbeef")))
	exec.engine.OnTick(vgctx, now)
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString(pubKey.PubKey.Key, dstypes.SignerTypePubKey),
	}

	// provide settlement data for first market
	vgctx = vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("2deadbeef")))
	exec.oracleEngine.BroadcastData(vgctx, dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"prices.ETH.value": "100"},
	})
	// then create snapshot of market
	state, _, _ := exec.engine.GetState("")

	exec2 := getEngine(t, paths.New(t.TempDir()), now)
	snap := &snapshot.Payload{}
	proto.Unmarshal(state, snap)
	vgctx = vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("3deadbeef")))
	_, _ = exec2.engine.LoadState(vgctx, types.PayloadFromProto(snap))

	state2, _, _ := exec2.engine.GetState("")
	// the states should match
	require.True(t, bytes.Equal(state, state2))

	// restore collateral
	accountsState, _, _ := exec.collateralEngine.GetState("accounts")
	accountsSnap := &snapshot.Payload{}
	proto.Unmarshal(accountsState, accountsSnap)

	_, _ = exec2.collateralEngine.LoadState(context.Background(), types.PayloadFromProto(accountsSnap))

	vgctx = vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("3deadbeef")))
	exec.oracleEngine.BroadcastData(vgctx, dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"trading.terminated": "true"},
	})

	exec2.oracleEngine.BroadcastData(vgctx, dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"trading.terminated": "true"},
	})

	marketState1, _ := exec.engine.GetMarketState("MarketID")
	marketState2, _ := exec2.engine.GetMarketState("MarketID")

	// markets should both be settled
	require.Equal(t, marketState1, marketState2)
	require.Equal(t, types.MarketStateSettled, marketState1)
	require.Equal(t, types.MarketStateSettled, marketState2)
}

// TestSnapshotOraclesTerminatingMarketFromSnapshotAfterSettlementData sets up a market that gets the settlement data first.
// Then a snapshot is taken and another node is restored from this snapshot. Finally trading termination data is received and both markets
// are expected to get settled.
func TestSnapshotOraclesTerminatingMarketFromSnapshotAfterSettlementData(t *testing.T) {
	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
	}

	now := time.Now()
	exec := getEngine(t, paths.New(t.TempDir()), now)
	mkt := newMarket("MarketID", pubKeys[0].Signer.(*dstypes.SignerPubKey))
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	err = exec.engine.StartOpeningAuction(context.Background(), mkt.ID)
	require.NoError(t, err)
	mktState, err := exec.engine.GetMarketState("MarketID")
	require.NoError(t, err)
	require.Equal(t, types.MarketStateActive, mktState)

	// set up market to get to continuous trading

	// settlement data arrives first
	exec.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"prices.ETH.value": "100"},
	})

	// take a snapshot
	state, _, _ := exec.engine.GetState("")

	// load from the snapshot
	exec2 := getEngine(t, paths.New(t.TempDir()), now)
	snap := &snapshot.Payload{}
	proto.Unmarshal(state, snap)
	_, _ = exec2.engine.LoadState(context.Background(), types.PayloadFromProto(snap))

	// take a snapshot on the loaded engine
	state2, _, _ := exec2.engine.GetState("")
	require.True(t, bytes.Equal(state, state2))

	// restore collateral
	accountsState, _, _ := exec.collateralEngine.GetState("accounts")
	accountsSnap := &snapshot.Payload{}
	proto.Unmarshal(accountsState, accountsSnap)

	_, _ = exec2.collateralEngine.LoadState(context.Background(), types.PayloadFromProto(accountsSnap))

	// terminate the market to lead to settlement
	exec.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"trading.terminated": "true"},
	})

	exec2.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
		Signers: pubKeys,
		Data:    map[string]string{"trading.terminated": "true"},
	})

	// take snapshot for both engines, and verify they're both settled
	marketState1, _ := exec.engine.GetMarketState("MarketID")
	marketState2, _ := exec2.engine.GetMarketState("MarketID")
	require.Equal(t, marketState1, marketState2)
	require.Equal(t, types.MarketStateSettled, marketState1)
	require.Equal(t, types.MarketStateSettled, marketState2)
}

// TestLoadTerminatedMarketFromSnapshot terminates markets, loads them using the snapshot engine and then settles them successfully.
func TestLoadTerminatedMarketFromSnapshot(t *testing.T) {
	ctx := vgtest.VegaContext("chainid", 100)

	now := time.Now()
	vegaPath := paths.New(t.TempDir())
	executionEngine1 := getEngine(t, vegaPath, now)
	snapshotEngine1CloseFn := vgtest.OnlyOnce(executionEngine1.snapshotEngine.Close)
	defer snapshotEngine1CloseFn()

	require.NoError(t, executionEngine1.snapshotEngine.Start(ctx))

	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("0xDEADBEFF", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("0xDEADBFFF", dstypes.SignerTypePubKey),
	}
	marketIDs := []string{"market1", "market2", "market3"}

	// submit and terminate all markets
	for i := 0; i < 3; i++ {
		mkt := newMarket(marketIDs[i], pubKeys[i].Signer.(*dstypes.SignerPubKey))
		err := executionEngine1.engine.SubmitMarket(ctx, mkt, "", time.Now())
		require.NoError(t, err)

		// verify markets are terminated
		marketState, err := executionEngine1.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, types.MarketStateProposed, marketState)

		err = executionEngine1.engine.StartOpeningAuction(context.Background(), mkt.ID)
		require.NoError(t, err)
		marketState, err = executionEngine1.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, marketState, types.MarketStateActive)

		// terminate all markets
		require.NoError(t, executionEngine1.oracleEngine.BroadcastData(ctx, dstypes.Data{
			Signers: []*dstypes.Signer{pubKeys[i]},
			Data:    map[string]string{"trading.terminated": "true"},
		}))

		marketState, err = executionEngine1.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, types.MarketStateTradingTerminated, marketState)
	}

	// we now have 3 terminated markets in the execution engine
	// let's take a snapshot
	hash1, err := executionEngine1.snapshotEngine.SnapshotNow(ctx)
	require.NoError(t, err)

	executionEngine1.timeService.SetTime(now.Add(2 * time.Second))

	for i := 0; i < 3; i++ {
		require.NoError(t, executionEngine1.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
			Signers: []*dstypes.Signer{pubKeys[i]},
			Data:    map[string]string{"prices.ETH.value": "100"},
		}))

		marketState1, err := executionEngine1.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, types.MarketStateSettled, marketState1)
	}

	state1 := map[string][]byte{}
	for _, key := range executionEngine1.engine.Keys() {
		state, additionalProvider, err := executionEngine1.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state1[key] = state
	}

	snapshotEngine1CloseFn()

	// now let's start from this snapshot
	executionEngine2 := getEngine(t, vegaPath, now)
	defer executionEngine2.snapshotEngine.Close()

	// This triggers the state restoration from the local snapshot.
	require.NoError(t, executionEngine2.snapshotEngine.Start(context.Background()))

	// Comparing the hash after restoration, to ensure it produces the same result.
	hash2, _, _ := executionEngine2.snapshotEngine.Info()
	require.Equal(t, hash1, hash2)

	// progress time to trigger any side effect on time ticks
	executionEngine2.timeService.SetTime(now.Add(2 * time.Second))

	// settle the markets
	for i := 0; i < 3; i++ {
		require.NoError(t, executionEngine2.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
			Signers: []*dstypes.Signer{pubKeys[i]},
			Data:    map[string]string{"prices.ETH.value": "100"},
		}))

		marketState2, err := executionEngine2.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, types.MarketStateSettled, marketState2)
	}

	state2 := map[string][]byte{}
	for _, key := range executionEngine2.engine.Keys() {
		state, additionalProvider, err := executionEngine2.engine.GetState(key)
		require.NoError(t, err)
		assert.Empty(t, additionalProvider)
		state2[key] = state
	}

	for key := range state1 {
		assert.Equalf(t, state1[key], state2[key], "Key %q does not have the same data", key)
	}
}

func newMarket(ID string, pubKey *dstypes.SignerPubKey) *types.Market {
	return newMarketWithAuctionDuration(ID, pubKey, nil)
}

func newMarketWithAuctionDuration(ID string, pubKey *dstypes.SignerPubKey, auctionDuration *types.AuctionDuration) *types.Market {
	return &types.Market{
		ID: ID, // ID will be generated
		PriceMonitoringSettings: &types.PriceMonitoringSettings{
			Parameters: &types.PriceMonitoringParameters{
				Triggers: []*types.PriceMonitoringTrigger{
					{
						Horizon:          1000,
						HorizonDec:       num.DecimalFromFloat(1000.0),
						Probability:      num.DecimalFromFloat(0.3),
						AuctionExtension: 10000,
					},
				},
			},
		},
		LiquidityMonitoringParameters: &types.LiquidityMonitoringParameters{
			TargetStakeParameters: &types.TargetStakeParameters{
				TimeWindow:    100,
				ScalingFactor: num.DecimalFromFloat(1.0),
			},
		},
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          num.DecimalFromFloat(0.1),
				InfrastructureFee: num.DecimalFromFloat(0.1),
				LiquidityFee:      num.DecimalFromFloat(0.1),
			},
			LiquidityFeeSettings: &types.LiquidityFeeSettings{
				Method: vega.LiquidityFeeSettings_METHOD_MARGINAL_COST,
			},
		},
		TradableInstrument: &types.TradableInstrument{
			MarginCalculator: &types.MarginCalculator{
				ScalingFactors: &types.ScalingFactors{
					SearchLevel:       num.DecimalFromFloat(1.2),
					InitialMargin:     num.DecimalFromFloat(1.3),
					CollateralRelease: num.DecimalFromFloat(1.4),
				},
			},
			Instrument: &types.Instrument{
				ID:   "Crypto/ETHUSD/Futures/Dec19",
				Code: "FX:ETHUSD/DEC19",
				Name: "December 2019 ETH vs USD future",
				Metadata: &types.InstrumentMetadata{
					Tags: []string{
						"asset_class:fx/crypto",
						"product:futures",
					},
				},
				Product: &types.InstrumentFuture{
					Future: &types.Future{
						SettlementAsset: "Ethereum/Ether",
						DataSourceSpecForSettlementData: &datasource.Spec{
							ID: hex.EncodeToString(crypto.Hash([]byte(ID + "price"))),
							Data: datasource.NewDefinition(
								datasource.ContentTypeOracle,
							).SetOracleConfig(
								&signedoracle.SpecConfiguration{
									Signers: []*dstypes.Signer{dstypes.CreateSignerFromString(pubKey.PubKey.Key, dstypes.SignerTypePubKey)},
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
							ID: hex.EncodeToString(crypto.Hash([]byte(ID + "tt"))),
							Data: datasource.NewDefinition(
								datasource.ContentTypeOracle,
							).SetOracleConfig(
								&signedoracle.SpecConfiguration{
									Signers: []*dstypes.Signer{dstypes.CreateSignerFromString(pubKey.PubKey.Key, dstypes.SignerTypePubKey)},
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
			},
			RiskModel: &types.TradableInstrumentLogNormalRiskModel{
				LogNormalRiskModel: &types.LogNormalRiskModel{
					RiskAversionParameter: num.DecimalFromFloat(0.01),
					Tau:                   num.DecimalFromFloat(1.0 / 365.25 / 24),
					Params: &types.LogNormalModelParams{
						Mu:    num.DecimalZero(),
						R:     num.DecimalFromFloat(0.016),
						Sigma: num.DecimalFromFloat(0.09),
					},
				},
			},
		},
		LiquiditySLAParams: &types.LiquiditySLAParams{
			PriceRange:                  num.DecimalFromFloat(0.95),
			CommitmentMinTimeFraction:   num.NewDecimalFromFloat(0.5),
			PerformanceHysteresisEpochs: 4,
			SlaCompetitionFactor:        num.NewDecimalFromFloat(0.5),
		},
		State: types.MarketStateActive,
		MarkPriceConfiguration: &types.CompositePriceConfiguration{
			DecayWeight:              num.DecimalZero(),
			DecayPower:               num.DecimalZero(),
			CashAmount:               num.UintZero(),
			SourceWeights:            []num.Decimal{num.DecimalFromFloat(0.1), num.DecimalFromFloat(0.2), num.DecimalFromFloat(0.3), num.DecimalFromFloat(0.4)},
			SourceStalenessTolerance: []time.Duration{0, 0, 0, 0},
			CompositePriceType:       types.CompositePriceTypeByLastTrade,
		},
		TickSize:       num.UintOne(),
		OpeningAuction: auctionDuration,
	}
}

func getEngine(t *testing.T, vegaPath paths.Paths, now time.Time) *snapshotTestData {
	t.Helper()
	cfg := execution.NewDefaultConfig()
	log := logging.NewTestLogger()
	broker := stubs.NewBrokerStub()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	collateralEngine := collateral.New(log, collateral.NewDefaultConfig(), timeService, broker)
	oracleEngine := spec.NewEngine(log, spec.NewDefaultConfig(), timeService, broker)

	epochEngine := epochtime.NewService(log, epochtime.NewDefaultConfig(), broker)
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	bc := mocks.NewMockAccountBalanceChecker(ctrl)
	marketActivityTracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, bc, broker, collateralEngine)
	epochEngine.NotifyOnEpoch(marketActivityTracker.OnEpochEvent, marketActivityTracker.OnEpochRestore)

	ethAsset := types.Asset{
		ID: "Ethereum/Ether",
		Details: &types.AssetDetails{
			Name:    "Ethereum/Ether",
			Symbol:  "Ethereum/Ether",
			Quantum: num.DecimalFromInt64(1),
		},
	}
	require.NoError(t, collateralEngine.EnableAsset(context.Background(), ethAsset))
	referralDiscountReward := fmock.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscount := fmock.NewMockVolumeDiscountService(ctrl)
	volumeRebate := fmock.NewMockVolumeRebateService(ctrl)
	referralDiscountReward.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	referralDiscountReward.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscount.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	referralDiscountReward.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	banking := mocks.NewMockBanking(ctrl)
	parties := mocks.NewMockParties(ctrl)
	delayTarget := mocks.NewMockDelayTransactionsTarget(ctrl)
	delayTarget.EXPECT().MarketDelayRequiredUpdated(gomock.Any(), gomock.Any()).AnyTimes()
	eng := execution.NewEngine(
		log,
		cfg,
		timeService,
		collateralEngine,
		oracleEngine,
		broker,
		stubs.NewStateVar(),
		marketActivityTracker,
		stubs.NewAssetStub(),
		referralDiscountReward,
		volumeDiscount,
		volumeRebate,
		banking,
		parties,
		delayTarget,
	)

	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.DefaultConfig()
	snapshotEngine, err := snp.NewEngine(vegaPath, config, log, timeService, statsData.Blockchain)
	require.NoError(t, err)
	snapshotEngine.AddProviders(eng)
	snapshotEngine.AddProviders(collateralEngine)

	return &snapshotTestData{
		engine:           eng,
		oracleEngine:     oracleEngine,
		snapshotEngine:   snapshotEngine,
		timeService:      timeService,
		collateralEngine: collateralEngine,
	}
}

func getEngineWithParties(t *testing.T, now time.Time, balance *num.Uint, parties ...string) *snapshotTestData {
	t.Helper()
	// ctrl := gomock.NewController(t)
	cfg := execution.NewDefaultConfig()
	log := logging.NewTestLogger()
	broker := stubs.NewBrokerStub()
	timeService := stubs.NewTimeStub()
	timeService.SetTime(now)
	collateralEngine := collateral.New(log, collateral.NewDefaultConfig(), timeService, broker)
	oracleEngine := spec.NewEngine(log, spec.NewDefaultConfig(), timeService, broker)

	epochEngine := epochtime.NewService(log, epochtime.NewDefaultConfig(), broker)
	ctrl := gomock.NewController(t)
	teams := mocks.NewMockTeams(ctrl)
	bc := mocks.NewMockAccountBalanceChecker(ctrl)
	marketActivityTracker := common.NewMarketActivityTracker(logging.NewTestLogger(), teams, bc, broker, collateralEngine)
	epochEngine.NotifyOnEpoch(marketActivityTracker.OnEpochEvent, marketActivityTracker.OnEpochRestore)

	ethAsset := types.Asset{
		ID: "Ethereum/Ether",
		Details: &types.AssetDetails{
			Name:    "Ethereum/Ether",
			Symbol:  "Ethereum/Ether",
			Quantum: num.DecimalFromInt64(1),
		},
	}
	collateralEngine.EnableAsset(context.Background(), ethAsset)
	for _, p := range parties {
		_, _ = collateralEngine.Deposit(context.Background(), p, ethAsset.ID, balance.Clone())
	}
	referralDiscountReward := fmock.NewMockReferralDiscountRewardService(ctrl)
	volumeDiscount := fmock.NewMockVolumeDiscountService(ctrl)
	volumeRebate := fmock.NewMockVolumeRebateService(ctrl)

	referralDiscountReward.EXPECT().ReferralDiscountFactorsForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	referralDiscountReward.EXPECT().RewardsFactorsMultiplierAppliedForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	volumeDiscount.EXPECT().VolumeDiscountFactorForParty(gomock.Any()).Return(types.EmptyFactors).AnyTimes()
	referralDiscountReward.EXPECT().GetReferrer(gomock.Any()).Return(types.PartyID(""), errors.New("not a referrer")).AnyTimes()
	banking := mocks.NewMockBanking(ctrl)
	partiesMock := mocks.NewMockParties(ctrl)
	delayTarget := mocks.NewMockDelayTransactionsTarget(ctrl)
	delayTarget.EXPECT().MarketDelayRequiredUpdated(gomock.Any(), gomock.Any()).AnyTimes()
	eng := execution.NewEngine(
		log,
		cfg,
		timeService,
		collateralEngine,
		oracleEngine,
		broker,
		stubs.NewStateVar(),
		marketActivityTracker,
		stubs.NewAssetStub(),
		referralDiscountReward,
		volumeDiscount,
		volumeRebate,
		banking,
		partiesMock,
		delayTarget,
	)

	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.DefaultConfig()
	config.Storage = "memory"
	snapshotEngine, _ := snp.NewEngine(&paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngine.AddProviders(eng)

	return &snapshotTestData{
		engine:           eng,
		oracleEngine:     oracleEngine,
		snapshotEngine:   snapshotEngine,
		timeService:      timeService,
		collateralEngine: collateralEngine,
	}
}

func (s *stubIDGen) NextID() string {
	s.calls++
	return hex.EncodeToString([]byte(fmt.Sprintf("deadb33f%d", s.calls)))
}
