package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/subscribers"
	protoapi "code.vegaprotocol.io/protos/vega/api"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"google.golang.org/grpc/codes"
)

type eventObserver struct {
	log          *logging.Logger
	eventService EventService
	Config       Config
}

func (e *eventObserver) ObserveEventBus(
	stream protoapi.TradingService_ObserveEventBusServer) error {
	ctx, cfunc := context.WithCancel(stream.Context())
	defer cfunc()

	// now we start listening for a few seconds in order to get at least the very first message
	// this will be blocking until the connection by the client is closed
	// and we will not start processing any events until we receive the original request
	// indicating filters and batch size.
	req, err := e.recvEventRequest(stream)
	if err != nil {
		// client exited, nothing to do
		return nil
	}

	if err := req.Validate(); err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}

	// now we will aggregate filter out of the initial request
	types, err := events.ProtoToInternal(req.Type...)
	if err != nil {
		return apiError(codes.InvalidArgument, ErrMalformedRequest, err)
	}
	filters := []subscribers.EventFilter{}
	if len(req.MarketId) > 0 && len(req.PartyId) > 0 {
		filters = append(filters, events.GetPartyAndMarketFilter(req.MarketId, req.PartyId))
	} else {
		if len(req.MarketId) > 0 {
			filters = append(filters, events.GetMarketIDFilter(req.MarketId))
		}
		if len(req.PartyId) > 0 {
			filters = append(filters, events.GetPartyIDFilter(req.PartyId))
		}
	}

	// number of retries to -1 to have pretty much unlimited retries
	ch, bCh := e.eventService.ObserveEvents(ctx, e.Config.StreamRetries, types, int(req.BatchSize), filters...)
	defer close(bCh)

	if req.BatchSize > 0 {
		err := e.observeEventsWithAck(ctx, stream, req.BatchSize, ch, bCh)
		return err

	}
	err = e.observeEvents(ctx, stream, ch)
	return err
}

func (e *eventObserver) observeEvents(
	ctx context.Context,
	stream protoapi.TradingService_ObserveEventBusServer,
	ch <-chan []*eventspb.BusEvent,
) error {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			resp := &protoapi.ObserveEventBusResponse{
				Events: data,
			}
			if err := stream.Send(resp); err != nil {
				e.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}
	}
}

func (e *eventObserver) recvEventRequest(
	stream protoapi.TradingService_ObserveEventBusServer,
) (*protoapi.ObserveEventBusRequest, error) {
	readCtx, cfunc := context.WithTimeout(stream.Context(), 5*time.Second)
	oebCh := make(chan protoapi.ObserveEventBusRequest)
	var err error
	go func() {
		defer close(oebCh)
		nb := protoapi.ObserveEventBusRequest{}
		if err = stream.RecvMsg(&nb); err != nil {
			cfunc()
			return
		}
		oebCh <- nb
	}()
	select {
	case <-readCtx.Done():
		if err != nil {
			// this means the client disconnected
			return nil, err
		}
		// this mean we timedout
		return nil, readCtx.Err()
	case nb := <-oebCh:
		return &nb, nil
	}
}

func (e *eventObserver) observeEventsWithAck(
	ctx context.Context,
	stream protoapi.TradingService_ObserveEventBusServer,
	batchSize int64,
	ch <-chan []*eventspb.BusEvent,
	bCh chan<- int,
) error {
	for {
		select {
		case data, ok := <-ch:
			if !ok {
				return nil
			}
			resp := &protoapi.ObserveEventBusResponse{
				Events: data,
			}
			if err := stream.Send(resp); err != nil {
				e.log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}

		// now we try to read again the new size / ack
		req, err := e.recvEventRequest(stream)
		if err != nil {
			return err
		}

		if req.BatchSize != batchSize {
			batchSize = req.BatchSize
			bCh <- int(batchSize)
		}
	}
}
