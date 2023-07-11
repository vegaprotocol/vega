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

package execution_test

import (
	"bytes"
	"context"
	"encoding/hex"
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
	"code.vegaprotocol.io/vega/core/integration/stubs"
	snp "code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/stats"
	"code.vegaprotocol.io/vega/core/types"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/paths"
	datapb "code.vegaprotocol.io/vega/protos/vega/data/v1"
	snapshot "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"
	"github.com/stretchr/testify/require"
)

type snapshotTestData struct {
	engine         *execution.Engine
	oracleEngine   *spec.Engine
	snapshotEngine *snp.Engine
	timeService    *stubs.TimeStub
}

type stubIDGen struct {
	calls int
}

// TestSnapshotOraclesTerminatingMarketFromSnapshot tests that market loaded from snapshot can be terminated with its oracle.
func TestSnapshotOraclesTerminatingMarketFromSnapshot(t *testing.T) {
	now := time.Now()
	exec := getEngine(t, now)
	pubKey := &dstypes.SignerPubKey{
		PubKey: &dstypes.PubKey{
			Key: "0xDEADBEEF",
		},
	}
	mkt := newMarket("MarketID", pubKey)
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	state, _, _ := exec.engine.GetState("")

	exec2 := getEngine(t, now)
	snap := &snapshot.Payload{}
	proto.Unmarshal(state, snap)
	_, _ = exec2.engine.LoadState(context.Background(), types.PayloadFromProto(snap))

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

	require.True(t, bytes.Equal(state, state2))
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
	mkt := newMarket("MarketID", pubKey)
	err := exec.engine.SubmitMarket(context.Background(), mkt, "", time.Now())
	require.NoError(t, err)

	err = exec.engine.StartOpeningAuction(context.Background(), mkt.ID)
	require.NoError(t, err)
	mktState, err := exec.engine.GetMarketState("MarketID")
	require.NoError(t, err)
	require.Equal(t, types.MarketStateActive, mktState)

	idgen := &stubIDGen{}
	// now let's submit some orders and get market to trade continuously
	lpSubmission := &types.LiquidityProvisionSubmission{
		MarketID:         mkt.ID,
		CommitmentAmount: num.NewUint(1000000),
		Fee:              num.DecimalFromFloat(0.01),
		Reference:        "lp1",
		Buys: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 10, 5),
		},
		Sells: []*types.LiquidityOrder{
			newLiquidityOrder(types.PeggedReferenceMid, 10, 5),
		},
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

	exec2 := getEngine(t, now)
	snap := &snapshot.Payload{}
	proto.Unmarshal(state, snap)
	vgctx = vgcontext.WithTraceID(context.Background(), hex.EncodeToString([]byte("3deadbeef")))
	_, _ = exec2.engine.LoadState(vgctx, types.PayloadFromProto(snap))

	state2, _, _ := exec2.engine.GetState("")
	// the states should match
	require.True(t, bytes.Equal(state, state2))

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
	exec := getEngine(t, now)
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
	exec2 := getEngine(t, now)
	snap := &snapshot.Payload{}
	proto.Unmarshal(state, snap)
	_, _ = exec2.engine.LoadState(context.Background(), types.PayloadFromProto(snap))

	// take a snapshot on the loaded engine
	state2, _, _ := exec2.engine.GetState("")
	require.True(t, bytes.Equal(state, state2))

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
	now := time.Now()
	exec := getEngine(t, now)
	defer exec.snapshotEngine.Close()
	ctx := vgcontext.WithTraceID(vgcontext.WithBlockHeight(context.Background(), 100), "0xDEADBEEF")
	ctx = vgcontext.WithChainID(ctx, "chainid")

	pubKeys := []*dstypes.Signer{
		dstypes.CreateSignerFromString("0xDEADBEEF", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("0xDEADBEFF", dstypes.SignerTypePubKey),
		dstypes.CreateSignerFromString("0xDEADBFFF", dstypes.SignerTypePubKey),
	}
	marketIDs := []string{"market1", "market2", "market3"}

	// submit and terminate all markets
	for i := 0; i < 3; i++ {
		mkt := newMarket(marketIDs[i], pubKeys[i].Signer.(*dstypes.SignerPubKey))
		err := exec.engine.SubmitMarket(ctx, mkt, "", time.Now())
		require.NoError(t, err)

		// verify markets are terminated
		marketState, err := exec.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, types.MarketStateProposed, marketState)

		err = exec.engine.StartOpeningAuction(context.Background(), mkt.ID)
		require.NoError(t, err)
		marketState, err = exec.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, marketState, types.MarketStateActive)

		// terminate all markets
		exec.oracleEngine.BroadcastData(ctx, dstypes.Data{
			Signers: []*dstypes.Signer{pubKeys[i]},
			Data:    map[string]string{"trading.terminated": "true"},
		})

		marketState, err = exec.engine.GetMarketState(marketIDs[i])
		require.NoError(t, err)
		require.Equal(t, types.MarketStateTradingTerminated, marketState)
	}

	// we now have 3 terminated markets in the execution engine
	// let's take a snapshot
	_, err := exec.snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	snaps, err := exec.snapshotEngine.List()
	require.NoError(t, err)
	snap1 := snaps[0]

	// now let's start from this snapshot
	exec2 := getEngine(t, now)
	defer exec2.snapshotEngine.Close()
	exec2.snapshotEngine.ReceiveSnapshot(snap1)
	exec2.snapshotEngine.ApplySnapshot(ctx)
	exec2.snapshotEngine.CheckLoaded()

	// progress time to trigger any side effect on time ticks
	exec.timeService.SetTime(now.Add(2 * time.Second))
	exec2.timeService.SetTime(now.Add(2 * time.Second))

	// finally take a snapshot of both and compare them
	snp, err := exec.snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	snp2, err := exec2.snapshotEngine.Snapshot(ctx)
	require.NoError(t, err)
	require.True(t, bytes.Equal(snp, snp2))

	// settle the markets
	for i := 0; i < 3; i++ {
		exec.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
			Signers: []*dstypes.Signer{pubKeys[i]},
			Data:    map[string]string{"prices.ETH.value": "100"},
		})
		exec2.oracleEngine.BroadcastData(context.Background(), dstypes.Data{
			Signers: []*dstypes.Signer{pubKeys[i]},
			Data:    map[string]string{"prices.ETH.value": "100"},
		})

		marketState1, _ := exec.engine.GetMarketState(marketIDs[i])
		marketState2, _ := exec2.engine.GetMarketState(marketIDs[i])
		require.Equal(t, marketState1.String(), marketState2.String())
		require.Equal(t, types.MarketStateSettled, marketState1)
		require.Equal(t, types.MarketStateSettled, marketState2)

		// finally take a snapshot of both and compare them
		snp, err := exec.snapshotEngine.Snapshot(ctx)
		require.NoError(t, err)
		snp2, err := exec2.snapshotEngine.Snapshot(ctx)
		require.NoError(t, err)
		require.True(t, bytes.Equal(snp, snp2))
	}
}

func newMarket(ID string, pubKey *dstypes.SignerPubKey) *types.Market {
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
			TriggeringRatio:  num.DecimalFromFloat(0.9),
			AuctionExtension: 10000,
		},
		Fees: &types.Fees{
			Factors: &types.FeeFactors{
				MakerFee:          num.DecimalFromFloat(0.1),
				InfrastructureFee: num.DecimalFromFloat(0.1),
				LiquidityFee:      num.DecimalFromFloat(0.1),
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
			PriceRange:                      num.DecimalFromFloat(0.95),
			CommitmentMinTimeFraction:       num.NewDecimalFromFloat(0.5),
			ProvidersFeeCalculationTimeStep: time.Second * 5,
			PerformanceHysteresisEpochs:     4,
			SlaCompetitionFactor:            num.NewDecimalFromFloat(0.5),
		},
		State: types.MarketStateActive,
	}
}

func getEngine(t *testing.T, now time.Time) *snapshotTestData {
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
	marketActivityTracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochEngine)

	ethAsset := types.Asset{
		ID: "Ethereum/Ether",
		Details: &types.AssetDetails{
			Name:   "Ethereum/Ether",
			Symbol: "Ethereum/Ether",
		},
	}
	collateralEngine.EnableAsset(context.Background(), ethAsset)

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
	)

	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.NewDefaultConfig()
	config.Storage = "memory"
	snapshotEngine, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngine.AddProviders(eng)
	snapshotEngine.ClearAndInitialise()

	return &snapshotTestData{
		engine:         eng,
		oracleEngine:   oracleEngine,
		snapshotEngine: snapshotEngine,
		timeService:    timeService,
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
	marketActivityTracker := common.NewMarketActivityTracker(logging.NewTestLogger(), epochEngine)

	ethAsset := types.Asset{
		ID: "Ethereum/Ether",
		Details: &types.AssetDetails{
			Name:   "Ethereum/Ether",
			Symbol: "Ethereum/Ether",
		},
	}
	collateralEngine.EnableAsset(context.Background(), ethAsset)
	for _, p := range parties {
		_, _ = collateralEngine.Deposit(context.Background(), p, ethAsset.ID, balance.Clone())
	}

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
	)

	statsData := stats.New(log, stats.NewDefaultConfig())
	config := snp.NewDefaultConfig()
	config.Storage = "memory"
	snapshotEngine, _ := snp.New(context.Background(), &paths.DefaultPaths{}, config, log, timeService, statsData.Blockchain)
	snapshotEngine.AddProviders(eng)
	snapshotEngine.ClearAndInitialise()

	return &snapshotTestData{
		engine:         eng,
		oracleEngine:   oracleEngine,
		snapshotEngine: snapshotEngine,
		timeService:    timeService,
	}
}

func (s *stubIDGen) NextID() string {
	s.calls++
	return hex.EncodeToString([]byte(fmt.Sprintf("deadb33f%d", s.calls)))
}

func newLiquidityOrder(reference types.PeggedReference, offset uint64, proportion uint32) *types.LiquidityOrder {
	return &types.LiquidityOrder{
		Reference:  reference,
		Proportion: proportion,
		Offset:     num.NewUint(offset),
	}
}
