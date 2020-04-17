package orders_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/orders/mocks"
	types "code.vegaprotocol.io/vega/proto"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/mock/gomock"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

var (
	orderSubmission = types.OrderSubmission{
		Id:          "order_id",
		MarketID:    "market_id",
		PartyID:     "party",
		Price:       10000,
		Size:        1,
		Side:        types.Side(1),
		TimeInForce: types.Order_GTT,
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

func TestPrepareOrder(t *testing.T) {
	t.Run("Create order with reference - successful", testPrepareOrderSuccess)
	t.Run("Create order without reference - successful", testPrepareOrderRefSuccess)
	t.Run("Create order - expired", testPrepareOrderExpired)
	t.Run("Prepare submit order with nil point", testPrepareSubmitOrderWithNilPointer)
	t.Run("Prepare cancel order with nil point", testPrepareCancelOrderWithNilPointer)
	t.Run("Prepare amend order with nil point", testPrepareAmendOrderWithNilPointer)
}

func TestPrepareCancelOrder(t *testing.T) {
	t.Run("Successfully cancel an order", testPrepareCancelOrderSuccess)
	t.Run("Fail to cancel an order for any number of reasons", testPrepareCancelOrderFail)
}

func TestCreateOrder(t *testing.T) {
	t.Run("Create order - successful", testOrderSuccess)
	t.Run("Create order - expired", testOrderExpired)
	t.Run("Create order - error expiry set for non gtt", testCreateOrderFailExpirySetForNonGTT)
}

func TestGetByOrderID(t *testing.T) {
	t.Run("Get by order ID - fetch default version", testGetByOrderIDDefaultVersion)
	t.Run("Get by order ID - fetch first version", testGetByOrderIDFirstVersion)
}

func testPrepareSubmitOrderWithNilPointer(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	ret, err := svc.svc.PrepareSubmitOrder(context.Background(), nil)
	assert.Nil(t, ret)
	assert.EqualError(t, err, orders.ErrEmptyPrepareRequest.Error())
}

func testPrepareAmendOrderWithNilPointer(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	ret, err := svc.svc.PrepareCancelOrder(context.Background(), nil)
	assert.Nil(t, ret)
	assert.EqualError(t, err, orders.ErrEmptyPrepareRequest.Error())
}

func testPrepareCancelOrderWithNilPointer(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	ret, err := svc.svc.PrepareAmendOrder(context.Background(), nil)
	assert.Nil(t, ret)
	assert.EqualError(t, err, orders.ErrEmptyPrepareRequest.Error())
}

func testPrepareOrderSuccess(t *testing.T) {
	// now
	now := vegatime.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	pre := &types.PendingOrder{
		Reference: "order_reference",
	}
	order := orderSubmission
	// set a specific reference
	order.Reference = pre.Reference
	order.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	// ensure the blockchain client is not called
	ret, err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.NotNil(t, ret)
	assert.NoError(t, err)
	assert.Equal(t, pre.Reference, ret.Reference)
}

func testPrepareOrderRefSuccess(t *testing.T) {
	// now
	now := vegatime.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	order := orderSubmission
	// DO NOT set a specific reference
	order.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	// ensure the blockchain client is not called
	ret, err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.NotNil(t, ret)
	assert.NoError(t, err)
	assert.NotEqual(t, "", order.Reference)
}

func testOrderSuccess(t *testing.T) {
	// now
	now := vegatime.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	order := orderSubmission
	order.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	pendingOrder, err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.NotNil(t, pendingOrder)
	if pendingOrder == nil {
		t.FailNow()
	}
	assert.NoError(t, err)
	assert.NotEmpty(t, pendingOrder.Reference)
}

func testPrepareOrderExpired(t *testing.T) {
	// now
	now := vegatime.Now()
	order := orderSubmission
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	svc.timeSvc.EXPECT().GetTimeNow().Times(1).Return(now, nil)
	pendingOrder, err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
}

func testCreateOrderFailExpirySetForNonGTT(t *testing.T) {
	order := orderSubmission
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	order.ExpiresAt = 12346
	order.TimeInForce = types.Order_GTC
	pendingOrder, err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.Nil(t, pendingOrder)
	assert.EqualError(t, err, orders.ErrNonGTTOrderWithExpiry.Error())
	pendingOrder, err = svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.Nil(t, pendingOrder)
	assert.EqualError(t, err, orders.ErrNonGTTOrderWithExpiry.Error())

	// ensure it works with a 0 expiry
	order.ExpiresAt = 0
	pendingOrder, err = svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.NotNil(t, pendingOrder)
	assert.NoError(t, err)
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
	pendingOrder, err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.Nil(t, pendingOrder)
	assert.Error(t, err)
}

func testPrepareCancelOrderSuccess(t *testing.T) {
	order := &types.Order{
		Id:          orderSubmission.Id,
		MarketID:    orderSubmission.MarketID,
		PartyID:     orderSubmission.PartyID,
		Side:        orderSubmission.Side,
		Price:       orderSubmission.Price,
		Size:        orderSubmission.Size,
		TimeInForce: orderSubmission.TimeInForce,
		Status:      types.Order_Active,   // order still is active
		Remaining:   orderSubmission.Size, // order not filled
	}
	cancel := types.OrderCancellation{
		OrderID:  order.Id,
		MarketID: order.MarketID,
		PartyID:  order.PartyID,
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.orderStore.EXPECT().GetByMarketAndID(gomock.Any(), cancel.MarketID, cancel.OrderID).Times(1).Return(order, nil)
	// ensure the blockchain client is not called
	ret, err := svc.svc.PrepareCancelOrder(context.Background(), &cancel)
	assert.NoError(t, err)
	assert.Equal(t, cancel.OrderID, ret.Id) // check that the ID matches the original one
	assert.Equal(t, order.Price, ret.Price) // check that the price matches the value from store
}

func testPrepareCancelOrderFail(t *testing.T) {
	data := map[string]*types.Order{
		"invalid status": &types.Order{
			Id:          orderSubmission.Id,
			MarketID:    orderSubmission.MarketID,
			PartyID:     orderSubmission.PartyID,
			Price:       orderSubmission.Price,
			Size:        orderSubmission.Size,
			Side:        orderSubmission.Side,
			TimeInForce: orderSubmission.TimeInForce,
			Status:      types.Order_Cancelled,
			Remaining:   orderSubmission.Size,
		},
		"order filled": &types.Order{
			Id:          orderSubmission.Id,
			MarketID:    orderSubmission.MarketID,
			PartyID:     orderSubmission.PartyID,
			Price:       orderSubmission.Price,
			Size:        orderSubmission.Size,
			Side:        orderSubmission.Side,
			TimeInForce: orderSubmission.TimeInForce,
			Status:      types.Order_Filled,
			Remaining:   0,
		},
		"wrong party": &types.Order{
			Id:          orderSubmission.Id,
			MarketID:    orderSubmission.MarketID,
			PartyID:     "someone else...",
			Price:       orderSubmission.Price,
			Size:        orderSubmission.Size,
			Side:        orderSubmission.Side,
			TimeInForce: orderSubmission.TimeInForce,
			Status:      types.Order_Active,
			Remaining:   orderSubmission.Size,
		},
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	// ensure the blockchain client is not called
	for set, order := range data {
		var (
			err    error
			cancel types.OrderCancellation
		)
		if order == nil {
			cancel = types.OrderCancellation{
				OrderID:  "123",
				MarketID: "346",
				PartyID:  "foobar",
			}
			err = errors.New(set)
		} else {
			err = nil
			cancel = types.OrderCancellation{
				OrderID:  order.Id,
				MarketID: order.MarketID,
				PartyID:  orderSubmission.PartyID, // this always is the same party, but the return from store could be different
			}
		}
		svc.orderStore.EXPECT().GetByMarketAndID(gomock.Any(), cancel.MarketID, cancel.OrderID).Times(1).Return(order, err)
		ret, rerr := svc.svc.PrepareCancelOrder(context.Background(), &cancel)
		assert.NotNil(t, ret)
		assert.NoError(t, rerr)
		if err != nil {
			assert.Equal(t, err, rerr)
		}
	}
}

func testGetByOrderIDDefaultVersion(t *testing.T) {
	order := &types.Order{
		Id:          orderSubmission.Id,
		MarketID:    orderSubmission.MarketID,
		PartyID:     orderSubmission.PartyID,
		Side:        orderSubmission.Side,
		Price:       orderSubmission.Price,
		Size:        orderSubmission.Size,
		TimeInForce: orderSubmission.TimeInForce,
		Status:      types.Order_Active,
		Remaining:   orderSubmission.Size,
		Version:     execution.InitialOrderVersion,
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.orderStore.EXPECT().GetByOrderID(gomock.Any(), order.Id, gomock.Nil()).Times(1).Return(order, nil)

	ret, err := svc.svc.GetByOrderID(context.Background(), order.Id, 0)
	assert.NoError(t, err)
	assert.Equal(t, order.Id, ret.Id)
	assert.Equal(t, order.Version, ret.Version)
}

func testGetByOrderIDFirstVersion(t *testing.T) {
	order := &types.Order{
		Id:          orderSubmission.Id,
		MarketID:    orderSubmission.MarketID,
		PartyID:     orderSubmission.PartyID,
		Side:        orderSubmission.Side,
		Price:       orderSubmission.Price,
		Size:        orderSubmission.Size,
		TimeInForce: orderSubmission.TimeInForce,
		Status:      types.Order_Active,
		Remaining:   orderSubmission.Size,
		Version:     execution.InitialOrderVersion,
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	svc.orderStore.EXPECT().GetByOrderID(gomock.Any(), order.Id, gomock.Not(nil)).Times(1).Return(order, nil)

	ret, err := svc.svc.GetByOrderID(context.Background(), order.Id, 1)
	assert.NoError(t, err)
	assert.Equal(t, order.Id, ret.Id)
	assert.Equal(t, order.Version, ret.Version)
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
