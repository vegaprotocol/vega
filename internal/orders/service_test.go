package orders

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/orders/newmocks"
	"code.vegaprotocol.io/vega/internal/vegatime"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/golang/mock/gomock"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	orderSubmission = types.OrderSubmission{
		Id:       "order_id",
		MarketId: "market_id",
		Party:    "party",
		Price:    10000,
		Size:     1,
		Side:     types.Side(1),
		Type:     types.Order_GTT,
	}
)

type testService struct {
	ctrl       *gomock.Controller
	orderStore *newmocks.MockOrderStore
	timeSvc    *newmocks.MockTimeService
	block      *newmocks.MockBlockchain
	conf       *Config
	svc        Service
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
	now := time.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	orderRef := "order_reference"
	order := orderSubmission
	order.ExpirationDatetime = expires.Format(time.RFC3339)
	matcher := orderMatcher{
		e: types.Order{
			Id:                  order.Id,
			Market:              order.MarketId,
			Party:               order.Party,
			Price:               order.Price,
			Size:                order.Size,
			Side:                order.Side,
			Type:                order.Type,
			ExpirationDatetime:  order.ExpirationDatetime,
			ExpirationTimestamp: uint64(time.Duration(expires.Unix()) * time.Second),
		},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(vegatime.Stamp(now.UnixNano()), now, nil)
	svc.block.EXPECT().CreateOrder(gomock.Any(), matcher).Times(1).Return(true, orderRef, nil)
	success, reference, err := svc.svc.CreateOrder(context.Background(), &order)
	assert.True(t, success)
	assert.NoError(t, err)
	assert.Equal(t, orderRef, reference)
}

func testOrderExpired(t *testing.T) {
	// now
	now := time.Now()
	//expired 2 hours ago
	expires := now.Add(time.Hour * -2)
	order := orderSubmission
	order.ExpirationDatetime = expires.Format(time.RFC3339)
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(vegatime.Stamp(now.UnixNano()), now, nil)
	success, reference, err := svc.svc.CreateOrder(context.Background(), &order)
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, "", reference)
}

func testOrderBlockchainError(t *testing.T) {
	// now
	now := time.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	bcErr := errors.New("blockchain error")
	order := orderSubmission
	order.ExpirationDatetime = expires.Format(time.RFC3339)
	matcher := orderMatcher{
		e: types.Order{
			Id:                  order.Id,
			Market:              order.MarketId,
			Party:               order.Party,
			Price:               order.Price,
			Size:                order.Size,
			Side:                order.Side,
			Type:                order.Type,
			ExpirationDatetime:  order.ExpirationDatetime,
			ExpirationTimestamp: uint64(time.Duration(expires.Unix()) * time.Second),
		},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(vegatime.Stamp(now.UnixNano()), now, nil)
	svc.block.EXPECT().CreateOrder(gomock.Any(), matcher).Times(1).Return(false, "", bcErr)
	success, reference, err := svc.svc.CreateOrder(context.Background(), &order)
	assert.False(t, success)
	assert.Error(t, err)
	assert.Equal(t, bcErr, err)
	assert.Equal(t, "", reference)
}

func getTestService(t *testing.T) *testService {
	log := logging.NewLoggerFromEnv("dev")
	ctrl := gomock.NewController(t)
	orderStore := newmocks.NewMockOrderStore(ctrl)
	timeSvc := newmocks.NewMockTimeService(ctrl)
	block := newmocks.NewMockBlockchain(ctrl)
	conf := NewDefaultConfig(log)
	svc, err := NewOrderService(conf, orderStore, timeSvc, block)
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
	if m.e.Id != v.Id && m.e.Market != v.Market {
		return false
	}
	if m.e.Party != v.Party {
		return false
	}
	if m.e.ExpirationDatetime != v.ExpirationDatetime {
		return false
	}
	return (m.e.ExpirationTimestamp == v.ExpirationTimestamp)
}
