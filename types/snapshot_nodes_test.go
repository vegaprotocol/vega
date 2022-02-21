package types_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/protos/vega"
	ov1 "code.vegaprotocol.io/protos/vega/oracles/v1"
	v1 "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/snapshot"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"

	"github.com/cosmos/iavl"
	"github.com/golang/protobuf/proto"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
	db "github.com/tendermint/tm-db"
)

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
							Quantum:     num.Zero(),
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
							Quantum:     num.Zero(),
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
							Amount:  num.Zero(),
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
							Quantum:     num.Zero(),
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
				Proposals: []*types.ProposalData{
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
											Quantum:     num.NewUint(10),
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
				Proposals: []*types.ProposalData{
					{
						Proposal: &types.Proposal{
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
											Quantum:     num.NewUint(10),
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

func TestPayloadConversion(t *testing.T) {
	t.Parallel()
	all := types.Chunk{
		Data: make([]*types.Payload, 0, 42),
	}
	all.Data = append(all.Data, &types.Payload{
		Data: &types.PayloadActiveAssets{
			ActiveAssets: &types.ActiveAssets{},
		},
	}, &types.Payload{
		Data: &types.PayloadPendingAssets{
			PendingAssets: &types.PendingAssets{},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingWithdrawals{
			BankingWithdrawals: &types.BankingWithdrawals{},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingDeposits{
			BankingDeposits: &types.BankingDeposits{},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingSeen{
			BankingSeen: &types.BankingSeen{},
		},
	}, &types.Payload{
		Data: &types.PayloadBankingAssetActions{
			BankingAssetActions: &types.BankingAssetActions{},
		},
	}, &types.Payload{
		Data: &types.PayloadCheckpoint{
			Checkpoint: &types.CPState{},
		},
	}, &types.Payload{
		Data: &types.PayloadCollateralAccounts{
			CollateralAccounts: &types.CollateralAccounts{},
		},
	}, &types.Payload{
		Data: &types.PayloadCollateralAssets{
			CollateralAssets: &types.CollateralAssets{},
		},
	}, &types.Payload{
		Data: &types.PayloadAppState{
			AppState: &types.AppState{},
		},
	}, &types.Payload{
		Data: &types.PayloadNetParams{
			NetParams: &types.NetParams{},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationActive{
			DelegationActive: &types.DelegationActive{},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationPending{
			DelegationPending: &types.DelegationPending{},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationAuto{
			DelegationAuto: &types.DelegationAuto{},
		},
	}, &types.Payload{
		Data: &types.PayloadDelegationLastReconTime{
			LastReconcilicationTime: time.Time{},
		},
	}, &types.Payload{
		Data: &types.PayloadGovernanceActive{
			GovernanceActive: &types.GovernanceActive{},
		},
	}, &types.Payload{
		Data: &types.PayloadGovernanceEnacted{
			GovernanceEnacted: &types.GovernanceEnacted{},
		},
	}, &types.Payload{
		Data: &types.PayloadGovernanceNode{
			GovernanceNode: &types.GovernanceNode{},
		},
	}, &types.Payload{
		Data: &types.PayloadMarketPositions{
			MarketPositions: &types.MarketPositions{
				MarketID: "key",
			},
		},
	}, &types.Payload{
		Data: &types.PayloadMatchingBook{
			MatchingBook: &types.MatchingBook{
				MarketID:        "key",
				LastTradedPrice: num.Zero(),
			},
		},
	}, &types.Payload{
		Data: &types.PayloadExecutionMarkets{
			ExecutionMarkets: &types.ExecutionMarkets{},
		},
	}, &types.Payload{
		Data: &types.PayloadStakingAccounts{
			StakingAccounts: &types.StakingAccounts{
				StakingAssetTotalSupply: num.NewUint(0),
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
	asProto := all.IntoProto()
	conv := types.ChunkFromProto(asProto)
	for i, c := range conv.Data {
		expect := all.Data[i]
		require.Equal(t, expect.Key(), c.Key())
		require.Equal(t, expect.Namespace(), c.Namespace())
	}
}

func TestSnapFromTree(t *testing.T) {
	t.Parallel()
	tree := createTree(t)
	data := getDummyData()
	// load the tree up with the data
	for _, n := range data.Data {
		k := n.GetTreeKey()
		serialised, err := proto.Marshal(n.IntoProto())
		require.NoError(t, err)
		_ = tree.Set([]byte(k), serialised)
	}
	// now get immutable tree
	hash, v, err := tree.SaveVersion()
	require.NoError(t, err)
	require.NotEmpty(t, hash) // @TODO see if storing it again produces the same hash
	immutable, err := tree.GetImmutable(v)
	require.NoError(t, err)
	snap, err := types.SnapshotFromTree(immutable)
	require.NoError(t, err)
	require.NotNil(t, snap)
}

func TestListSnapFromTree(t *testing.T) {
	t.Parallel()
	tree := createTree(t)
	data := getDummyData()
	for _, n := range data.Data {
		k := n.GetTreeKey()
		serialised, err := proto.Marshal(n.IntoProto())
		require.NoError(t, err)
		_ = tree.Set([]byte(k), serialised)
	}
	// now get immutable tree
	hash, v, err := tree.SaveVersion()
	require.NoError(t, err)
	require.NotEmpty(t, hash) // @TODO see if storing it again produces the same hash

	snapshotsHeights, invalidVersions, err := snapshot.SnapshotsHeightsFromTree(tree)

	require.NoError(t, err)
	require.Empty(t, invalidVersions)

	var expectedHeight uint64 = 2

	require.Equal(t, 1, len(snapshotsHeights))
	require.Equal(t, v, snapshotsHeights[0].Version)
	require.Equal(t, hash, snapshotsHeights[0].Hash)
	require.Equal(t, expectedHeight, snapshotsHeights[0].Height)
}

func createTree(t *testing.T) *iavl.MutableTree {
	t.Helper()
	db := db.NewMemDB()
	tree, err := iavl.NewMutableTreeWithOpts(db, 0, nil)
	require.NoError(t, err)
	return tree
}
