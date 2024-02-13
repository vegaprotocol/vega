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

package events

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	vgcontext "code.vegaprotocol.io/vega/libs/context"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
)

var ErrInvalidEventType = errors.New("invalid proto event type")

type Type int

// simple interface for event filtering on market ID.
type marketFilterable interface {
	Event
	MarketID() string
}

// simple interface for event filtering on party ID.
type partyFilterable interface {
	Event
	IsParty(id string) bool
}

// simple interface for event filtering by party and market ID.
type marketPartyFilterable interface {
	Event
	MarketID() string
	PartyID() string
}

// Base common denominator all event-bus events share.
type Base struct {
	ctx     context.Context
	traceID string
	chainID string
	txHash  string
	blockNr int64
	seq     uint64
	et      Type
}

// Event - the base event interface type, add sequence ID setter here, because the type assertions in broker
// seem to be a bottleneck. Change its behaviour so as to only set the sequence ID once.
type Event interface {
	Type() Type
	Context() context.Context
	TraceID() string
	TxHash() string
	ChainID() string
	Sequence() uint64
	SetSequenceID(s uint64)
	BlockNr() int64
	StreamMessage() *eventspb.BusEvent
	// used for events like ExpiredOrders. It is used to increment the sequence ID by the number of records
	// this event will produce to ensure history tables using time + sequence number to function properly.
	CompositeCount() uint64
	Replace(context.Context)
}

const (
	// All event type -> used by subscribers to just receive all events, has no actual corresponding event payload.
	All Type = iota
	// other event types that DO have corresponding event types.
	TimeUpdate
	LedgerMovementsEvent
	PositionResolution
	MarketEvent // this event is not used for any specific event, but by subscribers that aggregate all market events (e.g. for logging)
	OrderEvent
	LiquidityProvisionEvent
	AccountEvent
	PartyEvent
	TradeEvent
	MarginLevelsEvent
	ProposalEvent
	VoteEvent
	MarketDataEvent
	NodeSignatureEvent
	LossSocializationEvent
	SettlePositionEvent
	SettleDistressedEvent
	MarketCreatedEvent
	MarketUpdatedEvent
	AssetEvent
	MarketTickEvent
	AuctionEvent
	WithdrawalEvent
	DepositEvent
	RiskFactorEvent
	NetworkParameterEvent
	TxErrEvent
	OracleSpecEvent
	OracleDataEvent
	EpochUpdate
	DelegationBalanceEvent
	StakeLinkingEvent
	ValidatorUpdateEvent
	RewardPayoutEvent
	CheckpointEvent
	ValidatorScoreEvent
	KeyRotationEvent
	StateVarEvent
	NetworkLimitsEvent
	TransferEvent
	ValidatorRankingEvent
	ERC20MultiSigThresholdSetEvent
	ERC20MultiSigSignerEvent
	ERC20MultiSigSignerAddedEvent
	ERC20MultiSigSignerRemovedEvent
	PositionStateEvent
	EthereumKeyRotationEvent
	ProtocolUpgradeEvent
	BeginBlockEvent
	EndBlockEvent
	ProtocolUpgradeStartedEvent
	SettleMarketEvent
	TransactionResultEvent
	CoreSnapshotEvent
	ProtocolUpgradeDataNodeReadyEvent
	DistressedOrdersClosedEvent
	ExpiredOrdersEvent
	DistressedPositionsEvent
	SpotLiquidityProvisionEvent
	StopOrderEvent
	FundingPeriodEvent
	FundingPeriodDataPointEvent
	TeamCreatedEvent
	TeamUpdatedEvent
	RefereeSwitchedTeamEvent
	RefereeJoinedTeamEvent
	ReferralProgramStartedEvent
	ReferralProgramEndedEvent
	ReferralProgramUpdatedEvent
	ReferralSetCreatedEvent
	RefereeJoinedReferralSetEvent
	PartyActivityStreakEvent
	VolumeDiscountProgramStartedEvent
	VolumeDiscountProgramEndedEvent
	VolumeDiscountProgramUpdatedEvent
	ReferralSetStatsUpdatedEvent
	VestingStatsUpdatedEvent
	VolumeDiscountStatsUpdatedEvent
	FeesStatsEvent
	FundingPaymentsEvent
	PaidLiquidityFeesStatsEvent
	VestingBalancesSummaryEvent
	TransferFeesEvent
	TransferFeesDiscountUpdatedEvent
	PartyMarginModeUpdatedEvent
	PartyProfileUpdatedEvent
	TeamsStatsUpdatedEvent
)

var (
	marketEvents = []Type{
		PositionResolution,
		MarketCreatedEvent,
		MarketUpdatedEvent,
		MarketTickEvent,
		AuctionEvent,
	}

	protoMap = map[eventspb.BusEventType]Type{
		eventspb.BusEventType_BUS_EVENT_TYPE_ALL:                               All,
		eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:                       TimeUpdate,
		eventspb.BusEventType_BUS_EVENT_TYPE_LEDGER_MOVEMENTS:                  LedgerMovementsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION:               PositionResolution,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET:                            MarketEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ORDER:                             OrderEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT:                           AccountEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PARTY:                             PartyEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRADE:                             TradeEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:                     MarginLevelsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROPOSAL:                          ProposalEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VOTE:                              VoteEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:                       MarketDataEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:                    NodeSignatureEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:                LossSocializationEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:                   SettlePositionEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:                 SettleDistressedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:                    MarketCreatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:                    MarketUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ASSET:                             AssetEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:                       MarketTickEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:                        WithdrawalEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT:                           DepositEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_AUCTION:                           AuctionEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR:                       RiskFactorEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_PARAMETER:                 NetworkParameterEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION:               LiquidityProvisionEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TX_ERROR:                          TxErrEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC:                       OracleSpecEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_DATA:                       OracleDataEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_EPOCH_UPDATE:                      EpochUpdate,
		eventspb.BusEventType_BUS_EVENT_TYPE_REWARD_PAYOUT_EVENT:               RewardPayoutEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_DELEGATION_BALANCE:                DelegationBalanceEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_SCORE:                   ValidatorScoreEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_STAKE_LINKING:                     StakeLinkingEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_UPDATE:                  ValidatorUpdateEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_CHECKPOINT:                        CheckpointEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_KEY_ROTATION:                      KeyRotationEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_STATE_VAR:                         StateVarEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_LIMITS:                    NetworkLimitsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER:                          TransferEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_RANKING:                 ValidatorRankingEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SET_THRESHOLD:     ERC20MultiSigThresholdSetEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_EVENT:      ERC20MultiSigSignerEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_ADDED:      ERC20MultiSigSignerAddedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_REMOVED:    ERC20MultiSigSignerRemovedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_STATE:                    PositionStateEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ETHEREUM_KEY_ROTATION:             EthereumKeyRotationEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_PROPOSAL:         ProtocolUpgradeEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_BEGIN_BLOCK:                       BeginBlockEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_END_BLOCK:                         EndBlockEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_STARTED:          ProtocolUpgradeStartedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_MARKET:                     SettleMarketEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRANSACTION_RESULT:                TransactionResultEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SNAPSHOT_TAKEN:                    CoreSnapshotEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_DATA_NODE_READY:  ProtocolUpgradeDataNodeReadyEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_DISTRESSED_ORDERS_CLOSED:          DistressedOrdersClosedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_EXPIRED_ORDERS:                    ExpiredOrdersEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_DISTRESSED_POSITIONS:              DistressedPositionsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_STOP_ORDER:                        StopOrderEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PERIOD:                    FundingPeriodEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PERIOD_DATA_POINT:         FundingPeriodDataPointEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TEAM_CREATED:                      TeamCreatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TEAM_UPDATED:                      TeamUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_SWITCHED_TEAM:             RefereeSwitchedTeamEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_JOINED_TEAM:               RefereeJoinedTeamEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_STARTED:          ReferralProgramStartedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_ENDED:            ReferralProgramEndedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_UPDATED:          ReferralProgramUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_SET_CREATED:              ReferralSetCreatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_JOINED_REFERRAL_SET:       RefereeJoinedReferralSetEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_ACTIVITY_STREAK:             PartyActivityStreakEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_STARTED:   VolumeDiscountProgramStartedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_ENDED:     VolumeDiscountProgramEndedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_UPDATED:   VolumeDiscountProgramUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_SET_STATS_UPDATED:        ReferralSetStatsUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VESTING_STATS_UPDATED:             VestingStatsUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_STATS_UPDATED:     VolumeDiscountStatsUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_FEES_STATS_UPDATED:                FeesStatsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PAYMENTS:                  FundingPaymentsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PAID_LIQUIDITY_FEES_STATS_UPDATED: PaidLiquidityFeesStatsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VESTING_SUMMARY:                   VestingBalancesSummaryEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_FEES_PAID:                TransferFeesEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_FEES_DISCOUNT_UPDATED:    TransferFeesDiscountUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_MARGIN_MODE_UPDATED:         PartyMarginModeUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_PROFILE_UPDATED:             PartyProfileUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TEAMS_STATS_UPDATED:               TeamsStatsUpdatedEvent,
		// If adding a type here, please also add it to datanode/broker/convert.go
	}

	toProto = map[Type]eventspb.BusEventType{
		ValidatorRankingEvent:             eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_RANKING,
		TimeUpdate:                        eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE,
		LedgerMovementsEvent:              eventspb.BusEventType_BUS_EVENT_TYPE_LEDGER_MOVEMENTS,
		PositionResolution:                eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION,
		MarketEvent:                       eventspb.BusEventType_BUS_EVENT_TYPE_MARKET,
		OrderEvent:                        eventspb.BusEventType_BUS_EVENT_TYPE_ORDER,
		AccountEvent:                      eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT,
		PartyEvent:                        eventspb.BusEventType_BUS_EVENT_TYPE_PARTY,
		TradeEvent:                        eventspb.BusEventType_BUS_EVENT_TYPE_TRADE,
		MarginLevelsEvent:                 eventspb.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS,
		ProposalEvent:                     eventspb.BusEventType_BUS_EVENT_TYPE_PROPOSAL,
		VoteEvent:                         eventspb.BusEventType_BUS_EVENT_TYPE_VOTE,
		MarketDataEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_DATA,
		NodeSignatureEvent:                eventspb.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE,
		LossSocializationEvent:            eventspb.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION,
		SettlePositionEvent:               eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION,
		SettleDistressedEvent:             eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED,
		MarketCreatedEvent:                eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED,
		MarketUpdatedEvent:                eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED,
		AssetEvent:                        eventspb.BusEventType_BUS_EVENT_TYPE_ASSET,
		MarketTickEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_TICK,
		WithdrawalEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL,
		DepositEvent:                      eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT,
		AuctionEvent:                      eventspb.BusEventType_BUS_EVENT_TYPE_AUCTION,
		RiskFactorEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR,
		NetworkParameterEvent:             eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_PARAMETER,
		LiquidityProvisionEvent:           eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION,
		TxErrEvent:                        eventspb.BusEventType_BUS_EVENT_TYPE_TX_ERROR,
		OracleSpecEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC,
		OracleDataEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_DATA,
		EpochUpdate:                       eventspb.BusEventType_BUS_EVENT_TYPE_EPOCH_UPDATE,
		DelegationBalanceEvent:            eventspb.BusEventType_BUS_EVENT_TYPE_DELEGATION_BALANCE,
		StakeLinkingEvent:                 eventspb.BusEventType_BUS_EVENT_TYPE_STAKE_LINKING,
		ValidatorUpdateEvent:              eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_UPDATE,
		RewardPayoutEvent:                 eventspb.BusEventType_BUS_EVENT_TYPE_REWARD_PAYOUT_EVENT,
		CheckpointEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_CHECKPOINT,
		ValidatorScoreEvent:               eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_SCORE,
		KeyRotationEvent:                  eventspb.BusEventType_BUS_EVENT_TYPE_KEY_ROTATION,
		StateVarEvent:                     eventspb.BusEventType_BUS_EVENT_TYPE_STATE_VAR,
		NetworkLimitsEvent:                eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_LIMITS,
		TransferEvent:                     eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER,
		ERC20MultiSigThresholdSetEvent:    eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SET_THRESHOLD,
		ERC20MultiSigSignerEvent:          eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_EVENT,
		ERC20MultiSigSignerAddedEvent:     eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_ADDED,
		ERC20MultiSigSignerRemovedEvent:   eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_REMOVED,
		PositionStateEvent:                eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_STATE,
		EthereumKeyRotationEvent:          eventspb.BusEventType_BUS_EVENT_TYPE_ETHEREUM_KEY_ROTATION,
		ProtocolUpgradeEvent:              eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_PROPOSAL,
		BeginBlockEvent:                   eventspb.BusEventType_BUS_EVENT_TYPE_BEGIN_BLOCK,
		EndBlockEvent:                     eventspb.BusEventType_BUS_EVENT_TYPE_END_BLOCK,
		ProtocolUpgradeStartedEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_STARTED,
		SettleMarketEvent:                 eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_MARKET,
		TransactionResultEvent:            eventspb.BusEventType_BUS_EVENT_TYPE_TRANSACTION_RESULT,
		CoreSnapshotEvent:                 eventspb.BusEventType_BUS_EVENT_TYPE_SNAPSHOT_TAKEN,
		ProtocolUpgradeDataNodeReadyEvent: eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_DATA_NODE_READY,
		DistressedOrdersClosedEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_DISTRESSED_ORDERS_CLOSED,
		ExpiredOrdersEvent:                eventspb.BusEventType_BUS_EVENT_TYPE_EXPIRED_ORDERS,
		DistressedPositionsEvent:          eventspb.BusEventType_BUS_EVENT_TYPE_DISTRESSED_POSITIONS,
		StopOrderEvent:                    eventspb.BusEventType_BUS_EVENT_TYPE_STOP_ORDER,
		FundingPeriodEvent:                eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PERIOD,
		FundingPeriodDataPointEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PERIOD_DATA_POINT,
		TeamCreatedEvent:                  eventspb.BusEventType_BUS_EVENT_TYPE_TEAM_CREATED,
		TeamUpdatedEvent:                  eventspb.BusEventType_BUS_EVENT_TYPE_TEAM_UPDATED,
		RefereeSwitchedTeamEvent:          eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_SWITCHED_TEAM,
		RefereeJoinedTeamEvent:            eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_JOINED_TEAM,
		ReferralProgramStartedEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_STARTED,
		ReferralProgramEndedEvent:         eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_ENDED,
		ReferralProgramUpdatedEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_PROGRAM_UPDATED,
		ReferralSetCreatedEvent:           eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_SET_CREATED,
		RefereeJoinedReferralSetEvent:     eventspb.BusEventType_BUS_EVENT_TYPE_REFEREE_JOINED_REFERRAL_SET,
		PartyActivityStreakEvent:          eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_ACTIVITY_STREAK,
		VolumeDiscountProgramStartedEvent: eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_STARTED,
		VolumeDiscountProgramEndedEvent:   eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_ENDED,
		VolumeDiscountProgramUpdatedEvent: eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_PROGRAM_UPDATED,
		ReferralSetStatsUpdatedEvent:      eventspb.BusEventType_BUS_EVENT_TYPE_REFERRAL_SET_STATS_UPDATED,
		VestingStatsUpdatedEvent:          eventspb.BusEventType_BUS_EVENT_TYPE_VESTING_STATS_UPDATED,
		VolumeDiscountStatsUpdatedEvent:   eventspb.BusEventType_BUS_EVENT_TYPE_VOLUME_DISCOUNT_STATS_UPDATED,
		FeesStatsEvent:                    eventspb.BusEventType_BUS_EVENT_TYPE_FEES_STATS_UPDATED,
		FundingPaymentsEvent:              eventspb.BusEventType_BUS_EVENT_TYPE_FUNDING_PAYMENTS,
		PaidLiquidityFeesStatsEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_PAID_LIQUIDITY_FEES_STATS_UPDATED,
		VestingBalancesSummaryEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_VESTING_SUMMARY,
		TransferFeesEvent:                 eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_FEES_PAID,
		TransferFeesDiscountUpdatedEvent:  eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER_FEES_DISCOUNT_UPDATED,
		PartyMarginModeUpdatedEvent:       eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_MARGIN_MODE_UPDATED,
		PartyProfileUpdatedEvent:          eventspb.BusEventType_BUS_EVENT_TYPE_PARTY_PROFILE_UPDATED,
		TeamsStatsUpdatedEvent:            eventspb.BusEventType_BUS_EVENT_TYPE_TEAMS_STATS_UPDATED,
		// If adding a type here, please also add it to datanode/broker/convert.go
	}

	eventStrings = map[Type]string{
		All:                               "ALL",
		TimeUpdate:                        "TimeUpdate",
		LedgerMovementsEvent:              "LedgerMovements",
		PositionResolution:                "PositionResolution",
		MarketEvent:                       "MarketEvent",
		OrderEvent:                        "OrderEvent",
		AccountEvent:                      "AccountEvent",
		PartyEvent:                        "PartyEvent",
		TradeEvent:                        "TradeEvent",
		MarginLevelsEvent:                 "MarginLevelsEvent",
		ProposalEvent:                     "ProposalEvent",
		VoteEvent:                         "VoteEvent",
		MarketDataEvent:                   "MarketDataEvent",
		NodeSignatureEvent:                "NodeSignatureEvent",
		LossSocializationEvent:            "LossSocializationEvent",
		SettlePositionEvent:               "SettlePositionEvent",
		SettleDistressedEvent:             "SettleDistressedEvent",
		MarketCreatedEvent:                "MarketCreatedEvent",
		MarketUpdatedEvent:                "MarketUpdatedEvent",
		AssetEvent:                        "AssetEvent",
		MarketTickEvent:                   "MarketTickEvent",
		AuctionEvent:                      "AuctionEvent",
		WithdrawalEvent:                   "WithdrawalEvent",
		DepositEvent:                      "DepositEvent",
		RiskFactorEvent:                   "RiskFactorEvent",
		NetworkParameterEvent:             "NetworkParameterEvent",
		LiquidityProvisionEvent:           "LiquidityProvisionEvent",
		TxErrEvent:                        "TxErrEvent",
		OracleSpecEvent:                   "OracleSpecEvent",
		OracleDataEvent:                   "OracleDataEvent",
		EpochUpdate:                       "EpochUpdate",
		DelegationBalanceEvent:            "DelegationBalanceEvent",
		StakeLinkingEvent:                 "StakeLinkingEvent",
		ValidatorUpdateEvent:              "ValidatorUpdateEvent",
		RewardPayoutEvent:                 "RewardPayoutEvent",
		CheckpointEvent:                   "CheckpointEvent",
		ValidatorScoreEvent:               "ValidatorScoreEvent",
		KeyRotationEvent:                  "KeyRotationEvent",
		StateVarEvent:                     "StateVarEvent",
		NetworkLimitsEvent:                "NetworkLimitsEvent",
		TransferEvent:                     "TransferEvent",
		ValidatorRankingEvent:             "ValidatorRankingEvent",
		ERC20MultiSigSignerEvent:          "ERC20MultiSigSignerEvent",
		ERC20MultiSigThresholdSetEvent:    "ERC20MultiSigThresholdSetEvent",
		ERC20MultiSigSignerAddedEvent:     "ERC20MultiSigSignerAddedEvent",
		ERC20MultiSigSignerRemovedEvent:   "ERC20MultiSigSignerRemovedEvent",
		PositionStateEvent:                "PositionStateEvent",
		EthereumKeyRotationEvent:          "EthereumKeyRotationEvent",
		ProtocolUpgradeEvent:              "ProtocolUpgradeEvent",
		BeginBlockEvent:                   "BeginBlockEvent",
		EndBlockEvent:                     "EndBlockEvent",
		ProtocolUpgradeStartedEvent:       "ProtocolUpgradeStartedEvent",
		SettleMarketEvent:                 "SettleMarketEvent",
		TransactionResultEvent:            "TransactionResultEvent",
		CoreSnapshotEvent:                 "CoreSnapshotEvent",
		ProtocolUpgradeDataNodeReadyEvent: "UpgradeDataNodeEvent",
		DistressedOrdersClosedEvent:       "DistressedOrdersClosedEvent",
		ExpiredOrdersEvent:                "ExpiredOrdersEvent",
		DistressedPositionsEvent:          "DistressedPositionsEvent",
		StopOrderEvent:                    "StopOrderEvent",
		FundingPeriodEvent:                "FundingPeriodEvent",
		FundingPeriodDataPointEvent:       "FundingPeriodDataPointEvent",
		TeamCreatedEvent:                  "TeamCreatedEvent",
		TeamUpdatedEvent:                  "TeamUpdatedEvent",
		RefereeSwitchedTeamEvent:          "RefereeSwitchedTeamEvent",
		RefereeJoinedTeamEvent:            "RefereeJoinedTeamEvent",
		ReferralProgramStartedEvent:       "ReferralProgramStartedEvent",
		ReferralProgramEndedEvent:         "ReferralProgramEndedEvent",
		ReferralProgramUpdatedEvent:       "ReferralProgramUpdatedEvent",
		ReferralSetCreatedEvent:           "ReferralSetCreatedEvent",
		RefereeJoinedReferralSetEvent:     "RefereeJoinReferralSetEvent",
		PartyActivityStreakEvent:          "PartyActivityStreakEvent",
		VolumeDiscountProgramStartedEvent: "VolumeDiscountProgramStartedEvent",
		VolumeDiscountProgramEndedEvent:   "VolumeDiscountProgramEndedEvent",
		VolumeDiscountProgramUpdatedEvent: "VolumeDiscountProgramUpdatedEvent",
		ReferralSetStatsUpdatedEvent:      "ReferralSetStatsUpdatedEvent",
		VestingStatsUpdatedEvent:          "VestingStatsUpdatedEvent",
		VolumeDiscountStatsUpdatedEvent:   "VolumeDiscountStatsUpdatedEvent",
		FeesStatsEvent:                    "FeesStatsEvent",
		FundingPaymentsEvent:              "FundingPaymentsEvent",
		PaidLiquidityFeesStatsEvent:       "LiquidityFeesStatsEvent",
		VestingBalancesSummaryEvent:       "VestingBalancesSummaryEvent",
		PartyMarginModeUpdatedEvent:       "PartyMarginModeUpdatedEvent",
		PartyProfileUpdatedEvent:          "PartyProfileUpdatedEvent",
		TeamsStatsUpdatedEvent:            "TeamsStatsUpdatedEvent",
	}
)

// A base event holds no data, so the constructor will not be called directly.
func newBase(ctx context.Context, t Type) *Base {
	ctx, tID := vgcontext.TraceIDFromContext(ctx)
	cID, _ := vgcontext.ChainIDFromContext(ctx)
	h, _ := vgcontext.BlockHeightFromContext(ctx)
	txHash, _ := vgcontext.TxHashFromContext(ctx)
	return &Base{
		ctx:     ctx,
		traceID: tID,
		chainID: cID,
		txHash:  txHash,
		blockNr: int64(h),
		et:      t,
	}
}

// Replace updates the event to be based on the new given context.
func (b *Base) Replace(ctx context.Context) {
	nb := newBase(ctx, b.Type())
	*b = *nb
}

// CompositeCount on the base event will default to 1.
func (b Base) CompositeCount() uint64 {
	return 1
}

// TraceID returns the... traceID obviously.
func (b Base) TraceID() string {
	return b.traceID
}

func (b Base) ChainID() string {
	return b.chainID
}

func (b Base) TxHash() string {
	return b.txHash
}

func (b *Base) SetSequenceID(s uint64) {
	// sequence ID can only be set once
	if b.seq != 0 {
		return
	}
	b.seq = s
}

// Sequence returns event sequence number.
func (b Base) Sequence() uint64 {
	return b.seq
}

// Context returns context.
func (b Base) Context() context.Context {
	return b.ctx
}

// Type returns the event type.
func (b Base) Type() Type {
	return b.et
}

func (b Base) eventID() string {
	return fmt.Sprintf("%d-%d", b.blockNr, b.seq)
}

// BlockNr returns the current block number.
func (b Base) BlockNr() int64 {
	return b.blockNr
}

// MarketEvents return all the possible market events.
func MarketEvents() []Type {
	return marketEvents
}

// String get string representation of event type.
func (t Type) String() string {
	s, ok := eventStrings[t]
	if !ok {
		return "UNKNOWN EVENT"
	}
	return s
}

// TryFromString tries to parse a raw string into an event type, false indicates that.
func TryFromString(s string) (*Type, bool) {
	for k, v := range eventStrings {
		if strings.EqualFold(s, v) {
			return &k, true
		}
	}
	return nil, false
}

// ProtoToInternal converts the proto message enum to our internal constants
// we're not using a map to de-duplicate the event types here, so we can exploit
// duplicating the same event to control the internal subscriber channel buffer.
func ProtoToInternal(pTypes ...eventspb.BusEventType) ([]Type, error) {
	ret := make([]Type, 0, len(pTypes))
	for _, t := range pTypes {
		// all events -> subscriber should return a nil slice
		if t == eventspb.BusEventType_BUS_EVENT_TYPE_ALL {
			return nil, nil
		}
		it, ok := protoMap[t]
		if !ok {
			return nil, ErrInvalidEventType
		}
		if it == MarketEvent {
			ret = append(ret, marketEvents...)
		} else {
			ret = append(ret, it)
		}
	}
	return ret, nil
}

func GetMarketIDFilter(mID string) func(Event) bool {
	return func(e Event) bool {
		me, ok := e.(marketFilterable)
		if !ok {
			return false
		}
		return me.MarketID() == mID
	}
}

func GetPartyIDFilter(pID string) func(Event) bool {
	return func(e Event) bool {
		pe, ok := e.(partyFilterable)
		if !ok {
			return false
		}
		return pe.IsParty(pID)
	}
}

func GetPartyAndMarketFilter(mID, pID string) func(Event) bool {
	return func(e Event) bool {
		mpe, ok := e.(marketPartyFilterable)
		if !ok {
			return false
		}
		return mpe.MarketID() == mID && mpe.PartyID() == pID
	}
}

func (t Type) ToProto() eventspb.BusEventType {
	pt, ok := toProto[t]
	if !ok {
		panic(fmt.Sprintf("Converting events.Type %s to proto BusEventType: no corresponding value found", t))
	}
	return pt
}

func newBusEventFromBase(base *Base) *eventspb.BusEvent {
	event := &eventspb.BusEvent{
		Version: eventspb.Version,
		Id:      base.eventID(),
		Type:    base.Type().ToProto(),
		Block:   base.TraceID(),
		ChainId: base.ChainID(),
		TxHash:  base.TxHash(),
	}

	return event
}

func newBaseFromBusEvent(ctx context.Context, t Type, be *eventspb.BusEvent) *Base {
	evtCtx := vgcontext.WithTraceID(ctx, be.Block)
	evtCtx = vgcontext.WithChainID(evtCtx, be.ChainId)
	evtCtx = vgcontext.WithTxHash(evtCtx, be.TxHash)
	blockNr, seq := decodeEventID(be.Id)
	return &Base{
		ctx:     evtCtx,
		traceID: be.Block,
		chainID: be.ChainId,
		txHash:  be.TxHash,
		blockNr: blockNr,
		seq:     seq,
		et:      t,
	}
}

func decodeEventID(id string) (blockNr int64, seq uint64) {
	arr := strings.Split(id, "-")
	s1, s2 := arr[0], arr[1]
	blockNr, _ = strconv.ParseInt(s1, 10, 64)
	n, _ := strconv.ParseInt(s2, 10, 64)
	seq = uint64(n)
	return
}
