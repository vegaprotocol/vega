// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package broker_test

import (
	"context"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/core/types"
	"code.vegaprotocol.io/vega/datanode/broker"
	"code.vegaprotocol.io/vega/datanode/entities"
	"code.vegaprotocol.io/vega/datanode/networkhistory"
	"code.vegaprotocol.io/vega/datanode/service"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/logging"
	eventsv1 "code.vegaprotocol.io/vega/protos/vega/events/v1"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var logger = logging.NewTestLogger()

func TestBrokerShutsDownOnErrorFromErrorChannelWhenInRecovery(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{s1})

	beSource := newBlockEventSource()
	blockEvent1 := beSource.NextBeginBlockEvent()
	// make sure we move forward by ending the block, but discard as we're not going to use it
	_ = beSource.NextEndBlockEvent()
	blockEvent2 := beSource.NextBeginBlockEvent()
	// make sure we move forward by ending the block, but discard as we're not going to use it
	_ = beSource.NextEndBlockEvent()

	block1, _ := entities.BlockFromBeginBlock(blockEvent1)
	block2, _ := entities.BlockFromBeginBlock(blockEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	require.NoError(t, blockStore.Add(context.Background(), *block1))
	require.NoError(t, blockStore.Add(context.Background(), *block2))

	closedChan := make(chan bool)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	tes.eventsCh <- blockEvent1

	tes.errorsCh <- errors.New("Test error")
}

func TestBrokerShutsDownOnErrorFromErrorChannel(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{s1})

	closedChan := make(chan bool)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	beSource := newBlockEventSource()
	tes.eventsCh <- beSource.NextBeginBlockEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	tes.errorsCh <- errors.New("Test error")

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownOnErrorWhenInRecovery(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{s1})

	beSource := newBlockEventSource()
	blockEvent1 := beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()
	blockEvent2 := beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()
	_ = beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()
	blockEvent4 := beSource.NextBeginBlockEvent()

	block1, _ := entities.BlockFromBeginBlock(blockEvent1)
	block2, _ := entities.BlockFromBeginBlock(blockEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	closedChan := make(chan bool)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	tes.eventsCh <- blockEvent4

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownOnError(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)
	errorSubscriber := &errorTestSQLBrokerSubscriber{s1}

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{errorSubscriber})

	closedChan := make(chan bool)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	beSource := newBlockEventSource()
	tes.eventsCh <- beSource.NextBeginBlockEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)
	tes.eventsCh <- beSource.NextEndBlockEvent()

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownWhenContextCancelledWhenInRecovery(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)

	beSource := newBlockEventSource()
	blockEvent1 := beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()
	blockEvent2 := beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()

	block1, _ := entities.BlockFromBeginBlock(blockEvent1)
	block2, _ := entities.BlockFromBeginBlock(blockEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	tes, sb := createTestBroker(newTestTransactionManager(), blockStore, []broker.SQLBrokerSubscriber{s1})

	ctx, cancel := context.WithCancel(context.Background())

	closedChan := make(chan bool)
	go func() {
		sb.Receive(ctx)
		closedChan <- true
	}()

	tes.eventsCh <- blockEvent1

	cancel()

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownWhenContextCancelled(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{s1})
	ctx, cancel := context.WithCancel(context.Background())

	closedChan := make(chan bool)
	go func() {
		sb.Receive(ctx)
		closedChan <- true
	}()

	beSource := newBlockEventSource()
	tes.eventsCh <- beSource.NextBeginBlockEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	cancel()

	assert.Equal(t, true, <-closedChan)
}

func TestAnyEventsSentAheadOfFirstTimeEventAreIgnored(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{s1})
	go sb.Receive(context.Background())

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	beSource := newBlockEventSource()
	tes.eventsCh <- beSource.NextBeginBlockEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"}), <-s1.receivedCh)
}

func TestBlocksSentBeforeStartedAtBlockAreIgnored(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)

	beSource := newBlockEventSource()
	blockBeginEvent1 := beSource.NextBeginBlockEvent()
	blockEndEvent1 := beSource.NextEndBlockEvent()
	blockBeginEvent2 := beSource.NextBeginBlockEvent()
	blockEndEvent2 := beSource.NextEndBlockEvent()
	blockBeginEvent3 := beSource.NextBeginBlockEvent()

	block1, _ := entities.BlockFromBeginBlock(blockBeginEvent1)
	block2, _ := entities.BlockFromBeginBlock(blockBeginEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	tes, sb := createTestBroker(newTestTransactionManager(), blockStore, []broker.SQLBrokerSubscriber{s1})
	go sb.Receive(context.Background())

	tes.eventsCh <- blockBeginEvent1
	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})
	tes.eventsCh <- blockEndEvent1
	tes.eventsCh <- blockBeginEvent2
	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"})
	tes.eventsCh <- blockEndEvent2
	tes.eventsCh <- blockBeginEvent3
	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a3"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a3"}), <-s1.receivedCh)
}

func TestTimeUpdateWithTooHighHeightCauseFailure(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)

	beSource := newBlockEventSource()
	blockEvent1 := beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()
	blockEvent2 := beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()
	_ = beSource.NextBeginBlockEvent()
	_ = beSource.NextEndBlockEvent()
	blockEvent4 := beSource.NextBeginBlockEvent()

	block1, _ := entities.BlockFromBeginBlock(blockEvent1)
	block2, _ := entities.BlockFromBeginBlock(blockEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	tes, sb := createTestBroker(newTestTransactionManager(), blockStore, []broker.SQLBrokerSubscriber{s1})

	errCh := make(chan error)
	go func() {
		err := sb.Receive(context.Background())
		errCh <- err
	}()

	tes.eventsCh <- blockEvent4

	assert.NotNil(t, <-errCh)
}

func TestSqlBrokerSubscriberCallbacks(t *testing.T) {
	s1 := testSQLBrokerSubscriber{
		eventType:  events.AssetEvent,
		receivedCh: make(chan events.Event, 1),
		vegaTimeCh: make(chan time.Time),
		flushCh:    make(chan bool),
	}

	transactionManager := newTestTransactionManager()
	transactionManager.withTransactionCalls = make(chan bool)
	transactionManager.withConnectionCalls = make(chan bool, 1)
	transactionManager.commitCall = make(chan bool)

	blockStore := newTestBlockStore()

	tes, sb := createTestBroker(transactionManager, blockStore, []broker.SQLBrokerSubscriber{&s1})

	go sb.Receive(context.Background())

	beSource := newBlockEventSource()

	// BlockEnd event should cause a flush of subscribers, followed by commit and then an update to subscribers vegatime,
	// followed by initiating a new transaction and adding a block for the new time
	beginEvent := beSource.NextBeginBlockEvent()
	endEvent := beSource.NextEndBlockEvent()
	tes.eventsCh <- beginEvent

	assert.Equal(t, time.Unix(0, beginEvent.BeginBlock().Timestamp), <-s1.vegaTimeCh)
	assert.Equal(t, true, <-transactionManager.withTransactionCalls)
	assert.Equal(t, true, <-transactionManager.withConnectionCalls)

	hash, _ := hex.DecodeString(beginEvent.TraceID())
	expectedBlock := entities.Block{
		VegaTime: time.Unix(0, beginEvent.BeginBlock().Timestamp).Truncate(time.Microsecond),
		Hash:     hash,
		Height:   beginEvent.BlockNr(),
	}

	assert.Equal(t, expectedBlock, <-blockStore.blocks)

	tes.eventsCh <- endEvent
	assert.Equal(t, true, <-s1.flushCh)
	assert.Equal(t, true, <-transactionManager.commitCall)

	beginEvent = beSource.NextBeginBlockEvent()
	endEvent = beSource.NextEndBlockEvent()
	tes.eventsCh <- beginEvent

	assert.Equal(t, time.Unix(0, beginEvent.BeginBlock().Timestamp).Truncate(time.Microsecond), <-s1.vegaTimeCh)
	assert.Equal(t, true, <-transactionManager.withTransactionCalls)

	hash, _ = hex.DecodeString(beginEvent.TraceID())
	expectedBlock = entities.Block{
		VegaTime: time.Unix(0, beginEvent.BeginBlock().Timestamp).Truncate(time.Microsecond),
		Hash:     hash,
		Height:   beginEvent.BlockNr(),
	}

	assert.Equal(t, expectedBlock, <-blockStore.blocks)

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	tes.eventsCh <- endEvent

	assert.Equal(t, true, <-s1.flushCh)
	assert.Equal(t, true, <-transactionManager.commitCall)

	beginEvent = beSource.NextBeginBlockEvent()
	tes.eventsCh <- beginEvent

	assert.Equal(t, time.Unix(0, beginEvent.BeginBlock().Timestamp).Truncate(time.Microsecond), <-s1.vegaTimeCh)
	assert.Equal(t, true, <-transactionManager.withTransactionCalls)

	hash, _ = hex.DecodeString(beginEvent.TraceID())
	expectedBlock = entities.Block{
		VegaTime: time.Unix(0, beginEvent.BeginBlock().Timestamp).Truncate(time.Microsecond),
		Hash:     hash,
		Height:   beginEvent.BlockNr(),
	}

	assert.Equal(t, expectedBlock, <-blockStore.blocks)
}

func TestSqlBrokerEventDistribution(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)
	s2 := newTestSQLBrokerSubscriber(events.AssetEvent)
	s3 := newTestSQLBrokerSubscriber(events.AccountEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{s1, s2, s3})
	go sb.Receive(context.Background())

	beSource := newBlockEventSource()
	tes.eventsCh <- beSource.NextBeginBlockEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s2.receivedCh)

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"}), <-s1.receivedCh)
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"}), <-s2.receivedCh)

	tes.eventsCh <- events.NewAccountEvent(context.Background(), types.Account{ID: "acc1"})

	assert.Equal(t, events.NewAccountEvent(context.Background(), types.Account{ID: "acc1"}), <-s3.receivedCh)
}

func TestSqlBrokerTimeEventSentToAllSubscribers(t *testing.T) {
	s1 := newTestSQLBrokerSubscriber(events.AssetEvent)
	s2 := newTestSQLBrokerSubscriber(events.AssetEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{s1, s2})

	go sb.Receive(context.Background())

	beSource := newBlockEventSource()
	blockEvent := beSource.NextBeginBlockEvent()
	tes.eventsCh <- blockEvent

	assert.Equal(t, time.Unix(0, blockEvent.BeginBlock().Timestamp).
		Truncate(time.Microsecond), <-s1.vegaTimeCh)
	assert.Equal(t, time.Unix(0, blockEvent.BeginBlock().Timestamp).
		Truncate(time.Microsecond), <-s2.vegaTimeCh)
}

func TestSqlBrokerUpgradeBlock(t *testing.T) {
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), []broker.SQLBrokerSubscriber{})

	errCh := make(chan error)
	go func() {
		err := sb.Receive(context.Background())
		errCh <- err
	}()

	beSource := newBlockEventSource()

	// send through a full block
	blockEvent := beSource.NextBeginBlockEvent()
	tes.eventsCh <- blockEvent
	tes.eventsCh <- beSource.NextEndBlockEvent()

	// everything gets committed, now we start a new block
	blockEvent = beSource.NextBeginBlockEvent()
	tes.eventsCh <- blockEvent
	assert.False(t, tes.protocolUpgradeSvc.GetProtocolUpgradeStarted())

	// now protocol upgrade event comes through
	tes.eventsCh <- events.NewProtocolUpgradeStarted(context.Background(), eventsv1.ProtocolUpgradeStarted{
		LastBlockHeight: blockEvent.BeginBlock().Height,
	})
	assert.Nil(t, <-errCh)
	assert.True(t, tes.protocolUpgradeSvc.GetProtocolUpgradeStarted())
}

func createTestBroker(transactionManager broker.TransactionManager, blockStore broker.BlockStore, subs []broker.SQLBrokerSubscriber) (*testEventSource, broker.SQLStoreEventBroker) {
	conf := broker.NewDefaultConfig()
	log := logging.NewTestLogger()
	tes := &testEventSource{
		eventsCh:           make(chan events.Event),
		errorsCh:           make(chan error, 1),
		protocolUpgradeSvc: service.NewProtocolUpgrade(nil, log),
	}

	blockCommitedFunc := func(context.Context, string, int64, bool) {}

	protocolUpgradeHandler := networkhistory.NewProtocolUpgradeHandler(log,
		tes.protocolUpgradeSvc, tes, func(ctx context.Context, chainID string, toHeight int64) error {
			return nil
		})

	sb := broker.NewSQLStoreBroker(logger, conf, "", tes, transactionManager, blockStore, blockCommitedFunc, protocolUpgradeHandler, subs)

	return tes, sb
}

type testBlockStore struct {
	blocks    chan entities.Block
	blockLock sync.Mutex
	lastBlock *entities.Block
}

func newTestBlockStore() *testBlockStore {
	return &testBlockStore{
		blocks: make(chan entities.Block, 100),
	}
}

func (t *testBlockStore) Add(ctx context.Context, b entities.Block) error {
	t.blocks <- b
	t.blockLock.Lock()
	defer t.blockLock.Unlock()
	t.lastBlock = &b

	return nil
}

func (t *testBlockStore) GetLastBlock(ctx context.Context) (entities.Block, error) {
	t.blockLock.Lock()
	defer t.blockLock.Unlock()

	if t.lastBlock == nil {
		return entities.Block{}, entities.ErrNotFound
	}

	return *t.lastBlock, nil
}

type testTransactionManager struct {
	withTransactionCalls chan bool
	withConnectionCalls  chan bool
	commitCall           chan bool
}

func newTestTransactionManager() *testTransactionManager {
	return &testTransactionManager{
		withTransactionCalls: make(chan bool, 100),
		withConnectionCalls:  make(chan bool, 100),
		commitCall:           make(chan bool, 100),
	}
}

func (t *testTransactionManager) WithTransaction(ctx context.Context) (context.Context, error) {
	t.withTransactionCalls <- true
	return ctx, nil
}

func (t *testTransactionManager) WithConnection(ctx context.Context) (context.Context, error) {
	t.withConnectionCalls <- true
	return ctx, nil
}

func (t *testTransactionManager) Commit(ctx context.Context) error {
	t.commitCall <- true
	return nil
}

func (t *testTransactionManager) Rollback(ctx context.Context) error {
	return nil
}

func (t *testTransactionManager) RefreshMaterializedViews(_ context.Context) error {
	return nil
}

type errorTestSQLBrokerSubscriber struct {
	*testSQLBrokerSubscriber
}

func (e *errorTestSQLBrokerSubscriber) Flush(ctx context.Context) error {
	return errors.New("its broken")
}

type testSQLBrokerSubscriber struct {
	eventType  events.Type
	receivedCh chan events.Event
	flushCh    chan bool
	vegaTimeCh chan time.Time
}

func newTestSQLBrokerSubscriber(eventType events.Type) *testSQLBrokerSubscriber {
	return &testSQLBrokerSubscriber{
		eventType:  eventType,
		receivedCh: make(chan events.Event, 100),
		flushCh:    make(chan bool, 100),
		vegaTimeCh: make(chan time.Time, 100),
	}
}

func (t *testSQLBrokerSubscriber) SetVegaTime(vegaTime time.Time) {
	t.vegaTimeCh <- vegaTime
}

func (t *testSQLBrokerSubscriber) Flush(ctx context.Context) error {
	t.flushCh <- true
	return nil
}

func (t *testSQLBrokerSubscriber) Push(ctx context.Context, evt events.Event) error {
	t.receivedCh <- evt
	return nil
}

func (t *testSQLBrokerSubscriber) Types() []events.Type {
	return []events.Type{t.eventType}
}

type blockEventSource struct {
	vegaTime    time.Time
	blockHeight uint64
}

func newBlockEventSource() *blockEventSource {
	return &blockEventSource{
		vegaTime:    time.Now().Truncate(time.Millisecond),
		blockHeight: 1,
	}
}

func (s *blockEventSource) NextBeginBlockEvent() *events.BeginBlock {
	ctx := vgcontext.WithTraceID(context.Background(), "DEADBEEF")
	ctx = vgcontext.WithBlockHeight(ctx, s.blockHeight)

	event := events.NewBeginBlock(ctx, eventsv1.BeginBlock{
		Height:    s.blockHeight,
		Timestamp: s.vegaTime.UnixNano(),
	})

	return event
}

func (s *blockEventSource) NextEndBlockEvent() *events.EndBlock {
	ctx := vgcontext.WithTraceID(context.Background(), "DEADBEEF")
	ctx = vgcontext.WithBlockHeight(ctx, s.blockHeight)

	event := events.NewEndBlock(ctx, eventsv1.EndBlock{
		Height: s.blockHeight,
	})

	s.vegaTime = s.vegaTime.Add(1 * time.Second)
	s.blockHeight++

	return event
}
