package orders

import (
	"context"
	"sync/atomic"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/vegatime"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	// ErrInvalidExpirationDTFmt signals that the time format was invalid
	ErrInvalidExpirationDTFmt = errors.New("invalid expiration datetime format")
	// ErrInvalidExpirationDT signals that the time format was invalid
	ErrInvalidExpirationDT = errors.New("invalid expiration datetime (cannot be in the past)")
	// ErrInvalidTimeInForceForMarketOrder signals that the time in force used with a market order is not accepted
	ErrInvalidTimeInForceForMarketOrder = errors.New("invalid time in force for market order (expected: FOK/IOC)")
	// ErrInvalidPriceForLimitOrder signal that a a price was missing for a limit order
	ErrInvalidPriceForLimitOrder = errors.New("invalid limit order (missing required price)")
	// ErrInvalidPriceForMarketOrder signals that a price was specified for a market order but not price is required
	ErrInvalidPriceForMarketOrder = errors.New("invalid market order (no price required)")
	// ErrNonGTTOrderWithExpiracy signals that a non GTT order what set with an expiracy
	ErrNonGTTOrderWithExpiry = errors.New("non GTT order with expiry")
	// ErrInvalidAmendmentSizeDelta ...
	ErrInvalidAmendmentSizeDelta = errors.New("invalid amendment size delta")
	// ErrInvalidAmendOrderTIF ...
	ErrInvalidAmendOrderTIF = errors.New("invalid amend order tif (cannot be IOC and FOK)")
)

// TimeService ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/time_service_mock.go -package mocks code.vegaprotocol.io/vega/orders TimeService
type TimeService interface {
	GetTimeNow() (time.Time, error)
}

// OrderStore ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/order_store_mock.go -package mocks code.vegaprotocol.io/vega/orders  OrderStore
type OrderStore interface {
	GetByMarketAndID(ctx context.Context, market string, id string) (*types.Order, error)
	GetByPartyAndID(ctx context.Context, party, id string) (*types.Order, error)
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) ([]*types.Order, error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) ([]*types.Order, error)
	GetByReference(ctx context.Context, ref string) (*types.Order, error)
	GetByOrderID(ctx context.Context, id string) (*types.Order, error)
	Subscribe(orders chan<- []types.Order) uint64
	Unsubscribe(id uint64) error
}

// Blockchain ...
//go:generate go run github.com/golang/mock/mockgen -destination mocks/blockchain_mock.go -package mocks code.vegaprotocol.io/vega/orders  Blockchain
type Blockchain interface {
	SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error)
}

// Svc represents the order service
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

// ReloadConf update the internal configuration of the order service
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

func (s *Svc) SubmitTransaction(ctx context.Context, bundle *types.SignedBundle) (bool, error) {
	return s.blockchain.SubmitTransaction(ctx, bundle)
}

func (s *Svc) PrepareSubmitOrder(ctx context.Context, submission *types.OrderSubmission) (*types.PendingOrder, error) {
	if err := s.validateOrderSubmission(submission); err != nil {
		return nil, err
	}
	if submission.Reference == "" {
		submission.Reference = uuid.NewV4().String()
	}
	return &types.PendingOrder{
		Reference:   submission.Reference,
		Price:       submission.Price,
		TimeInForce: submission.TimeInForce,
		Side:        submission.Side,
		MarketID:    submission.MarketID,
		Size:        submission.Size,
		PartyID:     submission.PartyID,
		Id:          submission.Id,
		Type:        submission.Type,
		Status:      types.Order_Active,
	}, nil
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

	if sub.TimeInForce != types.Order_GTT && sub.ExpiresAt != 0 {
		return ErrNonGTTOrderWithExpiry
	}

	if sub.Type == types.Order_MARKET && sub.Price != 0 {
		return ErrInvalidPriceForMarketOrder
	}
	if sub.Type == types.Order_MARKET &&
		(sub.TimeInForce != types.Order_FOK && sub.TimeInForce != types.Order_IOC) {
		return ErrInvalidTimeInForceForMarketOrder
	}
	if sub.Type == types.Order_LIMIT && sub.Price == 0 {
		return ErrInvalidPriceForLimitOrder
	}

	return nil
}

func (s *Svc) PrepareCancelOrder(ctx context.Context, order *types.OrderCancellation) (*types.PendingOrder, error) {
	if err := order.Validate(); err != nil {
		return nil, errors.Wrap(err, "order cancellation invalid")
	}
	o, err := s.orderStore.GetByMarketAndID(ctx, order.MarketID, order.OrderID)
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

func (s *Svc) PrepareAmendOrder(ctx context.Context, amendment *types.OrderAmendment) (*types.PendingOrder, error) {
	if err := amendment.Validate(); err != nil {
		return nil, errors.Wrap(err, "order amendment validation failed")
	}
	// Validate order exists using read store
	o, err := s.orderStore.GetByPartyAndID(ctx, amendment.PartyID, amendment.OrderID)
	if err != nil {
		return nil, err
	}

	if o.Status != types.Order_Active {
		return nil, errors.New("order is not active")
	}

	// if order is GTT convert datetime to blockchain ts
	if amendment.TimeInForce == types.Order_GTT {
		_, err := s.validateOrderExpirationTS(amendment.ExpiresAt)
		if err != nil {
			s.log.Error("unable to get expiration time", logging.Error(err))
			return nil, err
		}
	} else if amendment.TimeInForce == types.Order_GTC {
		// this is cool, but we need to ensure and expiry is not set
		if amendment.ExpiresAt != 0 {
			return nil, ErrNonGTTOrderWithExpiry
		} else {
			// IOC and FOK are not acceptable for amend order
			return nil, ErrInvalidAmendOrderTIF
		}
	}

	// if size changes, make sure it does not get negative
	newSize := int64(o.Size) + amendment.SizeDelta
	if newSize <= 0 {
		return nil, ErrInvalidAmendmentSizeDelta
	}

	return &types.PendingOrder{
		Reference:   o.Reference,
		Price:       amendment.Price,
		TimeInForce: amendment.TimeInForce,
		Side:        o.Side,
		MarketID:    o.MarketID,
		Size:        uint64(newSize),
		PartyID:     o.PartyID,
		Status:      types.Order_Active,
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

// GetByReference find an order from the store using its reference
func (s *Svc) GetByReference(ctx context.Context, ref string) (*types.Order, error) {
	return s.orderStore.GetByReference(ctx, ref)
}

// GetByMarket returns a list of order for a given market
func (s *Svc) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error) {
	return s.orderStore.GetByMarket(ctx, market, skip, limit, descending, open)
}

// GetByParty returns a list of order for a given party
func (s *Svc) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool, open *bool) (orders []*types.Order, err error) {
	return s.orderStore.GetByParty(ctx, party, skip, limit, descending, open)
}

// GetByMarketAndID find a order using a marketID and an order id
func (s *Svc) GetByMarketAndID(ctx context.Context, market string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByMarketAndID(ctx, market, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

// GetByOrderID find a order using an orderID
func (s *Svc) GetByOrderID(ctx context.Context, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByOrderID(ctx, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

// GetByPartyAndID find an order using a party id and an order id
func (s *Svc) GetByPartyAndID(ctx context.Context, party string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByPartyAndID(ctx, party, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

// GetOrderSubscribersCount return the number of subscribers to the
// orders updates stream
func (s *Svc) GetOrderSubscribersCount() int32 {
	return atomic.LoadInt32(&s.subscriberCnt)
}

// ObserveOrders add a new subscriber to the stream of order updates
func (s *Svc) ObserveOrders(ctx context.Context, retries int, market *string, party *string) (<-chan []types.Order, uint64) {
	orders := make(chan []types.Order)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	var cancel func()
	ctx, cancel = context.WithCancel(ctx)
	go func() {
		atomic.AddInt32(&s.subscriberCnt, 1)
		defer atomic.AddInt32(&s.subscriberCnt, -1)
		ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
		defer cancel()
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
				retryCount := retries
				success := false
				for !success && retryCount >= 0 {
					select {
					case orders <- validatedOrders:
						s.log.Debug(
							"Orders for subscriber sent successfully",
							logging.Uint64("ref", ref),
							logging.String("ip-address", ip),
						)
						success = true
					default:
						retryCount--
						if retryCount >= 0 {
							s.log.Debug(
								"Orders for subscriber not sent",
								logging.Uint64("ref", ref),
								logging.String("ip-address", ip),
							)
						}
						time.Sleep(time.Duration(10) * time.Millisecond)
					}
				}
				if !success && retryCount <= 0 {
					s.log.Warn(
						"Order subscriber has hit the retry limit",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip),
						logging.Int("retries", retries),
					)
					cancel()
				}
			}
		}
	}()

	return orders, ref
}
