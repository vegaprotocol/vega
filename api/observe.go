package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/data-node/logging"
	"code.vegaprotocol.io/data-node/metrics"
	"google.golang.org/grpc/codes"
)

func observe[T any](ctx context.Context, log *logging.Logger, eventType string, eventsInChan <-chan []T,
	ref any, send func(T) error) error {
	defer metrics.StartActiveSubscriptionCountGRPC(eventType)()

	publishedEventStatTicker := time.NewTicker(time.Second)
	var publishedEvents int64

	var err error
	for {
		select {
		case <-publishedEventStatTicker.C:
			metrics.PublishedEventsAdd(eventType, float64(publishedEvents))
			publishedEvents = 0
		case events, ok := <-eventsInChan:
			if !ok {
				err = ErrChannelClosed
				log.Errorf("subscriber to %s, reference %v, error: %v", eventType, ref, err)
				return apiError(codes.Internal, err)
			}
			for _, event := range events {
				if err = send(event); err != nil {
					log.Errorf("rpc stream error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
					return apiError(codes.Internal, ErrStreamInternal, err)
				}
				publishedEvents++
			}
		case <-ctx.Done():
			err = ctx.Err()
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream ctx error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		}

		if eventsInChan == nil {
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream closed, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}

func observeBatch[T any](ctx context.Context, log *logging.Logger, eventType string,
	eventsInChan <-chan []T, ref any,
	send func([]T) error) error {
	defer metrics.StartActiveSubscriptionCountGRPC(eventType)()

	publishedEventStatTicker := time.NewTicker(time.Second)
	var publishedEvents int64
	var err error
	for {
		select {
		case <-publishedEventStatTicker.C:
			metrics.PublishedEventsAdd(eventType, float64(publishedEvents))
			publishedEvents = 0
		case events, ok := <-eventsInChan:
			if !ok {
				err = ErrChannelClosed
				log.Errorf("subscriber to %s, reference %v, error: %v", eventType, ref, err)
				return apiError(codes.Internal, err)
			}
			err = send(events)
			if err != nil {
				log.Errorf("rpc stream error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
				return apiError(codes.Internal, ErrStreamInternal, err)
			}
			publishedEvents = publishedEvents + int64(len(events))
		case <-ctx.Done():
			err = ctx.Err()
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream ctx error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return apiError(codes.Internal, ErrStreamInternal, err)
		}

		if eventsInChan == nil {
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream closed, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return apiError(codes.Internal, ErrStreamClosed)
		}
	}
}
