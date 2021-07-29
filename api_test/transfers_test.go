package api_test

import (
	"context"
	"io"
	"testing"

	"code.vegaprotocol.io/data-node/events"
	apipb "code.vegaprotocol.io/protos/data-node/api/v1"
	pb "code.vegaprotocol.io/protos/vega"
	eventspb "code.vegaprotocol.io/protos/vega/events/v1"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestObserveTransferResponses(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimout)
	defer cancel()

	conn, broker := NewTestServer(t, ctx, true)

	client := apipb.NewTradingDataServiceClient(conn)
	require.NotNil(t, client)

	// we need to subscribe to the stream prior to publishing the events
	stream, err := client.TransferResponsesSubscribe(ctx, &apipb.TransferResponsesSubscribeRequest{})
	assert.NoError(t, err)

	PublishEvents(t, ctx, broker, func(be *eventspb.BusEvent) (events.Event, error) {
		tr := be.GetTransferResponses()
		require.NotNil(t, tr)
		var responses []*pb.TransferResponse
		for _, resp := range tr.Responses {
			responses = append(responses, &pb.TransferResponse{
				Transfers: resp.Transfers,
				Balances:  resp.Balances,
			})
		}
		e := events.NewTransferResponse(ctx, TransferResponsesFromProto(responses))
		return e, nil
	}, "transfer-responses-events.golden")

	// we only receive one response from the stream and assert it
	var resp *apipb.TransferResponsesSubscribeResponse
	done := make(chan struct{})

	go func() {
		for {
			select {
			case <-ctx.Done():
				close(done)
				return
			default:
				resp, err = stream.Recv()
				if err == io.EOF {
					close(done)
					return
				}
				require.NoError(t, err)
				close(done)
				return
			}
		}
	}()
	<-done

	require.NotNil(t, resp)
	require.Equal(t, "076BB86A5AA41E3E*6d9d35f657589e40ddfb448b7ad4a7463b66efb307527fedd2aa7df1bbd5ea616", resp.Response.Transfers[0].FromAccount)
	require.Equal(t, "076BB86A5AA41E3E0f3d86044f8e7efff27131227235fb6db82574e24f788c30723d67f888b51d616d9d35f657589e40ddfb448b7ad4a7463b66efb307527fedd2aa7df1bbd5ea613", resp.Response.Transfers[0].ToAccount)
	require.Equal(t, uint64(10412267), resp.Response.Transfers[0].Amount)
	require.Equal(t, "TRANSFER_TYPE_LIQUIDITY_FEE_DISTRIBUTE", resp.Response.Transfers[0].Reference)
	require.Equal(t, "settlement", resp.Response.Transfers[0].Type)
	require.Equal(t, int64(1622563663355188728), resp.Response.Transfers[0].Timestamp)

	require.Equal(t, "076BB86A5AA41E3E0f3d86044f8e7efff27131227235fb6db82574e24f788c30723d67f888b51d616d9d35f657589e40ddfb448b7ad4a7463b66efb307527fedd2aa7df1bbd5ea613", resp.Response.Balances[0].Account.Id)
}
