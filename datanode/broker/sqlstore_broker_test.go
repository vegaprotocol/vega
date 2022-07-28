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

package broker_test

import (
	"context"
	"encoding/hex"
	"sync"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/sqlstore"
	"code.vegaprotocol.io/vega/events"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/types"
	"github.com/pkg/errors"

	"github.com/stretchr/testify/assert"
)

var logger = logging.NewTestLogger()

func TestBrokerShutsDownOnErrorFromErrorChannelWhenInRecovery(t *testing.T) {

	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), s1)

	timeEventSource := newTimeEventSource()
	timeEvent1 := timeEventSource.NextTimeEvent()
	timeEvent2 := timeEventSource.NextTimeEvent()

	block1, _ := entities.BlockFromTimeUpdate(timeEvent1)
	block2, _ := entities.BlockFromTimeUpdate(timeEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	closedChan := make(chan bool, 0)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	tes.eventsCh <- timeEvent1

	tes.errorsCh <- errors.New("Test error")
}

func TestBrokerShutsDownOnErrorFromErrorChannel(t *testing.T) {

	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), s1)

	closedChan := make(chan bool, 0)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	timeEventSource := newTimeEventSource()
	tes.eventsCh <- timeEventSource.NextTimeEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	tes.errorsCh <- errors.New("Test error")

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownOnErrorWhenInRecovery(t *testing.T) {

	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), s1)

	timeEventSource := newTimeEventSource()
	timeEvent1 := timeEventSource.NextTimeEvent()
	timeEvent2 := timeEventSource.NextTimeEvent()
	timeEventSource.NextTimeEvent()
	timeEvent4 := timeEventSource.NextTimeEvent()

	block1, _ := entities.BlockFromTimeUpdate(timeEvent1)
	block2, _ := entities.BlockFromTimeUpdate(timeEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	closedChan := make(chan bool, 0)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	tes.eventsCh <- timeEvent4

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownOnError(t *testing.T) {

	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)
	errorSubscriber := &errorTestSqlBrokerSubscriber{s1}

	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), errorSubscriber)

	closedChan := make(chan bool, 0)
	go func() {
		err := sb.Receive(context.Background())
		assert.NotNil(t, err)
		closedChan <- true
	}()

	timeEventSource := newTimeEventSource()
	tes.eventsCh <- timeEventSource.NextTimeEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	tes.eventsCh <- timeEventSource.NextTimeEvent()

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownWhenContextCancelledWhenInRecovery(t *testing.T) {

	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)

	timeEventSource := newTimeEventSource()
	timeEvent1 := timeEventSource.NextTimeEvent()
	timeEvent2 := timeEventSource.NextTimeEvent()

	block1, _ := entities.BlockFromTimeUpdate(timeEvent1)
	block2, _ := entities.BlockFromTimeUpdate(timeEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	tes, sb := createTestBroker(newTestTransactionManager(), blockStore, s1)

	ctx, cancel := context.WithCancel(context.Background())

	closedChan := make(chan bool, 0)
	go func() {
		sb.Receive(ctx)
		closedChan <- true
	}()

	tes.eventsCh <- timeEvent1

	cancel()

	assert.Equal(t, true, <-closedChan)
}

func TestBrokerShutsDownWhenContextCancelled(t *testing.T) {

	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), s1)
	ctx, cancel := context.WithCancel(context.Background())

	closedChan := make(chan bool, 0)
	go func() {
		sb.Receive(ctx)
		closedChan <- true
	}()

	timeEventSource := newTimeEventSource()
	tes.eventsCh <- timeEventSource.NextTimeEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	cancel()

	assert.Equal(t, true, <-closedChan)
}

func TestAnyEventsSentAheadOfFirstTimeEventAreIgnored(t *testing.T) {
	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), s1)
	go sb.Receive(context.Background())

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	timeEventSource := newTimeEventSource()
	tes.eventsCh <- timeEventSource.NextTimeEvent()

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"}), <-s1.receivedCh)
}

func TestBlocksSentBeforeStartedAtBlockAreIgnored(t *testing.T) {
	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)

	timeEventSource := newTimeEventSource()
	timeEvent1 := timeEventSource.NextTimeEvent()
	timeEvent2 := timeEventSource.NextTimeEvent()
	timeEvent3 := timeEventSource.NextTimeEvent()

	block1, _ := entities.BlockFromTimeUpdate(timeEvent1)
	block2, _ := entities.BlockFromTimeUpdate(timeEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	tes, sb := createTestBroker(newTestTransactionManager(), blockStore, s1)
	go sb.Receive(context.Background())

	tes.eventsCh <- timeEvent1

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})

	tes.eventsCh <- timeEvent2

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a2"})

	tes.eventsCh <- timeEvent3

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a3"})

	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a3"}), <-s1.receivedCh)

}

func TestTimeUpdateWithTooHighHeightCauseFailure(t *testing.T) {

	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)

	timeEventSource := newTimeEventSource()
	timeEvent1 := timeEventSource.NextTimeEvent()
	timeEvent2 := timeEventSource.NextTimeEvent()
	timeEventSource.NextTimeEvent()
	timeEvent4 := timeEventSource.NextTimeEvent()

	block1, _ := entities.BlockFromTimeUpdate(timeEvent1)
	block2, _ := entities.BlockFromTimeUpdate(timeEvent2)

	blockStore := newTestBlockStore()

	// Set the block store to have already processed the first 2 blocks
	blockStore.Add(context.Background(), *block1)
	blockStore.Add(context.Background(), *block2)

	tes, sb := createTestBroker(newTestTransactionManager(), blockStore, s1)

	errCh := make(chan error, 0)
	go func() {
		err := sb.Receive(context.Background())
		errCh <- err
	}()

	tes.eventsCh <- timeEvent4

	assert.NotNil(t, <-errCh)

}

func TestSqlBrokerSubscriberCallbacks(t *testing.T) {
	s1 := testSqlBrokerSubscriber{eventType: events.AssetEvent, receivedCh: make(chan events.Event, 1),
		vegaTimeCh: make(chan time.Time, 0), flushCh: make(chan bool, 0)}

	transactionManager := newTestTransactionManager()
	transactionManager.withTransactionCalls = make(chan bool, 0)
	transactionManager.withConnectionCalls = make(chan bool, 1)
	transactionManager.commitCall = make(chan bool, 0)

	blockStore := newTestBlockStore()

	tes, sb := createTestBroker(transactionManager, blockStore, &s1)

	go sb.Receive(context.Background())

	timeEventSource := newTimeEventSource()

	// Time event should cause a flush of subscribers, followed by commit and then an update to subscribers vegatime,
	// followed by initiating a new transaction and adding a block for the new time
	timeEvent := timeEventSource.NextTimeEvent()
	tes.eventsCh <- timeEvent

	assert.Equal(t, timeEvent.Time(), <-s1.vegaTimeCh)
	assert.Equal(t, true, <-transactionManager.withTransactionCalls)
	assert.Equal(t, true, <-transactionManager.withConnectionCalls)

	hash, _ := hex.DecodeString(timeEvent.TraceID())
	expectedBlock := entities.Block{
		VegaTime: timeEvent.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   timeEvent.BlockNr(),
	}

	assert.Equal(t, expectedBlock, <-blockStore.blocks)

	timeEvent = timeEventSource.NextTimeEvent()
	tes.eventsCh <- timeEvent

	assert.Equal(t, true, <-s1.flushCh)
	assert.Equal(t, true, <-transactionManager.commitCall)

	assert.Equal(t, timeEvent.Time(), <-s1.vegaTimeCh)
	assert.Equal(t, true, <-transactionManager.withTransactionCalls)

	hash, _ = hex.DecodeString(timeEvent.TraceID())
	expectedBlock = entities.Block{
		VegaTime: timeEvent.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   timeEvent.BlockNr(),
	}

	assert.Equal(t, expectedBlock, <-blockStore.blocks)

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	timeEvent = timeEventSource.NextTimeEvent()
	tes.eventsCh <- timeEvent

	assert.Equal(t, true, <-s1.flushCh)
	assert.Equal(t, true, <-transactionManager.commitCall)

	assert.Equal(t, timeEvent.Time(), <-s1.vegaTimeCh)
	assert.Equal(t, true, <-transactionManager.withTransactionCalls)

	hash, _ = hex.DecodeString(timeEvent.TraceID())
	expectedBlock = entities.Block{
		VegaTime: timeEvent.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   timeEvent.BlockNr(),
	}

	assert.Equal(t, expectedBlock, <-blockStore.blocks)

}

func TestSqlBrokerEventDistribution(t *testing.T) {
	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)
	s2 := newTestSqlBrokerSubscriber(events.AssetEvent)
	s3 := newTestSqlBrokerSubscriber(events.AccountEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), s1, s2, s3)
	go sb.Receive(context.Background())

	timeEventSource := newTimeEventSource()
	tes.eventsCh <- timeEventSource.NextTimeEvent()

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
	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)
	s2 := newTestSqlBrokerSubscriber(events.AssetEvent)
	tes, sb := createTestBroker(newTestTransactionManager(), newTestBlockStore(), s1, s2)

	go sb.Receive(context.Background())

	timeEventSource := newTimeEventSource()
	timeEvent := timeEventSource.NextTimeEvent()
	tes.eventsCh <- timeEvent

	assert.Equal(t, timeEvent.Time(), <-s1.vegaTimeCh)
	assert.Equal(t, timeEvent.Time(), <-s2.vegaTimeCh)
}

func createTestBroker(transactionManager broker.TransactionManager, blockStore broker.BlockStore, subs ...broker.SqlBrokerSubscriber) (*testEventSource, broker.SqlStoreEventBroker) {
	conf := broker.NewDefaultConfig()
	testChainInfo := testChainInfo{chainId: ""}
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error, 1),
	}

	sb := broker.NewSqlStoreBroker(logger, conf, testChainInfo, tes, transactionManager, blockStore,
		subs...)

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
		return entities.Block{}, sqlstore.ErrNoLastBlock
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

type errorTestSqlBrokerSubscriber struct {
	*testSqlBrokerSubscriber
}

func (e *errorTestSqlBrokerSubscriber) Flush(ctx context.Context) error {
	return errors.New("its broken")
}

type testSqlBrokerSubscriber struct {
	eventType  events.Type
	receivedCh chan events.Event
	flushCh    chan bool
	vegaTimeCh chan time.Time
}

func newTestSqlBrokerSubscriber(eventType events.Type) *testSqlBrokerSubscriber {
	return &testSqlBrokerSubscriber{
		eventType:  eventType,
		receivedCh: make(chan events.Event, 100),
		flushCh:    make(chan bool, 100),
		vegaTimeCh: make(chan time.Time, 100),
	}
}

func (t *testSqlBrokerSubscriber) SetVegaTime(vegaTime time.Time) {
	t.vegaTimeCh <- vegaTime
}

func (t *testSqlBrokerSubscriber) Flush(ctx context.Context) error {
	t.flushCh <- true
	return nil
}

func (t *testSqlBrokerSubscriber) Push(ctx context.Context, evt events.Event) error {
	t.receivedCh <- evt
	return nil
}

func (t *testSqlBrokerSubscriber) Types() []events.Type {
	return []events.Type{t.eventType}
}

type testChainInfo struct {
	chainId string
}

func (t testChainInfo) SetChainID(s string) error {
	t.chainId = s
	return nil
}

func (t testChainInfo) GetChainID() (string, error) {
	return t.chainId, nil
}

type timeEventSource struct {
	vegaTime    time.Time
	blockHeight int64
}

func newTimeEventSource() *timeEventSource {
	return &timeEventSource{
		vegaTime: time.Now().Truncate(time.Millisecond),
	}
}

func (tes *timeEventSource) NextTimeEvent() *events.Time {
	ctx := vgcontext.WithTraceID(context.Background(), "DEADBEEF")
	ctx = vgcontext.WithBlockHeight(ctx, tes.blockHeight)

	event := events.NewTime(ctx, tes.vegaTime)
	tes.vegaTime = tes.vegaTime.Add(1 * time.Second)
	tes.blockHeight++
	return event
}
