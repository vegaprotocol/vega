package snapshot_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/protos/vega"
	ov1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	v1 "code.vegaprotocol.io/protos/vega/snapshot/v1"
	vegactx "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/libs/crypto"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/snapshot/mocks"
	"code.vegaprotocol.io/vega/types"
	tmocks "code.vegaprotocol.io/vega/types/mocks"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/golang/mock/gomock"
	"github.com/golang/protobuf/proto"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
)

type tstEngine struct {
	*snapshot.Engine
	ctx   context.Context
	cfunc context.CancelFunc
	ctrl  *gomock.Controller
	time  *mocks.MockTimeService
}

func getTestEngine(t *testing.T) *tstEngine {
	t.Helper()
	ctx, cfunc := context.WithCancel(context.Background())
	ctrl := gomock.NewController(t)
	time := mocks.NewMockTimeService(ctrl)
	eng, err := snapshot.New(context.Background(), nil, snapshot.NewTestConfig(), logging.NewTestLogger(), time)
	require.NoError(t, err)
	ctx = vegactx.WithTraceID(vegactx.WithBlockHeight(ctx, 1), "0xDEADBEEF")
	return &tstEngine{
		ctx:    ctx,
		cfunc:  cfunc,
		Engine: eng,
		ctrl:   ctrl,
		time:   time,
	}
}

// basic engine functionality tests.
func TestEngine(t *testing.T) {
	t.Run("Adding a provider calls what we expect on the state provider", testAddProviders)
	t.Run("Adding provider with duplicate key in same namespace: first come, first serve", testAddProvidersDuplicateKeys)
	t.Run("Create a snapshot, if nothing changes, we don't get the data and the hash remains unchanged", testTakeSnapshot)
	t.Run("Fill DB with fake snapshots, check that listing snapshots works", testListSnapshot)
}

func TestRestore(t *testing.T) {
	t.Run("Restoring a snapshot from chain works as expected", testReloadSnapshot)
	t.Run("Restoring replay protectors replaces the provider as expected", testReloadReplayProtectors)
	t.Run("Restoring a snapshot calls the post-restore callback if available", testReloadRestore)
}

func testAddProviders(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return([]string{"all"})
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)
}

func testAddProvidersDuplicateKeys(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys1 := []string{
		"foo",
		"bar",
	}
	keys2 := []string{
		keys1[0],
		"bar2",
	}
	prov1 := engine.getNewProviderMock()
	prov2 := engine.getNewProviderMock()
	prov1.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	prov2.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	prov1.EXPECT().Keys().Times(1).Return(keys1)
	prov2.EXPECT().Keys().Times(1).Return(keys2)
	// first come-first serve
	engine.AddProviders(prov1, prov2)
	hash1 := [][]byte{
		[]byte("foo"),
		[]byte("bar"),
	}
	data1 := hash1
	hash2 := [][]byte{
		[]byte("bar2"),
	}
	data2 := hash2
	for i, k := range keys1 {
		prov1.EXPECT().GetHash(k).Times(1).Return(hash1[i], nil)
		prov1.EXPECT().GetState(k).Times(1).Return(data1[i], nil, nil)
	}
	// duplicate key is skipped
	prov2.EXPECT().GetHash(keys2[1]).Times(1).Return(hash2[0], nil)
	prov2.EXPECT().GetState(keys2[1]).Times(1).Return(data2[0], nil, nil)

	engine.time.EXPECT().GetTimeNow().Times(1).Return(time.Now())
	_, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
}

func testTakeSnapshot(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys := []string{
		"all",
	}
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return(keys)
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)

	// now take a snapshot
	now := time.Now()
	engine.time.EXPECT().GetTimeNow().Times(2).Return(now)
	state := map[string]*types.Payload{
		keys[0]: {
			Data: &types.PayloadCheckpoint{
				Checkpoint: &types.CPState{
					NextCp: now.Add(time.Hour).Unix(),
				},
			},
		},
	}
	// set up provider to return state
	for _, k := range keys {
		pl := state[k]
		data, err := proto.Marshal(pl.IntoProto())
		require.NoError(t, err)
		hash := crypto.Hash(data)
		prov.EXPECT().GetHash(k).Times(2).Return(hash, nil)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}

	// take the snapshot knowing state has changed:
	// we need the ctx that goes with the mock, because that has block height and hash set
	hash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	secondHash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.EqualValues(t, hash, secondHash)
}

func getDummyData() *types.Chunk {
	all := types.Chunk{
		Data: make([]*types.Payload, 0, 42),
	}
	all.Data = append(all.Data, &types.Payload{
		Data: &types.PayloadActiveAssets{
			ActiveAssets: &types.ActiveAssets{
				Assets: []*types.Asset{
					{
						ID: "asset",
						Details: &types.AssetDetails{
							Name:        "asset",
							Symbol:      "AST",
							TotalSupply: num.Zero(),
							Decimals:    0,
							MinLpStake:  num.Zero(),
							Source: &types.AssetDetailsBuiltinAsset{
								BuiltinAsset: &types.BuiltinAsset{
									MaxFaucetAmountMint: num.Zero(),
								},
							},
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadPendingAssets{
			PendingAssets: &types.PendingAssets{
				Assets: []*types.Asset{
					{
						ID: "asset2",
						Details: &types.AssetDetails{
							Name:        "asset2",
							Symbol:      "AS2",
							TotalSupply: num.Zero(),
							Decimals:    0,
							MinLpStake:  num.Zero(),
							Source: &types.AssetDetailsBuiltinAsset{
								BuiltinAsset: &types.BuiltinAsset{
									MaxFaucetAmountMint: num.Zero(),
								},
							},
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingWithdrawals{
			BankingWithdrawals: &types.BankingWithdrawals{
				Withdrawals: []*types.RWithdrawal{
					{
						Ref: "RWRef",
						Withdrawal: &types.Withdrawal{
							ID:      "RW1",
							PartyID: "p1",
							Amount:  num.NewUint(10),
							Asset:   "AST",
							Status:  0,
							Ref:     "rw1",
							TxHash:  "abcdef091235456",
							Ext: &vega.WithdrawExt{
								Ext: nil,
							},
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingDeposits{
			BankingDeposits: &types.BankingDeposits{
				Deposit: []*types.BDeposit{
					{
						ID: "BD1",
						Deposit: &types.Deposit{
							ID:      "BD1",
							Status:  0,
							PartyID: "p1",
							Asset:   "AST",
							Amount:  num.NewUint(10),
							TxHash:  "abcdef1234567890",
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingSeen{
			BankingSeen: &types.BankingSeen{
				Refs: []*types.TxRef{}, // nothing needed
			},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingAssetActions{
			BankingAssetActions: &types.BankingAssetActions{
				AssetAction: []*types.AssetAction{
					{
						ID:          "AA1",
						Asset:       "AST",
						BlockNumber: 1,
						TxIndex:     1,
						Hash:        "abcdef123",
						BuiltinD: &types.BuiltinAssetDeposit{
							VegaAssetID: "AST",
							PartyID:     "P1",
							Amount:      num.NewUint(1),
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadCheckpoint{
			Checkpoint: &types.CPState{
				NextCp: 100000000,
			},
		},
	}, &types.Payload{
		Data: &types.PayloadCollateralAccounts{
			CollateralAccounts: &types.CollateralAccounts{
				Accounts: []*types.Account{
					{
						ID:       "",
						Owner:    "party1",
						Balance:  num.Zero(),
						Asset:    "AST",
						MarketID: "",
						Type:     types.AccountTypeGeneral,
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadCollateralAssets{
			CollateralAssets: &types.CollateralAssets{
				Assets: []*types.Asset{
					{
						ID: "asset",
						Details: &types.AssetDetails{
							Name:        "asset",
							Symbol:      "AST",
							TotalSupply: num.Zero(),
							Decimals:    0,
							MinLpStake:  num.Zero(),
							Source: &types.AssetDetailsBuiltinAsset{
								BuiltinAsset: &types.BuiltinAsset{
									MaxFaucetAmountMint: num.Zero(),
								},
							},
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadAppState{
			AppState: &types.AppState{
				Height: 2,
				Block:  "abcdef123456889",
				Time:   1000010,
			},
		},
	}, &types.Payload{
		Data: &types.PayloadNetParams{
			NetParams: &types.NetParams{
				Params: []*types.NetworkParameter{
					{
						Key:   "foo",
						Value: "bar",
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationActive{
			DelegationActive: &types.DelegationActive{
				Delegations: []*types.Delegation{
					{
						Party:    "party1",
						NodeID:   "node1",
						Amount:   num.NewUint(1),
						EpochSeq: "1",
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationPending{
			DelegationPending: &types.DelegationPending{
				Delegations: []*types.Delegation{
					{
						Party:    "party1",
						NodeID:   "node2",
						Amount:   num.NewUint(1),
						EpochSeq: "2",
					},
				},
				Undelegation: []*types.Delegation{
					{
						Party:    "party1",
						NodeID:   "node1",
						Amount:   num.NewUint(1),
						EpochSeq: "2",
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationAuto{
			DelegationAuto: &types.DelegationAuto{
				Parties: []string{
					"party2",
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationLastReconTime{
			LastReconcilicationTime: time.Time{},
		},
	}, &types.Payload{
		Data: &types.PayloadGovernanceActive{
			GovernanceActive: &types.GovernanceActive{
				Proposals: []*types.PendingProposal{
					{
						Proposal: &types.Proposal{
							ID:        "prop1",
							Reference: "prop1",
							Party:     "party_animal",
							State:     types.ProposalStateOpen,
							Terms: &types.ProposalTerms{
								ClosingTimestamp:   10000000000,
								EnactmentTimestamp: 10000000002,
								Change: &types.ProposalTerms_NewAsset{
									NewAsset: &types.NewAsset{
										Changes: &types.AssetDetails{
											Name:        "foocoin2",
											Symbol:      "FO2",
											TotalSupply: num.NewUint(1000000),
											Decimals:    5,
											MinLpStake:  num.NewUint(10),
											Source: &types.AssetDetailsBuiltinAsset{
												BuiltinAsset: &types.BuiltinAsset{
													MaxFaucetAmountMint: num.NewUint(1),
												},
											},
										},
									},
								},
							},
						},
						Yes: []*types.Vote{
							{
								PartyID:                     "party1",
								ProposalID:                  "prop1",
								Value:                       types.VoteValueYes,
								TotalGovernanceTokenBalance: num.NewUint(10),
								TotalGovernanceTokenWeight:  num.NewDecimalFromFloat(.1),
							},
						},
						No: []*types.Vote{
							{
								PartyID:                     "party2",
								ProposalID:                  "prop1",
								Value:                       types.VoteValueNo,
								TotalGovernanceTokenBalance: num.NewUint(10),
								TotalGovernanceTokenWeight:  num.NewDecimalFromFloat(.1),
							},
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadGovernanceEnacted{
			GovernanceEnacted: &types.GovernanceEnacted{
				Proposals: []*types.Proposal{
					{
						ID:        "propA",
						Reference: "foo",
						Party:     "party_animal",
						State:     types.ProposalStateEnacted,
						Terms: &types.ProposalTerms{
							Change: &types.ProposalTerms_NewAsset{
								NewAsset: &types.NewAsset{
									Changes: &types.AssetDetails{
										Name:        "foocoin",
										Symbol:      "FOO",
										TotalSupply: num.NewUint(1000000),
										Decimals:    5,
										MinLpStake:  num.NewUint(10),
										Source: &types.AssetDetailsBuiltinAsset{
											BuiltinAsset: &types.BuiltinAsset{
												MaxFaucetAmountMint: num.NewUint(1),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadGovernanceNode{
			GovernanceNode: &types.GovernanceNode{
				Proposals: []*types.Proposal{},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadMarketPositions{
			MarketPositions: &types.MarketPositions{
				MarketID: "key",
				Positions: []*types.MarketPosition{
					{
						PartyID: "party1",
						Size:    10,
						Buy:     0,
						Sell:    0,
						Price:   num.NewUint(10),
						VwBuy:   num.Zero(),
						VwSell:  num.Zero(),
					},
					{
						PartyID: "party2",
						Size:    -10,
						Buy:     0,
						Sell:    0,
						Price:   num.NewUint(10),
						VwBuy:   num.Zero(),
						VwSell:  num.Zero(),
					},
				},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadMatchingBook{
			MatchingBook: &types.MatchingBook{
				MarketID:        "key",
				LastTradedPrice: num.NewUint(10),
			},
		},
	}, &types.Payload{
		Data: &types.PayloadExecutionMarkets{
			ExecutionMarkets: &types.ExecutionMarkets{
				Markets: []*types.ExecMarket{
					{
						Market: &types.Market{
							ID: "key",
							TradableInstrument: &types.TradableInstrument{
								Instrument: &types.Instrument{
									ID:   "",
									Code: "",
									Name: "",
									Metadata: &types.InstrumentMetadata{
										Tags: []string{},
									},
									Product: &types.Instrument_Future{
										Future: &types.Future{
											Maturity:        "1",
											SettlementAsset: "AST",
											QuoteName:       "AST",
											OracleSpecForSettlementPrice: &ov1.OracleSpec{
												Id:      "o1",
												PubKeys: []string{},
												Filters: []*ov1.Filter{},
											},
											OracleSpecForTradingTermination: &ov1.OracleSpec{
												Id:      "os1",
												Filters: []*ov1.Filter{},
											},
											OracleSpecBinding: &types.OracleSpecToFutureBinding{},
										},
									},
								},
								MarginCalculator: &types.MarginCalculator{
									ScalingFactors: &types.ScalingFactors{
										SearchLevel:       decimal.Decimal{},
										InitialMargin:     decimal.Decimal{},
										CollateralRelease: decimal.Decimal{},
									},
								},
								RiskModel: &types.TradableInstrumentSimpleRiskModel{
									SimpleRiskModel: &types.SimpleRiskModel{
										Params: &types.SimpleModelParams{
											FactorLong:           num.DecimalZero(),
											FactorShort:          num.DecimalZero(),
											MaxMoveUp:            num.DecimalZero(),
											MinMoveDown:          num.DecimalZero(),
											ProbabilityOfTrading: num.DecimalZero(),
										},
									},
								},
							},
						},
						PriceMonitor: &types.PriceMonitor{},
						AuctionState: &types.AuctionState{
							Mode:        types.MarketTradingModeContinuous,
							DefaultMode: types.MarketTradingModeContinuous,
							Begin:       time.Time{},
							End:         nil,
						},
						LastBestBid:          num.NewUint(10),
						LastBestAsk:          num.NewUint(10),
						LastMidBid:           num.NewUint(10),
						LastMidAsk:           num.NewUint(10),
						LastMarketValueProxy: num.NewDecimalFromFloat(10),
						EquityShare: &types.EquityShare{
							Mvp:                 num.NewDecimalFromFloat(10),
							OpeningAuctionEnded: true,
						},
						CurrentMarkPrice: num.NewUint(10),
					},
				},
				Batches:   0,
				Orders:    2,
				Proposals: 2,
			},
		},
	}, &types.Payload{
		Data: &types.PayloadStakingAccounts{
			StakingAccounts: &types.StakingAccounts{
				Accounts: []*types.StakingAccount{},
			},
		},
	}, &types.Payload{
		Data: &types.PayloadStakeVerifierDeposited{},
	}, &types.Payload{
		Data: &types.PayloadStakeVerifierRemoved{},
	}, &types.Payload{
		Data: &types.PayloadEpoch{
			EpochState: &types.EpochState{},
		},
	}, &types.Payload{
		Data: &types.PayloadLimitState{
			LimitState: &types.LimitState{},
		},
	}, &types.Payload{
		Data: &types.PayloadNotary{
			Notary: &types.Notary{},
		},
	}, &types.Payload{
		Data: &types.PayloadWitness{
			Witness: &types.Witness{},
		},
	}, &types.Payload{
		Data: &types.PayloadTopology{
			Topology: &types.Topology{},
		},
	}, &types.Payload{
		Data: &types.PayloadReplayProtection{},
	}, &types.Payload{
		Data: &types.PayloadEventForwarder{},
	}, &types.Payload{
		Data: &types.PayloadLiquidityParameters{
			Parameters: &v1.LiquidityParameters{
				MarketId: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadLiquidityPendingProvisions{
			PendingProvisions: &v1.LiquidityPendingProvisions{
				MarketId: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadLiquidityPartiesLiquidityOrders{
			PartiesLiquidityOrders: &v1.LiquidityPartiesLiquidityOrders{
				MarketId: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadLiquidityPartiesOrders{
			PartiesOrders: &v1.LiquidityPartiesOrders{
				MarketId: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadLiquidityProvisions{
			Provisions: &v1.LiquidityProvisions{
				MarketId: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadLiquidityTarget{
			Target: &v1.LiquidityTarget{
				MarketId: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadLiquiditySupplied{
			LiquiditySupplied: &v1.LiquiditySupplied{
				MarketId: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadVoteSpamPolicy{
			VoteSpamPolicy: &types.VoteSpamPolicy{
				MinVotingTokensFactor: num.Zero(),
			},
		},
	}, &types.Payload{
		Data: &types.PayloadSimpleSpamPolicy{
			SimpleSpamPolicy: &types.SimpleSpamPolicy{},
		},
	}, &types.Payload{
		Data: &types.PayloadRewardsPayout{
			RewardsPendingPayouts: &types.RewardsPendingPayouts{},
		},
	},
	)
	return &all
}

func testListSnapshot(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()

	t.Parallel()

	data := getDummyData()
	// load the tree up with the data
	for _, n := range data.Data {
		_, err := engine.Engine.SetTreeNode(n)
		require.NoError(t, err)
	}

	hash, err := engine.Engine.SaveCurrentTree()
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	listed, err := engine.Engine.List()
	require.Equal(t, 1, len(listed))
	require.NoError(t, err)

	require.Equal(t, []byte{}, listed[0].Hash)

}

func testReloadSnapshot(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys := []string{
		"all",
	}
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return(keys)
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)

	now := time.Now()
	engine.time.EXPECT().GetTimeNow().Times(1).Return(now)
	state := map[string]*types.Payload{
		keys[0]: {
			Data: &types.PayloadCheckpoint{
				Checkpoint: &types.CPState{
					NextCp: now.Add(time.Hour).Unix(),
				},
			},
		},
	}
	for _, k := range keys {
		pl := state[k]
		data, err := proto.Marshal(pl.IntoProto())
		require.NoError(t, err)
		hash := crypto.Hash(data)
		prov.EXPECT().GetHash(k).Times(1).Return(hash, nil)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}
	hash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// get the snapshot list
	snaps, err := engine.List()
	require.NoError(t, err)
	require.NotEmpty(t, snaps)
	require.Equal(t, 1, len(snaps))

	// create a new engine which will restore the snapshot
	eng2 := getTestEngine(t)
	defer eng2.Finish()
	p2 := eng2.getNewProviderMock()
	p2.EXPECT().Keys().Times(1).Return(keys)
	p2.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	eng2.AddProviders(p2)

	// calls we expect to see when reloading
	eng2.time.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(1).Do(func(_ context.Context, newT time.Time) {
		require.Equal(t, newT.Unix(), now.Unix())
	})
	// ensure we're passing the right state
	p2.EXPECT().LoadState(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil).Do(func(_ context.Context, pl *types.Payload) {
		require.EqualValues(t, pl.Data, state[keys[0]].Data)
	})

	// start receiving the snapshot
	snap := snaps[0]
	require.NoError(t, eng2.ReceiveSnapshot(snap))
	ready := false
	for i := uint32(0); i < snap.Chunks; i++ {
		chunk, err := engine.LoadSnapshotChunk(snap.Height, uint32(snap.Format), i)
		require.NoError(t, err)
		ready, err = eng2.ApplySnapshotChunk(chunk)
		require.NoError(t, err)
	}
	require.True(t, ready)

	// OK, our snapshot is ready to load
	require.NoError(t, eng2.ApplySnapshot(eng2.ctx))
}

func testReloadReplayProtectors(t *testing.T) {
	e := getTestEngine(t)
	defer e.Finish()
	rpPl := types.PayloadReplayProtection{
		Blocks: []*types.ReplayBlockTransactions{
			{
				Transactions: []string{"foo", "bar"},
			},
		},
	}
	payload := types.Payload{
		Data: &rpPl,
	}
	data, err := proto.Marshal(payload.IntoProto())
	require.NoError(t, err)
	hash := crypto.Hash(data)
	rpl := e.getNewProviderMock()
	rpl.EXPECT().Namespace().Times(1).Return(payload.Namespace())
	rpl.EXPECT().Keys().Times(1).Return([]string{payload.Key()})

	old := e.getNewProviderMock()
	old.EXPECT().Namespace().Times(2).Return(payload.Namespace())
	old.EXPECT().Keys().Times(3).Return([]string{payload.Key()}) // this gets called a second time when replacing
	old.EXPECT().GetHash(payload.Key()).Times(1).Return(hash, nil)
	old.EXPECT().GetState(payload.Key()).Times(1).Return(data, nil, nil)
	old.EXPECT().LoadState(gomock.Any(), gomock.Any()).Times(1).DoAndReturn(func(_ context.Context, pl *types.Payload) ([]types.StateProvider, error) {
		switch dt := pl.Data.(type) {
		case *types.PayloadReplayProtection:
			require.Equal(t, len(dt.Blocks), len(rpPl.Blocks))
			require.EqualValues(t, dt.Blocks[0], rpPl.Blocks[0])
		default:
			t.Fatal("Incorrect payload type passed")
		}
		return []types.StateProvider{rpl}, nil
	})
	// call is made when creating the snapshot
	now := time.Now()
	e.time.EXPECT().GetTimeNow().Times(1).Return(now)

	// add old provider
	e.AddProviders(old)
	// take a snapshot
	snapHash, err := e.Snapshot(e.ctx)
	require.NoError(t, err)
	require.NotNil(t, snapHash)
	// now get the snapshot we've just created
	snaps, err := e.List()
	require.NoError(t, err)
	require.NotEmpty(t, snaps)
	require.Equal(t, 1, len(snaps))

	snap := snaps[0]

	// now reload the snapshot on a new engine
	e2 := getTestEngine(t)
	defer e2.Finish()
	e2.time.EXPECT().GetTimeNow().Times(1).Return(now)
	// calls we expect to see when reloading
	e2.time.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(1).Do(func(_ context.Context, newT time.Time) {
		require.Equal(t, newT.Unix(), now.Unix())
	})
	e2.AddProviders(old)
	require.NoError(t, e2.ReceiveSnapshot(snap))
	ready := false
	for i := uint32(0); i < snap.Chunks; i++ {
		chunk, err := e.LoadSnapshotChunk(snap.Height, uint32(snap.Format), i)
		require.NoError(t, err)
		ready, err = e2.ApplySnapshotChunk(chunk)
		require.NoError(t, err)
	}
	require.True(t, ready)
	// OK, snapshot is ready to be applied
	require.NoError(t, e2.ApplySnapshot(e.ctx))
	// so now we can check if taking a snapshot calls the methods on the replacement provider

	rpl.EXPECT().GetHash(payload.Key()).Times(1).Return(hash, nil)
	snapHash2, err := e2.Snapshot(e.ctx)
	require.NoError(t, err)
	require.NotNil(t, snapHash2)
	require.EqualValues(t, snapHash, snapHash2)
}

func testReloadRestore(t *testing.T) {
	engine := getTestEngine(t)
	defer engine.Finish()
	keys := []string{
		"all",
	}
	prov := engine.getNewProviderMock()
	prov.EXPECT().Keys().Times(1).Return(keys)
	prov.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	engine.AddProviders(prov)

	now := time.Now()
	engine.time.EXPECT().GetTimeNow().Times(1).Return(now)
	state := map[string]*types.Payload{
		keys[0]: {
			Data: &types.PayloadCheckpoint{
				Checkpoint: &types.CPState{
					NextCp: now.Add(time.Hour).Unix(),
				},
			},
		},
	}
	for _, k := range keys {
		pl := state[k]
		data, err := proto.Marshal(pl.IntoProto())
		require.NoError(t, err)
		hash := crypto.Hash(data)
		prov.EXPECT().GetHash(k).Times(1).Return(hash, nil)
		prov.EXPECT().GetState(k).Times(1).Return(data, nil, nil)
	}
	hash, err := engine.Snapshot(engine.ctx)
	require.NoError(t, err)
	require.NotEmpty(t, hash)

	// get the snapshot list
	snaps, err := engine.List()
	require.NoError(t, err)
	require.NotEmpty(t, snaps)
	require.Equal(t, 1, len(snaps))

	// create a new engine which will restore the snapshot
	eng2 := getTestEngine(t)
	defer eng2.Finish()
	p2 := eng2.getRestoreMock()
	p2.EXPECT().Keys().Times(1).Return(keys)
	p2.EXPECT().Namespace().Times(1).Return(types.CheckpointSnapshot)
	eng2.AddProviders(p2)

	// calls we expect to see when reloading
	eng2.time.EXPECT().SetTimeNow(gomock.Any(), gomock.Any()).Times(1).Do(func(_ context.Context, newT time.Time) {
		require.Equal(t, newT.Unix(), now.Unix())
	})
	// ensure we're passing the right state
	p2.EXPECT().LoadState(gomock.Any(), gomock.Any()).Times(1).Return(nil, nil).Do(func(_ context.Context, pl *types.Payload) {
		require.EqualValues(t, pl.Data, state[keys[0]].Data)
	})

	// start receiving the snapshot
	snap := snaps[0]
	require.NoError(t, eng2.ReceiveSnapshot(snap))
	ready := false
	for i := uint32(0); i < snap.Chunks; i++ {
		chunk, err := engine.LoadSnapshotChunk(snap.Height, uint32(snap.Format), i)
		require.NoError(t, err)
		ready, err = eng2.ApplySnapshotChunk(chunk)
		require.NoError(t, err)
	}
	require.True(t, ready)
	p2.EXPECT().OnStateLoaded(gomock.Any()).Times(1).Return(nil)

	// OK, our snapshot is ready to load
	require.NoError(t, eng2.ApplySnapshot(eng2.ctx))
}

func (t *tstEngine) getNewProviderMock() *tmocks.MockStateProvider {
	return tmocks.NewMockStateProvider(t.ctrl)
}

func (t *tstEngine) getRestoreMock() *tmocks.MockPostRestore {
	return tmocks.NewMockPostRestore(t.ctrl)
}

func (t *tstEngine) Finish() {
	t.cfunc()
	t.ctrl.Finish()
}
