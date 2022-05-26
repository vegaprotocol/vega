package broker_test

import (
	"context"
	"encoding/hex"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/entities"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
	vgcontext "code.vegaprotocol.io/vega/libs/context"
	"code.vegaprotocol.io/vega/types"

	"github.com/stretchr/testify/assert"
)

var logger = logging.NewTestLogger()

func TestSqlBrokerSubscriberCallbacks(t *testing.T) {
	s1 := testSqlBrokerSubscriber{eventType: events.AssetEvent, receivedCh: make(chan events.Event, 1),
		vegaTimeCh: make(chan time.Time, 0), flushCh: make(chan bool, 0)}

	transactionManager := newTestTransactionManager()
	transactionManager.withTransactionCalls = make(chan bool, 0)
	transactionManager.commitCall = make(chan bool, 0)

	blockStore := newTestBlockStore()
	blockStore.blocks = make(chan entities.Block, 0)

	tes, sb := createTestBroker(t, transactionManager, blockStore, &s1)

	go sb.Receive(context.Background())

	now := time.Now()

	assert.Equal(t, true, <-transactionManager.withTransactionCalls)

	// Time event should cause a flush of subscribers, followed by commit and then an update to subscribers vegatime,
	// followed by initiating a new transaction and adding a block for the new time
	timeEvent := events.NewTime(vgcontext.WithTraceID(context.Background(), "DEADBEEF"), now)
	timeEvent.TraceID()
	tes.eventsCh <- timeEvent

	assert.Equal(t, true, <-s1.flushCh)
	assert.Equal(t, true, <-transactionManager.commitCall)

	hash, _ := hex.DecodeString(timeEvent.TraceID())

	expectedBlock := entities.Block{
		VegaTime: timeEvent.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   timeEvent.BlockNr(),
	}

	assert.Equal(t, now, <-s1.vegaTimeCh)

	assert.Equal(t, true, <-transactionManager.withTransactionCalls)
	assert.Equal(t, expectedBlock, <-blockStore.blocks)

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	now2 := time.Now()

	timeEvent2 := events.NewTime(vgcontext.WithTraceID(context.Background(), "DEADBEEF"), now2)
	timeEvent2.TraceID()
	hash, _ = hex.DecodeString(timeEvent2.TraceID())
	expectedBlock = entities.Block{
		VegaTime: timeEvent2.Time().Truncate(time.Microsecond),
		Hash:     hash,
		Height:   timeEvent2.BlockNr(),
	}

	tes.eventsCh <- timeEvent2

	assert.Equal(t, true, <-s1.flushCh)
	assert.Equal(t, true, <-transactionManager.commitCall)
	assert.Equal(t, now2, <-s1.vegaTimeCh)
	assert.Equal(t, true, <-transactionManager.withTransactionCalls)
	assert.Equal(t, expectedBlock, <-blockStore.blocks)

}

func TestSqlBrokerEventDistribution(t *testing.T) {
	s1 := newTestSqlBrokerSubscriber(events.AssetEvent)
	s2 := newTestSqlBrokerSubscriber(events.AssetEvent)
	s3 := newTestSqlBrokerSubscriber(events.AccountEvent)
	tes, sb := createTestBroker(t, newTestTransactionManager(), newTestBlockStore(), s1, s2, s3)
	go sb.Receive(context.Background())

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
	tes, sb := createTestBroker(t, newTestTransactionManager(), newTestBlockStore(), s1, s2)

	go sb.Receive(context.Background())

	now := time.Now()

	timeEvent := events.NewTime(vgcontext.WithTraceID(context.Background(), "DEADBEEF"), now)

	timeEvent.TraceID()

	tes.eventsCh <- timeEvent

	assert.Equal(t, now, <-s1.vegaTimeCh)
	assert.Equal(t, now, <-s2.vegaTimeCh)
}

func createTestBroker(t *testing.T, transactionManager broker.TransactionManager, blockStore broker.BlockStore, subs ...broker.SqlBrokerSubscriber) (*testEventSource, broker.SqlStoreEventBroker) {
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
	blocks chan entities.Block
}

func newTestBlockStore() *testBlockStore {
	return &testBlockStore{
		blocks: make(chan entities.Block, 100),
	}
}

func (t *testBlockStore) Add(ctx context.Context, b entities.Block) error {
	t.blocks <- b
	return nil
}

type testTransactionManager struct {
	withTransactionCalls chan bool
	commitCall           chan bool
}

func newTestTransactionManager() *testTransactionManager {
	return &testTransactionManager{
		withTransactionCalls: make(chan bool, 100),
		commitCall:           make(chan bool, 100),
	}
}

func (t *testTransactionManager) WithTransaction(ctx context.Context) (context.Context, error) {
	t.withTransactionCalls <- true
	return ctx, nil
}

func (t *testTransactionManager) Commit(ctx context.Context) error {
	t.commitCall <- true
	return nil
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

func (t testSqlBrokerSubscriber) SetVegaTime(vegaTime time.Time) {
	t.vegaTimeCh <- vegaTime
}

func (t testSqlBrokerSubscriber) Flush(ctx context.Context) error {
	t.flushCh <- true
	return nil
}

func (t testSqlBrokerSubscriber) Push(ctx context.Context, evt events.Event) error {
	t.receivedCh <- evt
	return nil
}

func (t testSqlBrokerSubscriber) Types() []events.Type {
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
