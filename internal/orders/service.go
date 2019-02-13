package orders

import (
	"context"
	"time"

	types "vega/proto"

	"vega/internal/blockchain"
	"vega/internal/filtering"
	"vega/internal/logging"
	"vega/internal/storage"
	"vega/internal/vegatime"

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
	ObserveOrders(ctx context.Context, market *string, party *string) (orders <-chan []types.Order, ref uint64)
}

type orderService struct {
	*Config
	orderStore  storage.OrderStore
	blockchain  blockchain.Client
	timeService vegatime.Service
}

// NewOrderService creates an Orders service with the necessary dependencies
func NewOrderService(config *Config, store storage.OrderStore, time vegatime.Service) (Service, error) {

	// todo (cdm): come back and pass configs in including blockchain config or blockchain client
	bcConfig := blockchain.NewConfig(config.log)
	client, err := blockchain.NewClient(bcConfig)
	if err != nil {
		config.log.Fatalf("error creating blockchain client %s", err)
		// todo(cdm): return this error or fatal?
	}

	return &orderService{
		config,
		store,
		client,
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
			s.log.Errorf("error loading current time when creating order: %s", err)
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
	o, err := s.orderStore.GetByMarketAndId(order.Market, order.Id)
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
	o, err := s.orderStore.GetByPartyAndId(amendment.Party, amendment.Id)
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
			s.log.Errorf("error loading current time when amending order: %s", err)
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
	o, err := s.orderStore.GetByMarket(market, filters)
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
	o, err := s.orderStore.GetByParty(party, filters)
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
	o, err := s.orderStore.GetByMarketAndId(market, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

func (s *orderService) GetByPartyAndId(ctx context.Context, party string, id string) (order *types.Order, err error) {
	o, err := s.orderStore.GetByPartyAndId(party, id)
	if err != nil {
		return &types.Order{}, err
	}
	return o, err
}

func (s *orderService) ObserveOrders(ctx context.Context, market *string, party *string) (<-chan []types.Order, uint64) {
	orders := make(chan []types.Order)
	internal := make(chan []types.Order)
	ref := s.orderStore.Subscribe(internal)

	go func(id uint64, internal chan []types.Order, ctx context.Context) {
		ip := logging.IPAddressFromContext(ctx)
		<-ctx.Done()
		s.log.Debugf("OrderService -> Subscriber closed connection: %d [%s]", id, ip)
		err := s.orderStore.Unsubscribe(id)
		if err != nil {
			s.log.Errorf("Error un-subscribing when context.Done() on OrderService for subscriber %d [%s]: %s", id, ip, err)
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
					s.log.Debugf("OrderService -> Orders for subscriber %d [%s] sent successfully", ref, ip)
				default:
					s.log.Debugf("OrderService -> Orders for subscriber %d [%s] not sent", ref, ip)
				}
			}
		}
		s.log.Debugf("OrderService -> Channel for subscriber %d [%s] has been closed", ref, ip)
	}(ref, ctx)

	return orders, ref
}
