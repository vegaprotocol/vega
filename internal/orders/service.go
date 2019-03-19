package orders

import (
	"context"
	"time"

	types "code.vegaprotocol.io/vega/proto"

	"code.vegaprotocol.io/vega/internal/blockchain"
	"code.vegaprotocol.io/vega/internal/filtering"
	"code.vegaprotocol.io/vega/internal/logging"
	"code.vegaprotocol.io/vega/internal/storage"
	"code.vegaprotocol.io/vega/internal/vegatime"

	"github.com/pkg/errors"
)

type Service interface {
	CreateOrder(ctx context.Context, order *types.Order) (success bool, orderReference string, err error)
	AmendOrder(ctx context.Context, amendment *types.Amendment) (success bool, err error)
	CancelOrder(ctx context.Context, order *types.Order) (success bool, err error)
	GetByMarket(ctx context.Context, market string, filters *filtering.OrderQueryFilters) (orders []*types.Order, err error)
	GetByParty(ctx context.Context, party string, filters *filtering.OrderQueryFilters) (orders []*types.Order, err error)
	GetByMarketAndId(ctx context.Context, market string, id string) (order *types.Order, err error)
	GetByPartyAndId(ctx context.Context, party string, id string) (order *types.Order, err error)
	GetLastOrder(ctx context.Context) (order *types.Order)
	ObserveOrders(ctx context.Context, market *string, party *string) (orders <-chan []types.Order, ref uint64)
}

type orderService struct {
	*Config
	blockchain  blockchain.Client
	orderStore  storage.OrderStore
	timeService vegatime.Service
}

// NewOrderService creates an Orders service with the necessary dependencies
func NewOrderService(config *Config, store storage.OrderStore, time vegatime.Service, client blockchain.Client) (Service, error) {
	if client == nil {
		return nil, errors.New("blockchain client is nil when calling NewOrderService in OrderService")
	}
	return &orderService{
		config,
		client,
		store,
		time,
	}, nil
}

func (s *orderService) CreateOrder(ctx context.Context, order *types.Order) (success bool, orderReference string, err error) {
	// Set defaults, prevent unwanted external manipulation
	order.Remaining = order.Size
	order.Status = types.Order_Active
	order.Timestamp = 0
	order.Reference = ""

	// if order is GTT convert datetime to blockchain timestamp
	if order.Type == types.Order_GTT {
		expirationDateTime, err := time.Parse(time.RFC3339, order.ExpirationDatetime)
		if err != nil {
			return false, "", errors.New("invalid expiration datetime format")
		}
		timeNow, _, err := s.timeService.GetTimeNow()
		if err != nil {
			s.log.Error("Failed to obtain current time when creating order in Order Service", logging.Error(err))
			return false, "", err
		}
		expirationTimestamp := expirationDateTime.UnixNano()
		if expirationTimestamp <= timeNow.UnixNano() {
			return false, "", errors.New("invalid expiration datetime error")
		}
		order.ExpirationTimestamp = uint64(expirationTimestamp)
	}

	// Call out to the blockchain package/layer and use internal client to gain consensus
	return s.blockchain.CreateOrder(ctx, order)
}

// CancelOrder requires valid ID, Market, Party on an attempt to cancel the given active order via consensus
func (s *orderService) CancelOrder(ctx context.Context, order *types.Order) (success bool, err error) {
	// Validate order exists using read store
	o, err := s.orderStore.GetByMarketAndId(ctx, order.Market, order.Id)
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

func (s *orderService) AmendOrder(ctx context.Context, amendment *types.Amendment) (success bool, err error) {

	// Validate order exists using read store
	o, err := s.orderStore.GetByPartyAndId(ctx, amendment.Party, amendment.Id)
	if err != nil {
		return false, err
	}

	if o.Status != types.Order_Active {
		return false, errors.New("order is not active")
	}

	// if order is GTT convert datetime to block chain timestamp
	if amendment.ExpirationDatetime != "" {
		expirationDateTime, err := time.Parse(time.RFC3339, amendment.ExpirationDatetime)
		if err != nil {
			return false, errors.New("invalid format expiration datetime")
		}
		_, currentDateTime, err := s.timeService.GetTimeNow()
		if err != nil {
			s.log.Error("Failed to obtain current time when amending order in Order Service", logging.Error(err))
			return false, err
		}
		if expirationDateTime.Before(currentDateTime) || expirationDateTime.Equal(currentDateTime) {
			return false, errors.New("invalid expiration datetime")
		}
		amendment.ExpirationTimestamp = uint64(expirationDateTime.UnixNano())
	}

	// Send edit request by consensus
	return s.blockchain.AmendOrder(ctx, amendment)
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

func (s *orderService) GetLastOrder(ctx context.Context) (order *types.Order) {
 	return s.orderStore.GetLastOrder(ctx)
}

func (s *orderService) ObserveOrders(ctx context.Context, market *string, party *string) (<-chan []types.Order, uint64) {
	orders := make(chan []types.Order)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Order, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		s.log.Debug("Orders subscriber closed connection",
			logging.Uint64("id", id),
			logging.String("ip-address", ip))
		err := s.orderStore.Unsubscribe(id)
		if err != nil {
			s.log.Error("Failure un-subscribing orders subscriber when context.Done()",
				logging.Uint64("id", id),
				logging.String("ip-address", ip),
				logging.Error(err))
		}
	}(ref, internal, ctx)

	go func(id uint64, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		// read internal channel
		for v := range internal {

			validatedOrders := make([]types.Order, 0)
			for _, item := range v {
				if market != nil && item.Market != *market {
					continue
				}
				if party != nil && item.Party != *party {
					continue
				}
				validatedOrders = append(validatedOrders, item)
			}
			if len(validatedOrders) > 0 {
				select {
				case orders <- validatedOrders:
					s.log.Debug("Orders for subscriber sent successfully",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				default:
					s.log.Debug("Orders for subscriber not sent",
						logging.Uint64("ref", ref),
						logging.String("ip-address", ip))
				}
			}
		}
		s.log.Debug("Orders subscriber channel has been closed",
			logging.Uint64("ref", ref),
			logging.String("ip-address", ip))
	}(ref, ctx)

	return orders, ref
}
