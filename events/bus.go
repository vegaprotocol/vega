package events

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	types "code.vegaprotocol.io/vega/proto/gen/golang"

	"github.com/pkg/errors"
)

var (
	ErrUnsuportedEvent  = errors.New("unknown payload for event")
	ErrInvalidEventType = errors.New("invalid proto event type")
)

type Type int

// simple interface for event filtering on market ID
type marketFilterable interface {
	Event
	MarketID() string
}

// simple interface for event filtering on party ID
type partyFilterable interface {
	Event
	IsParty(id string) bool
}

// simple interface for event filtering by party and market ID
type marketPartyFilterable interface {
	Event
	MarketID() string
	PartyID() string
}

// Base common denominator all event-bus events share
type Base struct {
	ctx     context.Context
	traceID string
	blockNr int64
	seq     uint64
	et      Type
}

// Event - the base event interface type, add sequence ID setter here, because the type assertions in broker
// seem to be a bottleneck. Change its behaviour so as to only set the sequence ID once
type Event interface {
	Type() Type
	Context() context.Context
	TraceID() string
	Sequence() uint64
	SetSequenceID(s uint64)
}

const (
	// All event type -> used by subscrubers to just receive all events, has no actual corresponding event payload
	All Type = iota
	// other event types that DO have corresponding event types
	TimeUpdate
	TransferResponses
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
)

var (
	marketEvents = []Type{
		PositionResolution,
		MarketCreatedEvent,
		MarketUpdatedEvent,
		MarketTickEvent,
		AuctionEvent,
	}

	protoMap = map[types.BusEventType]Type{
		types.BusEventType_BUS_EVENT_TYPE_ALL:                 All,
		types.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE:         TimeUpdate,
		types.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES:  TransferResponses,
		types.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION: PositionResolution,
		types.BusEventType_BUS_EVENT_TYPE_MARKET:              MarketEvent,
		types.BusEventType_BUS_EVENT_TYPE_ORDER:               OrderEvent,
		types.BusEventType_BUS_EVENT_TYPE_ACCOUNT:             AccountEvent,
		types.BusEventType_BUS_EVENT_TYPE_PARTY:               PartyEvent,
		types.BusEventType_BUS_EVENT_TYPE_TRADE:               TradeEvent,
		types.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS:       MarginLevelsEvent,
		types.BusEventType_BUS_EVENT_TYPE_PROPOSAL:            ProposalEvent,
		types.BusEventType_BUS_EVENT_TYPE_VOTE:                VoteEvent,
		types.BusEventType_BUS_EVENT_TYPE_MARKET_DATA:         MarketDataEvent,
		types.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE:      NodeSignatureEvent,
		types.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION:  LossSocializationEvent,
		types.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION:     SettlePositionEvent,
		types.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED:   SettleDistressedEvent,
		types.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED:      MarketCreatedEvent,
		types.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED:      MarketUpdatedEvent,
		types.BusEventType_BUS_EVENT_TYPE_ASSET:               AssetEvent,
		types.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:         MarketTickEvent,
		types.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:          WithdrawalEvent,
		types.BusEventType_BUS_EVENT_TYPE_DEPOSIT:             DepositEvent,
		types.BusEventType_BUS_EVENT_TYPE_AUCTION:             AuctionEvent,
		types.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR:         RiskFactorEvent,
		types.BusEventType_BUS_EVENT_TYPE_NETWORK_PARAMETER:   NetworkParameterEvent,
		types.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION: LiquidityProvisionEvent,
		types.BusEventType_BUS_EVENT_TYPE_TX_ERROR:            TxErrEvent,
	}

	toProto = map[Type]types.BusEventType{
		TimeUpdate:              types.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE,
		TransferResponses:       types.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES,
		PositionResolution:      types.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION,
		MarketEvent:             types.BusEventType_BUS_EVENT_TYPE_MARKET,
		OrderEvent:              types.BusEventType_BUS_EVENT_TYPE_ORDER,
		AccountEvent:            types.BusEventType_BUS_EVENT_TYPE_ACCOUNT,
		PartyEvent:              types.BusEventType_BUS_EVENT_TYPE_PARTY,
		TradeEvent:              types.BusEventType_BUS_EVENT_TYPE_TRADE,
		MarginLevelsEvent:       types.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS,
		ProposalEvent:           types.BusEventType_BUS_EVENT_TYPE_PROPOSAL,
		VoteEvent:               types.BusEventType_BUS_EVENT_TYPE_VOTE,
		MarketDataEvent:         types.BusEventType_BUS_EVENT_TYPE_MARKET_DATA,
		NodeSignatureEvent:      types.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE,
		LossSocializationEvent:  types.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION,
		SettlePositionEvent:     types.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION,
		SettleDistressedEvent:   types.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED,
		MarketCreatedEvent:      types.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED,
		MarketUpdatedEvent:      types.BusEventType_BUS_EVENT_TYPE_MARKET_UPDATED,
		AssetEvent:              types.BusEventType_BUS_EVENT_TYPE_ASSET,
		MarketTickEvent:         types.BusEventType_BUS_EVENT_TYPE_MARKET_TICK,
		WithdrawalEvent:         types.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL,
		DepositEvent:            types.BusEventType_BUS_EVENT_TYPE_DEPOSIT,
		AuctionEvent:            types.BusEventType_BUS_EVENT_TYPE_AUCTION,
		RiskFactorEvent:         types.BusEventType_BUS_EVENT_TYPE_RISK_FACTOR,
		NetworkParameterEvent:   types.BusEventType_BUS_EVENT_TYPE_NETWORK_PARAMETER,
		LiquidityProvisionEvent: types.BusEventType_BUS_EVENT_TYPE_LIQUIDITY_PROVISION,
		TxErrEvent:              types.BusEventType_BUS_EVENT_TYPE_TX_ERROR,
	}

	eventStrings = map[Type]string{
		All:                     "ALL",
		TimeUpdate:              "TimeUpdate",
		TransferResponses:       "TransferResponses",
		PositionResolution:      "PositionResolution",
		MarketEvent:             "MarketEvent",
		OrderEvent:              "OrderEvent",
		AccountEvent:            "AccountEvent",
		PartyEvent:              "PartyEvent",
		TradeEvent:              "TradeEvent",
		MarginLevelsEvent:       "MarginLevelsEvent",
		ProposalEvent:           "ProposalEvent",
		VoteEvent:               "VoteEvent",
		MarketDataEvent:         "MarketDataEvent",
		NodeSignatureEvent:      "NodeSignatureEvent",
		LossSocializationEvent:  "LossSocializationEvent",
		SettlePositionEvent:     "SettlePositionEvent",
		SettleDistressedEvent:   "SettleDistressedEvent",
		MarketCreatedEvent:      "MarketCreatedEvent",
		MarketUpdatedEvent:      "MarketUpdatedEvent",
		AssetEvent:              "AssetEvent",
		MarketTickEvent:         "MarketTickEvent",
		AuctionEvent:            "AuctionEvent",
		WithdrawalEvent:         "WithdrawalEvent",
		DepositEvent:            "DepositEvent",
		RiskFactorEvent:         "RiskFactorEvent",
		NetworkParameterEvent:   "NetworkParameterEvent",
		LiquidityProvisionEvent: "LiquidityProvisionEvent",
		TxErrEvent:              "TxErrEvent",
	}
)

// New is a generic constructor - based on the type of v, the specific event will be returned
func New(ctx context.Context, v interface{}) (interface{}, error) {
	switch tv := v.(type) {
	case *time.Time:
		e := NewTime(ctx, *tv)
		return e, nil
	case time.Time:
		e := NewTime(ctx, tv)
		return e, nil
	case []*types.TransferResponse:
		e := NewTransferResponse(ctx, tv)
		return e, nil
	case *types.Order:
		e := NewOrderEvent(ctx, tv)
		return e, nil
	case types.Account:
		e := NewAccountEvent(ctx, tv)
		return e, nil
	case types.Party:
		e := NewPartyEvent(ctx, tv)
		return e, nil
	case types.Trade:
		e := NewTradeEvent(ctx, tv)
		return e, nil
	case types.MarginLevels:
		e := NewMarginLevelsEvent(ctx, tv)
		return e, nil
	case types.Proposal:
		e := NewProposalEvent(ctx, tv)
		return e, nil
	case types.Vote:
		e := NewVoteEvent(ctx, tv)
		return e, nil
	case types.MarketData:
		e := NewMarketDataEvent(ctx, tv)
		return e, nil
	case types.NodeSignature:
		e := NewNodeSignatureEvent(ctx, tv)
		return e, nil
	case types.Asset:
		e := NewAssetEvent(ctx, tv)
		return e, nil
	case types.Withdrawal:
		e := NewWithdrawalEvent(ctx, tv)
		return e, nil
	case types.Deposit:
		e := NewDepositEvent(ctx, tv)
		return e, nil
	case types.RiskFactor:
		e := NewRiskFactorEvent(ctx, tv)
		return e, nil
	case types.LiquidityProvision:
		e := NewLiquidityProvisionEvent(ctx, &tv)
		return e, nil
	}
	return nil, ErrUnsuportedEvent
}

// A base event holds no data, so the constructor will not be called directly
func newBase(ctx context.Context, t Type) *Base {
	ctx, tID := contextutil.TraceIDFromContext(ctx)
	h, _ := contextutil.BlockHeightFromContext(ctx)
	return &Base{
		ctx:     ctx,
		traceID: tID,
		blockNr: h,
		et:      t,
	}
}

// TraceID returns the... traceID obviously
func (b Base) TraceID() string {
	return b.traceID
}

func (b *Base) SetSequenceID(s uint64) {
	// sequence ID can only be set once
	if b.seq != 0 {
		return
	}
	b.seq = s
}

// Sequence returns event sequence number
func (b Base) Sequence() uint64 {
	return b.seq
}

// Context returns context
func (b Base) Context() context.Context {
	return b.ctx
}

// Type returns the event type
func (b Base) Type() Type {
	return b.et
}

func (b Base) eventID() string {
	return fmt.Sprintf("%d-%d", b.blockNr, b.seq)
}

// MarketEvents return all the possible market events
func MarketEvents() []Type {
	return marketEvents
}

// String get string representation of event type
func (t Type) String() string {
	s, ok := eventStrings[t]
	if !ok {
		return "UNKNOWN EVENT"
	}
	return s
}

// ProtoToInternal converts the proto message enum to our internal constants
// we're not using a map to de-duplicate the event types here, so we can exploit
// duplicating the same event to control the internal subscriber channel buffer
func ProtoToInternal(pTypes ...types.BusEventType) ([]Type, error) {
	ret := make([]Type, 0, len(pTypes))
	for _, t := range pTypes {
		// all events -> subscriber should return a nil slice
		if t == types.BusEventType_BUS_EVENT_TYPE_ALL {
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
		return (me.MarketID() == mID)
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
		return (mpe.MarketID() == mID && mpe.PartyID() == pID)
	}
}

func (t Type) ToProto() types.BusEventType {
	pt, ok := toProto[t]
	if !ok {
		panic(fmt.Sprintf("Converting events.Type %s to proto BusEventType: no corresponding value found", t))
	}
	return pt
}
