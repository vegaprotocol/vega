package positions

import (
	"context"

	"code.vegaprotocol.io/vega/logging"
)

type svc struct {
	ctx context.Context
	pos *Pos
	log *logging.Logger
}

func newService(ctx context.Context, log *logging.Logger, pos *Pos) *svc {
	return &svc{
		ctx: ctx,
		pos: pos,
		log: log,
	}
}

func (s *svc) PositionsByMarket(_ context.Context, req *proto.GetPositionsByMarketRequest) (*proto.PositionsByMarketResponse, error) {
}
