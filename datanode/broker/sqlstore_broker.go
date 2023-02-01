// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"github.com/pkg/errors"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
)

type SQLBrokerSubscriber interface {
	SetVegaTime(vegaTime time.Time)
	Flush(ctx context.Context) error
	Push(ctx context.Context, val events.Event) error
	Types() []events.Type
}

type SQLStoreEventBroker interface {
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

type ProtocolUpgradeHandler interface {
	OnProtocolUpgradeEvent(ctx context.Context, chainID string, lastCommittedBlockHeight int64) error
	GetProtocolUpgradeStarted() bool
}

const (
	slowTimeUpdateThreshold = 2 * time.Second
)

// SQLStoreBroker : push events to each subscriber with a single go routine across all types.
type SQLStoreBroker struct {
	config                       Config
	log                          *logging.Logger
	subscribers                  []SQLBrokerSubscriber
	typeToSubs                   map[events.Type][]SQLBrokerSubscriber
	eventSource                  EventReceiver
	transactionManager           TransactionManager
	blockStore                   BlockStore
	onBlockCommitted             func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool)
	protocolUpdateHandler        ProtocolUpgradeHandler
	chainID                      string
	lastBlock                    *entities.Block
	slowTimeUpdateTicker         *time.Ticker
	receivedProtocolUpgradeEvent bool
	snapshotTaken                bool
}

func NewSQLStoreBroker(
	log *logging.Logger,
	config Config,
	chainID string,
	eventsource EventReceiver,
	transactionManager TransactionManager,
	blockStore BlockStore,
	onBlockCommitted func(ctx context.Context, chainId string, lastCommittedBlockHeight int64, snapshotTaken bool),
	protocolUpdateHandler ProtocolUpgradeHandler,
	subs []SQLBrokerSubscriber,
) *SQLStoreBroker {
	b := &SQLStoreBroker{
		config:                config,
		log:                   log,
		subscribers:           subs,
		typeToSubs:            map[events.Type][]SQLBrokerSubscriber{},
		eventSource:           eventsource,
		transactionManager:    transactionManager,
		blockStore:            blockStore,
		chainID:               chainID,
		onBlockCommitted:      onBlockCommitted,
		protocolUpdateHandler: protocolUpdateHandler,
		slowTimeUpdateTicker:  time.NewTicker(slowTimeUpdateThreshold),
	}

	for _, s := range subs {
		for _, evtType := range s.Types() {
			b.typeToSubs[evtType] = append(b.typeToSubs[evtType], s)
		}
	}

	return b
}

func (b *SQLStoreBroker) Receive(ctx context.Context) error {
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

		b.onBlockCommitted(ctx, b.chainID, b.lastBlock.Height, b.snapshotTaken)

		if b.receivedProtocolUpgradeEvent {
			return b.protocolUpdateHandler.OnProtocolUpgradeEvent(ctx, b.chainID, b.lastBlock.Height)
		}
	}
}

// waitForFirstBlock processes all events until a new block is encountered and returns the new block. A 'new' block is one for which
// events have not already been processed by this datanode.
func (b *SQLStoreBroker) waitForFirstBlock(ctx context.Context, errCh <-chan error, receiveCh <-chan events.Event) (*entities.Block, error) {
	lastProcessedBlock, err := b.blockStore.GetLastBlock(ctx)

	if err == nil {
		b.log.Infof("waiting for first unprocessed block, last processed block height: %d", lastProcessedBlock.Height)
	} else if errors.Is(err, entities.ErrNotFound) {
		lastProcessedBlock = entities.Block{
			VegaTime: time.Time{},
			// TODO: This is making the assumption that the first block will be height 1. This is not necessarily true.
			//       The node can start at any time given to Tendermint through the genesis file.
			Height: 0,
			Hash:   nil,
		}
	} else {
		return nil, err
	}

	var beginBlock entities.BeginBlockEvent
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case err = <-errCh:
			return nil, err
		case e := <-receiveCh:
			if e.Type() == events.BeginBlockEvent {
				beginBlock = e.(entities.BeginBlockEvent)
				metrics.EventCounterInc(beginBlock.Type().String())

				if beginBlock.BlockNr() > lastProcessedBlock.Height+1 {
					return nil, fmt.Errorf("block height on begin block, %d, is too high, the height of the last processed block is %d",
						beginBlock.BlockNr(), lastProcessedBlock.Height)
				}

				if beginBlock.BlockNr() > lastProcessedBlock.Height {
					b.log.Info("first unprocessed block received, starting block processing")
					return entities.BlockFromBeginBlock(beginBlock)
				}
			}
		}
	}
}

// processBlock processes all events in the current block up to the next time update.  The next time block is returned when processing of the block is done.
func (b *SQLStoreBroker) processBlock(ctx context.Context, dbContext context.Context, block *entities.Block, eventsCh <-chan events.Event, errCh <-chan error) (*entities.Block, error) {
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
	blockCtx, cancel = context.WithCancel(dbContext)
	defer cancel()

	blockCtx, err := b.transactionManager.WithTransaction(blockCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to add transaction to context:%w", err)
	}

	if err = b.addBlock(blockCtx, block); err != nil {
		return nil, fmt.Errorf("failed to add block:%w", err)
	}

	defer b.slowTimeUpdateTicker.Stop()
	b.snapshotTaken = false
	betweenBlocks := false
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
		case <-b.slowTimeUpdateTicker.C:
			b.log.Warningf("slow time update detected, time between checks %v, block height: %d, total block processing time: %v", slowTimeUpdateThreshold,
				block.Height, blockTimer.duration)
		case e := <-eventsCh:
			if b.protocolUpdateHandler.GetProtocolUpgradeStarted() {
				return nil, errors.New("received event after protocol upgrade started")
			}

			if b.config.Level.Level == logging.DebugLevel {
				b.log.Debug("received event", logging.String("type", e.Type().String()), logging.String("trace-id", e.TraceID()))
			}
			metrics.EventCounterInc(e.Type().String())
			blockTimer.startTimer()

			switch e.Type() {
			case events.EndBlockEvent:
				err = b.flushAllSubscribers(blockCtx)
				if err != nil {
					return nil, err
				}

				err = b.transactionManager.Commit(blockCtx)
				if err != nil {
					return nil, fmt.Errorf("failed to commit transactional context:%w", err)
				}
				b.slowTimeUpdateTicker.Reset(slowTimeUpdateThreshold)
				betweenBlocks = true
			case events.BeginBlockEvent:
				beginBlock := e.(entities.BeginBlockEvent)
				return entities.BlockFromBeginBlock(beginBlock)
			case events.CoreSnapshotEvent:
				// if a snapshot is taken on a protocol upgrade block, we want it to be taken synchronously as part of handling of protocol upgrade
				b.snapshotTaken = !e.StreamMessage().GetCoreSnapshotEvent().ProtocolUpgradeBlock
				if err = b.handleEvent(blockCtx, e); err != nil {
					return nil, err
				}
			case events.ProtocolUpgradeStartedEvent:
				b.receivedProtocolUpgradeEvent = true
				// we've received a protocol upgrade event which is the last event core will have sent out
				// so we can leave now
				return nil, nil
			default:
				if betweenBlocks {
					// we should only be receiving a BeginBlockEvent immediately after an EndBlockEvent
					panic(fmt.Errorf("received event %s between end block and begin block", e.Type().String()))
				}
				if err = b.handleEvent(blockCtx, e); err != nil {
					return nil, err
				}
			}
		}
	}
}

func (b *SQLStoreBroker) flushAllSubscribers(blockCtx context.Context) error {
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

func (b *SQLStoreBroker) addBlock(ctx context.Context, block *entities.Block) error {
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

func (b *SQLStoreBroker) handleEvent(ctx context.Context, e events.Event) error {
	if err := checkChainID(b.chainID, e.ChainID()); err != nil {
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

func (b *SQLStoreBroker) push(ctx context.Context, sub SQLBrokerSubscriber, e events.Event) error {
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
		t.duration = t.duration + time.Since(*t.startTime)
		t.startTime = nil
	}
}
