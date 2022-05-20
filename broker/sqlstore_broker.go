package broker

import (
	"context"
	"encoding/hex"
	"fmt"
	"reflect"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

type TimeUpdateEvent interface {
	events.Event
	Time() time.Time
}

type SqlBrokerSubscriber interface {
	SetVegaTime(vegaTime time.Time)
	Flush(ctx context.Context) error
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

type BlockStore interface {
	Add(ctx context.Context, b entities.Block) error
}

// sqlStoreBroker : push events to each subscriber with a single go routine across all types
type sqlStoreBroker struct {
	config             Config
	log                *logging.Logger
	subscribers        []SqlBrokerSubscriber
	typeToSubs         map[events.Type][]SqlBrokerSubscriber
	eventSource        eventSource
	transactionManager TransactionManager
	blockStore         BlockStore
	chainInfo          ChainInfoI
	nextBlockTime      TimeUpdateEvent
	lastBlock          *entities.Block
}

func NewSqlStoreBroker(log *logging.Logger, config Config, chainInfo ChainInfoI,
	eventsource eventSource, transactionManager TransactionManager, blockStore BlockStore, subs ...SqlBrokerSubscriber,
) *sqlStoreBroker {
	b := &sqlStoreBroker{
		config:             config,
		log:                log,
		subscribers:        subs,
		typeToSubs:         map[events.Type][]SqlBrokerSubscriber{},
		eventSource:        eventsource,
		transactionManager: transactionManager,
		blockStore:         blockStore,
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

	if b.nextBlockTime != nil {
		b.addBlock(blockCtx, b.nextBlockTime)
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
			if e.Type() == events.TimeUpdate {
				return b.handleBlockEnd(blockCtx, e.(TimeUpdateEvent))
			} else {
				err = b.handleEvent(blockCtx, e)
				if err != nil {
					return err
				}
			}

		}
	}
}

func (b *sqlStoreBroker) handleBlockEnd(blockCtx context.Context, timeUpdate TimeUpdateEvent) error {
	metrics.EventCounterInc(timeUpdate.Type().String())
	metrics.BlockCounterInc()
	for _, subscriber := range b.subscribers {
		subscriber.Flush(blockCtx)
	}

	err := b.transactionManager.Commit(blockCtx)
	if err != nil {
		return fmt.Errorf("failed to commit transactional context:%w", err)
	}

	b.nextBlockTime = timeUpdate
	for _, subscriber := range b.subscribers {
		subscriber.SetVegaTime(b.nextBlockTime.Time())
	}

	return nil
}

func (b *sqlStoreBroker) addBlock(ctx context.Context, te TimeUpdateEvent) error {
	hash, err := hex.DecodeString(te.TraceID())
	if err != nil {
		b.log.Panic("Trace ID is not valid hex string",
			logging.String("traceId", te.TraceID()))
	}

	// Postgres only stores timestamps in microsecond resolution
	block := entities.Block{
		VegaTime: te.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   te.BlockNr(),
	}

	// At startup we get time updates that have the same time to microsecond precision which causes
	// a primary key restraint failure, this code is to handle this scenario
	if b.lastBlock == nil || !block.VegaTime.Equal(b.lastBlock.VegaTime) {
		b.lastBlock = &block
		err = b.blockStore.Add(ctx, block)
		if err != nil {
			return errors.Wrap(err, "failed to add block")
		}
	}

	return nil
}

func (b *sqlStoreBroker) handleEvent(ctx context.Context, e events.Event) error {
	metrics.EventCounterInc(e.Type().String())

	if err := checkChainID(b.chainInfo, e.ChainID()); err != nil {
		return err
	}

	if subs, ok := b.typeToSubs[e.Type()]; ok {
		for _, sub := range subs {
			if err := b.push(ctx, sub, e); err != nil {
				return err
			}
		}
	}

	return nil
}

func (b *sqlStoreBroker) push(ctx context.Context, sub SqlBrokerSubscriber, e events.Event) error {
	subName := reflect.TypeOf(sub).Elem().Name()
	timer := metrics.NewTimeCounter("sql", subName, e.Type().String())
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
