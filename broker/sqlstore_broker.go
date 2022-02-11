package broker

import (
	"context"
	"sync"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
)

type SqlBrokerSubscriber interface {
	Push(val events.Event)
	Types() []events.Type
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
	if err := b.eventSource.Listen(); err != nil {
		return err
	}

	receiveCh, errCh := b.eventSource.Receive(ctx)
	b.startSendingEvents(ctx)

	for e := range receiveCh {
		if err := checkChainID(b.chainInfo, e.ChainID()); err != nil {
			return err
		}

		if ch, ok := b.typeToEvtCh[e.Type()]; ok {
			ch <- e
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
	for _, t := range s.Types() {
		if _, exists := b.typeToEvtCh[t]; !exists {
			ch := make(chan events.Event, b.eventTypeBufferSize)
			b.typeToEvtCh[t] = ch
		}
	}

	types := s.Types()
	for _, t := range types {
		b.typeToSubs[t] = append(b.typeToSubs[t], s)
	}
}

func (b *SqlStoreBroker) startSendingEvents(ctx context.Context) {
	b.startedGuard.Lock()
	defer b.startedGuard.Unlock()
	b.startedSendingEvents = true

	for _, ch := range b.typeToEvtCh {
		go func(ch chan events.Event) {
			for {
				select {
				case <-ctx.Done():
					return
				case evt := <-ch:
					subs, _ := b.typeToSubs[evt.Type()]
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
		}(ch)
	}
}
