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

func (b *sqlStoreBroker) Receive(ctx context.Context) error {
	if err := b.eventSource.Listen(); err != nil {
		return err
	}

	receiveCh, errCh := b.eventSource.Receive(ctx)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := b.receiveBlock(ctx, receiveCh, errCh); err != nil {
				return err
			}
		}
	}
}

func (b *sqlStoreBroker) receiveBlock(ctx context.Context, receiveCh <-chan events.Event, errCh <-chan error) error {
	// Don't use our parent context for as a parent of the database operation; if we get cancelled
	// by e.g. a shutdown request then let the last database operation complete.
	blockCtx, cancel := context.WithTimeout(context.Background(), b.config.BlockProcessingTimeout.Duration)
	defer cancel()

	blockCtx, err := b.transactionManager.WithTransaction(blockCtx)
	if err != nil {
		return fmt.Errorf("failed to add transaction to context:%w", err)
	}

	for {
		// Do a pre-check on ctx.Done() since select() cases are randomized, this reduces
		// the number of things we'll keep trying to handle after we are cancelled.
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-receiveCh:
			nextBlock, err := b.handleEvent(blockCtx, e)
			if err != nil {
				return err
			}
			if nextBlock {
				err := b.transactionManager.Commit(blockCtx)
				if err != nil {
					return fmt.Errorf("failed to commit transactional context:%w", err)
				}
				return nil
			}
		}
	}
}

func (b *sqlStoreBroker) handleEvent(ctx context.Context, e events.Event) (bool, error) {
	if err := checkChainID(b.chainInfo, e.ChainID()); err != nil {
		return false, err
	}

	metrics.EventCounterInc(e.Type().String())

	// If the event is a time event send it to all subscribers, this indicates a new block start
	if e.Type() == events.TimeUpdate {
		metrics.BlockCounterInc()
		for _, subs := range b.typeToSubs {
			for _, sub := range subs {
				if err := b.push(ctx, sub, e); err != nil {
					return false, err
				}
			}
		}
		return true, nil
	} else {
		if subs, ok := b.typeToSubs[e.Type()]; ok {
			for _, sub := range subs {
				if err := b.push(ctx, sub, e); err != nil {
					return false, err
				}
			}
		}
	}
	return false, nil
}

func (b *sqlStoreBroker) push(ctx context.Context, sub SqlBrokerSubscriber, e events.Event) error {
	sub_name := reflect.TypeOf(sub).Elem().Name()
	timer := metrics.NewTimeCounter("sql", sub_name, e.Type().String())
	err := sub.Push(ctx, e)
	timer.EventTimeCounterAdd()

	if err != nil {
		errMsg := fmt.Sprintf("failed to process event %v error:%+v", e.StreamMessage(), err)
		b.log.Error(errMsg)
		if b.config.PanicOnError {
			return err
		}
	}

	return nil
}
