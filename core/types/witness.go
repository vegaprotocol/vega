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

package types

import proto "code.vegaprotocol.io/vega/protos/vega/commands/v1"

type NodeVoteType = proto.NodeVote_Type

const (
	NodeVoteTypeUnspecified                NodeVoteType = proto.NodeVote_TYPE_UNSPECIFIED
	NodeVoteTypeStakeDeposited             NodeVoteType = proto.NodeVote_TYPE_STAKE_DEPOSITED
	NodeVoteTypeStakeRemoved               NodeVoteType = proto.NodeVote_TYPE_STAKE_REMOVED
	NodeVoteTypeFundsDeposited             NodeVoteType = proto.NodeVote_TYPE_FUNDS_DEPOSITED
	NodeVoteTypeSignerAdded                NodeVoteType = proto.NodeVote_TYPE_SIGNER_ADDED
	NodeVoteTypeSignerRemoved              NodeVoteType = proto.NodeVote_TYPE_SIGNER_REMOVED
	NodeVoteTypeBridgeStopped              NodeVoteType = proto.NodeVote_TYPE_BRIDGE_STOPPED
	NodeVoteTypeBridgeResumed              NodeVoteType = proto.NodeVote_TYPE_BRIDGE_RESUMED
	NodeVoteTypeAssetListed                NodeVoteType = proto.NodeVote_TYPE_ASSET_LISTED
	NodeVoteTypeAssetLimitsUpdated         NodeVoteType = proto.NodeVote_TYPE_LIMITS_UPDATED
	NodeVoteTypeStakeTotalSupply           NodeVoteType = proto.NodeVote_TYPE_STAKE_TOTAL_SUPPLY
	NodeVoteTypeSignerThresholdSet         NodeVoteType = proto.NodeVote_TYPE_SIGNER_THRESHOLD_SET
	NodeVoteTypeGovernanceValidateAsset    NodeVoteType = proto.NodeVote_TYPE_GOVERNANCE_VALIDATE_ASSET
	NodeVoteTypeEthereumContractCallResult NodeVoteType = proto.NodeVote_TYPE_ETHEREUM_CONTRACT_CALL_RESULT
	NodeVoteTypeEthereumHeartbeat          NodeVoteType = proto.NodeVote_TYPE_ETHEREUM_HEARTBEAT
)
