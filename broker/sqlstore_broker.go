package broker

import (
	"context"
	"fmt"
	"reflect"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/vega/events"
)

type SqlBrokerSubscriber interface {
	Push(ctx context.Context, val events.Event) error
	Types() []events.Type
}

type SqlStoreEventBroker interface {
	Receive(ctx context.Context) error
}

type TransactionManager interface {
	WithTransaction(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
}

// sqlStoreBroker : push events to each subscriber with a single go routine across all types
type sqlStoreBroker struct {
	config             Config
	log                *logging.Logger
	typeToSubs         map[events.Type][]SqlBrokerSubscriber
	eventSource        eventSource
	transactionManager TransactionManager
	chainInfo          ChainInfoI
}

func NewSqlStoreBroker(log *logging.Logger, config Config, chainInfo ChainInfoI,
	eventsource eventSource, transactionManager TransactionManager, subs ...SqlBrokerSubscriber,
) *sqlStoreBroker {
	b := &sqlStoreBroker{
		config:             config,
		log:                log,
		typeToSubs:         map[events.Type][]SqlBrokerSubscriber{},
		eventSource:        eventsource,
		transactionManager: transactionManager,
		chainInfo:          chainInfo,
	}

	for _, s := range subs {
		for _, evtType := range s.Types() {
			b.typeToSubs[evtType] = append(b.typeToSubs[evtType], s)
		}
	}
	return b
}

func (b *sqlStoreBroker) Receive(receiveContext context.Context) error {
	if err := b.eventSource.Listen(); err != nil {
		return err
	}

	receiveCh, errCh := b.eventSource.Receive(receiveContext)

	var blockContext context.Context
	var blockContextCancelFn context.CancelFunc

	for e := range receiveCh {
		if err := checkChainID(b.chainInfo, e.ChainID()); err != nil {
			return err
		}
		metrics.EventCounterInc(e.Type().String())

		if blockContext == nil {
			var err error
			blockContext, blockContextCancelFn = context.WithTimeout(receiveContext, b.config.BlockProcessingTimeout.Duration)
			defer blockContextCancelFn()
			blockContext, err = b.transactionManager.WithTransaction(blockContext)
			if err != nil {
				return fmt.Errorf("failed to add transaction to context:%w", err)
			}
		}

		// If the event is a time event send it to all subscribers, this indicates a new block start
		if e.Type() == events.TimeUpdate {
			metrics.BlockCounterInc()
			for _, subs := range b.typeToSubs {
				for _, sub := range subs {
					b.push(blockContext, sub, e)
				}
			}

			err := b.transactionManager.Commit(blockContext)
			if err != nil {
				return fmt.Errorf("failed to commit transactional context:%w", err)
			}
			blockContextCancelFn()
			blockContext = nil

		} else {
			if subs, ok := b.typeToSubs[e.Type()]; ok {
				for _, sub := range subs {
					b.push(blockContext, sub, e)
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

func (b *sqlStoreBroker) push(ctx context.Context, sub SqlBrokerSubscriber, e events.Event) {
	sub_name := reflect.TypeOf(sub).Elem().Name()
	timer := metrics.NewTimeCounter("sql", sub_name, e.Type().String())
	err := sub.Push(ctx, e)
	if err != nil {
		b.OnPushEventError(e, err)
	}
	timer.EventTimeCounterAdd()
}

func (b *sqlStoreBroker) OnPushEventError(evt events.Event, err error) {
	errMsg := fmt.Sprintf("failed to process event %v error:%+v", evt.StreamMessage(), err)
	if b.config.PanicOnError {
		b.log.Panic(errMsg)
	} else {
		b.log.Error(errMsg)
	}
}
