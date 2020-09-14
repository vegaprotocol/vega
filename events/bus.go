package events

import (
	"context"
	"fmt"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	types "code.vegaprotocol.io/vega/proto"

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
	PartyID() string
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
	seq     int
	et      Type
}

type Event interface {
	Type() Type
	Context() context.Context
	TraceID() string
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
	AssetEvent
	MarketTickEvent
	AuctionEvent
	WithdrawalEvent
	DepositEvent
)

var (
	marketEvents = []Type{
		PositionResolution,
		MarketCreatedEvent,
		MarketTickEvent,
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
		types.BusEventType_BUS_EVENT_TYPE_ASSET:               AssetEvent,
		types.BusEventType_BUS_EVENT_TYPE_MARKET_TICK:         MarketTickEvent,
		types.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL:          WithdrawalEvent,
		types.BusEventType_BUS_EVENT_TYPE_DEPOSIT:             DepositEvent,
	}

	toProto = map[Type]types.BusEventType{
		TimeUpdate:             types.BusEventType_BUS_EVENT_TYPE_TIME_UPDATE,
		TransferResponses:      types.BusEventType_BUS_EVENT_TYPE_TRANSFER_RESPONSES,
		PositionResolution:     types.BusEventType_BUS_EVENT_TYPE_POSITION_RESOLUTION,
		MarketEvent:            types.BusEventType_BUS_EVENT_TYPE_MARKET,
		OrderEvent:             types.BusEventType_BUS_EVENT_TYPE_ORDER,
		AccountEvent:           types.BusEventType_BUS_EVENT_TYPE_ACCOUNT,
		PartyEvent:             types.BusEventType_BUS_EVENT_TYPE_PARTY,
		TradeEvent:             types.BusEventType_BUS_EVENT_TYPE_TRADE,
		MarginLevelsEvent:      types.BusEventType_BUS_EVENT_TYPE_MARGIN_LEVELS,
		ProposalEvent:          types.BusEventType_BUS_EVENT_TYPE_PROPOSAL,
		VoteEvent:              types.BusEventType_BUS_EVENT_TYPE_VOTE,
		MarketDataEvent:        types.BusEventType_BUS_EVENT_TYPE_MARKET_DATA,
		NodeSignatureEvent:     types.BusEventType_BUS_EVENT_TYPE_NODE_SIGNATURE,
		LossSocializationEvent: types.BusEventType_BUS_EVENT_TYPE_LOSS_SOCIALIZATION,
		SettlePositionEvent:    types.BusEventType_BUS_EVENT_TYPE_SETTLE_POSITION,
		SettleDistressedEvent:  types.BusEventType_BUS_EVENT_TYPE_SETTLE_DISTRESSED,
		MarketCreatedEvent:     types.BusEventType_BUS_EVENT_TYPE_MARKET_CREATED,
		AssetEvent:             types.BusEventType_BUS_EVENT_TYPE_ASSET,
		MarketTickEvent:        types.BusEventType_BUS_EVENT_TYPE_MARKET_TICK,
		WithdrawalEvent:        types.BusEventType_BUS_EVENT_TYPE_WITHDRAWAL,
		DepositEvent:           types.BusEventType_BUS_EVENT_TYPE_DEPOSIT,
	}

	eventStrings = map[Type]string{
		All:                    "ALL",
		TimeUpdate:             "TimeUpdate",
		TransferResponses:      "TransferResponses",
		PositionResolution:     "PositionResolution",
		MarketEvent:            "MarketEvent",
		OrderEvent:             "OrderEvent",
		AccountEvent:           "AccountEvent",
		PartyEvent:             "PartyEvent",
		TradeEvent:             "TradeEvent",
		MarginLevelsEvent:      "MarginLevelsEvent",
		ProposalEvent:          "ProposalEvent",
		VoteEvent:              "VoteEvent",
		MarketDataEvent:        "MarketDataEvent",
		NodeSignatureEvent:     "NodeSignatureEvent",
		LossSocializationEvent: "LossSocializationEvent",
		SettlePositionEvent:    "SettlePositionEvent",
		SettleDistressedEvent:  "SettleDistressedEvent",
		MarketCreatedEvent:     "MarketCreatedEvent",
		AssetEvent:             "AssetEvent",
		MarketTickEvent:        "MarketTickEvent",
		AuctionEvent:           "AuctionEvent",
		WithdrawalEvent:        "WithdrawalEvent",
		DepositEvent:           "DepositEvent",
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
	}
	return nil, ErrUnsuportedEvent
}

// A base event holds no data, so the constructor will not be called directly
func newBase(ctx context.Context, t Type) *Base {
	ctx, tID := contextutil.TraceIDFromContext(ctx)
	return &Base{
		ctx:     ctx,
		traceID: tID,
		et:      t,
	}
}

// TraceID returns the... traceID obviously
func (b Base) TraceID() string {
	return b.traceID
}

// Sequence returns event sequence number
func (b Base) Sequence() int {
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
		return (pe.PartyID() == pID)
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
