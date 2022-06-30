// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE file and at https://www.mariadb.com/bsl11.
//
// Change Date: 18 months from the later of the date of the first publicly
// available Distribution of this version of the repository, and 25 June 2022.
//
// On the date above, in accordance with the Business Source License, use
// of this software will be governed by version 3 or later of the GNU General
// Public License.

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
	defer metrics.StartActiveEventBusConnection()()

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
	metrics.StartEventBusActiveSubscriptionCount(types)
	defer metrics.StopEventBusActiveSubscriptionCount(types)

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
	publishedEvents := eventStats{}

	for {
		select {
		case <-sentEventStatTicker.C:
			publishedEvents.publishStats()
			publishedEvents = eventStats{}
		case data, ok := <-ch:
			if !ok {
				return nil
			}

			if err := stream.Send(data); err != nil {
				log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
			publishedEvents.updateStats(data)
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
	publishedEvents := eventStats{}

	for {
		select {
		case <-sentEventStatTicker.C:
			publishedEvents.publishStats()
			publishedEvents = eventStats{}
		case data, ok := <-ch:
			if !ok {
				return nil
			}

			if err := stream.Send(data); err != nil {
				log.Error("Error sending event on stream", logging.Error(err))
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
			publishedEvents.updateStats(data)
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

// this needs to be greater than the highest eventspb.BusEvent event type
const maxEventTypeOrdinal = 299

type eventStats struct {
	eventCount    [maxEventTypeOrdinal + 1]int
	ignoredEvents bool
}

func (s *eventStats) updateStats(events []*eventspb.BusEvent) {
	for _, event := range events {
		eventType := event.Type
		if int(eventType) > maxEventTypeOrdinal {
			eventType = maxEventTypeOrdinal
		}
		s.eventCount[eventType] = s.eventCount[eventType] + 1
	}
}

func (s eventStats) publishStats() {
	for idx, count := range s.eventCount {
		if count > 0 {
			if idx == maxEventTypeOrdinal {
				metrics.EventBusPublishedEventsAdd("Unknown", float64(count))
			}

			eventName := eventspb.BusEventType_name[int32(idx)]
			metrics.EventBusPublishedEventsAdd(eventName, float64(count))
		}
	}
}
