package orders

import (
	"context"
	"sync/atomic"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/pkg/errors"
)

var (
	ErrInvalidExpirationDTFmt           = errors.New("invalid expiration datetime format")
	ErrInvalidExpirationDT              = errors.New("invalid expiration datetime")
	ErrInvalidTimeInForceForMarketOrder = errors.New("invalid time in for for market order (expected: FOK/IOC)")
	ErrInvalidPriceForLimitOrder        = errors.New("invalid limit order (missing required price)")
	ErrInvalidPriceForMarketOrder       = errors.New("invalid market order (no price required)")
)

//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/internal/orders TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_store_mock.go -package mocks code.vegaprotocol.io/vega/internal/orders  OrderStore
type OrderStore interface {
	GetByMarketAndId(ctx context.Context, market string, id string) (*types.Order, error)
	GetByPartyAndId(ctx context.Context, party, id string) (*types.Order, error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) ([]*types.Order, error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) ([]*types.Order, error)
	GetByReference(ctx context.Context, ref string) (*types.Order, error)
	Subscribe(orders chan<- []types.Order) uint64
	Unsubscribe(id uint64) error
}

//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/internal/orders  Blockchain
type Blockchain interface {
	CreateOrder(ctx context.Context, order *types.Order) (*types.PendingOrder, error)
	CancelOrder(ctx context.Context, order *types.Order) (success bool, err error)
	AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (success bool, err error)
}

type Svc struct {
	Config
	log *logging.Logger

	blockchain    Blockchain
	orderStore    OrderStore
	timeService   TimeService
	subscriberCnt int32
}

// NewService creates an Orders service with the necessary dependencies
func NewService(log *logging.Logger, config Config, store OrderStore, time TimeService, client Blockchain) (*Svc, error) {
	if client == nil {
		return nil, errors.New("blockchain client is nil when calling NewService in OrderService")
	}

	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:         log,
		Config:      config,
		blockchain:  client,
		orderStore:  store,
		timeService: time,
	}, nil
}

func (s *Svc) ReloadConf(cfg Config) {
	s.log.Info("reloading configuration")
	if s.log.GetLevel() != cfg.Level.Get() {
		s.log.Info("updating log level",
			logging.String("old", s.log.GetLevel().String()),
			logging.String("new", cfg.Level.String()),
		)
		s.log.SetLevel(cfg.Level.Get())
	}

	s.Config = cfg
}

func (s *Svc) CreateOrder(
	ctx context.Context,
	orderSubmission *types.OrderSubmission,
) (*types.PendingOrder, error) {
	if err := s.validateOrderSubmission(orderSubmission); err != nil {
		return nil, err
	}
	order := types.Order{
		Id:          orderSubmission.Id,
		MarketID:    orderSubmission.MarketID,
		PartyID:     orderSubmission.PartyID,
		Price:       orderSubmission.Price,
		Size:        orderSubmission.Size,
		Side:        orderSubmission.Side,
		TimeInForce: orderSubmission.TimeInForce,
		Type:        orderSubmission.Type,
		ExpiresAt:   orderSubmission.ExpiresAt,
	}

	// Set defaults, prevent unwanted external manipulation
	order.Remaining = orderSubmission.Size
	order.Status = types.Order_Active
	order.CreatedAt = 0
	order.Reference = ""
	// Call out to the blockchain package/layer and use internal client to gain consensus
	return s.blockchain.CreateOrder(ctx, &order)
}

func (s *Svc) validateOrderSubmission(sub *types.OrderSubmission) error {
	if err := sub.Validate(); err != nil {
		return errors.Wrap(err, "order validation failed")
	}
	if sub.TimeInForce == types.Order_GTT {
		_, err := s.validateOrderExpirationTS(sub.ExpiresAt)
		if err != nil {
			s.log.Error("unable to get expiration time", logging.Error(err))
			return err
		}
	}

	if sub.Type == types.Order_MARKET && sub.Price != 0 {
		return ErrInvalidPriceForMarketOrder
	}
	if sub.Type == types.Order_MARKET &&
		(sub.TimeInForce != types.Order_FOK && sub.TimeInForce != types.Order_IOC) {
		return ErrInvalidTimeInForceForMarketOrder
	}
	if sub.Type == types.Order_LIMIT && sub.Price != 0 {
		return ErrInvalidPriceForLimitOrder
	}

	return nil
}

// CancelOrder requires valid ID, Market, Party on an attempt to cancel the given active order via consensus
func (s *Svc) CancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.PendingOrder, error) {
	if err := order.Validate(); err != nil {
		return nil, errors.Wrap(err, "order cancellation validation failed")
	}
	// Validate order exists using read store
	o, err := s.orderStore.GetByMarketAndId(ctx, order.MarketID, order.OrderID)
	if err != nil {
		return nil, err
	}
	if o.Status == types.Order_Cancelled {
		return nil, errors.New("order has already been cancelled")
	}
	if o.Remaining == 0 {
		return nil, errors.New("order has been fully filled")
	}
	if o.PartyID != order.PartyID {
		return nil, errors.New("party mis-match cannot cancel order")
	}
	// Send cancellation request by consensus
	if _, err := s.blockchain.CancelOrder(ctx, o); err != nil {
		return nil, err
	}

	return &types.PendingOrder{
		Reference:   o.Reference,
		Price:       o.Price,
		TimeInForce: o.TimeInForce,
		Side:        o.Side,
		MarketID:    o.MarketID,
		Size:        o.Size,
		PartyID:     o.PartyID,
		Status:      types.Order_Cancelled,
		Id:          o.Id,
	}, nil
}

func (s *Svc) AmendOrder(ctx context.Context, amendment *types.OrderAmendment) (*types.PendingOrder, error) {
	if err := amendment.Validate(); err != nil {
		return nil, errors.Wrap(err, "order amendment validation failed")
	}
	// Validate order exists using read store
	o, err := s.orderStore.GetByPartyAndId(ctx, amendment.PartyID, amendment.OrderID)
	if err != nil {
		return nil, err
	}

	if o.PartyID != amendment.PartyID {
		return nil, errors.New("party mis-match cannot cancel order")
	}

	if o.Status != types.Order_Active {
		return nil, errors.New("order is not active")
	}

	// if order is GTT convert datetime to blockchain ts
	if o.TimeInForce == types.Order_GTT {
		_, err := s.validateOrderExpirationTS(amendment.ExpiresAt)
		if err != nil {
			s.log.Error("unable to get expiration time", logging.Error(err))
			return nil, err
		}
	}

	// Send edit request by consensus
	if _, err := s.blockchain.AmendOrder(ctx, amendment); err != nil {
		return nil, err
	}

	return &types.PendingOrder{
		Reference:   o.Reference,
		Price:       amendment.Price,
		TimeInForce: o.TimeInForce,
		Side:        o.Side,
		MarketID:    o.MarketID,
		Size:        amendment.Size,
		PartyID:     o.PartyID,
		Status:      types.Order_Cancelled,
		Id:          o.Id,
	}, nil
}

func (s *Svc) validateOrderExpirationTS(expiresAt int64) (time.Time, error) {
	exp := vegatime.UnixNano(expiresAt)

	now, err := s.timeService.GetTimeNow()
	if err != nil {
		return time.Time{}, err
	}

	if exp.Before(now) || exp.Equal(now) {
		return time.Time{}, ErrInvalidExpirationDT
	}

	return exp, nil
}

func (s *Svc) GetByReference(ctx context.Context, ref string) (*types.Order, error) {
	return s.orderStore.GetByReference(ctx, ref)
}

func (s *Svc) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error) {
	return s.orderStore.GetByMarket(ctx, market, skip, limit, descending, open)
}

func (s *Svc) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error) {
	return s.orderStore.GetByParty(ctx, party, skip, limit, descending, open)
}

func (s *Svc) GetByMarketAndId(ctx context.Context, market string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByMarketAndId(ctx, market, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

func (s *Svc) GetByPartyAndId(ctx context.Context, party string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByPartyAndId(ctx, party, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

func (s *Svc) GetOrderSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}

func (s *Svc) ObserveOrders(ctx context.Context, retries int, market *string, party *string) (<-chan []types.Order, uint64) {
	orders := make(chan []types.Order)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	retryCount := retries
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
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
					if (market == nil || item.MarketID == *market) && (party == nil || item.PartyID == *party) {
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
