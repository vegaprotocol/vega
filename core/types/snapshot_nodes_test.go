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

package types_test

import (
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/snapshot"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/libs/num"
	"code.vegaprotocol.io/vega/libs/proto"
	vegapb "code.vegaprotocol.io/vega/protos/vega"
	v1 "code.vegaprotocol.io/vega/protos/vega/snapshot/v1"

	cometbftdb "github.com/cometbft/cometbft-db"
	"github.com/cosmos/iavl"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/require"
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
							Name:     "asset",
							Symbol:   "AST",
							Decimals: 0,
							Quantum:  num.DecimalZero(),
							Source: &types.AssetDetailsBuiltinAsset{
								BuiltinAsset: &types.BuiltinAsset{
									MaxFaucetAmountMint: num.UintZero(),
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
							Name:     "asset2",
							Symbol:   "AS2",
							Decimals: 0,
							Quantum:  num.DecimalZero(),
							Source: &types.AssetDetailsBuiltinAsset{
								BuiltinAsset: &types.BuiltinAsset{
									MaxFaucetAmountMint: num.UintZero(),
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
							Amount:  num.UintZero(),
							Asset:   "AST",
							Status:  0,
							Ref:     "rw1",
							TxHash:  "abcdef091235456",
							Ext: &types.WithdrawExt{
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
				Refs: []string{}, // nothing needed
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
						Balance:  num.UintZero(),
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
							Name:     "asset",
							Symbol:   "AST",
							Decimals: 0,
							Quantum:  num.DecimalZero(),
							Source: &types.AssetDetailsBuiltinAsset{
								BuiltinAsset: &types.BuiltinAsset{
									MaxFaucetAmountMint: num.UintZero(),
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
								Change: &types.ProposalTermsNewAsset{
									NewAsset: &types.NewAsset{
										Changes: &types.AssetDetails{
											Name:     "foocoin2",
											Symbol:   "FO2",
											Decimals: 5,
											Quantum:  num.DecimalFromFloat(10),
											Source: &types.AssetDetailsBuiltinAsset{
												BuiltinAsset: &types.BuiltinAsset{
													MaxFaucetAmountMint: num.NewUint(1),
												},
											},
										},
									},
								},
							},
							Rationale: &types.ProposalRationale{
								Description: "some description",
								Title:       "0xdeadbeef",
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
								Change: &types.ProposalTermsNewAsset{
									NewAsset: &types.NewAsset{
										Changes: &types.AssetDetails{
											Name:     "foocoin",
											Symbol:   "FOO",
											Decimals: 5,
											Quantum:  num.DecimalFromFloat(10),
											Source: &types.AssetDetailsBuiltinAsset{
												BuiltinAsset: &types.BuiltinAsset{
													MaxFaucetAmountMint: num.NewUint(1),
												},
											},
										},
									},
								},
							},
							Rationale: &types.ProposalRationale{
								Description: "some description",
								Title:       "0xdeadbeef",
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
						PartyID:        "party1",
						Size:           10,
						Buy:            0,
						Sell:           0,
						Price:          num.NewUint(10),
						BuySumProduct:  num.UintZero(),
						SellSumProduct: num.UintZero(),
					},
					{
						PartyID:        "party2",
						Size:           -10,
						Buy:            0,
						Sell:           0,
						Price:          num.NewUint(10),
						BuySumProduct:  num.UintZero(),
						SellSumProduct: num.UintZero(),
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
									Product: &types.InstrumentFuture{
										Future: &types.Future{
											SettlementAsset: "AST",
											QuoteName:       "AST",
											DataSourceSpecForSettlementData: &types.DataSourceSpec{
												ID: "o1",
												Data: types.NewDataSourceDefinition(
													vegapb.DataSourceDefinitionTypeExt,
												).SetOracleConfig(
													&types.DataSourceSpecConfiguration{
														Signers: []*types.Signer{},
														Filters: []*types.DataSourceSpecFilter{},
													},
												),
											},
											DataSourceSpecForTradingTermination: &types.DataSourceSpec{
												ID: "os1",
												Data: types.NewDataSourceDefinition(
													vegapb.DataSourceDefinitionTypeExt,
												).SetOracleConfig(
													&types.DataSourceSpecConfiguration{
														Signers: []*types.Signer{},
														Filters: []*types.DataSourceSpecFilter{},
													},
												),
											},
											DataSourceSpecBinding: &types.DataSourceSpecBindingForFuture{},
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
							LPPriceRange: num.DecimalFromFloat(0.95),
						},
						PeggedOrders: &types.PeggedOrdersState{},
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
						LastTradedPrice:  num.NewUint(10),
						FeeSplitter: &types.FeeSplitter{
							TimeWindowStart: time.Now(),
							TradeValue:      num.NewUint(1000),
						},
					},
				},
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
				MinVotingTokensFactor: num.UintZero(),
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
				LastTradedPrice: num.UintZero(),
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
				MinVotingTokensFactor: num.UintZero(),
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
		_, _ = tree.Set([]byte(k), serialised)
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
		_, _ = tree.Set([]byte(k), serialised)
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
	db := cometbftdb.NewMemDB()
	tree, err := iavl.NewMutableTreeWithOpts(db, 0, nil, false)
	require.NoError(t, err)
	return tree
}
