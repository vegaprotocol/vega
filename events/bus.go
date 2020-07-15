package events

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/contextutil"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrUnsuportedEvent = errors.New("unknown payload for event")
)

type Type int

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
)

var (
	marketEvents = []Type{
		PositionResolution,
		MarketCreatedEvent,
		MarketTickEvent,
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
