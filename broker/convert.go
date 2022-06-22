// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package broker

import (
	"context"

	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
)

func toEvent(ctx context.Context, be *eventspb.BusEvent) events.Event {
	switch be.Type {
	case eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:
		return events.TimeEventFromStream(ctx, be)
	case eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES:
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
	}

	return nil
}
