// Copyright (c) 2022 Gobalsky Labs Limited
//
// Use of this software is governed by the Business Source License included
// in the LICENSE.DATANODE file and at https://www.mariadb.com/bsl11.
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

	"code.vegaprotocol.io/vega/libs/subscribers"

	"code.vegaprotocol.io/vega/core/events"
	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
	protoapi "code.vegaprotocol.io/vega/protos/vega/api/v1"
	eventspb "code.vegaprotocol.io/vega/protos/vega/events/v1"
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
	return observeEventBus(e.log, e.Config, coreServiceEventBusServer{stream}, e.eventService)
}

func observeEventBus(log *logging.Logger, config Config, eventBusServer eventBusServer, eventService EventService) error {
	defer metrics.StartActiveEventBusConnection()()

	ctx, cfunc := context.WithCancel(eventBusServer.Context())
	defer cfunc()

	// now we start listening for a few seconds in order to get at least the very first message
	// this will be blocking until the connection by the client is closed
	// and we will not start processing any events until we receive the original request
	// indicating filters and batch size.
	req, err := recvEventRequest(ctx, defaultReqTimeout, eventBusServer)
	if err != nil {
		log.Error("Error receiving event request", logging.Error(err))
		// client exited, nothing to do
		return nil //nolint:nilerr
	}

	// now we will aggregate filter out of the initial request
	types, err := events.ProtoToInternal(req.Type...)
	if err != nil {
		return formatE(ErrMalformedRequest, err)
	}

	metrics.StartEventBusActiveSubscriptionCount(types)
	defer metrics.StopEventBusActiveSubscriptionCount(types)

	var filters []subscribers.EventFilter
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
		return observeEventsWithAck(ctx, log, eventBusServer, req.BatchSize, ch, bCh)
	}

	return observeEvents(ctx, log, eventBusServer, ch)
}

func observeEvents(
	ctx context.Context,
	log *logging.Logger,
	stream eventBusServer,
	ch <-chan []*eventspb.BusEvent,
) error {
	sentEventStatTicker := time.NewTicker(time.Second)
	defer sentEventStatTicker.Stop()
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
				return formatE(ErrStreamInternal, err)
			}
			publishedEvents.updateStats(data)
		case <-ctx.Done():
			return formatE(ErrStreamInternal, ctx.Err())
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
	defer sentEventStatTicker.Stop()
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
				return formatE(ErrStreamInternal, err)
			}
			publishedEvents.updateStats(data)
		case <-ctx.Done():
			return formatE(ErrStreamInternal, ctx.Err())
		}

		// now we try to read again the new size / ack
		req, err := recvEventRequest(ctx, defaultReqTimeout, stream)
		if err != nil {
			return formatE(ErrStreamInternal, err)
		}

		if req.BatchSize != batchSize {
			batchSize = req.BatchSize
			bCh <- int(batchSize)
		}
	}
}

const defaultReqTimeout = time.Second * 5

func recvEventRequest(ctx context.Context, timeout time.Duration, stream eventBusServer) (*protoapi.ObserveEventBusRequest, error) {
	type resp struct {
		nb  *protoapi.ObserveEventBusRequest
		err error
	}

	oebCh := make(chan resp, 1)

	go func() {
		defer close(oebCh)

		nb := &protoapi.ObserveEventBusRequest{}
		err := stream.RecvMsg(nb)

		oebCh <- resp{nb: nb, err: err}
	}()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	select {
	case rsp := <-oebCh:
		return rsp.nb, rsp.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// this needs to be greater than the highest eventspb.BusEvent event type.
const maxEventTypeOrdinal = 299

type eventStats struct {
	eventCount [maxEventTypeOrdinal + 1]int
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
