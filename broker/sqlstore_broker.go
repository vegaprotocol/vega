// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

package broker

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/vega/events"
	"github.com/pkg/errors"
)

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
	WithConnection(ctx context.Context) (context.Context, error)
	WithTransaction(ctx context.Context) (context.Context, error)
	Commit(ctx context.Context) error
}

type BlockStore interface {
	Add(ctx context.Context, b entities.Block) error
	GetLastBlock(ctx context.Context) (entities.Block, error)
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

	nextBlock, err := b.waitForFirstBlock(ctx, errCh, receiveCh)
	if err != nil {
		return err
	}

	dbContext, err := b.transactionManager.WithConnection(context.Background())
	if err != nil {
		return err
	}

	for {
		if nextBlock, err = b.processBlock(ctx, dbContext, nextBlock, receiveCh, errCh); err != nil {
			return err
		}
	}

}

// waitForFirstBlock processes all events until a new block is encountered and returns the new block. A 'new' block is one for which
// events have not already been processed by this datanode.
func (b *sqlStoreBroker) waitForFirstBlock(ctx context.Context, errCh <-chan error, receiveCh <-chan events.Event) (*entities.Block, error) {

	lastProcessedBlock, err := b.blockStore.GetLastBlock(ctx)

	if err == nil {
		b.log.Infof("waiting for first unprocessed block, last processed block: %v", lastProcessedBlock)
	} else if errors.Is(err, sqlstore.ErrNoLastBlock) {
		lastProcessedBlock = entities.Block{
			VegaTime: time.Time{},
			Height:   -1,
			Hash:     nil,
		}
	} else {
		return nil, err
	}

	var timeUpdate entities.TimeUpdateEvent
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err = <-errCh:
			return nil, err
		case e := <-receiveCh:
			if e.Type() == events.TimeUpdate {
				timeUpdate = e.(entities.TimeUpdateEvent)
				metrics.EventCounterInc(timeUpdate.Type().String())

				if timeUpdate.BlockNr() > lastProcessedBlock.Height+1 {
					return nil, fmt.Errorf("block height on time update, %d, is too high, the height of the last processed block is %d",
						timeUpdate.BlockNr(), lastProcessedBlock.Height)
				}

				if timeUpdate.BlockNr() > lastProcessedBlock.Height {
					b.log.Info("first unprocessed block received, starting block processing")
					return entities.BlockFromTimeUpdate(timeUpdate)
				}
			}
		}
	}

}

// processBlock processes all events in the current block up to the next time update.  The next time block is returned when processing of the block is done.
func (b *sqlStoreBroker) processBlock(ctx context.Context, dbContext context.Context, block *entities.Block, eventsCh <-chan events.Event, errCh <-chan error) (*entities.Block, error) {
	metrics.BlockCounterInc()
	metrics.SetBlockHeight(float64(block.Height))

	blockTimer := blockTimer{}
	blockTimer.startTimer()
	defer func() {
		blockTimer.stopTimer()
		metrics.AddBlockHandlingTime(blockTimer.duration)
	}()

	for _, subscriber := range b.subscribers {
		subscriber.SetVegaTime(block.VegaTime)
	}

	// Don't use our parent context as a parent of the database operation; if we get cancelled
	// by e.g. a shutdown request then let the last database operation complete.
	var blockCtx context.Context
	var cancel context.CancelFunc
	blockCtx, cancel = context.WithTimeout(dbContext, b.config.BlockProcessingTimeout.Duration)
	defer cancel()

	blockCtx, err := b.transactionManager.WithTransaction(blockCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to add transaction to context:%w", err)
	}

	if err = b.addBlock(blockCtx, block); err != nil {
		return nil, fmt.Errorf("failed to add block:%w", err)
	}

	slowTimeUpdateThreshold := 2 * time.Second
	slowTimeUpdateTicker := time.NewTicker(slowTimeUpdateThreshold)
	defer slowTimeUpdateTicker.Stop()

	for {
		// Do a pre-check on ctx.Done() since select() cases are randomized, this reduces
		// the number of things we'll keep trying to handle after we are cancelled.
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		blockTimer.stopTimer()
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err = <-errCh:
			return nil, err
		case <-slowTimeUpdateTicker.C:
			b.log.Warningf("slow time update detected, time between checks %v, block height: %d, total block processing time: %v", slowTimeUpdateThreshold,
				block.Height, blockTimer.duration)
		case e := <-eventsCh:
			metrics.EventCounterInc(e.Type().String())
			blockTimer.startTimer()
			if e.Type() == events.TimeUpdate {

				timeUpdate := e.(entities.TimeUpdateEvent)

				err = b.flushAllSubscribers(blockCtx)
				if err != nil {
					return nil, err
				}

				err = b.transactionManager.Commit(blockCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to commit transactional context:%w", err)
				}

				return entities.BlockFromTimeUpdate(timeUpdate)
			} else {
				if err = b.handleEvent(blockCtx, e); err != nil {
					return nil, err
				}
			}
		}
	}
}

func (b *sqlStoreBroker) flushAllSubscribers(blockCtx context.Context) error {
	for _, subscriber := range b.subscribers {
		subName := reflect.TypeOf(subscriber).Elem().Name()
		timer := metrics.NewTimeCounter(subName)
		err := subscriber.Flush(blockCtx)
		timer.FlushTimeCounterAdd()
		if err != nil {
			return fmt.Errorf("failed to flush subscriber:%w", err)
		}
	}
	return nil
}

func (b *sqlStoreBroker) addBlock(ctx context.Context, block *entities.Block) error {
	// At startup we get time updates that have the same time to microsecond precision which causes
	// a primary key restraint failure, this code is to handle this scenario
	if b.lastBlock == nil || !block.VegaTime.Equal(b.lastBlock.VegaTime) {
		b.lastBlock = block
		err := b.blockStore.Add(ctx, *block)
		if err != nil {
			return errors.Wrap(err, "failed to add block")
		}
	}

	return nil
}

func (b *sqlStoreBroker) handleEvent(ctx context.Context, e events.Event) error {

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

type blockTimer struct {
	duration  time.Duration
	startTime *time.Time
}

func (t *blockTimer) startTimer() {
	now := time.Now()
	t.startTime = &now
}

func (t *blockTimer) stopTimer() {
	if t.startTime != nil {
		t.duration = t.duration + time.Now().Sub(*t.startTime)
		t.startTime = nil
	}
}
