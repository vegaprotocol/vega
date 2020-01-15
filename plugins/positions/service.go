package positions

import (
	"context"

	"code.vegaprotocol.io/vega/logging"
	api "code.vegaprotocol.io/vega/plugins/positions/proto"
	types "code.vegaprotocol.io/vega/proto"
)

type svc struct {
	ctx context.Context
	pos *Pos
	log *logging.Logger
}

func newService(ctx context.Context, log *logging.Logger, pos *Pos) *svc {
	s := &svc{
		ctx: ctx,
		pos: pos,
		log: log,
	}
	api.RegisterPositionsServer(pos.srv, s)
	return s
}

func (s *svc) PositionsByParty(_ context.Context, req *api.PositionsByPartyRequest) (*api.PositionsByPartyResponse, error) {
	if req.MarketID != "" && req.PartyID != "" {
		pos, err := s.pos.GetPositionsByMarketAndParty(req.MarketID, req.PartyID)
		if err != nil {
			return nil, err
		}
		resp := &api.PositionsByPartyResponse{Positions: []*types.Position{pos}}
		return resp, nil
	}
	return nil, nil
}

func (s *svc) PositionsSubscribe(req *api.PositionsSubscribeRequest, sub api.Positions_PositionsSubscribeServer) error {
	return nil
}
