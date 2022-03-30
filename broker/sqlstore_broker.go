package broker

import (
	"context"
	"fmt"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
)

type SqlBrokerSubscriber interface {
	Push(val events.Event) error
	Types() []events.Type
}

type SqlStoreEventBroker interface {
	Receive(ctx context.Context) error
}

func NewSqlStoreBroker(log *logging.Logger, config Config, chainInfo ChainInfoI,
	eventsource eventSource, eventTypeBufferSize int, subs ...SqlBrokerSubscriber,
) SqlStoreEventBroker {
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())
	if config.UseSequentialSqlStoreBroker {
		return newSequentialSqlStoreBroker(log, chainInfo, eventsource, eventTypeBufferSize, subs, bool(config.PanicOnError))
	} else {
		return newConcurrentSqlStoreBroker(log, chainInfo, eventsource, eventTypeBufferSize, subs, bool(config.PanicOnError))
	}
}

// concurrentSqlStoreBroker : push events to each subscriber with a go-routine per type
type concurrentSqlStoreBroker struct {
	log                 *logging.Logger
	typeToSubs          map[events.Type][]SqlBrokerSubscriber
	typeToEvtCh         map[events.Type]chan events.Event
	eventSource         eventSource
	chainInfo           ChainInfoI
	eventTypeBufferSize int
	panicOnError        bool
}

func newConcurrentSqlStoreBroker(log *logging.Logger, chainInfo ChainInfoI, eventsource eventSource, eventTypeBufferSize int,
	subs []SqlBrokerSubscriber, panicOnError bool,
) *concurrentSqlStoreBroker {
	b := &concurrentSqlStoreBroker{
		log:                 log,
		typeToSubs:          map[events.Type][]SqlBrokerSubscriber{},
		typeToEvtCh:         map[events.Type]chan events.Event{},
		eventSource:         eventsource,
		chainInfo:           chainInfo,
		eventTypeBufferSize: eventTypeBufferSize,
		panicOnError:        panicOnError,
	}

	for _, s := range subs {
		b.subscribe(s)
	}
	return b
}

func (b *concurrentSqlStoreBroker) Receive(ctx context.Context) error {
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

func (b *concurrentSqlStoreBroker) subscribe(s SqlBrokerSubscriber) {
	for _, evtType := range s.Types() {
		if _, exists := b.typeToEvtCh[evtType]; !exists {
			ch := make(chan events.Event, b.eventTypeBufferSize)
			b.typeToEvtCh[evtType] = ch
		}

		b.typeToSubs[evtType] = append(b.typeToSubs[evtType], s)
	}
}

func (b *concurrentSqlStoreBroker) startSendingEvents(ctx context.Context) {
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
							err := sub.Push(time)
							if err != nil {
								b.OnPushEventError(time, err)
							}
						}
					} else {
						for _, sub := range subs {
							select {
							case <-ctx.Done():
								return
							default:
								err := sub.Push(evt)
								if err != nil {
									b.OnPushEventError(evt, err)
								}
							}
						}
					}
				}
			}
		}(ch, b.typeToSubs[t])
	}
}

func (b *concurrentSqlStoreBroker) OnPushEventError(evt events.Event, err error) {
	errMsg := fmt.Sprintf("failed to process event %v error:%+v", evt.StreamMessage(), err)
	if b.panicOnError {
		b.log.Panic(errMsg)
	} else {
		b.log.Error(errMsg)
	}

}

// sequentialSqlStoreBroker : push events to each subscriber with a single go routine across all types
type sequentialSqlStoreBroker struct {
	log                 *logging.Logger
	typeToSubs          map[events.Type][]SqlBrokerSubscriber
	eventSource         eventSource
	chainInfo           ChainInfoI
	eventTypeBufferSize int
	panicOnError        bool
}

func newSequentialSqlStoreBroker(log *logging.Logger, chainInfo ChainInfoI,
	eventsource eventSource, eventTypeBufferSize int, subs []SqlBrokerSubscriber,
	panicOnError bool,
) *sequentialSqlStoreBroker {
	b := &sequentialSqlStoreBroker{
		log:                 log,
		typeToSubs:          map[events.Type][]SqlBrokerSubscriber{},
		eventSource:         eventsource,
		chainInfo:           chainInfo,
		eventTypeBufferSize: eventTypeBufferSize,
		panicOnError:        panicOnError,
	}

	for _, s := range subs {
		for _, evtType := range s.Types() {
			b.typeToSubs[evtType] = append(b.typeToSubs[evtType], s)
		}
	}
	return b
}

func (b *sequentialSqlStoreBroker) Receive(ctx context.Context) error {
	if err := b.eventSource.Listen(); err != nil {
		return err
	}

	receiveCh, errCh := b.eventSource.Receive(ctx)

	for e := range receiveCh {
		if err := checkChainID(b.chainInfo, e.ChainID()); err != nil {
			return err
		}

		// If the event is a time event send it to all subscribers, this indicates a new block start (for now)
		if e.Type() == events.TimeUpdate {
			for _, subs := range b.typeToSubs {
				for _, sub := range subs {
					err := sub.Push(e)
					if err != nil {
						b.OnPushEventError(e, err)
					}
				}
			}
		} else {
			if subs, ok := b.typeToSubs[e.Type()]; ok {
				for _, sub := range subs {
					err := sub.Push(e)
					if err != nil {
						b.OnPushEventError(e, err)
					}
				}
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

func (b *sequentialSqlStoreBroker) OnPushEventError(evt events.Event, err error) {
	errMsg := fmt.Sprintf("failed to process event %v error:%+v", evt.StreamMessage(), err)
	if b.panicOnError {
		b.log.Panic(errMsg)
	} else {
		b.log.Error(errMsg)
	}

}
