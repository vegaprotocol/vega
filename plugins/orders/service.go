package orders

import (
	"context"
	"errors"
	"sync/atomic"

	"code.vegaprotocol.io/vega/contextutil"
	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins/orders/proto"
	types "code.vegaprotocol.io/vega/proto"
)

var (
	// ErrChannelClosed signals that the channel streaming data is closed
	ErrChannelClosed = errors.New("channel closed")
	// ErrServerShutdown signals to the client that the server  is shutting down
	ErrServerShutdown = errors.New("server shutdown")
	// ErrStreamClosed signals to the users that the grpc stream is closing
	ErrStreamClosed    = errors.New("stream closed")
	ErrMissingPartyID  = errors.New("missing party id")
	ErrMissingMarketID = errors.New("missing market id")
	ErrMissingOrderID  = errors.New("missing order id")
)

type service struct {
	log   *logging.Logger
	ctx   context.Context
	store *orderStore

	subscriberCnt int32
}

func newService(ctx context.Context, log *logging.Logger, store *orderStore) *service {
	return &service{
		log:   log,
		ctx:   ctx,
		store: store,
	}
}

func (s *service) OrdersByParty(_ context.Context, req *proto.OrdersByPartyRequest) (*proto.OrdersByPartyResponse, error) {
	if len(req.PartyID) <= 0 {
		return nil, ErrMissingPartyID
	}

	o, err := s.store.GetByPartyID(req.PartyID)
	if err != nil {
		return nil, err
	}
	return &proto.OrdersByPartyResponse{
		Orders: o,
	}, nil
}

func (s *service) OrdersByPartyAndMarket(_ context.Context, req *proto.OrdersByPartyAndMarketRequest) (*proto.OrdersByPartyAndMarketResponse, error) {
	if len(req.PartyID) <= 0 {
		return nil, ErrMissingPartyID
	}
	if len(req.MarketID) <= 0 {
		return nil, ErrMissingMarketID
	}
	o, err := s.store.GetByPartyAndMarketID(req.PartyID, req.MarketID)
	if err != nil {
		return nil, err
	}
	return &proto.OrdersByPartyAndMarketResponse{
		Orders: o,
	}, nil

}

func (s *service) OrderByID(_ context.Context, req *proto.OrderByIDRequest) (*proto.OrderByIDResponse, error) {
	if len(req.OrderID) <= 0 {
		return nil, ErrMissingOrderID
	}
	o, err := s.store.GetByID(req.OrderID)
	if err != nil {
		return nil, err
	}
	return &proto.OrderByIDResponse{
		Order: o,
	}, nil

}

func (s *service) Subscribe(
	req *proto.SubscribeRequest, srv proto.OrdersCore_SubscribeServer) error {
	// wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	// increment counter of subscribers
	atomic.AddInt32(&s.subscriberCnt, 1)
	defer atomic.AddInt32(&s.subscriberCnt, -1)

	// subscribe to the orders
	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
	ordersChan := make(chan []types.Order)
	defer close(ordersChan)
	ref := s.store.Subscribe(ordersChan)
	defer func() {
		if err := s.store.Unsubscribe(ref); err != nil {
			s.log.Error(
				"Failure un-subscribing orders subscriber when context.Done()",
				logging.Uint64("id", ref),
				logging.String("ip-address", ip),
				logging.Error(err))
		}
	}()
	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case orders := <-ordersChan:
			// ensure channel is not closed
			if orders == nil {
				s.log.Error("Orders subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return ErrChannelClosed
			}

			// filter orders
			out := make([]*types.Order, 0, len(orders))
			for _, v := range orders {
				// if market is not set, or equals item market and party is not set or equals item party
				if (len(req.MarketID) <= 0 || v.MarketID == req.MarketID) &&
					(len(req.PartyID) <= 0 || v.PartyID == req.PartyID) {
					v := v
					out = append(out, &v)
				}
			}
			if err := srv.Send(&proto.OrdersStream{Orders: out}); err != nil {
				s.log.Error("Orders subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref))
				return err
			}
		case <-ctx.Done():
			if s.log.GetLevel() == logging.DebugLevel {
				s.log.Debug("Orders subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref))
			}
			return ctx.Err()
		case <-s.ctx.Done():
			return ErrServerShutdown
		}

		if ordersChan == nil {
			if s.log.GetLevel() == logging.DebugLevel {
				s.log.Debug("Orders subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return ErrStreamClosed
		}
	}
}

func (s *service) SubscribeMarketDepth(
	req *proto.SubscribeMarketDepthRequest, srv proto.OrdersCore_SubscribeMarketDepthServer) error {
	if len(req.MarketID) <= 0 {
		return ErrMissingMarketID
	}

	// wrap context from the request into cancellable. We can close internal chan on error.
	ctx, cancel := context.WithCancel(srv.Context())
	defer cancel()

	// increment counter of subscribers
	atomic.AddInt32(&s.subscriberCnt, 1)
	defer atomic.AddInt32(&s.subscriberCnt, -1)

	// subscribe to the orders
	ip, _ := contextutil.RemoteIPAddrFromContext(ctx)
	ordersChan := make(chan []types.Order)
	defer close(ordersChan)
	ref := s.store.Subscribe(ordersChan)
	defer func() {
		if err := s.store.Unsubscribe(ref); err != nil {
			s.log.Error(
				"Failure un-subscribing orders subscriber when context.Done()",
				logging.Uint64("id", ref),
				logging.String("ip-address", ip),
				logging.Error(err))
		}
	}()
	if s.log.GetLevel() == logging.DebugLevel {
		s.log.Debug("Orders subscriber - new rpc stream", logging.Uint64("ref", ref))
	}

	var err error
	for {
		select {
		case ords := <-ordersChan:
			// ensure channel is not closed
			if ords == nil {
				s.log.Error("Orders subscriber",
					logging.Error(err),
					logging.Uint64("ref", ref),
				)
				return ErrChannelClosed
			}

			d, err := s.store.GetMarketDepth(ctx, req.MarketID)
			if err != nil {
				s.log.Debug(
					"Failure calculating market depth for subscriber",
					logging.Uint64("ref", ref),
					logging.String("ip-address", ip),
					logging.Error(err))
				continue
			}
			if err := srv.Send(d); err != nil {
				s.log.Error("Orders subscriber - rpc stream error",
					logging.Error(err),
					logging.Uint64("ref", ref))
				return err
			}
		case <-ctx.Done():
			if s.log.GetLevel() == logging.DebugLevel {
				s.log.Debug("Orders subscriber - rpc stream ctx error",
					logging.Error(err),
					logging.Uint64("ref", ref))
			}
			return ctx.Err()
		case <-s.ctx.Done():
			return ErrServerShutdown
		}

		if ordersChan == nil {
			if s.log.GetLevel() == logging.DebugLevel {
				s.log.Debug("Orders subscriber - rpc stream closed",
					logging.Uint64("ref", ref),
				)
			}
			return ErrStreamClosed
		}
	}
}
