package broker_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/data-node/broker"
	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/vega/events"
	"code.vegaprotocol.io/vega/types"

	"github.com/stretchr/testify/assert"
)

var logger = logging.NewTestLogger()

func TestEventDistribution(t *testing.T) {
	tes, sb := createTestBroker(t)

	s1 := testSqlBrokerSubscriber{types: []events.Type{events.AssetEvent}, receivedCh: make(chan events.Event)}
	s2 := testSqlBrokerSubscriber{types: []events.Type{events.AssetEvent}, receivedCh: make(chan events.Event)}
	s3 := testSqlBrokerSubscriber{types: []events.Type{events.AccountEvent}, receivedCh: make(chan events.Event)}
	sb.SubscribeBatch(s1, s2, s3)
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

func TestSubscriptionAfterBrokerStartPanics(t *testing.T) {
	defer func() {
		r := recover()
		if r == nil {
			t.Fatalf("expected to panic")
		}
	}()

	tes, sb := createTestBroker(t)

	s1 := testSqlBrokerSubscriber{types: []events.Type{events.AssetEvent}, receivedCh: make(chan events.Event)}
	sb.SubscribeBatch(s1)
	go sb.Receive(context.Background())

	tes.eventsCh <- events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"})
	assert.Equal(t, events.NewAssetEvent(context.Background(), types.Asset{ID: "a1"}), <-s1.receivedCh)

	s2 := testSqlBrokerSubscriber{types: []events.Type{events.AssetEvent}, receivedCh: make(chan events.Event)}
	sb.SubscribeBatch(s2)
}

func createTestBroker(t *testing.T) (*testEventSource, *broker.SqlStoreBroker) {
	conf := broker.NewDefaultConfig()
	testChainInfo := testChainInfo{chainId: ""}
	tes := &testEventSource{
		eventsCh: make(chan events.Event),
		errorsCh: make(chan error, 1),
	}

	sb, err := broker.NewSqlStoreBroker(logger, conf, testChainInfo, tes, 0)
	if err != nil {
		t.Fatalf("failed to create broker:%s", err)
	}
	return tes, sb
}

type testSqlBrokerSubscriber struct {
	types      []events.Type
	receivedCh chan events.Event
}

func (t testSqlBrokerSubscriber) Push(evt events.Event) {
	t.receivedCh <- evt
}

func (t testSqlBrokerSubscriber) Types() []events.Type {
	return t.types
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
