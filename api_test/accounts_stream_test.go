package api_test

import (
	"context"
	"io"
	"testing"

	"code.vegaprotocol.io/vega/events"
	pb "code.vegaprotocol.io/vega/proto"
	apipb "code.vegaprotocol.io/vega/proto/api"
	eventspb "code.vegaprotocol.io/vega/proto/events/v1"
	"github.com/stretchr/testify/require"
)

func TestStreamAccountEvents(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	ebc, err := client.ObserveEventBus(ctx)
	require.NoError(t, err)

	done := make(chan struct{})
	// all events will be aggregated here
	evts := []*eventspb.BusEvent{}
	go func() {
		for {
			resp, err := ebc.Recv()
			if err == io.EOF {
				close(done)
				return
			}
			if err != nil {
				t.Errorf("Failed to read from stream: %v\n", err)
				return
			}
			evts = append(evts, resp.Events...)
			if len(evts) > 0 {
				close(done)
				return
			}
		}
	}()

	msg := &apipb.ObserveEventBusRequest{
		Type: []eventspb.BusEventType{
			eventspb.BusEventType_BUS_EVENT_TYPE_ACCOUNT,
		},
		// BatchSize: 10,
	}
	// keep flushing stream
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-done:
				ebc.CloseSend()
				return
			default:
				require.NoError(t, ebc.Send(msg))
			}
		}
	}()
	// send the events
	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		acc := be.GetAccount()
		e := events.NewAccountEvent(ctx, pb.Account{
			Id:       acc.Id,
			Owner:    acc.Owner,
			Balance:  acc.Balance,
			Asset:    acc.Asset,
			MarketId: acc.MarketId,
			Type:     acc.Type,
		})
		return e, nil
	})
	<-done
	require.NotEmpty(t, evts)
}
