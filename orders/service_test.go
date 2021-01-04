package orders_test

import (
	"context"
	"testing"
	"time"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/orders/mocks"
	types "code.vegaprotocol.io/vega/proto/gen/golang"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

var (
	orderSubmission = types.OrderSubmission{
		Type:        types.Order_TYPE_LIMIT,
		Id:          "order_id",
		MarketID:    "market_id",
		PartyID:     "party",
		Price:       10000,
		Size:        1,
		Side:        types.Side(1),
		TimeInForce: types.Order_TIF_GTT,
	}
)

type testService struct {
	ctrl       *gomock.Controller
	orderStore *mocks.MockOrderStore
	timeSvc    *mocks.MockTimeService
	svc        *orders.Svc
}

func TestPrepareOrder(t *testing.T) {
	t.Run("Create order with reference - successful", testPrepareOrderSuccess)
	t.Run("Create order without reference - successful", testPrepareOrderRefSuccess)
	t.Run("Prepare submit order with nil point", testPrepareSubmitOrderWithNilPointer)
	t.Run("Prepare cancel order with nil point", testPrepareCancelOrderWithNilPointer)
	t.Run("Prepare amend order with nil point", testPrepareAmendOrderWithNilPointer)
}

func TestPrepareCancelOrder(t *testing.T) {
	t.Run("Successfully cancel an order", testPrepareCancelOrderSuccess)
}

func TestCreateOrder(t *testing.T) {
	t.Run("Create order - successful", testOrderSuccess)
	t.Run("Create order - error expiry set for non gtt", testCreateOrderFailExpirySetForNonGTT)
	t.Run("Create order - error use network order type", testCreateOrderFailNetworkOrderType)
}

func TestGetByOrderID(t *testing.T) {
	t.Run("Get by order ID - fetch default version", testGetByOrderIDDefaultVersion)
	t.Run("Get by order ID - fetch first version", testGetByOrderIDFirstVersion)
}

func testPrepareSubmitOrderWithNilPointer(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareSubmitOrder(context.Background(), nil)
	assert.EqualError(t, err, orders.ErrEmptyPrepareRequest.Error())
}

func testPrepareAmendOrderWithNilPointer(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareCancelOrder(context.Background(), nil)
	assert.EqualError(t, err, orders.ErrEmptyPrepareRequest.Error())
}

func testPrepareCancelOrderWithNilPointer(t *testing.T) {
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareAmendOrder(context.Background(), nil)
	assert.EqualError(t, err, orders.ErrEmptyPrepareRequest.Error())
}

func testPrepareOrderSuccess(t *testing.T) {
	// now
	now := vegatime.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	order := orderSubmission
	// set a specific reference
	order.Reference = "test-reference"
	order.ExpiresAt = expires.UnixNano()
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.NoError(t, err)
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

	err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
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

	err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.NoError(t, err)
}

func testCreateOrderFailExpirySetForNonGTT(t *testing.T) {
	order := orderSubmission
	svc := getTestService(t)
	defer svc.ctrl.Finish()
	order.ExpiresAt = 12346
	order.TimeInForce = types.Order_TIF_GTC
	err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.EqualError(t, err, orders.ErrNonGTTOrderWithExpiry.Error())

	// ensure it works with a 0 expiry
	order.ExpiresAt = 0
	err = svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.NoError(t, err)
}

func testCreateOrderFailNetworkOrderType(t *testing.T) {
	// now
	now := vegatime.Now()
	// expires 2 hours from now
	expires := now.Add(time.Hour * 2)
	order := orderSubmission
	order.ExpiresAt = expires.UnixNano()
	order.Type = types.Order_TYPE_NETWORK
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareSubmitOrder(context.Background(), &order)
	assert.EqualError(t, err, orders.ErrUnAuthorizedOrderType.Error())
}

func testPrepareCancelOrderSuccess(t *testing.T) {
	cancel := types.OrderCancellation{
		OrderID:  "order.Id",
		MarketID: "order.MarketID",
		PartyID:  "order.PartyID",
	}
	svc := getTestService(t)
	defer svc.ctrl.Finish()

	err := svc.svc.PrepareCancelOrder(context.Background(), &cancel)
	assert.NoError(t, err)
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
		Status:      types.Order_STATUS_ACTIVE,
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
		Status:      types.Order_STATUS_ACTIVE,
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
	conf := orders.NewDefaultConfig()
	svc, err := orders.NewService(log, conf, orderStore, timeSvc)
	if err != nil {
		t.Fatalf("Failed to get test service: %+v", err)
	}
	return &testService{
		ctrl:       ctrl,
		orderStore: orderStore,
		timeSvc:    timeSvc,
		svc:        svc,
	}
}
