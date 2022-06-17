package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"code.vegaprotocol.io/data-node/subscribers"
	protoapi "code.vegaprotocol.io/protos/vega/api/v1"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"
	"code.vegaprotocol.io/vega/events"
	"google.golang.org/grpc/codes"
)

type eventBusServer interface {
	RecvMsg(m interface{}) error
	Context() context.Context
	Send(data []*eventspb.BusEvent) error
}

type eventObserver struct {
	log          *logging.Logger
	eventService EventService
	Config       Config
}

type coreServiceEventBusServer struct {
	stream protoapi.CoreService_ObserveEventBusServer
}

func (t coreServiceEventBusServer) RecvMsg(m interface{}) error {
	return t.stream.RecvMsg(m)
}

func (t coreServiceEventBusServer) Context() context.Context {
	return t.stream.Context()
}

func (t coreServiceEventBusServer) Send(data []*eventspb.BusEvent) error {
	resp := &protoapi.ObserveEventBusResponse{
		Events: data,
	}
	return t.stream.Send(resp)
}

func (e *eventObserver) ObserveEventBus(
	stream protoapi.CoreService_ObserveEventBusServer,
) error {
	server := coreServiceEventBusServer{stream}
	eventService := e.eventService

	return observeEventBus(e.log, e.Config, server, eventService)
}

func observeEventBus(log *logging.Logger, config Config, eventBusServer eventBusServer, eventService EventService) error {
	defer metrics.StartActiveSubscriptionCountGRPC("EventBus")()

	ctx, cfunc := context.WithCancel(eventBusServer.Context())
	defer cfunc()

	// now we start listening for a few seconds in order to get at least the very first message
	// this will be blocking until the connection by the client is closed
	// and we will not start processing any events until we receive the original request
	// indicating filters and batch size.
	req, err := recvEventRequest(eventBusServer)
	if err != nil {
		// client exited, nothing to do
		return nil
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
	ch, bCh := eventService.ObserveEvents(ctx, config.StreamRetries, types, int(req.BatchSize), filters...)
	defer close(bCh)

	if req.BatchSize > 0 {
		err := observeEventsWithAck(ctx, log, eventBusServer, req.BatchSize, ch, bCh)
		return err

	}
	err = observeEvents(ctx, log, eventBusServer, ch)
	return err
}

func observeEvents(
	ctx context.Context,
	log *logging.Logger,
	stream eventBusServer,
	ch <-chan []*eventspb.BusEvent,
) error {
	sentEventStatTicker := time.NewTicker(time.Second)
	var sentEvents int64

	for {
		select {
		case <-sentEventStatTicker.C:
			metrics.PublishedEventsAdd("EventBus", float64(sentEvents))
			sentEvents = 0
		case data, ok := <-ch:
			if !ok {
				return nil
			}

			if err := stream.Send(data); err != nil {
				log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
			sentEvents = sentEvents + int64(len(data))
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}
	}
}

func observeEventsWithAck(
	ctx context.Context,
	log *logging.Logger,
	stream eventBusServer,
	batchSize int64,
	ch <-chan []*eventspb.BusEvent,
	bCh chan<- int,
) error {
	sentEventStatTicker := time.NewTicker(time.Second)
	var sentEvents int64

	for {
		select {
		case <-sentEventStatTicker.C:
			metrics.PublishedEventsAdd("EventBus", float64(sentEvents))
			sentEvents = 0
		case data, ok := <-ch:
			if !ok {
				return nil
			}

			if err := stream.Send(data); err != nil {
				log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
			sentEvents = sentEvents + int64(len(data))
		case <-ctx.Done():
			return apiError(codes.Internal, ErrStreamInternal, ctx.Err())
		}

		// now we try to read again the new size / ack
		req, err := recvEventRequest(stream)
		if err != nil {
			return err
		}

		if req.BatchSize != batchSize {
			batchSize = req.BatchSize
			bCh <- int(batchSize)
		}
	}
}

func recvEventRequest(
	stream eventBusServer,
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
