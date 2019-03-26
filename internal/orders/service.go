package orders

import (
	"context"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/pkg/errors"
)

var (
	ErrInvalidExpirationDTFmt = errors.New("invalid expiration datetime format")
	ErrInvalidExpirationDT    = errors.New("invalid expiration datetime")
)

//go:generate go run github.com/golang/mock/mockgen -destination newmocks/time_service_mock.go -package newmocks code.vegaprotocol.io/vega/internal/orders TimeService
type TimeService interface {
	GetTimeNow() (epochTimeNano vegatime.Stamp, datetime time.Time, err error)
}

//go:generate go run github.com/golang/mock/mockgen -destination newmocks/order_store_mock.go -package newmocks code.vegaprotocol.io/vega/internal/orders  OrderStore
type OrderStore interface {
	GetByMarketAndId(ctx context.Context, market string, id string) (*types.Order, error)
	GetByPartyAndId(ctx context.Context, party, id string) (*types.Order, error)
	GetByMarket(ctx context.Context, market string, filters *filtering.OrderQueryFilters) ([]*types.Order, error)
	GetByParty(ctx context.Context, party string, filters *filtering.OrderQueryFilters) ([]*types.Order, error)
	Subscribe(orders chan<- []types.Order) uint64
	Unsubscribe(id uint64) error
}

//go:generate go run github.com/golang/mock/mockgen -destination newmocks/blockchain_mock.go -package newmocks code.vegaprotocol.io/vega/internal/orders  Blockchain
type Blockchain interface {
	CreateOrder(ctx context.Context, order *types.Order) (success bool, orderReference string, err error)
	CancelOrder(ctx context.Context, order *types.Order) (success bool, err error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
}

type Service interface {
	CreateOrder(ctx context.Context, order *types.OrderSubmission) (success bool, orderReference string, err error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
	CancelOrder(ctx context.Context, order *types.OrderCancellation) (success bool, err error)
	GetByMarket(ctx context.Context, market string, filters *filtering.OrderQueryFilters) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, filters *filtering.OrderQueryFilters) (orders []*types.Order, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (order *types.Order, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (order *types.Order, err error)
	ObserveOrders(ctx context.Context, retries int, market *string, party *string) (orders <-chan []types.Order, ref uint64)
}

type orderService struct {
	*Config
	blockchain  Blockchain
	orderStore  OrderStore
	timeService TimeService
}

// NewOrderService creates an Orders service with the necessary dependencies
func NewOrderService(config *Config, store OrderStore, time TimeService, client Blockchain) (Service, error) {
	if client == nil {
		return nil, errors.New("blockchain client is nil when calling NewOrderService in OrderService")
	}
	return &orderService{
		Config:      config,
		blockchain:  client,
		orderStore:  store,
		timeService: time,
	}, nil
}

func (s *orderService) CreateOrder(ctx context.Context, orderSubmission *types.OrderSubmission) (success bool, orderReference string, err error) {
	if err := orderSubmission.Validate(); err != nil {
		return false, "", errors.Wrap(err, "order validation failed")
	}
	order := types.Order{
		Id:                 orderSubmission.Id,
		Market:             orderSubmission.MarketId,
		Party:              orderSubmission.Party,
		Price:              orderSubmission.Price,
		Size:               orderSubmission.Size,
		Side:               orderSubmission.Side,
		Type:               orderSubmission.Type,
		ExpirationDatetime: orderSubmission.ExpirationDatetime,
	}

	// Set defaults, prevent unwanted external manipulation
	order.Remaining = orderSubmission.Size
	order.Status = types.Order_Active
	order.Timestamp = 0
	order.Reference = ""

	if order.Type == types.Order_GTT {
		t, err := s.validateOrderExpirationTS(order.ExpirationDatetime)
		if err != nil {
			s.log.Error("unable to get expiration time", logging.Error(err))
			return false, "", err
		}
		order.ExpirationTimestamp = uint64(t.UnixNano())
	}

	// Call out to the blockchain package/layer and use internal client to gain consensus
	return s.blockchain.CreateOrder(ctx, &order)
}

// CancelOrder requires valid ID, Market, Party on an attempt to cancel the given active order via consensus
func (s *orderService) CancelOrder(ctx context.Context, order *types.OrderCancellation) (success bool, err error) {
	if err := order.Validate(); err != nil {
		return false, errors.Wrap(err, "order cancellation validation failed")
	}
	// Validate order exists using read store
	o, err := s.orderStore.GetByMarketAndId(ctx, order.MarketId, order.Id)
	if err != nil {
		return false, err
	}
	if o.Status == types.Order_Cancelled {
		return false, errors.New("order has already been cancelled")
	}
	if o.Remaining == 0 {
		return false, errors.New("order has been fully filled")
	}
	if o.Party != order.Party {
		return false, errors.New("party mis-match cannot cancel order")
	}
	// Send cancellation request by consensus
	return s.blockchain.CancelOrder(ctx, o)
}

func (s *orderService) AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error) {
	if err := amendment.Validate(); err != nil {
		return false, errors.Wrap(err, "order amendment validation failed")
	}
	// Validate order exists using read store
	o, err := s.orderStore.GetByPartyAndId(ctx, amendment.Party, amendment.Id)
	if err != nil {
		return false, err
	}

	if o.Status != types.Order_Active {
		return false, errors.New("order is not active")
	}

	// if order is GTT convert datetime to blockchain ts
	if o.Type == types.Order_GTT {
		t, err := s.validateOrderExpirationTS(amendment.ExpirationDatetime)
		if err != nil {
			s.log.Error("unable to get expiration time", logging.Error(err))
			return false, err
		}
		amendment.ExpirationTimestamp = uint64(t.UnixNano())
	}

	// Send edit request by consensus
	return s.blockchain.AmendOrder(ctx, amendment)
}

func (s *orderService) validateOrderExpirationTS(expdt string) (time.Time, error) {
	exp, err := time.Parse(time.RFC3339, expdt)
	if err != nil {
		return time.Time{}, ErrInvalidExpirationDTFmt
	}

	_, now, err := s.timeService.GetTimeNow()
	if err != nil {
		return time.Time{}, err
	}

	if exp.Before(now) || exp.Equal(now) {
		return time.Time{}, ErrInvalidExpirationDT
	}

	return exp, nil
}

func (s *orderService) GetByMarket(ctx context.Context, market string, filters *filtering.OrderQueryFilters) (orders []*types.Order, err error) {
	o, err := s.orderStore.GetByMarket(ctx, market, filters)
	if err != nil {
		return nil, err
	}
	filterOpen := filters != nil && filters.Open == true
	result := make([]*types.Order, 0)
	for _, order := range o {
		if filterOpen && (order.Remaining == 0 || order.Status != types.Order_Active) {
			continue
		}
		result = append(result, order)
	}
	return result, err
}

func (s *orderService) GetByParty(ctx context.Context, party string, filters *filtering.OrderQueryFilters) (orders []*types.Order, err error) {
	o, err := s.orderStore.GetByParty(ctx, party, filters)
	if err != nil {
		return nil, err
	}
	filterOpen := filters != nil && filters.Open == true
	result := make([]*types.Order, 0)
	for _, order := range o {
		if filterOpen && (order.Remaining == 0 || order.Status != types.Order_Active) {
			continue
		}
		result = append(result, order)
	}
	return result, err
}

func (s *orderService) GetByMarketAndId(ctx context.Context, market string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByMarketAndId(ctx, market, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

func (s *orderService) GetByPartyAndId(ctx context.Context, party string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByPartyAndId(ctx, party, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

func (s *orderService) ObserveOrders(ctx context.Context, retries int, market *string, party *string) (<-chan []types.Order, uint64) {
	orders := make(chan []types.Order)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	retryCount := retries
	go func() {
		ip := logging.IPAddressFromContext(ctx)
		ctx, cfunc := context.WithCancel(ctx)
		defer cfunc()
		for {
			select {
			case <-ctx.Done():
				s.log.Debug(
					"Orders subscriber closed connection",
					logging.Uint64("id", ref),
					logging.String("ip-address", ip),
				)
				// this error only happens when the subscriber reference doesn't exist
				// so we can still safely close the channels
				if err := s.orderStore.Unsubscribe(ref); err != nil {
					s.log.Error(
						"Failure un-subscribing orders subscriber when context.Done()",
						logging.Uint64("id", ref),
						logging.String("ip-address", ip),
						logging.Error(err),
					)
				}
				close(internal)
				close(orders)
				return
			case v := <-internal:
				// max cap for this slice is the length of the slice we read from the channel
				validatedOrders := make([]types.Order, 0, len(v))
				for _, item := range v {
					// if market is not set, or equals item market and party is not set or equals item party
					if (market == nil || item.Market == *market) && (party == nil || item.Party == *party) {
						validatedOrders = append(validatedOrders, item)
					}
				}
				if len(validatedOrders) == 0 {
					continue
				}
				select {
				case orders <- validatedOrders:
					retryCount = retries
					s.log.Debug(
						"Orders for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				default:
					retryCount--
					if retryCount == 0 {
						s.log.Warn(
							"Order subscriber has hit the retry limit",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
							logging.Int("retries", retries),
						)
						cfunc()
					}
					// retry counter here
					s.log.Debug(
						"Orders for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
					)
				}
			}
		}
	}()

	return orders, ref
}
