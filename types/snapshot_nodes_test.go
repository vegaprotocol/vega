package types_test

import (
	"testing"
	"time"

	v1 "code.vegaprotocol.io/protos/vega/snapshot/v1"
	"code.vegaprotocol.io/vega/types"
	"code.vegaprotocol.io/vega/types/num"
	"github.com/stretchr/testify/require"
)

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
			StakingAccounts: &types.StakingAccounts{},
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
	}, &types.Payload{
		Data: &types.PayloadOracleData{},
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
