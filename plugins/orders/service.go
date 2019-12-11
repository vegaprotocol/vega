package orders

import (
	"context"
	"errors"

	"code.vegaprotocol.io/vega/logging"
	"code.vegaprotocol.io/vega/plugins/orders/proto"
)

var (
	ErrMissingPartyID  = errors.New("missing party id")
	ErrMissingMarketID = errors.New("missing market id")
	ErrMissingOrderID  = errors.New("missing order id")
)

type service struct {
	log   *logging.Logger
	ctx   context.Context
	store *orderStore
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
