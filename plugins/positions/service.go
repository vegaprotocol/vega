package positions

import (
	"context"

	"code.vegaprotocol.io/vega/logging"
	api "code.vegaprotocol.io/vega/plugins/positions/proto"
	types "code.vegaprotocol.io/vega/proto"

	"github.com/pkg/errors"
)

var (
	ErrStreamExpired = errors.New("stream has expired - retry count reached")
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
	// register this service to handle specific gRPC calls
	api.RegisterPositionsServer(pos.srv, s)
	return s
}

func (s *svc) PositionsByParty(_ context.Context, req *api.PositionsByPartyRequest) (*api.PositionsByPartyResponse, error) {
	if req.MarketID != "" && req.PartyID != "" {
		pos, err := s.pos.GetPositionsByMarketAndParty(req.MarketID, req.PartyID)
		if err != nil {
			return nil, err
		}
		resp := &api.PositionsByPartyResponse{Positions: []*types.MarketPosition{positionToMarketPosition(pos)}}
		return resp, nil
	}
	var (
		ps  []*types.Position
		err error
	)
	if req.PartyID != "" {
		// get by party, regardless of market
		ps, err = s.pos.GetPositionsByParty(req.PartyID)
	} else {
		// get all positions for a market. The validator on the request doesn' allow this ATM
		// but best provide this here
		ps, err = s.pos.GetPositionsByMarket(req.MarketID)
	}
	if err != nil {
		return nil, err
	}
	mp := make([]*types.MarketPosition, 0, len(ps))
	for _, pos := range ps {
		mp = append(mp, positionToMarketPosition(pos))
	}
	return &api.PositionsByPartyResponse{
		Positions: mp,
	}, nil
}

func (s *svc) PositionsSubscribe(req *api.PositionsSubscribeRequest, sub api.Positions_PositionsSubscribeServer) error {
	// the subscription channel, buffered to stream retries count
	// if the buffer is full, the stream should be cancelled
	internal := make(chan struct{}, s.pos.conf.StreamRetries) // the channel used for subscription
	id, full := s.pos.Subscribe(internal)
	ctx, cfunc := context.WithCancel(sub.Context())
	defer func() {
		cfunc()
		s.pos.Unsubscribe(id)
		close(internal)
	}()
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-full:
			return ErrStreamExpired
		case <-internal:
			ps, err := s.pos.GetPositionsByParty(req.PartyID)
			if err != nil {
				return err
			}
			for _, pos := range ps {
				// while sending, make sure we're sure the stream still is active
				select {
				case <-ctx.Done():
					return nil
				case <-full:
					return ErrStreamExpired
				default:
					sub.Send(positionToMarketPosition(pos))
				}
			}
		}
	}
	return nil
}

func positionToMarketPosition(pos *types.Position) *types.MarketPosition {
	return &types.MarketPosition{
		MarketID:          pos.MarketID,
		RealisedVolume:    pos.OpenVolume,
		RealisedPNL:       pos.RealisedPNL,
		UnrealisedPNL:     pos.UnrealisedPNL,
		AverageEntryPrice: pos.AverageEntryPrice,
		PartyID:           pos.PartyID,
	}
}
