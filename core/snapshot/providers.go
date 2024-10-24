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

package snapshot

import (
	"code.vegaprotocol.io/vega/core/types"
)

// providersInCallOrder holds the providers namespace in the order in which
// they must be called.
var providersInCallOrder = []types.SnapshotNamespace{
	types.TxCacheSnapshot,
	types.EpochSnapshot,
	types.AssetsSnapshot,  // Needs to happen before banking.
	types.WitnessSnapshot, // Needs to happen before banking and governance.
	types.NetParamsSnapshot,
	types.GovernanceSnapshot,
	types.BankingSnapshot,
	types.CollateralSnapshot,
	types.TopologySnapshot,
	types.NotarySnapshot,
	types.CheckpointSnapshot,
	types.DelegationSnapshot,
	types.FloatingPointConsensusSnapshot, // Shouldn't matter but maybe best before the markets are restored.
	types.ExecutionSnapshot,              // Creates the markets, returns matching and positions engines for state providers.
	types.PositionsSnapshot,              // Requires markets.
	types.MatchingSnapshot,               // Requires markets, and positions so that AMM's evaluate properly
	types.SettlementSnapshot,             // Requires markets.
	types.LiquidationSnapshot,            // Requires markets.
	types.HoldingAccountTrackerSnapshot,
	types.EthereumOracleVerifierSnapshot,
	types.L2EthereumOraclesSnapshot,
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
	types.EVMMultiSigTopologiesSnapshot,
	types.EVMMultiSigTopologiesSnapshot,
	types.PoWSnapshot,
	types.ProtocolUpgradeSnapshot,
	types.TeamsSnapshot,
	types.VestingSnapshot,
	types.ReferralProgramSnapshot,
	types.ActivityStreakSnapshot,
	types.VolumeDiscountProgramSnapshot,
	types.PartiesSnapshot,
	types.EVMHeartbeatSnapshot,
	types.VolumeRebateProgramSnapshot,
	types.VaultSnapshot,
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
