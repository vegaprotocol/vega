package orders

import (
	"context"
	"sync/atomic"
	"time"

	"code.vegaprotocol.io/vega/commands"
	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	types "code.vegaprotocol.io/vega/proto"
	commandspb "code.vegaprotocol.io/vega/proto/commands/v1"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var (
	// ErrEmptyPrepareRequest empty prepare request
	ErrEmptyPrepareRequest = errors.New("empty prepare request")
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
	GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) ([]*types.Order, error)
	GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool) ([]*types.Order, error)
	GetByReference(ctx context.Context, ref string) (*types.Order, error)
	GetByOrderID(ctx context.Context, id string, version *uint64) (*types.Order, error)
	GetAllVersionsByOrderID(ctx context.Context, id string, skip, limit uint64, descending bool) ([]*types.Order, error)
	Subscribe(orders chan<- []types.Order) uint64
	Unsubscribe(id uint64) error
}

// Svc represents the order service
type Svc struct {
	Config
	log *logging.Logger

	orderStore    OrderStore
	timeService   TimeService
	subscriberCnt int32
}

// NewService creates an Orders service with the necessary dependencies
func NewService(log *logging.Logger, config Config, store OrderStore, time TimeService) (*Svc, error) {
	// setup logger
	log = log.Named(namedLogger)
	log.SetLevel(config.Level.Get())

	return &Svc{
		log:         log,
		Config:      config,
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

func (s *Svc) PrepareSubmitOrder(_ context.Context, cmd *commandspb.OrderSubmission) error {
	if cmd == nil {
		return ErrEmptyPrepareRequest
	}

	if cmd.Reference == "" {
		cmd.Reference = uuid.NewV4().String()
	}

	return commands.CheckOrderSubmission(cmd)
}

func (s *Svc) PrepareCancelOrder(_ context.Context, cmd *commandspb.OrderCancellation) error {
	return nil
}

func (s *Svc) PrepareAmendOrder(_ context.Context, cmd *commandspb.OrderAmendment) error {
	if cmd == nil {
		return ErrEmptyPrepareRequest
	}

	return commands.CheckOrderAmendment(cmd)
}

// GetByOrderID find an order using its orderID
func (s *Svc) GetByOrderID(ctx context.Context, id string, version uint64) (order *types.Order, err error) {
	if version == 0 {
		return s.orderStore.GetByOrderID(ctx, id, nil)
	}
	return s.orderStore.GetByOrderID(ctx, id, &version)
}

// GetByReference find an order from the store using its reference
func (s *Svc) GetByReference(ctx context.Context, ref string) (*types.Order, error) {
	return s.orderStore.GetByReference(ctx, ref)
}

// GetByMarket returns a list of order for a given market
func (s *Svc) GetByMarket(ctx context.Context, market string, skip, limit uint64, descending bool) (orders []*types.Order, err error) {
	return s.orderStore.GetByMarket(ctx, market, skip, limit, descending)
}

// GetByParty returns a list of order for a given party
func (s *Svc) GetByParty(ctx context.Context, party string, skip, limit uint64, descending bool) (orders []*types.Order, err error) {
	return s.orderStore.GetByParty(ctx, party, skip, limit, descending)
}

// GetByMarketAndID find a order using a marketID and an order id
func (s *Svc) GetByMarketAndID(ctx context.Context, market string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByMarketAndID(ctx, market, id)
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

// GetAllVersionsByOrderID returns all available versions for the order specified by id
func (s *Svc) GetAllVersionsByOrderID(ctx context.Context, id string, skip, limit uint64, descending bool) (orders []*types.Order, err error) {
	return s.orderStore.GetAllVersionsByOrderID(ctx, id, skip, limit, descending)
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
					if (market == nil || item.MarketId == *market) && (party == nil || item.PartyId == *party) {
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
