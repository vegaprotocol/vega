package orders_test

import (
	"context"
	"testing"

	"code.vegaprotocol.io/vega/execution"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/orders"
	"code.vegaprotocol.io/vega/orders/mocks"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"
	"github.com/golang/mock/gomock"

	"github.com/stretchr/testify/assert"
)

var (
	orderSubmission = commandspb.OrderSubmission{
		Type:        types.Order_TYPE_LIMIT,
		MarketId:    "market_id",
		Price:       10000,
		Size:        1,
		Side:        types.Side(1),
		TimeInForce: types.Order_TIME_IN_FORCE_GTT,
	}
)

type testService struct {
	ctrl       *gomock.Controller
	orderStore *mocks.MockOrderStore
	timeSvc    *mocks.MockTimeService
	svc        *orders.Svc
}

func TestGetByOrderID(t *testing.T) {
	t.Run("Get by order ID - fetch default version", testGetByOrderIDDefaultVersion)
	t.Run("Get by order ID - fetch first version", testGetByOrderIDFirstVersion)
}

func testGetByOrderIDDefaultVersion(t *testing.T) {
	order := &types.Order{
		MarketId:    orderSubmission.MarketId,
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
		MarketId:    orderSubmission.MarketId,
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
