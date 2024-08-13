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

package broker

import (
	"context"

	"code.vegaprotocol.io/vega/core/events"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
)

func toEvent(ctx context.Context, be *eventspb.BusEvent) events.Event {
	switch be.Type {
	case eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return events.TimeEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_LEDGER_MOVEMENTS:
		return events.TransferResponseEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION:
		return events.PositionResolutionEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ORDER:
		return events.OrderEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT:
		return events.AccountEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PARTY:
		return events.PartyEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRADE:
		return events.TradeEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:
		return events.MarginLevelsEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PROPOSAL:
		return events.ProposalEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOTE:
		return events.VoteEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:
		return events.MarketDataEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:
		return events.NodeSignatureEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:
		return events.LossSocializationEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:
		return events.SettlePositionEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:
		return events.SettleDistressedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:
		return events.MarketCreatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ASSET:
		return events.AssetEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:
		return events.MarketTickEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:
		return events.WithdrawalEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT:
		return events.DepositEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_AUCTION:
		return events.AuctionEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR:
		return events.RiskFactorEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_PARAMETER:
		return events.NetworkParameterEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION:
		return events.LiquidityProvisionEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:
		return events.MarketUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC:
		return events.OracleSpecEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_DATA:
		return events.OracleDataEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TX_ERROR:
		return events.TxErrEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_EPOCH_UPDATE:
		return events.EpochEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_DELEGATION_BALANCE:
		return events.DelegationBalanceEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_UPDATE:
		return events.ValidatorUpdateEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REWARD_PAYOUT_EVENT:
		return events.RewardPayoutEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_STAKE_LINKING:
		return events.StakeLinkingFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_SCORE:
		return events.ValidatorScoreEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_CHECKPOINT:
		return events.CheckpointEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_KEY_ROTATION:
		return events.KeyRotationEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_STATE_VAR:
		return events.StateVarEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_LIMITS:
		return events.NetworkLimitsEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER:
		return events.TransferFundsEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_RANKING:
		return events.ValidatorRankingEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SET_THRESHOLD:
		return events.ERC20MultiSigThresholdSetFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_EVENT:
		return events.ERC20MultiSigSignerFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_ADDED:
		return events.ERC20MultiSigSignerAddedFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_REMOVED:
		return events.ERC20MultiSigSignerRemovedFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_STATE:
		return events.PositionStateEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_ETHEREUM_KEY_ROTATION:
		return events.EthereumKeyRotationEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_PROPOSAL:
		return events.ProtocolUpgradeProposalEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_BEGIN_BLOCK:
		return events.BeginBlockEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_END_BLOCK:
		return events.EndBlockEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_STARTED:
		return events.ProtocolUpgradeStartedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_MARKET:
		return events.SettleMarketEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSACTION_RESULT:
		return events.TransactionResultEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_SNAPSHOT_TAKEN:
		return events.SnapthostTakenEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_DISTRESSED_ORDERS_CLOSED:
		return events.DistressedOrdersEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_EXPIRED_ORDERS:
		return events.ExpiredOrdersEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_DISTRESSED_POSITIONS:
		return events.DistressedPositionsEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_STOP_ORDER:
		return events.StopOrderEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PERIOD:
		return events.FundingPeriodEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PERIOD_DATA_POINT:
		return events.FundingPeriodDataPointEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TEAM_CREATED:
		return events.TeamCreatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TEAM_UPDATED:
		return events.TeamUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_SWITCHED_TEAM:
		return events.RefereeSwitchedTeamEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_JOINED_TEAM:
		return events.RefereeJoinedTeamEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_STARTED:
		return events.ReferralProgramStartedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_ENDED:
		return events.ReferralProgramEndedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_UPDATED:
		return events.ReferralProgramUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_SET_CREATED:
		return events.ReferralSetCreatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_JOINED_REFERRAL_SET:
		return events.RefereeJoinedReferralSetEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_ACTIVITY_STREAK:
		return events.PartyActivityStreakEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_STARTED:
		return events.VolumeDiscountProgramStartedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_ENDED:
		return events.VolumeDiscountProgramEndedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_UPDATED:
		return events.VolumeDiscountProgramUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_SET_STATS_UPDATED:
		return events.ReferralSetStatsUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VESTING_STATS_UPDATED:
		return events.VestingStatsUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_STATS_UPDATED:
		return events.VolumeDiscountStatsUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_FEES_STATS_UPDATED:
		return events.FeesStatsEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PAYMENTS:
		return events.FundingPaymentEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PAID_LIQUIDITY_FEES_STATS_UPDATED:
		return events.PaidLiquidityFeesStatsEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VESTING_SUMMARY:
		return events.VestingBalancesSummaryEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_FEES_PAID:
		return events.TransferFeesEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_FEES_DISCOUNT_UPDATED:
		return events.TransferFeesDiscountUpdatedFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_MARGIN_MODE_UPDATED:
		return events.PartyMarginModeUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_PROFILE_UPDATED:
		return events.PartyProfileUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TEAMS_STATS_UPDATED:
		return events.TeamsStatsUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TIME_WEIGHTED_NOTIONAL_POSITION_UPDATED:
		return events.TimeWeightedNotionalPositionUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_CANCELLED_ORDERS:
		return events.CancelledOrdersEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_GAME_SCORES:
		return events.GameScoresEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_AMM:
		return events.AMMPoolEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_REBATE_PROGRAM_STARTED:
		return events.VolumeRebateProgramStartedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_REBATE_PROGRAM_ENDED:
		return events.VolumeRebateProgramEndedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_REBATE_PROGRAM_UPDATED:
		return events.VolumeRebateProgramUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_REBATE_STATS_UPDATED:
		return events.VolumeRebateStatsUpdatedEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_COMMUNITY_TAG:
		return events.MarketCommunityTagsEventFromStream(ctx, be)
	}

	return nil
}
