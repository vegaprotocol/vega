package broker

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
)

type SqlBrokerSubscriber interface {
	Push(val events.Event)
	Type() events.Type
}

// SqlStoreBroker : push events to each subscriber with a go-routine per type
type SqlStoreBroker struct {
	startedGuard         sync.Mutex
	typeToSubs           map[events.Type][]SqlBrokerSubscriber
	typeToEvtCh          map[events.Type]chan events.Event
	eventSource          eventSource
	chainInfo            ChainInfoI
	startedSendingEvents bool
	eventTypeBufferSize  int
}

func NewSqlStoreBroker(log *logging.Logger, config Config, chainInfo ChainInfoI,
	eventsource eventSource, eventTypeBufferSize int) (*SqlStoreBroker, error) {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	b := &SqlStoreBroker{
		typeToSubs:          map[events.Type][]SqlBrokerSubscriber{},
		typeToEvtCh:         map[events.Type]chan events.Event{},
		eventSource:         eventsource,
		chainInfo:           chainInfo,
		eventTypeBufferSize: eventTypeBufferSize,
	}

	return b, nil
}

func (b *SqlStoreBroker) Receive(ctx context.Context) error {
	b.startedGuard.Lock()
	defer b.startedGuard.Unlock()
	b.startedSendingEvents = true

	if err := b.eventSource.Listen(); err != nil {
		return err
	}

	receiveCh, errCh := b.eventSource.Receive(ctx)
	b.startSendingEvents(ctx)

	for e := range receiveCh {
		if err := checkChainID(b.chainInfo, e.ChainID()); err != nil {
			return err
		}

		// If the event is a time event send it to all type channels, this indicates a new block start (for now)
		if e.Type() == events.TimeUpdate {
			for _, ch := range b.typeToEvtCh {
				ch <- e
			}
		} else {
			if ch, ok := b.typeToEvtCh[e.Type()]; ok {
				ch <- e
			}
		}
	}

	select {
	case err := <-errCh:
		return err
	default:
		return nil
	}
}

func (b *SqlStoreBroker) SubscribeBatch(subs ...SqlBrokerSubscriber) {
	b.startedGuard.Lock()
	defer b.startedGuard.Unlock()
	if b.startedSendingEvents {
		panic("too late to subscribe, events are already being sent")
	}

	for _, s := range subs {
		b.subscribe(s)
	}
}

func (b *SqlStoreBroker) subscribe(s SqlBrokerSubscriber) {
	if _, exists := b.typeToEvtCh[s.Type()]; !exists {
		ch := make(chan events.Event, b.eventTypeBufferSize)
		b.typeToEvtCh[s.Type()] = ch
	}

	b.typeToSubs[s.Type()] = append(b.typeToSubs[s.Type()], s)
}

func (b *SqlStoreBroker) startSendingEvents(ctx context.Context) {
	for t, ch := range b.typeToEvtCh {
		go func(ch chan events.Event, subs []SqlBrokerSubscriber) {
			for {
				select {
				case <-ctx.Done():
					return
				case evt := <-ch:
					if evt.Type() == events.TimeUpdate {
						time := evt.(*events.Time)
						for _, sub := range subs {
							sub.Push(time)
						}
					} else {
						for _, sub := range subs {
							select {
							case <-ctx.Done():
								return
							default:
								sub.Push(evt)
							}
						}
					}
				}
			}
		}(ch, b.typeToSubs[t])
	}
}
