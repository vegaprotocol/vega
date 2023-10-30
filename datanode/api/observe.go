// Copyright (C) 2023 Gobalsky Labs Limited
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as
// published by the Free Software Foundation, either version 3 of the
// License, or (at your option) any later version.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program.  If not, see <http://www.gnu.org/licenses/>.

package api

import (
	"context"
	"time"

	"code.vegaprotocol.io/vega/datanode/metrics"
	"code.vegaprotocol.io/vega/logging"
)

func observe[T any](ctx context.Context, log *logging.Logger, eventType string, eventsInChan <-chan []T,
	ref any, send func(T) error,
) error {
	defer metrics.StartActiveSubscriptionCountGRPC(eventType)()

	publishedEventStatTicker := time.NewTicker(time.Second)
	defer publishedEventStatTicker.Stop()

	var (
		publishedEvents int64
		err             error
	)
	for {
		select {
		case <-publishedEventStatTicker.C:
			metrics.PublishedEventsAdd(eventType, float64(publishedEvents))
			publishedEvents = 0
		case events, ok := <-eventsInChan:
			if !ok {
				err = ErrChannelClosed
				log.Errorf("subscriber to %s, reference %v, error: %v", eventType, ref, err)
				return formatE(ErrStreamInternal, err)
			}
			for _, event := range events {
				if err = send(event); err != nil {
					log.Errorf("rpc stream error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
					return formatE(ErrStreamInternal, err)
				}
				publishedEvents++
			}
		case <-ctx.Done():
			err = ctx.Err()
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream ctx error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return formatE(ErrStreamInternal, err)
		}

		if eventsInChan == nil {
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream closed, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return formatE(ErrStreamClosed)
		}
	}
}

func observeBatch[T any](ctx context.Context, log *logging.Logger, eventType string,
	eventsInChan <-chan []T, ref any,
	send func([]T) error,
) error {
	defer metrics.StartActiveSubscriptionCountGRPC(eventType)()

	publishedEventStatTicker := time.NewTicker(time.Second)
	defer publishedEventStatTicker.Stop()

	var (
		publishedEvents int64
		err             error
	)
	for {
		select {
		case <-publishedEventStatTicker.C:
			metrics.PublishedEventsAdd(eventType, float64(publishedEvents))
			publishedEvents = 0
		case events, ok := <-eventsInChan:
			if !ok {
				err = ErrChannelClosed
				log.Errorf("subscriber to %s, reference %v, error: %v", eventType, ref, err)
				return formatE(ErrStreamInternal, err)
			}
			err = send(events)
			if err != nil {
				log.Errorf("rpc stream error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
				return formatE(ErrStreamInternal, err)
			}
			publishedEvents = publishedEvents + int64(len(events))
		case <-ctx.Done():
			err = ctx.Err()
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream ctx error, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return formatE(ErrStreamInternal, err)
		}

		if eventsInChan == nil {
			if log.GetLevel() == logging.DebugLevel {
				log.Debugf("rpc stream closed, subscriber to %s, reference %v, error: %v", eventType, ref, err)
			}
			return formatE(ErrStreamClosed)
		}
	}
}
