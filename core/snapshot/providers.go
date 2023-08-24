package snapshot

import (
	"code.vegaprotocol.io/vega/core/types"
)

// providersInCallOrder holds the providers namespace in the order in which
// they must be called.
var providersInCallOrder = []types.SnapshotNamespace{
	types.EpochSnapshot,
	types.AssetsSnapshot,  // Needs to happen before banking.
	types.WitnessSnapshot, // Needs to happen before banking and governance.
	types.GovernanceSnapshot,
	types.BankingSnapshot,
	types.CollateralSnapshot,
	types.TopologySnapshot,
	types.NotarySnapshot,
	types.NetParamsSnapshot,
	types.CheckpointSnapshot,
	types.DelegationSnapshot,
	types.FloatingPointConsensusSnapshot, // Shouldn't matter but maybe best before the markets are restored.
	types.ExecutionSnapshot,              // Creates the markets, returns matching and positions engines for state providers.
	types.MatchingSnapshot,               // Requires markets.
	types.PositionsSnapshot,              // Requires markets.
	types.SettlementSnapshot,             // Requires markets.
	types.HoldingAccountTrackerSnapshot,
	types.EthereumOracleVerifierSnapshot,
	types.LiquiditySnapshot,
	types.LiquidityV2Snapshot,
	types.LiquidityTargetSnapshot,
	types.StakingSnapshot,
	types.StakeVerifierSnapshot,
	types.SpamSnapshot,
	types.LimitSnapshot,
	types.RewardSnapshot,
	types.EventForwarderSnapshot,
	types.MarketActivityTrackerSnapshot,
	types.ERC20MultiSigTopologySnapshot,
	types.PoWSnapshot,
	types.ProtocolUpgradeSnapshot,
	types.TeamsSnapshot,
	types.VestingSnapshot,
	types.ReferralProgramSnapshot,
}

func groupPayloadsPerNamespace(payloads []*types.Payload) map[types.SnapshotNamespace][]*types.Payload {
	payloadsPerNamespace := make(map[types.SnapshotNamespace][]*types.Payload, len(providersInCallOrder))
	for _, payload := range payloads {
		providerNamespace := payload.Namespace()
		if _, ok := payloadsPerNamespace[providerNamespace]; !ok {
			payloadsPerNamespace[providerNamespace] = []*types.Payload{}
		}
		payloadsPerNamespace[providerNamespace] = append(payloadsPerNamespace[providerNamespace], payload)
	}
	return payloadsPerNamespace
}
