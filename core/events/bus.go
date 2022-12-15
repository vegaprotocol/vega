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

var (
	ErrUnsupportedEvent = errors.New("unknown payload for event")
	ErrInvalidEventType = errors.New("invalid proto event type")
)

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
		eventspb.BusEventType_BUS_EVENT_TYPE_ALL:                              All,
		eventspb.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:                      TimeUpdate,
		eventspb.BusEventType_BUS_EVENT_TYPE_LEDGER_MOVEMENTS:                 LedgerMovementsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION:              PositionResolution,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET:                           MarketEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ORDER:                            OrderEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT:                          AccountEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PARTY:                            PartyEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRADE:                            TradeEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:                    MarginLevelsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROPOSAL:                         ProposalEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VOTE:                             VoteEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:                      MarketDataEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:                   NodeSignatureEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:               LossSocializationEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:                  SettlePositionEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:                SettleDistressedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:                   MarketCreatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:                   MarketUpdatedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ASSET:                            AssetEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:                      MarketTickEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:                       WithdrawalEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_DEPOSIT:                          DepositEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_AUCTION:                          AuctionEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR:                      RiskFactorEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_PARAMETER:                NetworkParameterEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION:              LiquidityProvisionEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TX_ERROR:                         TxErrEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_SPEC:                      OracleSpecEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ORACLE_DATA:                      OracleDataEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_EPOCH_UPDATE:                     EpochUpdate,
		eventspb.BusEventType_BUS_EVENT_TYPE_REWARD_PAYOUT_EVENT:              RewardPayoutEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_DELEGATION_BALANCE:               DelegationBalanceEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_SCORE:                  ValidatorScoreEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_STAKE_LINKING:                    StakeLinkingEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_UPDATE:                 ValidatorUpdateEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_CHECKPOINT:                       CheckpointEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_KEY_ROTATION:                     KeyRotationEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_STATE_VAR:                        StateVarEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_NETWORK_LIMITS:                   NetworkLimitsEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRANSFER:                         TransferEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_VALIDATOR_RANKING:                ValidatorRankingEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SET_THRESHOLD:    ERC20MultiSigThresholdSetEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_EVENT:     ERC20MultiSigSignerEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_ADDED:     ERC20MultiSigSignerAddedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ERC20_MULTI_SIG_SIGNER_REMOVED:   ERC20MultiSigSignerRemovedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_POSITION_STATE:                   PositionStateEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_ETHEREUM_KEY_ROTATION:            EthereumKeyRotationEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_PROPOSAL:        ProtocolUpgradeEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_BEGIN_BLOCK:                      BeginBlockEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_END_BLOCK:                        EndBlockEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_STARTED:         ProtocolUpgradeStartedEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SETTLE_MARKET:                    SettleMarketEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_TRANSACTION_RESULT:               TransactionResultEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_SNAPSHOT_TAKEN:                   CoreSnapshotEvent,
		eventspb.BusEventType_BUS_EVENT_TYPE_PROTOCOL_UPGRADE_DATA_NODE_READY: ProtocolUpgradeDataNodeReadyEvent,

		// If adding a type here, please also add it to data-node/broker/convert.go
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
		blockNr: h,
		et:      t,
	}
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
