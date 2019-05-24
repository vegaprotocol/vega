package orders_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/orders"
	"code.vegaprotocol.io/vega/internal/orders/mocks"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	orderSubmission = types.OrderSubmission{
		Id:       "order_id",
		MarketID: "market_id",
		PartyID:  "party",
		Price:    10000,
		Size:     1,
		Side:     types.Side(1),
		Type:     types.Order_GTT,
	}
)

type testService struct {
	ctrl       *gomock.Controller
	orderStore *mocks.MockOrderStore
	timeSvc    *mocks.MockTimeService
	block      *mocks.MockBlockchain
	svc        *orders.Svc
}

type orderMatcher struct {
	e types.Order
}

func TestCreateOrder(t *testing.T) {
	t.Run("Create order - successful", testOrderSuccess)
	t.Run("Create order - expired", testOrderExpired)
	t.Run("Create order - blockchain error", testOrderBlockchainError)
}

func testOrderSuccess(t *testing.T) {
	// now
	now := vegatime.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	pre := &types.PendingOrder{
		Reference: "order_reference",
	}
	order := orderSubmission
	order.ExpiresAt = expires.UnixNano()
	matcher := orderMatcher{
		e: types.Order{
			Id:        order.Id,
			MarketID:  order.MarketID,
			PartyID:   order.PartyID,
			Price:     order.Price,
			Size:      order.Size,
			Side:      order.Side,
			Type:      order.Type,
			ExpiresAt: expires.UnixNano(),
		},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	svc.block.EXPECT().CreateOrder(gomock.Any(), matcher).Times(1).Return(pre, nil)
	pendingOrder, err := svc.svc.CreateOrder(context.Background(), &order)
	assert.NotNil(t, pendingOrder)
	assert.NoError(t, err)
	assert.Equal(t, pre.Reference, pendingOrder.Reference)
}

func testOrderExpired(t *testing.T) {
	// now
	now := vegatime.Now()
	//expired 2 hours ago
	// expires := now.Add(time.Hour * -2)
	order := orderSubmission
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	pendingOrder, err := svc.svc.CreateOrder(context.Background(), &order)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
}

func testOrderBlockchainError(t *testing.T) {
	// now
	now := vegatime.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	bcErr := errors.New("blockchain error")
	order := orderSubmission
	order.ExpiresAt = expires.UnixNano()
	matcher := orderMatcher{
		e: types.Order{
			Id:        order.Id,
			MarketID:  order.MarketID,
			PartyID:   order.PartyID,
			Price:     order.Price,
			Size:      order.Size,
			Side:      order.Side,
			Type:      order.Type,
			ExpiresAt: expires.UnixNano(),
		},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	svc.block.EXPECT().CreateOrder(gomock.Any(), matcher).Times(1).Return(nil, bcErr)
	pendingOrder, err := svc.svc.CreateOrder(context.Background(), &order)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
	assert.Equal(t, bcErr, err)
}

func getTestService(t *testing.T) *testService {
	log := logging.NewTestLogger()
	ctrl := gomock.NewController(t)
	orderStore := mocks.NewMockOrderStore(ctrl)
	timeSvc := mocks.NewMockTimeService(ctrl)
	block := mocks.NewMockBlockchain(ctrl)
	conf := orders.NewDefaultConfig()
	svc, err := orders.NewService(log, conf, orderStore, timeSvc, block)
	if err != nil {
		t.Fatalf("Failed to get test service: %+v", err)
	}
	return &testService{
		ctrl:       ctrl,
		orderStore: orderStore,
		timeSvc:    timeSvc,
		block:      block,
		svc:        svc,
	}
}

func (m orderMatcher) String() string {
	return fmt.Sprintf("%#v", m.e)
}

func (m orderMatcher) Matches(x interface{}) bool {
	var v types.Order
	switch val := x.(type) {
	case *types.Order:
		v = *val
	case types.Order:
		v = val
	default:
		return false
	}
	if m.e.Id != v.Id && m.e.MarketID != v.MarketID {
		return false
	}
	if m.e.PartyID != v.PartyID {
		return false
	}

	return (m.e.ExpiresAt == v.ExpiresAt)
}
