package broker_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/data-node/config/encoding"

	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"

	"github.com/stretchr/testify/assert"
)

var logger = logging.NewTestLogger()

func TestSqlBrokerEventDistribution(t *testing.T) {
	testSqlBrokerEventDistribution(t, false)
	testSqlBrokerEventDistribution(t, true)
}

func testSqlBrokerEventDistribution(t *testing.T, sequential bool) {
	s1 := testSqlBrokerSubscriber{eventType: events.AssetEvent, receivedCh: make(chan events.Event)}
	s2 := testSqlBrokerSubscriber{eventType: events.AssetEvent, receivedCh: make(chan events.Event)}
	s3 := testSqlBrokerSubscriber{eventType: events.AccountEvent, receivedCh: make(chan events.Event)}
	tes, sb := createTestBroker(t, sequential, &s1, &s2, &s3)
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
	testSqlBrokerTimeEventSentToAllSubscribers(t, false)
	testSqlBrokerTimeEventSentToAllSubscribers(t, true)
}

func testSqlBrokerTimeEventSentToAllSubscribers(t *testing.T, sequential bool) {
	s1 := testSqlBrokerSubscriber{eventType: events.AssetEvent, receivedCh: make(chan events.Event)}
	s2 := testSqlBrokerSubscriber{eventType: events.AssetEvent, receivedCh: make(chan events.Event)}
	tes, sb := createTestBroker(t, sequential, &s1, &s2)

	go sb.Receive(context.Background())

	timeEvent := events.NewTime(context.Background(), time.Now())

	tes.eventsCh <- timeEvent

	assert.Equal(t, timeEvent, <-s1.receivedCh)
	assert.Equal(t, timeEvent, <-s2.receivedCh)
}

func TestSqlBrokerTimeEventOnlySendOnceToTimeSubscribers(t *testing.T) {
	testNewSqlStoreBrokerestSqlBrokerTimeEventOnlySendOnceToTimeSubscribers(t, false)
	testNewSqlStoreBrokerestSqlBrokerTimeEventOnlySendOnceToTimeSubscribers(t, true)
}

func testNewSqlStoreBrokerestSqlBrokerTimeEventOnlySendOnceToTimeSubscribers(t *testing.T, seq bool) {
	s1 := testSqlBrokerSubscriber{eventType: events.TimeUpdate, receivedCh: make(chan events.Event)}
	tes, sb := createTestBroker(t, seq, &s1)

	go sb.Receive(context.Background())

	timeEvent := events.NewTime(context.Background(), time.Now())

	tes.eventsCh <- timeEvent

	assert.Equal(t, timeEvent, <-s1.receivedCh)
	assert.Equal(t, 0, len(s1.receivedCh))
}

func createTestBroker(t *testing.T, sequential bool, subs ...broker.SqlBrokerSubscriber) (*testEventSource, broker.SqlStoreEventBroker) {
	conf := broker.NewDefaultConfig()
	conf.UseSequentialSqlStoreBroker = encoding.Bool(sequential)
	testChainInfo := testChainInfo{chainId: ""}
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error, 1),
	}

	sb := broker.NewSqlStoreBroker(logger, conf, testChainInfo, tes, 0, subs...)

	return tes, sb
}

type testSqlBrokerSubscriber struct {
	eventType  events.Type
	receivedCh chan events.Event
}

func (t testSqlBrokerSubscriber) Push(evt events.Event) error {
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
